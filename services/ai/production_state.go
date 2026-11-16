package main

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"crypto/sha256"
	_ "embed"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"syscall"

	"github.com/project-horizon/horizon-core/services/ai/knowledge"
	"github.com/project-horizon/horizon-core/services/ai/runtime"
)

// Generated ONLY by scripts/prepare_native_continuation.py from the pinned
// archive. These are the original checkpoint/journal bytes, not a new Brain.
//
//go:embed canonical-ae1.tar.gz
var canonicalAE1Bundle []byte

const (
	ae1ArchiveSHA     = "7452a46b6231b5af409848bf928b1fd486fbe8e563ee44baeffbfefeeaf0c4d3"
	ae1CheckpointSHA  = "e73f8d34e80957dd442b1b5be81811270989e30ee50d03c783fab4dfeb137537"
	ae1JournalSHA     = "ebe2ff1ed74c3adf2c0176b2aa5ab8ead09c8fbdf10c07243e724de9b8354ede"
	ae1CheckpointName = "brain_memory.json.checkpoint.json"
	ae1JournalName    = "horizon_events.jsonl"
	ae1OriginName     = "ae1-origin.json"
)

type productionPaths struct {
	Directory, Memory, Journal, Checkpoint string
}

func directoryProductionPaths(dir string) productionPaths {
	return productionPaths{dir, filepath.Join(dir, "brain_memory.json"), filepath.Join(dir, ae1JournalName), filepath.Join(dir, ae1CheckpointName)}
}

// Explicit individual paths retain their existing deployment contract. New
// deployments use a single isolated state directory; ambiguous combinations
// are refused rather than pairing a checkpoint with another lineage's WAL.
func resolveProductionPaths(getenv func(string) string, cwd string) (productionPaths, error) {
	resolve := func(p string) string {
		if filepath.IsAbs(p) {
			return filepath.Clean(p)
		}
		return filepath.Join(cwd, p)
	}
	dir := strings.TrimSpace(getenv("HORIZON_STATE_DIR"))
	memory := strings.TrimSpace(getenv("HORIZON_BRAIN_MEMORY_PATH"))
	journal := strings.TrimSpace(getenv("HORIZON_EVENT_LOG_PATH"))
	checkpoint := strings.TrimSpace(getenv("HORIZON_CHECKPOINT_PATH"))
	individual := memory != "" || journal != "" || checkpoint != ""
	if dir != "" && individual {
		return productionPaths{}, errors.New("HORIZON_STATE_DIR cannot be combined with individual Brain paths")
	}
	if dir != "" {
		return directoryProductionPaths(resolve(dir)), nil
	}
	if memory == "" {
		memory = "brain_memory.json"
	}
	if journal == "" {
		journal = ae1JournalName
	}
	if checkpoint == "" {
		checkpoint = memory + ".checkpoint.json"
	}
	legacy := productionPaths{Memory: resolve(memory), Journal: resolve(journal), Checkpoint: resolve(checkpoint)}
	for _, path := range []string{legacy.Memory, legacy.Journal, legacy.Checkpoint} {
		exists, err := startupFileExists(path)
		if err != nil {
			return productionPaths{}, err
		}
		if exists {
			return legacy, nil
		}
	}
	if individual {
		return productionPaths{}, errors.New("empty individual Brain paths: use HORIZON_STATE_DIR for canonical A–E1 initialization")
	}
	return directoryProductionPaths(resolve("horizon-state")), nil
}

// The sibling lock survives atomic directory publication and is held for the
// process lifetime. Journal ownership remains enforced by OpenEventLog too.
func lockProductionDirectory(dir string) (*os.File, error) {
	if filepath.Dir(dir) == dir {
		return nil, errors.New("state directory cannot be a filesystem root")
	}
	if err := os.MkdirAll(filepath.Dir(dir), 0700); err != nil {
		return nil, err
	}
	lock, err := os.OpenFile(filepath.Join(filepath.Dir(dir), "."+filepath.Base(dir)+".startup.lock"), os.O_CREATE|os.O_RDWR, 0600)
	if err != nil {
		return nil, err
	}
	if err = syscall.Flock(int(lock.Fd()), syscall.LOCK_EX|syscall.LOCK_NB); err != nil {
		lock.Close()
		return nil, fmt.Errorf("state directory already in use: %w", err)
	}
	return lock, nil
}

type productionState struct {
	Paths productionPaths
	Log   *runtime.EventLog
	lock  *os.File
}

func (s *productionState) Close() {
	if s.Log != nil {
		_ = s.Log.Close()
	}
	if s.lock != nil {
		_ = s.lock.Close()
	}
}

