package runtime

import (
	"sync"
	"testing"
	"time"

	"github.com/project-horizon/horizon-core/services/ai/knowledge"
)

func TestFixedClockIsStableUntilAdvanced(t *testing.T) {
	initial := time.Date(2026, 9, 13, 12, 0, 0, 0, time.UTC)
	clock := NewFixedClock(initial)
	if got := clock.Now(); !got.Equal(initial) { t.Fatalf("initial clock: got %v want %v", got, initial) }
	if got := clock.Now(); !got.Equal(initial) { t.Fatalf("clock advanced unexpectedly: got %v want %v", got, initial) }
	next := initial.Add(time.Second)
	clock.Set(next)
	if got := clock.Now(); !got.Equal(next) { t.Fatalf("advanced clock: got %v want %v", got, next) }
}

func TestRuntimeUsesInjectedClockForMissingEventTimestamp(t *testing.T) {
	initial := time.Date(2026, 9, 13, 12, 0, 0, 0, time.UTC)
	brain := knowledge.NewBrain()
	brain.Store("air")
	r := NewBrainRuntime(brain)
	r.SetClock(NewFixedClock(initial))

	path := t.TempDir() + "/events.jsonl"
	log, err := OpenEventLog(path)
	if err != nil { t.Fatalf("open event log: %v", err) }
	defer log.Close()
	r.SetEventLog(log)

	if _, _, err := r.Process(Event{ID: "clock-test", Stimulus: []string{"air"}, Cycles: 1}); err != nil { t.Fatalf("process: %v", err) }
	events, err := ReadEventLog(path)
	if err != nil { t.Fatalf("read event log: %v", err) }
	if len(events) != 1 { t.Fatalf("events: got %d want 1", len(events)) }
	if !events[0].Timestamp.Equal(initial) { t.Fatalf("event timestamp: got %v want %v", events[0].Timestamp, initial) }
}

func TestRuntimeClockAccessIsSerialized(t *testing.T) {
	brain := knowledge.NewBrain()
	brain.Store("air")
	r := NewBrainRuntime(brain)
	var wg sync.WaitGroup
	const workers = 8
	wg.Add(workers * 2)
	for i := 0; i < workers; i++ {
		go func(i int) {
			defer wg.Done()
			r.SetClock(NewFixedClock(time.Date(2026, 9, 13, 12, 0, i, 0, time.UTC)))
		}(i)
		go func(i int) {
			defer wg.Done()
			_, _, err := r.CognitiveProcess(Event{ID: string(rune('a' + i)), Stimulus: []string{"air"}, Cycles: 1})
			if err != nil { t.Errorf("CognitiveProcess: %v", err) }
		}(i)
	}
	wg.Wait()
}
