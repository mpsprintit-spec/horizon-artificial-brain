package runtime

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/project-horizon/horizon-core/services/ai/internal/jsonstream"
)

type checkpointCounter struct {
	w io.Writer
	n int64
}

func (c *checkpointCounter) Write(p []byte) (int, error) {
	n, err := c.w.Write(p)
	c.n += int64(n)
	if err == nil && n != len(p) {
		err = io.ErrShortWrite
	}
	return n, err
}

// Write one complete compact envelope to a private temp file. The Brain is
// streamed ONCE to both the checksum's Brain field and the file's Brain field.
// Its checksum slot is patched before sync/rename; incomplete files never replace
// an existing checkpoint. The v5/v6/v7 logical checksum law is unchanged.
func (r *BrainRuntime) writeCheckpointLocked(path string, snapshot BrainSnapshot) error {
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}
	tmp, err := os.CreateTemp(filepath.Dir(path), ".horizon-checkpoint-*.tmp")
	if err != nil {
		return err
	}
	defer os.Remove(tmp.Name())
	defer tmp.Close()
	buffer := bufio.NewWriterSize(tmp, 64*1024)
	file := &checkpointCounter{w: buffer}
	var checksumOffset int64 = -1
	overrides := map[string]jsonstream.ValueWriter{
		"checksum": func(w io.Writer, _ bool, _ string) error {
			checksumOffset = file.n + 1
			return jsonstream.Value(w, strings.Repeat("0", 64), false, "")
		},
		"brain": func(_ io.Writer, _ bool, _ string) error {
			checksumLaw := checkpointChecksumV7WithBrain
			if snapshot.SchemaVersion >= neuralWrittenBrainSnapshotSchemaVersion { checksumLaw = checkpointChecksumV8WithBrain }
			if snapshot.SchemaVersion == CanonicalPropagationSnapshotSchemaVersion { checksumLaw = checkpointChecksumV9WithBrain }
			checksum, e := checksumLaw(snapshot, func(hash io.Writer, _ bool, _ string) error {
				return r.brain.WriteCanonical(io.MultiWriter(hash, file), false, "")
			})
			if e != nil {
				return e
			}
			snapshot.Checksum = checksum
			return nil
		},
	}
	if err := jsonstream.Object(file, snapshot, false, "", overrides); err != nil {
		return fmt.Errorf("stream complete checkpoint: %w", err)
	}
	if err := buffer.Flush(); err != nil {
		return err
	}
	if checksumOffset < 0 || len(snapshot.Checksum) != 64 {
		return fmt.Errorf("checkpoint checksum was not completed")
	}
	if n, err := tmp.WriteAt([]byte(snapshot.Checksum), checksumOffset); err != nil {
		return err
	} else if n != 64 {
		return io.ErrShortWrite
	}
	if err := tmp.Sync(); err != nil {
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	return os.Rename(tmp.Name(), path)
}
