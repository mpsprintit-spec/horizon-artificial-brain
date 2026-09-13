package runtime

import (
	"testing"
	"time"
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
