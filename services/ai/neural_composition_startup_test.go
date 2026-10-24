package main

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/project-horizon/horizon-core/services/ai/knowledge"
	"github.com/project-horizon/horizon-core/services/ai/runtime"
)

func TestProductionCompositionServingMemoryRecoversWithJournal(t *testing.T) {
	dir := t.TempDir()
	journalPath := filepath.Join(dir, "events.jsonl")
	memory := filepath.Join(dir, "memory.json")
	checkpoint := filepath.Join(dir, "checkpoint.json")
	log, err := runtime.OpenEventLog(journalPath)
	if err != nil {
		t.Fatal(err)
	}
	defer log.Close()
	h, _, err := initializeProductionHorizon(log, journalPath, memory, checkpoint)
	if err != nil {
		t.Fatal(err)
	}
	if err := h.InitializeNeuralComposition(knowledge.DefaultCompositionConfig(), filepath.Join(dir, "rollback.json"), time.Now().UTC()); err != nil {
		t.Fatal(err)
	}
	if err := h.SaveActiveBrain(memory); err != nil {
		t.Fatal(err)
	}
	if err := h.Supervisor().Close(); err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(checkpoint); err != nil {
		t.Fatal(err)
	}
	restarted, _, err := initializeProductionHorizon(log, journalPath, memory, checkpoint)
	if err != nil {
		t.Fatal(err)
	}
	state, err := restarted.CompositionStatus()
	if err != nil || state == nil {
		t.Fatalf("production startup lost complete versioned serving memory: %v", err)
	}
	if _, err := os.Stat(checkpoint); err != nil {
		t.Fatal("startup did not re-anchor recovered serving state")
	}
}