func openProductionState(paths productionPaths) (_ *productionState, err error) {
	s := &productionState{Paths: paths}
	defer func() {
		if err != nil {
			s.Close()
		}
	}()
	if paths.Directory != "" {
		s.lock, err = lockProductionDirectory(paths.Directory)
		if err != nil {
			return nil, err
		}
		info, statErr := os.Lstat(paths.Directory)
		if errors.Is(statErr, os.ErrNotExist) {
			err = installCanonicalAE1(paths.Directory)
		} else if statErr != nil {
			err = statErr
		} else if !info.IsDir() {
			err = errors.New("state directory must be a real directory, not a symlink")
		} else {
			entries, readErr := os.ReadDir(paths.Directory)
			if readErr != nil {
				err = readErr
			} else if len(entries) == 0 {
				err = installCanonicalAE1(paths.Directory)
			}
		}
		if err != nil {
			return nil, err
		}
		found := false
		for _, p := range []string{paths.Checkpoint, paths.Memory, paths.Journal} {
			exists, e := startupFileExists(p)
			if e != nil {
				return nil, e
			}
			found = found || exists
		}
		if !found {
			return nil, errors.New("nonempty state directory has no Brain; refusing to replace its contents")
		}
		if err = verifyAE1Origin(paths); err != nil {
			return nil, err
		}
	}
	// Opening a missing journal alongside a checkpoint would mutate a bad
	// pair before recovery can reject it. Refuse it first.
	cp, err := startupFileExists(paths.Checkpoint)
	if err != nil {
		return nil, err
	}
	journal, err := startupFileExists(paths.Journal)
	if err != nil {
		return nil, err
	}
	if cp && !journal {
		return nil, errors.New("checkpoint has no matching journal; refusing startup")
	}
	if !journal {
		memory, e := startupFileExists(paths.Memory)
		if e != nil {
			return nil, e
		}
		if memory {
			// A legacy standalone import may create its first journal only
			// after the existing loader has accepted it. A complete serving
			// checkpoint always requires its original matching journal.
			raw, e := os.ReadFile(paths.Memory)
			if e != nil {
				return nil, e
			}
			var envelope struct {
				Brain json.RawMessage `json:"brain"`
			}
			if e = json.Unmarshal(raw, &envelope); e != nil {
				return nil, fmt.Errorf("invalid standalone memory; refusing journal creation: %w", e)
			}
			if len(envelope.Brain) > 0 {
				return nil, errors.New("complete serving memory has no matching journal; refusing startup")
			}
			if e = knowledge.NewBrain().LoadStandalone(paths.Memory); e != nil {
				return nil, fmt.Errorf("invalid standalone memory; refusing journal creation: %w", e)
			}
		}
	}
	s.Log, err = runtime.OpenEventLog(paths.Journal)
	if err != nil {
		return nil, err
	}
	return s, nil
}

type ae1Origin struct {
	Archive      string `json:"archive_sha256"`
	Checkpoint   string `json:"checkpoint_sha256"`
	Journal      string `json:"journal_sha256"`
	JournalBytes int64  `json:"journal_bytes"`
	Identity     string `json:"brain_identity"`
	Sequence     uint64 `json:"final_sequence"`
}

// Provenance is immutable, while later checkpoints and journal suffixes evolve.
// Authenticate the original byte prefix, never require the full evolving WAL
// to equal its initial hash and never append another Brain's genesis to it.
func verifyAE1Origin(paths productionPaths) error {
	raw, err := os.ReadFile(filepath.Join(paths.Directory, ae1OriginName))
	if errors.Is(err, os.ErrNotExist) {
		return nil
	} // Existing non-A–E1 deployment.
	if err != nil {
		return err
	}
	var origin ae1Origin
	if err = json.Unmarshal(raw, &origin); err != nil {
		return err
	}
	if origin.Archive != ae1ArchiveSHA || origin.Checkpoint != ae1CheckpointSHA || origin.Journal != ae1JournalSHA || origin.Identity != runtime.BrainIdentity || origin.Sequence != 75331 || origin.JournalBytes <= 0 {
		return errors.New("invalid canonical A–E1 provenance")
	}
	checkpoint, err := startupFileExists(paths.Checkpoint)
	if err != nil {
		return err
	}
	memory, err := startupFileExists(paths.Memory)
	if err != nil {
		return err
	}
	if !checkpoint && !memory {
		return errors.New("A–E1 checkpoint and serving memory are missing; refusing genesis fallback")
	}
	anchor := paths.Checkpoint
	if !checkpoint {
		anchor = paths.Memory
	}
	state, err := os.ReadFile(anchor)
	if err != nil {
		return err
	}
	var header struct {
		Sequence uint64 `json:"sequence"`
		Identity string `json:"brain_identity"`
	}
	if err = json.Unmarshal(state, &header); err != nil {
		return err
	}
	if header.Sequence < 75331 || header.Identity != runtime.BrainIdentity {
		return errors.New("checkpoint does not continue A–E1 lineage")
	}
	journal, err := os.Open(paths.Journal)
	if err != nil {
		return err
	}
	defer journal.Close()
	hash := sha256.New()
	if _, err = io.CopyN(hash, journal, origin.JournalBytes); err != nil {
		return fmt.Errorf("A–E1 journal prefix incomplete: %w", err)
	}
	if hex.EncodeToString(hash.Sum(nil)) != ae1JournalSHA {
		return errors.New("A–E1 journal lineage prefix mismatch")
	}
	return nil
}

