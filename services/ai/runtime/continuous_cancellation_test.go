package runtime

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/project-horizon/horizon-core/services/ai/knowledge"
)

func TestContinuousServiceCancellationDoesNotCommitAnotherCycle(t *testing.T) {
	for _, beforeStart := range []bool{false, true} {
		r := NewBrainRuntime(knowledge.NewBrain())
		service := NewContinuousService(r, time.Nanosecond, 1)
		ctx, cancel := context.WithCancel(context.Background())
		if beforeStart {
			cancel()
		}
		calls := 0
		err := service.Run(ctx, func(_ CognitiveOutput, err error) {
			if err != nil {
				t.Error(err)
			}
			calls++
			if calls >= 2 {
				cancel()
			}
		})
		cancel()
		want := 2
		if beforeStart {
			want = 0
		}
		if !errors.Is(err, context.Canceled) || calls != want || r.LastSequence() != uint64(want) {
			t.Fatalf("cancelled service committed extra work: calls=%d seq=%d err=%v", calls, r.LastSequence(), err)
		}
	}
}
