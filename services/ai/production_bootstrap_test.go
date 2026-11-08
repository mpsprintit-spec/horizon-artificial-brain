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
    if err != nil { t.Fatalf("first bootstrap: %v", err) }
    if count == 0 || firstOutput.Sequence == 0 {
        t.Fatalf("bootstrap did not record initial event: count=%d sequence=%d", count, firstOutput.Sequence)
    }
    originalBrain := getActiveStartupState(t, first)
    originalMemory, err := os.ReadFile(memory)
    if err != nil { t.Fatal(err) }
    beforeEvents, err := runtime.ReadEventLog(journal)
    if err != nil { t.Fatal(err) }
    if len(beforeEvents) == 0 { t.Fatal("bootstrap never entered journal") }

    if err := first.Supervisor().Close(); err != nil { t.Fatal(err) }
    if err := firstLog.Close(); err != nil { t.Fatal(err) }

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
    if err != nil { t.Fatal(err) }
    if !bytes.Equal(originalMemory, afterMemory) {
        t.Fatal("resume rewrote canonical brain memory")
    }
    afterEvents, err := runtime.ReadEventLog(journal)
    if err != nil { t.Fatal(err) }
    if len(afterEvents) != len(beforeEvents) {
        t.Fatal("resume wrote an unexpected journal event")
    }
}
