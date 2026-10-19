package runtime

import (
	"bytes"
	"errors"
	"path/filepath"
	"reflect"
	"slices"
	"testing"
	"time"

	"github.com/project-horizon/horizon-core/services/ai/knowledge"
	"github.com/project-horizon/horizon-core/services/ai/learning"
)

func TestLegacyWrittenSurfaceRebindingIsReferencedJournaledAndRecoverable(t *testing.T) {
	log := legacySurfaceJournal(t)
	r, err := RebuildRuntimeFromCommittedJournal(log, knowledge.NewBrain)
	if err != nil {
		t.Fatal(err)
	}
	r.SetEventLog(log)
	owner, err := NewRuntimeSupervisor(r)
	if err != nil {
		t.Fatal(err)
	}
	defer owner.Close()
	beforePath := filepath.Join(t.TempDir(), "before-migration.json")
	if err := owner.WithMutation(func(active *BrainRuntime) error { return active.Checkpoint(beforePath) }); err != nil {
		t.Fatal(err)
	}
	before := continuationBytes(t, r)
	plan, err := r.PlanWrittenSurfaceRebinding("NAKU", "text")
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(before, continuationBytes(t, r)) {
		t.Fatal("migration planning mutated neural state")
	}
	brainBefore, _ := r.brain.CanonicalJSON()
	at := time.Date(2026, 10, 19, 10, 1, 0, 0, time.UTC)
	if err := owner.WithMutation(func(active *BrainRuntime) error { return active.ApplyWrittenSurfaceRebinding(plan, at) }); err != nil {
		t.Fatal(err)
	}
	brainAfter, _ := r.brain.CanonicalJSON()
	if !bytes.Equal(brainBefore, brainAfter) {
		t.Fatal("adapter rebinding rewrote neurons, learned connections or traces")
	}
	source, err := r.brain.LegacyWrittenSurface("NAKU", "text")
	if err != nil || !reflect.DeepEqual(source, plan.Source) {
		t.Fatal("migration discarded historical provenance")
	}
	var binding knowledge.SurfaceAnnotation
	for _, a := range r.brain.SurfaceAnnotationsSnapshot() {
		if a.Surface == "naku" && a.Encoding == knowledge.WrittenTextEncoding {
			binding = a
		}
	}
	if binding.ReferenceHash != plan.ReferenceHash || binding.ReboundFrom != "text:NAKU" || !slices.Equal(binding.Population, plan.Source.Population) {
		t.Fatal("migration receipt does not describe the chosen historical population")
	}
	path := log.file.Name()
	events, err := ReadEventLog(path)
	if err != nil {
		t.Fatal(err)
	}
	last := events[len(events)-1]
	if last.SchemaVersion != EventLogSchemaVersion || last.SurfaceRebinding == nil || !reflect.DeepEqual(*last.SurfaceRebinding, plan) {
		t.Fatal("migration is not fully represented in the journal")
	}
	observe := func(active *BrainRuntime, id string, at time.Time) error {
		out, _, err := NewCognitiveOrchestrator(active).ProcessObservation(Event{ID: id, Timestamp: at, Cycles: 1}, ObservationInput{Source: "new-keyboard", Modality: "text", Tokens: []string{"naku"}}, nil)
		if err != nil {
			return err
		}
		if len(out.GroundedRepresentations) != 1 || !slices.Equal(out.GroundedRepresentations[0].Population, plan.Source.Population) {
			t.Fatal("new input bypassed the selected legacy population")
		}
		return nil
	}
	if err := owner.WithMutation(func(active *BrainRuntime) error { return observe(active, "after-migration", at.Add(time.Second)) }); err != nil {
		t.Fatal(err)
	}
	genesis, err := RebuildRuntimeFromCommittedJournal(log, knowledge.NewBrain)
	if err != nil {
		t.Fatal(err)
	}
	suffix, err := RebuildRuntimeFromCheckpointAndJournal(log, beforePath)
	if err != nil {
		t.Fatal(err)
	}
	assertCompleteContinuation(t, r, genesis)
	assertCompleteContinuation(t, r, suffix)
	if err := owner.RecoverFromCheckpoint(log, beforePath); err != nil {
		t.Fatal(err)
	}
	if err := owner.WithMutation(func(active *BrainRuntime) error {
		if err := observe(active, "after-restart", at.Add(2*time.Second)); err != nil {
			return err
		}
		if _, err := active.LearnExperience(learning.Experience{Sequence: []string{"naku", "follow"}, Weight: .8, Confidence: .8}, at.Add(3*time.Second)); err != nil {
			return err
		}
		for _, p := range active.brain.Patterns.All() {
			if len(p.Members) == 2 && p.Members[0] == plan.Source.NodeID {
				return nil
			}
		}
		t.Fatal("new sequence learning ignored the explicit surface binding")
		return nil
	}); err != nil {
		t.Fatal(err)
	}
}

