package terminal

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"

	profile "github.com/project-horizon/horizon-core/models/grounded-dialogue-v1"
	"github.com/project-horizon/horizon-core/services/ai/runtime"
)

// BrainInfo describes the recovered, read-only session. SourceFingerprint names
// the files loaded at startup; CanonicalStateHash comes from the live Brain.
// Matching a published artifact is provenance, not a claim of general ability.
type BrainInfo struct {
	Directory          string
	Identity           string
	Sequence           uint64
	Composition        string
	CanonicalStateHash string
	SourceFingerprint  string
	Units              int
	InputForms         int
	OutputForms        int
	MatchesBundle      bool
	BundleTrainedOn    string
}

// BundledProfileDirectory selects the code's published training artifacts in a
// separate profile. Different training artifacts cannot overwrite old learning.
func BundledProfileDirectory(root string) (string, error) {
	release, err := profile.BundledRelease()
	if err != nil {
		return "", err
	}
	return filepath.Join(root, "trained", release.Files.Fingerprint()), nil
}

func fileDigest(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()
	digest := sha256.New()
	if _, err := io.Copy(digest, f); err != nil {
		return "", err
	}
	return hex.EncodeToString(digest.Sum(nil)), nil
}

func (s *Session) inspectBrain(directory string) (BrainInfo, error) {
	info := BrainInfo{Directory: directory, InputForms: len(s.codec.Input), OutputForms: len(s.codec.Output)}
	var files profile.ProfileFiles
	for name, target := range map[string]*string{"language-brain.json": &files.Checkpoint, "events.jsonl": &files.Journal, "codec.json": &files.Codec} {
		var err error
		*target, err = fileDigest(filepath.Join(directory, name))
		if err != nil {
			return info, err
		}
	}
	bundle, err := profile.BundledRelease()
	if err != nil {
		return info, err
	}
	s.release = bundle
	info.SourceFingerprint = files.Fingerprint()
	err = s.engine.Supervisor().With(func(rt *runtime.BrainRuntime) error {
		state, err := rt.MonitorSnapshot()
		if err != nil {
			return err
		}
		composition, err := rt.CompositionStatus()
		if err != nil {
			return err
		}
		if composition == nil {
			return fmt.Errorf("profil tidak mempunyai jalur komunikasi komposisional")
		}
		info.Identity, info.Sequence = state.BrainIdentity, state.StateRevision
		info.CanonicalStateHash, info.Units = state.CanonicalStateHash, state.Counts.NeuralUnits
		info.Composition = composition.Config.Version
		return nil
	})
	info.MatchesBundle = files == bundle.Files && info.Sequence == bundle.Sequence && s.codec.Version == bundle.Communication.Adapter
	if info.MatchesBundle {
		info.BundleTrainedOn = bundle.TrainedOn
	}
	return info, err
}

// BrainInfo returns a value detached from canonical state. Inference, UI
// context and simulated room controls do not change this session's Brain.
func (s *Session) BrainInfo() BrainInfo { return s.info }
