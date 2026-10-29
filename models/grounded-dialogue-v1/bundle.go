// Package groundeddialogue embeds the opt-in trained profile for offline use.
package groundeddialogue

import (
	"bytes"
	"compress/gzip"
	"crypto/sha256"
	"embed"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
)

//go:embed manifest.json language-brain.json.gz events.jsonl.gz codec.json
var assets embed.FS

// PrepareServingProfile verifies the three serving assets before creating a
// private profile. Evaluation cases and the replay seed are not in the binary.
// An existing directory (even an empty one) is never replaced or repaired.
func PrepareServingProfile(directory string) error { return prepare(assets, directory) }

func prepare(source fs.FS, directory string) error {
	var manifest struct {
		Version string `json:"version"`
		Files   []struct {
			Source         string `json:"source"`
			Target         string `json:"target"`
			Encoding       string `json:"encoding"`
			Bytes          int64  `json:"bytes"`
			SHA256         string `json:"sha256"`
			UnpackedBytes  int64  `json:"unpacked_bytes"`
			UnpackedSHA256 string `json:"unpacked_sha256"`
		} `json:"files"`
	}
	raw, err := fs.ReadFile(source, "manifest.json")
	if err != nil {
		return err
	}
	if err := json.Unmarshal(raw, &manifest); err != nil {
		return err
	}
	if manifest.Version != "grounded-dialogue-bundle-v1" {
		return fmt.Errorf("unsupported dialogue bundle")
	}
	required := map[string]bool{"language-brain.json": true, "events.jsonl": true, "codec.json": true}
	seen := map[string]bool{}
	verified := map[string][]byte{}
	digest := func(data []byte) string { sum := sha256.Sum256(data); return hex.EncodeToString(sum[:]) }
	for _, entry := range manifest.Files {
		if !fs.ValidPath(entry.Source) || filepath.Base(entry.Source) != entry.Source || seen[entry.Target] {
			return fmt.Errorf("invalid or repeated dialogue asset")
		}
		seen[entry.Target] = true
		if !required[entry.Target] {
			if entry.Target == "evaluation.json" || entry.Target == "start.json" {
				continue
			}
			return fmt.Errorf("unknown dialogue asset")
		}
		const budget = 16 << 20
		if entry.Bytes <= 0 || entry.Bytes > budget || entry.UnpackedBytes <= 0 || entry.UnpackedBytes > budget {
			return fmt.Errorf("dialogue asset size exceeds budget")
		}
		packed, err := fs.ReadFile(source, entry.Source)
		if err != nil {
			return err
		}
		if int64(len(packed)) != entry.Bytes || digest(packed) != entry.SHA256 {
			return fmt.Errorf("dialogue checksum mismatch: %s", entry.Source)
		}
		data := packed
		switch entry.Encoding {
		case "identity":
		case "gzip":
			reader, err := gzip.NewReader(bytes.NewReader(packed))
			if err != nil {
				return err
			}
			data, err = io.ReadAll(io.LimitReader(reader, entry.UnpackedBytes+1))
			closeErr := reader.Close()
			if err != nil {
				return err
			}
			if closeErr != nil {
				return closeErr
			}
		default:
			return fmt.Errorf("unsupported dialogue asset encoding")
		}
		if int64(len(data)) != entry.UnpackedBytes || digest(data) != entry.UnpackedSHA256 {
			return fmt.Errorf("unpacked dialogue checksum mismatch: %s", entry.Source)
		}
		verified[entry.Target] = data
	}
	if len(verified) != len(required) {
		return fmt.Errorf("incomplete dialogue bundle")
	}
	if err := os.Mkdir(directory, 0700); err != nil {
		return err
	}
	complete := false
	defer func() {
		if !complete {
			_ = os.RemoveAll(directory)
		}
	}()
	for name, data := range verified {
		if err := os.WriteFile(filepath.Join(directory, name), data, 0600); err != nil {
			return err
		}
	}
	complete = true
	return nil
}
