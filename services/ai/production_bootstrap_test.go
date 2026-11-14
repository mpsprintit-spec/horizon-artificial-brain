package main

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/project-horizon/horizon-core/services/ai/runtime"
)

// Regresses the production main.go bug where the previous condition
// `if !loadedBrain` tried to append bootstrap to a restored journal.
func TestProductionStartupBootstrapOnlyOnFreshBrain(t *testing.T) {
	dir := t.TempDir()
	journal := filepath.Join(dir, "events.jsonl")
	memory := filepath.Join(dir, "brain_memory.json")
	checkpoint := memory + ".checkpoint.json"

	firstLog := newStartupJournal(t, journal)
	first, shouldBootstrap, err := initializeProductionHorizon(firstLog, journal, memory, checkpoint)
	if err != nil || !shouldBootstrap {
		t.Fatalf("new Horizon bootstrap=%t err=%v", shouldBootstrap, err)
	}
	firstOutput, count, err := finishProductionStartup(first, shouldBootstrap, memory, checkpoint)
	if err != nil {
		t.Fatalf("first bootstrap: %v", err)
	}
	if count == 0 || firstOutput.Sequence == 0 {
		t.Fatalf("bootstrap did not record initial event: count=%d sequence=%d", count, firstOutput.Sequence)
	}
	originalBrain := getActiveStartupState(t, first)
	originalMemory, err := os.ReadFile(memory)
	if err != nil {
		t.Fatal(err)
	}
	beforeEvents, err := runtime.ReadEventLog(journal)
	if err != nil {
		t.Fatal(err)
	}
	if len(beforeEvents) == 0 {
		t.Fatal("bootstrap never entered journal")
	}

	if err := first.Supervisor().Close(); err != nil {
		t.Fatal(err)
	}
	if err := firstLog.Close(); err != nil {
		t.Fatal(err)
	}

	secondLog := newStartupJournal(t, journal)
	second, shouldBootstrap, err := initializeProductionHorizon(secondLog, journal, memory, checkpoint)
	if err != nil || shouldBootstrap {
		t.Fatalf("recovered Horizon bootstrap=%t err=%v", shouldBootstrap, err)
	}
	secondOutput, secondCount, err := finishProductionStartup(second, shouldBootstrap, memory, checkpoint)
	if err != nil {
		t.Fatalf("recovered production startup reissued bootstrap: %v", err)
	}
	if secondCount != 0 || secondOutput.Sequence != firstOutput.Sequence {
		t.Fatalf("unexpected bootstrap or sequence after restart: count=%d sequence=%d", secondCount, secondOutput.Sequence)
	}
	if !bytes.Equal(originalBrain, getActiveStartupState(t, second)) {
		t.Fatal("restart changed recovered neural substrate")
	}
	afterMemory, err := os.ReadFile(memory)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(originalMemory, afterMemory) {
		t.Fatal("resume rewrote canonical brain memory")
	}
	afterEvents, err := runtime.ReadEventLog(journal)
	if err != nil {
		t.Fatal(err)
	}
	if len(afterEvents) != len(beforeEvents) {
		t.Fatal("resume wrote an unexpected journal event")
	}
}

// Recovery must not require writing another memory/checkpoint file. This also
// catches a future caller accidentally issuing bootstrap on an already active
// recovered owner just because deployment snapshot paths are unavailable.
func TestProductionRecoveredStartupNeverWritesSnapshotPaths(t *testing.T) {
	dir := t.TempDir()
	journal := filepath.Join(dir, "events.jsonl")
	log := newStartupJournal(t, journal)
	defer log.Close()
	memory := filepath.Join(dir, "memory.json")
	horizon, fresh, err := initializeProductionHorizon(log, journal, memory, memory+".checkpoint.json")
	if err != nil || !fresh {
		t.Fatalf("foundation: %v fresh=%t", err, fresh)
	}
	if _, _, err = finishProductionStartup(horizon, fresh, memory, memory+".checkpoint.json"); err != nil {
		t.Fatal(err)
	}
	defer horizon.Supervisor().Close()
	brain := getActiveStartupState(t, horizon)
	before, err := os.ReadFile(journal)
	if err != nil {
		t.Fatal(err)
	}
	block := filepath.Join(dir, "regular-file")
	if err = os.WriteFile(block, []byte("preserve"), 0600); err != nil {
		t.Fatal(err)
	}
	var sequence uint64
	for i := 0; i < 3; i++ {
		out, count, err := finishProductionStartup(horizon, false, filepath.Join(block, "memory"), filepath.Join(block, "checkpoint"))
		if err != nil || count != 0 {
			t.Fatalf("resume attempted persistence: %v count=%d", err, count)
		}
		if i > 0 && out.Sequence != sequence {
			t.Fatal("read-only resume advanced sequence")
		}
		sequence = out.Sequence
	}
	after, err := os.ReadFile(journal)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(before, after) || !bytes.Equal(brain, getActiveStartupState(t, horizon)) {
		t.Fatal("resume mutated canonical state or journal")
	}
}
