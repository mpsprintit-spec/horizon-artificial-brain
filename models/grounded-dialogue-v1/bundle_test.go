package groundeddialogue

import (
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"testing/fstest"
)

func TestPrepareRefusesExistingAndCorruptedProfiles(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "profile")
	if err := Prepare(dir); err != nil {
		t.Fatal(err)
	}
	checkpoint := filepath.Join(dir, "language-brain.json")
	if err := os.WriteFile(checkpoint, []byte("existing learned state"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := Prepare(dir); !os.IsExist(err) {
		t.Fatalf("existing state was not refused: %v", err)
	}
	data, err := os.ReadFile(checkpoint)
	if err != nil || string(data) != "existing learned state" {
		t.Fatal("existing state was changed")
	}
	broken := fstest.MapFS{}
	if err := fs.WalkDir(assets, ".", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.IsDir() {
			data, e := assets.ReadFile(path)
			if e != nil {
				return e
			}
			broken[path] = &fstest.MapFile{Data: data}
		}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	broken["language-brain.json.gz"].Data[80] ^= 1
	other := filepath.Join(t.TempDir(), "profile")
	if err := prepare(broken, other); err == nil || !strings.Contains(err.Error(), "checksum mismatch") {
		t.Fatalf("corrupted checkpoint accepted: %v", err)
	}
	if _, err := os.Stat(other); !os.IsNotExist(err) {
		t.Fatal("failed verification created state")
	}
}
