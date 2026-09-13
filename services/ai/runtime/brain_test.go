package runtime

import (
	"sync"
	"testing"
	"time"

	"github.com/project-horizon/horizon-core/services/ai/knowledge"
)

func TestBrainRuntimeUsesSingleBrainAcrossEvents(t *testing.T) {
	brain := knowledge.NewBrain()
	r := NewBrainRuntime(brain)

	first, seq1, err := r.Process(Event{
		ID: "e1", Stimulus: []string{"saya", "ingin", "belajar"}, Cycles: 2,
		Timestamp: time.Unix(100, 0).UTC(),
	})
	if err != nil { t.Fatal(err) }
	if seq1 != 1 { t.Fatalf("first sequence = %d, want 1", seq1) }
	if len(first.Activations) == 0 { t.Fatal("first event produced no neural activation") }

	second, seq2, err := r.Process(Event{
		ID: "e2", Stimulus: []string{"saya", "ingin", "belajar"}, Cycles: 2,
		Timestamp: time.Unix(101, 0).UTC(),
	})
	if err != nil { t.Fatal(err) }
	if seq2 != 2 { t.Fatalf("second sequence = %d, want 2", seq2) }
	if len(second.Activations) == 0 { t.Fatal("second event produced no neural activation") }

	if got := len(brain.Registry.Nodes()); got != 3 {
		t.Fatalf("brain node count = %d, want 3; repeated experience must reuse the same substrate", got)
	}

	saya := brain.Fetch("saya")
	ingin := brain.Fetch("ingin")
	if saya == nil || ingin == nil { t.Fatal("expected learned nodes to remain in the same brain") }
	path := saya.SynapsesTo(ingin.ID)
	if len(path) != 1 { t.Fatalf("saya -> ingin synapse count = %d, want 1", len(path)) }
	if path[0].Dynamic.Frequency < 2 { t.Fatalf("synapse frequency = %d, want repeated reinforcement", path[0].Dynamic.Frequency) }
}

func TestBrainRuntimeContinuesWithoutExternalInput(t *testing.T) {
	brain := knowledge.NewBrain()
	r := NewBrainRuntime(brain)
	if _, _, err := r.Process(Event{ID: "seed", Stimulus: []string{"air", "dingin", "air"}, Cycles: 1, Timestamp: time.Unix(200, 0).UTC()}); err != nil { t.Fatal(err) }

	thought, seq, err := r.Think(3)
	if err != nil { t.Fatal(err) }
	if seq != 2 { t.Fatalf("internal thought sequence = %d, want 2", seq) }
	if len(thought.Activations) == 0 { t.Fatal("internal thought stopped despite an existing neural state") }
}

func TestBrainRuntimeSerializesConcurrentTransitions(t *testing.T) {
	brain := knowledge.NewBrain()
	r := NewBrainRuntime(brain)

	const workers = 16
	sequences := make(chan uint64, workers)
	errs := make(chan error, workers)
	var wg sync.WaitGroup
	wg.Add(workers)
	for i := 0; i < workers; i++ {
		go func(i int) {
			defer wg.Done()
			_, seq, err := r.Process(Event{
				ID: string(rune('a' + i)),
				Stimulus: []string{"shared", "path"},
				Cycles: 1,
				Timestamp: time.Unix(int64(i+1), 0).UTC(),
			})
			sequences <- seq
			errs <- err
		}(i)
	}
	wg.Wait()
	close(sequences)
	close(errs)

	for err := range errs { if err != nil { t.Fatal(err) } }
	seen := make(map[uint64]bool, workers)
	for seq := range sequences {
		if seen[seq] { t.Fatalf("duplicate sequence %d", seq) }
		seen[seq] = true
	}
	if len(seen) != workers { t.Fatalf("got %d unique sequences, want %d", len(seen), workers) }
	if got := r.LastSequence(); got != workers { t.Fatalf("last sequence = %d, want %d", got, workers) }
}
