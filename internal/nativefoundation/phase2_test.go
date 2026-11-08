package nativefoundation

import (
	"os"
	"path/filepath"
	"testing"
)

func TestPhase2PreservesExistingCheckpoint(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "trained-checkpoint.json")
	original := []byte("existing evidence must survive")
	if err := os.WriteFile(path, original, 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := Phase2Experiment(dir, 1709, "relational", 1, 1, 7200000, 1, .1); err == nil {
		t.Fatal("existing experiment accepted")
	}
	got, err := os.ReadFile(path)
	if err != nil || string(got) != string(original) {
		t.Fatal("checkpoint replaced", err)
	}
}