func syncStateDirectory(path string) error {
	dir, err := os.Open(path)
	if err != nil {
		return err
	}
	defer dir.Close()
	return dir.Sync()
}

// Called only under the sibling startup lock. Publish a complete verified
// directory by one rename; a crash/validation failure leaves the old directory
// untouched. Never install over any nonempty directory.
func installCanonicalAE1(destination string) error {
	if entries, err := os.ReadDir(destination); err == nil {
		if len(entries) != 0 {
			return errors.New("A–E1 destination is not empty; existing Brain preserved")
		}
	} else if !errors.Is(err, os.ErrNotExist) {
		return err
	}
	stage, err := os.MkdirTemp(filepath.Dir(destination), ".ae1-staging-")
	if err != nil {
		return err
	}
	defer os.RemoveAll(stage)
	compressed, err := gzip.NewReader(bytes.NewReader(canonicalAE1Bundle))
	if err != nil {
		return err
	}
	defer compressed.Close()
	archive := tar.NewReader(compressed)
	seen := map[string]bool{}
	for {
		entry, e := archive.Next()
		if e == io.EOF {
			break
		}
		if e != nil {
			return e
		}
		if seen[entry.Name] || entry.Typeflag != tar.TypeReg || entry.Size > 100<<20 || (entry.Name != ae1CheckpointName && entry.Name != ae1JournalName && entry.Name != ae1OriginName) {
			return errors.New("invalid canonical A–E1 bundle member")
		}
		seen[entry.Name] = true
		file, e := os.OpenFile(filepath.Join(stage, entry.Name), os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
		if e != nil {
			return e
		}
		hash := sha256.New()
		_, e = io.Copy(io.MultiWriter(file, hash), archive)
		if e == nil {
			e = file.Sync()
		}
		closeErr := file.Close()
		if e != nil {
			return e
		}
		if closeErr != nil {
			return closeErr
		}
		digest := hex.EncodeToString(hash.Sum(nil))
		if entry.Name == ae1CheckpointName && digest != ae1CheckpointSHA || entry.Name == ae1JournalName && digest != ae1JournalSHA {
			return errors.New("canonical A–E1 payload checksum mismatch")
		}
	}
	// Consume gzip's trailer too, verifying its CRC before publishing.
	if _, err = io.Copy(io.Discard, compressed); err != nil {
		return err
	}
	if len(seen) != 3 {
		return errors.New("incomplete canonical A–E1 bundle")
	}
	paths := directoryProductionPaths(stage)
	if err = verifyAE1Origin(paths); err != nil {
		return err
	}
	journal, err := runtime.OpenEventLog(paths.Journal)
	if err != nil {
		return err
	}
	defer journal.Close()
	horizon, bootstrap, err := initializeProductionHorizon(journal, paths.Journal, paths.Memory, paths.Checkpoint)
	if err != nil {
		return err
	}
	defer horizon.Supervisor().Close()
	if bootstrap {
		return errors.New("restored A–E1 unexpectedly requested bootstrap")
	}
	inspection, err := horizon.InspectActiveBrain()
	if err != nil {
		return err
	}
	units, synapses, patterns := inspection.Graph.Counts()
	if inspection.Revision != 75331 || units != 57 || synapses != 94 || patterns != 24 {
		return errors.New("restored A–E1 state disagrees with canonical inventory")
	}
	if err = horizon.Supervisor().Close(); err != nil {
		return err
	}
	if err = journal.Close(); err != nil {
		return err
	}
	if err = syncStateDirectory(stage); err != nil {
		return err
	}
	if err = os.Rename(stage, destination); err != nil {
		return err
	}
	return syncStateDirectory(filepath.Dir(destination))
}

func prepareCanonicalAE1(destination string) error {
	destination, err := filepath.Abs(destination)
	if err != nil {
		return err
	}
	lock, err := lockProductionDirectory(destination)
	if err != nil {
		return err
	}
	defer lock.Close()
	if _, err = os.Lstat(destination); !errors.Is(err, os.ErrNotExist) {
		return errors.New("activation requires a new directory; preserve the existing Brain and backup")
	}
	return installCanonicalAE1(destination)
}
