package main

import (
	"bytes"
	"path/filepath"
	"strings"
	"testing"
)

func runCLI(t *testing.T, input string) (string, string) {
	t.Helper()
	var out bytes.Buffer
	memoryPath := filepath.Join(t.TempDir(), "brain_memory.json")
	app := newCLI(strings.NewReader(input), &out, memoryPath)
	if err := app.startup(); err != nil {
		t.Fatalf("startup failed: %v", err)
	}
	app.run(t.Context())
	return out.String(), memoryPath
}

func TestCLIAcceptsMultipleInputsAndExit(t *testing.T) {
	out, _ := runCLI(t, "api menyebabkan panas\napa api panas?\nexit\n")
	if strings.Count(out, "> ") < 3 {
		t.Fatalf("REPL stopped too early; output: %s", out)
	}
	if !strings.Contains(out, "Shutting down Horizon") {
		t.Fatalf("exit command did not shut down cleanly; output: %s", out)
	}
	if !strings.Contains(out, "Session: 2 inputs") {
		t.Fatalf("session did not process two non-command inputs; output: %s", out)
	}
}

func TestSaveLoadCommands(t *testing.T) {
	out, memoryPath := runCLI(t, "langit biru\nsave\nload\nexit\n")
	if !strings.Contains(out, "Memory saved.") || !strings.Contains(out, "Memory loaded.") {
		t.Fatalf("save/load output missing; output: %s", out)
	}
	if memoryPath == "" {
		t.Fatal("memory path was not returned")
	}
}

func TestDebugModePrintsPipelineDetails(t *testing.T) {
	out, _ := runCLI(t, "debug on\nair minum segar\ndebug off\nexit\n")
	for _, want := range []string{"[debug] Activation:", "[debug] Understanding:", "[debug] Reasoning:", "[debug] Confidence:", "[debug] Context:", "[debug] Hypothesis:", "[debug] WebSearch:", "[debug] Learning:"} {
		if !strings.Contains(out, want) {
			t.Fatalf("debug output missing %q; output: %s", want, out)
		}
	}
	if !strings.Contains(out, "Debug disabled.") {
		t.Fatalf("debug off did not respond; output: %s", out)
	}
}
