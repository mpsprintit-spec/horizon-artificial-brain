package runtime

import (
	"errors"
	"path/filepath"
	"sync"
	"testing"
	"time"
)

func TestOwnerAttachesOnlyAnInitialVerifiedEmptyJournal(t *testing.T) {
	r := NewBrainRuntime(nil)
	owner, err := NewRuntimeSupervisor(r)
	if err != nil {
		t.Fatal(err)
	}
	defer owner.Close()
	journal := openKnowledgeJournal(t)
	if err := owner.AttachEventLog(journal); err != nil {
		t.Fatal(err)
	}
	if err := owner.AttachEventLog(journal); err != nil {
		t.Fatal("same writer attachment must be idempotent", err)
	}
	if r.eventLog != journal {
		t.Fatal("owner did not attach writer")
	}
	other := openKnowledgeJournal(t)
	if err := owner.AttachEventLog(other); err == nil {
		t.Fatal("owner silently replaced journal")
	}
	if r.eventLog != journal {
		t.Fatal("failed writer replacement changed active writer")
	}
	if err := owner.Close(); err != nil {
		t.Fatal(err)
	}
	if err := owner.AttachEventLog(journal); !errors.Is(err, ErrRuntimeSupervisorClosed) {
		t.Fatalf("closed owner accepted writer: %v", err)
	}
}

func TestOwnerRejectsJournalHistoryWithoutVerifiedRecovery(t *testing.T) {
	for _, advancedRuntime := range []bool{false, true} {
		r := NewBrainRuntime(nil)
		owner, err := NewRuntimeSupervisor(r)
		if err != nil {
			t.Fatal(err)
		}
		journal := openKnowledgeJournal(t)
		at := time.Unix(100, 0).UTC()
		if advancedRuntime {
			if _, _, err := r.Process(Event{ID: "unanchored", Timestamp: at, Cycles: 1}); err != nil {
				t.Fatal(err)
			}
		} else {
			event := Event{ID: "foreign-history", Timestamp: at, Cycles: 1}
			if err := journal.Append(LoggedEvent{Sequence: 1, Type: EventTypeProcess, Timestamp: at, Event: &event}); err != nil {
				t.Fatal(err)
			}
		}
		before := continuationBytes(t, r)
		if err := owner.AttachEventLog(journal); err == nil {
			t.Fatal("unverified state/journal pairing accepted")
		}
		if r.eventLog != nil || string(before) != string(continuationBytes(t, r)) || r.RecoveryRequired() {
			t.Fatal("failed attachment changed healthy runtime")
		}
		if err := owner.Close(); err != nil {
			t.Fatal(err)
		}
	}
}

func TestEventLogAppendAndCloseShareTheFileLifetimeLock(t *testing.T) {
	journal, err := OpenEventLog(filepath.Join(t.TempDir(), "events.jsonl"))
	if err != nil {
		t.Fatal(err)
	}
	defer journal.Close()
	start := make(chan struct{})
	var wg sync.WaitGroup
	for i := 0; i < 16; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			_ = journal.Append(LoggedEvent{Sequence: 1, Type: EventTypeThink, Event: &Event{Cycles: 1}})
		}()
	}
	wg.Add(1)
	go func() {
		defer wg.Done()
		<-start
		if err := journal.Close(); err != nil {
			t.Error(err)
		}
	}()
	close(start)
	wg.Wait()
	if err := journal.Append(LoggedEvent{}); err == nil {
		t.Fatal("closed writer accepted append")
	}
}
