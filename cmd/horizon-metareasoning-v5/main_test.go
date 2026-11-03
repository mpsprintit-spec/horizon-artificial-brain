package main

import (
	"encoding/json"
	k "github.com/project-horizon/horizon-core/services/ai/knowledge"
	r "github.com/project-horizon/horizon-core/services/ai/runtime"
	"os"
	"path/filepath"
	"testing"
)

func TestWorkbenchRequiresOptInBeforeFilesystemAccess(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "must-not-exist")
	if runArgs([]string{"-state", dir, "-command", "unused.json"}) == nil {
		t.Fatal("default enabled")
	}
	if _, e := os.Stat(dir); !os.IsNotExist(e) {
		t.Fatal("state touched before opt-in")
	}
}
func TestWorkbenchRecoversJournalOnlyInitializationAndIdempotentRetry(t *testing.T) {
	dir := t.TempDir()
	profile := filepath.Join(dir, "profile")
	cmdFile := filepath.Join(dir, "init.json")
	c := k.CognitiveModelConfig{Version: k.CognitiveModelVersion, Inputs: 3, Hidden: 3, Actions: 3, Horizon: 4, Seed: 91, LearningRate: .003, Gamma: 1, ValueWeight: .2, EntropyWeight: .01, AssessmentWeight: .3, WorldWeight: .2, RegulationSteps: 2, RegulationRate: .02}
	b, _ := json.Marshal(r.CognitiveModelCommand{ID: "initialize-experimental", Initialize: &c})
	os.WriteFile(cmdFile, b, 0600)
	args := []string{"-experimental-v5", "-state", profile, "-command", cmdFile}
	if e := runArgs(args); e != nil {
		t.Fatal(e)
	}
	if e := os.Remove(filepath.Join(profile, "brain.json")); e != nil {
		t.Fatal(e)
	}
	if e := runArgs(args); e != nil {
		t.Fatal(e)
	}
	events, e := r.ReadEventLog(filepath.Join(profile, "events.jsonl"))
	if e != nil || len(events) != 1 {
		t.Fatal("lost acknowledgement duplicated initialization", e, len(events))
	}
	rt := r.NewBrainRuntime(k.NewBrain())
	if e := rt.RestoreCheckpoint(filepath.Join(profile, "brain.json")); e != nil {
		t.Fatal(e)
	}
}
