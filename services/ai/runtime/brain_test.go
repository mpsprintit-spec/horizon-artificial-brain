package runtime

import (
	"os"
	"sync"
	"testing"
	"time"

	"github.com/project-horizon/horizon-core/services/ai/knowledge"
)

func TestBrainRuntimeUsesSingleBrainAcrossEvents(t *testing.T) {
	brain := knowledge.NewBrain()
	saya := brain.Store("saya")
	ingin := brain.Store("ingin")
	brain.Store("belajar")
	brain.Connect(saya, ingin, 0.6, 0.8, false)
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
	path := saya.SynapsesTo(ingin.ID)
	if len(path) != 1 { t.Fatalf("saya -> ingin synapse count = %d, want 1", len(path)) }
	if path[0].Dynamic.Frequency < 2 { t.Fatalf("synapse frequency = %d, want repeated reinforcement", path[0].Dynamic.Frequency) }
}

func TestBrainRuntimeContinuesWithoutExternalInput(t *testing.T) {
	brain := knowledge.NewBrain()
	brain.Store("air")
	brain.Store("dingin")
	r := NewBrainRuntime(brain)
	if _, _, err := r.Process(Event{ID: "seed", Stimulus: []string{"air", "dingin", "air"}, Cycles: 1, Timestamp: time.Unix(200, 0).UTC()}); err != nil { t.Fatal(err) }

	thought, seq, err := r.Think(3)
	if err != nil { t.Fatal(err) }
	if seq != 2 { t.Fatalf("internal thought sequence = %d, want 2", seq) }
	if len(thought.Activations) == 0 { t.Fatal("internal thought stopped despite an existing neural state") }
}

