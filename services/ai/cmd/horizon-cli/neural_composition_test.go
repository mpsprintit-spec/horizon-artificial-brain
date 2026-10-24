package main

import (
	"bytes"
	"context"
	"path/filepath"
	"strings"
	"testing"
)

func TestCLICompositionOptInAndCompleteRestart(t *testing.T) {
	var output bytes.Buffer
	dir := t.TempDir()
	app := newCLI(strings.NewReader(""), &output, filepath.Join(dir, "memory.json"))
	if !app.handleCommand("composition status") || !strings.Contains(output.String(), "nonaktif") {
		t.Fatal("CLI did not expose disabled state")
	}
	if !app.handleCommand("composition initialize " + filepath.Join(dir, "rollback.json")) {
		t.Fatal("CLI did not recognize explicit initialization")
	}
	state, err := app.horizon.CompositionStatus()
	if err != nil || state == nil {
		t.Fatalf("CLI initialization failed: %v: %s", err, output.String())
	}
	if err := app.horizon.SaveActiveBrain(app.memoryPath); err != nil {
		t.Fatal(err)
	}
	restarted := newCLI(strings.NewReader(""), &output, app.memoryPath)
	if err := restarted.horizon.LoadActiveBrain(app.memoryPath); err != nil {
		t.Fatal(err)
	}
	result := restarted.process(context.Background(), "nyalakan televisi")
	if !strings.Contains(result.Answer, "Komposisi:") || !strings.Contains(result.Answer, "belum dieksekusi") {
		t.Fatal("CLI hid the native numerical proposal status")
	}
}
