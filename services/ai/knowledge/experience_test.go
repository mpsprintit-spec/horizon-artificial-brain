package knowledge

import (
	"testing"
	"time"
)

func TestLearnNeuralTraceReusesSingleSubstrate(t *testing.T) {
	brain := NewBrain()
	a := brain.Store("unit-a")
	b := brain.Store("unit-b")
	c := brain.Store("unit-c")

	trace := NeuralTrace{
		Sequence: []PatternStep{
			{NodeID: a.ID, Position: 0, Activation: 1},
			{NodeID: b.ID, Position: 1, Activation: 0.8, Delta: 10 * time.Millisecond},
			{NodeID: c.ID, Position: 2, Activation: 0.6, Delta: 20 * time.Millisecond},
		},
		Weight:     0.7,
		Confidence: 0.9,
		Now:        time.Now().UTC(),
	}

	brain.LearnNeuralTrace(trace)
	brain.LearnNeuralTrace(trace)

	if len(a.Synapses[b.ID]) != 1 {
		t.Fatalf("expected one reused dynamic connection a->b, got %d", len(a.Synapses[b.ID]))
	}
	if len(b.Synapses[c.ID]) != 1 {
		t.Fatalf("expected one reused dynamic connection b->c, got %d", len(b.Synapses[c.ID]))
	}

	ab := a.FindDynamicSynapse(b.ID, false)
	bc := b.FindDynamicSynapse(c.ID, false)
	if ab == nil || bc == nil {
		t.Fatal("expected dynamic synapses to be present")
	}
	if ab.Dynamic.Frequency != 2 || bc.Dynamic.Frequency != 2 {
		t.Fatalf("expected repeated trace to reinforce existing structure: ab=%d bc=%d", ab.Dynamic.Frequency, bc.Dynamic.Frequency)
	}

	patterns := brain.Patterns.All()
	if len(patterns) != 1 {
		t.Fatalf("expected one reused temporal trace, got %d", len(patterns))
	}
	if patterns[0].Frequency != 2 {
		t.Fatalf("expected temporal trace frequency 2, got %d", patterns[0].Frequency)
	}
}

func TestLearnNeuralTraceDoesNotRequireSemanticRelationKind(t *testing.T) {
	brain := NewBrain()
	a := brain.Store("vision-unit")
	b := brain.Store("audio-unit")

	brain.LearnNeuralTrace(NeuralTrace{
		Sequence: []PatternStep{
			{NodeID: a.ID, Position: 0, Activation: 1},
			{NodeID: b.ID, Position: 1, Activation: 1},
		},
		Weight:     1,
		Confidence: 1,
	})

	synapse := a.FindDynamicSynapse(b.ID, false)
	if synapse == nil {
		t.Fatal("expected modality-neutral trace to create a dynamic connection")
	}
	if synapse.Kind != "" {
		t.Fatalf("expected no semantic relation kind, got %q", synapse.Kind)
	}
}
