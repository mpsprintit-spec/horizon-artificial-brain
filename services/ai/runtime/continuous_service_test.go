package runtime

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/project-horizon/horizon-core/services/ai/knowledge"
)

func TestContinuousServiceRunsImmediatelyAndStopsOnContextCancellation(t *testing.T) {
	brain := knowledge.NewKnowledgeBase()
	runtime := NewBrainRuntime(brain)
	if runtime == nil {
		t.Fatal("expected runtime")
	}

	ctx, cancel := context.WithCancel(context.Background())
	outputs := make(chan CognitiveOutput, 2)
	errs := make(chan error, 2)

	service := NewContinuousService(runtime, time.Hour, 1)

	done := make(chan error, 1)
	go func() {
		done <- service.Run(ctx, func(output CognitiveOutput, err error) {
			if err != nil {
				errs <- err
				return
			}
			outputs <- output
		})
	}()

	select {
	case <-outputs:
		// Cancel from the test goroutine after observing the output. This avoids
		// racing the service callback's output send with cancellation.
		cancel()
	case <-time.After(2 * time.Second):
		t.Fatal("continuous service did not execute its initial cognitive cycle")
	}

	select {
	case err := <-done:
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("expected context cancellation, got %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("continuous service did not stop after context cancellation")
	}

	select {
	case err := <-errs:
		t.Fatalf("continuous cognitive cycle failed: %v", err)
	default:
	}
}
