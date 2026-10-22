package runtime

import (
	"bytes"
	"errors"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/project-horizon/horizon-core/services/ai/knowledge"
	"github.com/project-horizon/horizon-core/services/ai/learning"
)

func TestOneBrainCannotAcquireMultipleRuntimeOrSupervisorOwners(t *testing.T) {
	brain := knowledge.NewBrain()
	r := NewBrainRuntime(brain)
	if duplicate := NewBrainRuntime(brain); duplicate != nil {
		t.Error("same Brain acquired a second runtime with independent sequence and activation")
	}
	const contenders = 8
	var wg sync.WaitGroup
	results := make(chan *RuntimeSupervisor, contenders)
	for i := 0; i < contenders; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			owner, _ := NewRuntimeSupervisor(r)
			if owner != nil {
				results <- owner
			}
		}()
	}
	wg.Wait()
	close(results)
	owners := 0
	for range results {
		owners++
	}
	if owners != 1 {
		t.Errorf("runtime acquired %d supervisors, want one", owners)
	}
}

func TestRetiredRuntimeRejectsDirectClaimPromotion(t *testing.T) {
	r := NewBrainRuntime(nil)
	grounded, err := r.GroundVectorAt(knowledge.NewNeuralVector([]float64{.2, .8}), time.Unix(100, 0).UTC())
	if err != nil {
		t.Fatal(err)
	}
	owner, err := NewRuntimeSupervisor(r)
	if err != nil {
		t.Fatal(err)
	}
	if err := owner.Close(); err != nil {
		t.Fatal(err)
	}
	before, _ := r.brain.CanonicalJSON()
	_, _, err = r.VerifyKnowledgeClaim(KnowledgeClaim{NodeID: grounded.NodeID}, learning.Evidence{Weight: .9, Confidence: .9, Reliability: .9, IndependentSources: 2}, time.Unix(101, 0).UTC())
	if !errors.Is(err, ErrRuntimeRecoveryRequired) {
		t.Errorf("retired runtime accepted promotion: %v", err)
	}
	after, _ := r.brain.CanonicalJSON()
	if !bytes.Equal(before, after) {
		t.Error("promotion mutated retired Brain")
	}
}

func TestManagedRuntimeDoesNotExportMutableBrainOrDetachItsJournal(t *testing.T) {
	journal, err := OpenEventLog(filepath.Join(t.TempDir(), "events.jsonl"))
	if err != nil {
		t.Fatal(err)
	}
	defer journal.Close()
	r := NewBrainRuntime(nil)
	r.SetEventLog(journal)
	owner, err := NewRuntimeSupervisor(r)
	if err != nil {
		t.Fatal(err)
	}
	defer owner.Close()
	before, err := r.brain.CanonicalJSON()
	if err != nil {
		t.Fatal(err)
	}
	called := false
	if err := r.WithBrainView(func(brain *knowledge.Brain) error { called = true; brain.Store("unlogged-mutation"); return nil }); err == nil || called {
		t.Error("managed callback exposed a mutable Brain")
	}
	if fabric := r.DNF(); fabric != nil {
		t.Error("managed DNF exposed its mutable Brain")
	}
	r.SetEventLog(nil)
	if r.eventLog != journal {
		t.Error("cached runtime detached owner journal")
	}
	after, err := r.brain.CanonicalJSON()
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(before, after) {
		t.Error("unsupported mutable access changed managed state")
	}
}

func TestManagedRuntimeCannotBeReplayedBehindItsOwner(t *testing.T) {
	r := NewBrainRuntime(nil)
	owner, err := NewRuntimeSupervisor(r)
	if err != nil {
		t.Fatal(err)
	}
	defer owner.Close()
	if err := ReplayEventLog(r, nil); err == nil {
		t.Error("managed runtime entered private replay and journal-detachment path")
	}
}
