package runtime

import (
	"testing"
	"time"

	"github.com/project-horizon/horizon-core/services/ai/knowledge"
)

func TestWriteAheadLogFailureDoesNotMutateProcessState(t *testing.T) {
	brain := knowledge.NewBrain()
	runtime := NewBrainRuntime(brain)
	log := &EventLog{}
	runtime.SetEventLog(log)

	before := len(brain.Registry.Nodes())
	_, sequence, err := runtime.Process(Event{Stimulus: []string{"air"}, Cycles: 1, Timestamp: time.Unix(100, 0).UTC()})
	if err == nil {
		t.Fatal("expected event log failure")
	}
	if sequence != 0 {
		t.Fatalf("sequence advanced after rejected event: got %d", sequence)
	}
	if got := len(brain.Registry.Nodes()); got != before {
		t.Fatalf("brain mutated despite rejected event: before=%d after=%d", before, got)
	}
}

func TestWriteAheadLogFailureDoesNotMutateLearningState(t *testing.T) {
	brain := knowledge.NewBrain()
	runtime := NewBrainRuntime(brain)
	log := &EventLog{}
	runtime.SetEventLog(log)

	before := len(brain.Registry.Nodes())
	sequence, err := runtime.LearnExperience(learningExperienceForWALTest(), time.Unix(101, 0).UTC())
	if err == nil {
		t.Fatal("expected event log failure")
	}
	if sequence != 0 {
		t.Fatalf("sequence advanced after rejected learning event: got %d", sequence)
	}
	if got := len(brain.Registry.Nodes()); got != before {
		t.Fatalf("brain mutated despite rejected learning event: before=%d after=%d", before, got)
	}
}

func learningExperienceForWALTest() learning.Experience {
	return learning.Experience{Sequence: []string{"saya", "belajar"}, Weight: 0.8, Confidence: 0.7, ExperienceID: "wal-test"}
}
