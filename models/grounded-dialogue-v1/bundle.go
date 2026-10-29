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

//go:embed manifest.json language-brain.json.gz events.jsonl.gz codec.json evaluation.json.gz start.json.gz
var assets embed.FS

// Prepare verifies every embedded asset before creating a new private profile.
// An existing directory (even an empty one) is never replaced or repaired.
func Prepare(directory string) error { return prepare(assets, directory) }

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
	required := map[string]bool{"language-brain.json": true, "events.jsonl": true, "codec.json": true, "evaluation.json": true, "start.json": true}
	verified := map[string][]byte{}
	digest := func(data []byte) string { sum := sha256.Sum256(data); return hex.EncodeToString(sum[:]) }
	for _, entry := range manifest.Files {
		if !fs.ValidPath(entry.Source) || filepath.Base(entry.Source) != entry.Source || !required[entry.Target] || verified[entry.Target] != nil {
			return fmt.Errorf("invalid or repeated dialogue asset")
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