func TestWrittenSurfaceRebindingRejectsStaleChangedAndRetiredPlans(t *testing.T) {
	log := legacySurfaceJournal(t)
	r, err := RebuildRuntimeFromCommittedJournal(log, knowledge.NewBrain)
	if err != nil {
		t.Fatal(err)
	}
	r.SetEventLog(log)
	plan, err := r.PlanWrittenSurfaceRebinding("NAKU", "text")
	if err != nil {
		t.Fatal(err)
	}
	before := continuationBytes(t, r)
	for _, mutate := range []func(*WrittenSurfaceRebinding){
		func(p *WrittenSurfaceRebinding) { p.ReferenceRevision++ },
		func(p *WrittenSurfaceRebinding) { p.ReferenceHash = "sha256:wrong-reference" },
		func(p *WrittenSurfaceRebinding) { p.Source.Population = []knowledge.NodeID{p.Source.NodeID + 1} },
	} {
		changed := plan
		mutate(&changed)
		if err := r.ApplyWrittenSurfaceRebinding(changed, time.Unix(100, 0)); err == nil {
			t.Fatal("invalid migration accepted")
		}
		if !bytes.Equal(before, continuationBytes(t, r)) {
			t.Fatal("invalid migration mutated state")
		}
	}
	if r.RecoveryRequired() {
		t.Fatal("invalid migration poisoned a healthy runtime")
	}
	events, err := ReadEventLog(log.file.Name())
	if err != nil || uint64(len(events)) != r.LastSequence() {
		t.Fatal("invalid migration appended a record")
	}
	owner, err := NewRuntimeSupervisor(r)
	if err != nil {
		t.Fatal(err)
	}
	if err := owner.Close(); err != nil {
		t.Fatal(err)
	}
	if _, err := r.PlanWrittenSurfaceRebinding("NAKU", "text"); !errors.Is(err, ErrRuntimeRecoveryRequired) {
		t.Fatal("retired runtime planned a migration")
	}
	if err := r.ApplyWrittenSurfaceRebinding(plan, time.Unix(100, 0)); !errors.Is(err, ErrRuntimeRecoveryRequired) {
		t.Fatal("retired runtime applied a migration")
	}
}

func TestWrittenSurfaceRebindingAppendFailureLeavesBindingUntouched(t *testing.T) {
	log := legacySurfaceJournal(t)
	r, err := RebuildRuntimeFromCommittedJournal(log, knowledge.NewBrain)
	if err != nil {
		t.Fatal(err)
	}
	r.SetEventLog(log)
	plan, err := r.PlanWrittenSurfaceRebinding("NAKU", "text")
	if err != nil {
		t.Fatal(err)
	}
	before := continuationBytes(t, r)
	log.beforeAppend = func(LoggedEvent) error { return errors.New("injected migration append failure") }
	if err := r.ApplyWrittenSurfaceRebinding(plan, time.Unix(100, 0)); !errors.Is(err, ErrRuntimeRecoveryRequired) {
		t.Fatalf("append failure=%v", err)
	}
	if !r.RecoveryRequired() || !bytes.Equal(before, continuationBytes(t, r)) {
		t.Fatal("failed write changed the surface binding or left runtime available")
	}
}
