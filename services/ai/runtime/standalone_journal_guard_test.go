package runtime

import (
	"bytes"
	"testing"
	"time"

	"github.com/project-horizon/horizon-core/services/ai/knowledge"
)

func TestAttachedJournalRejectsStandalonePartialObservationMutators(t *testing.T) {
	r := NewBrainRuntime(nil)
	r.SetEventLog(openKnowledgeJournal(t))
	before := continuationBytes(t, r)
	at := time.Date(2026, 10, 19, 9, 0, 0, 0, time.UTC)
	if _, err := r.GroundObservation("surface", "local", "vision"); err == nil {
		t.Fatal("standalone grounding bypassed journal")
	}
	if _, err := r.GroundObservationAt("surface", "local", "vision", at); err == nil {
		t.Fatal("standalone timed grounding bypassed journal")
	}
	if err := r.LearnObservedTransition([]knowledge.NodeID{1}, at); err == nil {
		t.Fatal("standalone temporal learning bypassed journal")
	}
	if _, err := r.PlanInquiry(.8, at); err == nil {
		t.Fatal("standalone inquiry planning bypassed journal")
	}
	if err := r.CompleteInquiry(.3, .4, at); err == nil {
		t.Fatal("standalone inquiry completion bypassed journal")
	}
	if !bytes.Equal(before, continuationBytes(t, r)) || r.RecoveryRequired() {
		t.Fatal("rejected partial mutator changed availability or state")
	}
}

func TestExplicitBootstrapActivationIsAReplayableProcessRevision(t *testing.T) {
	r := NewBrainRuntime(nil)
	journal := openKnowledgeJournal(t)
	r.SetEventLog(journal)
	at := time.Date(2026, 10, 19, 9, 0, 0, 0, time.UTC)
	if _, err := r.ActivateBootstrap(at); err != nil {
		t.Fatal(err)
	}
	if r.LastSequence() != 1 {
		t.Fatal("bootstrap activation changed recurrent state without a revision")
	}
	restored, err := RebuildRuntimeFromCommittedJournal(journal, knowledge.NewBootstrapBrain)
	if err != nil {
		t.Fatal(err)
	}
	assertCompleteContinuation(t, r, restored)
}
