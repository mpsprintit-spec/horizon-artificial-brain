package runtime

import (
	"context"
	"errors"
	"path/filepath"
	"testing"
	"time"

	"github.com/project-horizon/horizon-core/services/ai/knowledge"
)

func TestManagedContinuousServiceFollowsRuntimeHandover(t *testing.T) {
	path := filepath.Join(t.TempDir(), "events.jsonl")
	checkpoint := filepath.Join(t.TempDir(), "checkpoint.json")
	log, err := OpenEventLog(path)
	if err != nil { t.Fatal(err) }
	defer log.Close()
	old := NewBrainRuntime(knowledge.NewBootstrapBrain())
	old.SetEventLog(log)
	owner, err := NewRuntimeOwner(old, log)
	if err != nil { t.Fatal(err) }
	if err := old.Checkpoint(checkpoint); err != nil { t.Fatalf("initial checkpoint: %v", err) }

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	service := NewManagedContinuousService(owner, 10*time.Millisecond, 1)
	outputs := make(chan CognitiveOutput, 256)
	errorsCh := make(chan error, 256)
	done := make(chan error, 1)
	go func() {
		done <- service.Run(ctx, func(out CognitiveOutput, err error) {
			if err != nil { errorsCh <- err; return }
			outputs <- out
		})
	}()
	select {
	case out := <-outputs:
		if out.Sequence == 0 { t.Fatal("first cycle did not advance runtime") }
	case err := <-errorsCh:
		t.Fatalf("initial managed cycle failed: %v", err)
	case <-time.After(3*time.Second):
		t.Fatal("managed service did not run initial cycle")
	}

	if err := owner.RecoverFromCheckpoint(checkpoint); err != nil {
		t.Fatalf("recover live continuous service: %v", err)
	}
	if !old.Retired() { t.Fatal("continuous service old runtime was not retired") }
	seqAfterHandover := uint64(0)
	if err := owner.WithRuntime(func(r *BrainRuntime) error {
		seqAfterHandover = r.LastSequence()
		return nil
	}); err != nil { t.Fatal(err) }
	found := false
	timeout := time.After(3*time.Second)
	for !found {
		select {
		case out := <-outputs:
			if out.Sequence > seqAfterHandover { found = true }
		case err := <-errorsCh:
			t.Fatalf("managed continuous service failed after handover: %v", err)
		case <-timeout:
			t.Fatal("managed continuous service did not continue with new runtime")
		}
	}
	cancel()
	select {
	case err := <-done:
		if !errors.Is(err, context.Canceled) { t.Fatalf("service exit: %v", err) }
	case <-time.After(3*time.Second):
		t.Fatal("managed service failed to stop")
	}
	if _, err := old.CognitiveThink(1); !errors.Is(err, ErrRuntimeRetired) {
		t.Fatalf("retired runtime accepted a direct cognitive cycle: %v", err)
	}
}