func TestBrainRuntimeSerializesConcurrentTransitions(t *testing.T) {
	brain := knowledge.NewBrain()
	brain.Store("shared")
	brain.Store("path")
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

func TestBrainRuntimeCuriosityDriveTargetsUnderusedNeuralState(t *testing.T) {
	brain := knowledge.NewBrain()
	first := brain.Store("alpha")
	second := brain.Store("beta")
	if first == nil || second == nil {
		t.Fatal("expected bootstrap neural units")
	}
	first.Frequency = 20
	first.Activation = 0.9
	second.Frequency = 1
	second.Activation = 0.05
	r := NewBrainRuntime(brain)

	_, err := r.CognitiveThink(1)
	if err != nil {
		t.Fatal(err)
	}

	target := brain.BrainState.CuriosityState["target_node"]
	if int64(target) != int64(second.ID) {
		t.Fatalf("curiosity target = %v, want node %d", target, second.ID)
	}
	if brain.BrainState.CuriosityState["drive"] <= 0 {
		t.Fatal("curiosity drive did not activate")
	}
}

func TestBrainRuntimeAutonomousCyclesChangeStateWithoutInput(t *testing.T) {
	brain := knowledge.NewBrain()
	for _, surface := range []string{"a", "b", "c", "d"} {
		brain.Store(surface)
	}
	r := NewBrainRuntime(brain)

	first, err := r.CognitiveThink(1)
	if err != nil {
		t.Fatal(err)
	}
	second, err := r.CognitiveThink(1)
	if err != nil {
		t.Fatal(err)
	}
	if second.Sequence <= first.Sequence {
		t.Fatalf("sequence did not advance: %d -> %d", first.Sequence, second.Sequence)
	}
	if len(second.Activations) == 0 {
		t.Fatal("autonomous cycle produced no activation")
	}
	if brain.BrainState.CuriosityState["last_update_unix"] <= 0 {
		t.Fatal("curiosity state was not updated")
	}
}


func TestBrainRuntimePlanInquiryPersistsTrajectoryWithoutExecution(t *testing.T) {
	brain := knowledge.NewBrain()
	first := brain.Store("first")
	second := brain.Store("second")
	if first == nil || second == nil {
		t.Fatal("expected neural substrate")
	}
	first.Frequency = 20
	first.Activation = 0.9
	second.Frequency = 1
	second.Activation = 0.05
	r := NewBrainRuntime(brain)
	at := time.Unix(300, 0).UTC()

	agenda, err := r.PlanInquiry(0.9, at)
	if err != nil {
		t.Fatal(err)
	}
	if agenda.Uncertainty != 0.9 {
		t.Fatalf("agenda uncertainty = %v, want 0.9", agenda.Uncertainty)
	}
	if agenda.Selected == nil {
		t.Fatal("inquiry agenda has no selected candidate")
	}
	if r.LastSequence() != 0 {
		t.Fatalf("planning inquiry changed execution sequence: %d", r.LastSequence())
	}
	state := brain.BrainState.InquiryState
	if !state.Pending {
		t.Fatal("inquiry trajectory was not marked pending")
	}
	if state.SelectedAction == "" {
		t.Fatal("selected inquiry action was not persisted")
	}
	if state.Sequence != 0 {
		t.Fatalf("inquiry sequence = %d, want 0", state.Sequence)
	}
}

func TestBrainRuntimeInquiryOutcomeChangesAndPersistsLearningStrategy(t *testing.T) {
	brain := knowledge.NewBrain()
	brain.Store("a")
	r := NewBrainRuntime(brain)
	at := time.Unix(400, 0).UTC()

	if _, err := r.PlanInquiry(0.8, at); err != nil {
		t.Fatal(err)
	}
	before := brain.BrainState.LearningPolicyState
	if err := r.CompleteInquiry(0.2, 0.9, at.Add(time.Second)); err != nil {
		t.Fatal(err)
	}
	after := brain.BrainState.LearningPolicyState
	if !(after.ExplorationBias > before.ExplorationBias) {
		t.Fatalf("exploration bias did not increase: %v -> %v", before.ExplorationBias, after.ExplorationBias)
	}
	if !(after.RepeatObservationBias > before.RepeatObservationBias) {
		t.Fatalf("repeat-observation bias did not increase: %v -> %v", before.RepeatObservationBias, after.RepeatObservationBias)
	}
	if !(after.DeferConclusionBias > before.DeferConclusionBias) {
		t.Fatalf("defer-conclusion bias did not increase: %v -> %v", before.DeferConclusionBias, after.DeferConclusionBias)
	}
	if brain.BrainState.InquiryState.Pending {
		t.Fatal("inquiry trajectory remained pending after outcome")
	}
	if brain.BrainState.InquiryState.OutcomePredictionError != 0.9 {
		t.Fatalf("stored inquiry error = %v, want 0.9", brain.BrainState.InquiryState.OutcomePredictionError)
	}

	path := t.TempDir() + "/brain_memory.json"
	if err := brain.Save(path); err != nil {
		t.Fatal(err)
	}
	loaded := knowledge.NewBrain()
	if err := loaded.Load(path); err != nil {
		t.Fatal(err)
	}
	if loaded.BrainState.InquiryState.SelectedAction != brain.BrainState.InquiryState.SelectedAction {
		t.Fatalf("selected action was not persisted: %q -> %q", brain.BrainState.InquiryState.SelectedAction, loaded.BrainState.InquiryState.SelectedAction)
	}
	if loaded.BrainState.LearningPolicyState.ExplorationBias != after.ExplorationBias {
		t.Fatalf("exploration bias was not persisted: %v -> %v", after.ExplorationBias, loaded.BrainState.LearningPolicyState.ExplorationBias)
	}
	_ = os.Remove(path)
}


func TestBrainRuntimeSerializesAutonomousCognitionAndInquiryPlanning(t *testing.T) {
	brain := knowledge.NewBrain()
	for _, surface := range []string{"alpha", "beta", "gamma"} {
		brain.Store(surface)
	}
	r := NewBrainRuntime(brain)

	const workers = 8
	var wg sync.WaitGroup
	errs := make(chan error, workers*2)
	wg.Add(workers * 2)

	for i := 0; i < workers; i++ {
		go func() {
			defer wg.Done()
			_, err := r.CognitiveThink(1)
			errs <- err
		}()
		go func() {
			defer wg.Done()
			_, err := r.PlanInquiry(0.7, time.Unix(500, 0).UTC())
			errs <- err
		}()
	}
	wg.Wait()
	close(errs)

	for err := range errs {
		if err != nil {
			t.Fatal(err)
		}
	}
	if got := r.LastSequence(); got != workers {
		t.Fatalf("autonomous sequence = %d, want %d", got, workers)
	}
	if !brain.BrainState.InquiryState.Pending {
		t.Fatal("inquiry planning did not persist a pending trajectory")
	}
}
