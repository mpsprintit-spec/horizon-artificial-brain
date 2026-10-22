package runtime

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"testing"
	"time"

	"github.com/project-horizon/horizon-core/services/ai/knowledge"
	"github.com/project-horizon/horizon-core/services/ai/learning"
)

func pr37Journal(t *testing.T) *EventLog {
	t.Helper()
	data, err := os.ReadFile("testdata/stable_projection_v1/commands.jsonl")
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), "commands.jsonl")
	if err = os.WriteFile(path, data, 0600); err != nil {
		t.Fatal(err)
	}
	log, err := OpenEventLog(path)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = log.Close() })
	return log
}

func TestEvidenceProvenancePreservesActualPR37StateAndNextHistoricalOutcome(t *testing.T) {
	data, err := os.ReadFile("testdata/stable_projection_v1/reference.json")
	if err != nil {
		t.Fatal(err)
	}
	var ref struct {
		Commit   string       `json:"commit"`
		Checksum string       `json:"checksum"`
		Next     OutcomeEvent `json:"next_outcome"`
		After    string       `json:"after_next_checksum"`
	}
	if err = json.Unmarshal(data, &ref); err != nil {
		t.Fatal(err)
	}
	if ref.Commit != "b5b697f5570495094c9a2f2db7432708c3d166c1" {
		t.Fatal("wrong historical source")
	}
	log := pr37Journal(t)
	genesis, err := RebuildRuntimeFromCommittedJournal(log, knowledge.NewBrain)
	if err != nil {
		t.Fatal(err)
	}
	suffix, err := RebuildRuntimeFromCheckpointAndJournal(log, "testdata/stable_projection_v1/prefix-checkpoint.json")
	if err != nil {
		t.Fatal(err)
	}
	checksum := func(r *BrainRuntime) string {
		s, e := readCheckpointForTest(orderedSave(t, r))
		if e != nil {
			t.Fatal(e)
		}
		s.SchemaVersion = observedFeedbackBrainSnapshotSchemaVersion
		sum, e := checkpointChecksumV7(s)
		if e != nil {
			t.Fatal(e)
		}
		return sum
	}
	for _, r := range []*BrainRuntime{genesis, suffix} {
		r.SetClock(NewFixedClock(time.Date(2026, 10, 20, 9, 0, 0, 0, time.UTC)))
		if checksum(r) != ref.Checksum || r.dynamicsEncoding != knowledge.CurrentDynamics {
			t.Fatal("historical full state or serving default changed")
		}
		if r.evidence.Counts(1).LegacyObservations != 2 {
			t.Fatal("legacy observations were reclassified as independent")
		}
		r.dynamicsEncoding = knowledge.StableProjectionDynamics
		if _, err := r.ObserveOutcome(ref.Next); err != nil {
			t.Fatal(err)
		}
		r.dynamicsEncoding = knowledge.CurrentDynamics
		if checksum(r) != ref.After {
			t.Fatal("next historical outcome changed")
		}
	}
}

func TestEvidenceProvenanceRepeatedCameraDoesNotPromoteAndDeclaredOriginsDo(t *testing.T) {
	for _, shared := range []bool{false, true} {
		t.Run(fmt.Sprint("shared-cause-", shared), func(t *testing.T) {
			r := NewBrainRuntime(knowledge.NewBrain())
			at := time.Unix(100, 0).UTC()
			g, err := r.GroundVectorAt(knowledge.NewNeuralVector([]float64{.2, .8}), at)
			if err != nil {
				t.Fatal(err)
			}
			node := r.brain.Registry.GetByID(g.NodeID)
			importance := node.Importance
			for i := 0; i < 8; i++ {
				id := fmt.Sprintf("request-%d", i)
				if err := r.RegisterActionBinding(ActionBinding{RequestID: id, TargetNodeIDs: []knowledge.NodeID{g.NodeID}}); err != nil {
					t.Fatal(err)
				}
				source, group, cause := "same-camera", "one-origin", "one-cause"
				if shared {
					source = fmt.Sprintf("relay-%d", i)
					group = fmt.Sprintf("declaration-%d", i)
				}
				if _, err := r.ObserveOutcome(OutcomeEvent{RequestID: id, Source: source, Success: true, Observation: []string{"seen"}, Reliability: .9, ObservedAt: at.Add(time.Duration(i+1) * time.Second), EpisodeID: fmt.Sprintf("e-%d", i), IndependenceGroup: group, CausalGroup: cause}); err != nil {
					t.Fatal(err)
				}
			}
			counts := r.evidence.Counts(g.NodeID)
			if counts.Observations != 8 || counts.Episodes != 8 || counts.IndependentGroups != 1 || counts.CausalGroups != 1 || node.Importance != importance {
				t.Fatalf("pseudo-independent promotion: %+v importance %g -> %g", counts, importance, node.Importance)
			}
			if err := r.RegisterActionBinding(ActionBinding{RequestID: "independent", TargetNodeIDs: []knowledge.NodeID{g.NodeID}}); err != nil {
				t.Fatal(err)
			}
			o := OutcomeEvent{RequestID: "independent", Source: "independent-observer", Success: true, Observation: []string{"confirmed"}, Reliability: .9, ObservedAt: at.Add(10 * time.Second), EpisodeID: "another-episode", IndependenceGroup: "another-origin", CausalGroup: "another-cause"}
			if _, err := r.ObserveOutcome(o); err != nil {
				t.Fatal(err)
			}
			if r.evidence.Counts(g.NodeID).IndependentGroups != 2 || node.Importance <= importance {
				t.Fatal("declared independent confirmation did not reach neural promotion")
			}
			before, _ := r.brain.CanonicalJSON()
			ledger := r.evidence.Snapshot()
			if _, err := r.ObserveOutcome(o); err != nil {
				t.Fatal(err)
			}
			after, _ := r.brain.CanonicalJSON()
			if !bytes.Equal(before, after) || !reflect.DeepEqual(ledger, r.evidence.Snapshot()) {
				t.Fatal("duplicate outcome repeated plasticity")
			}
			t.Logf("observations=%d sources=%d independent_groups=%d importance %.2f -> %.2f", counts.Observations, counts.Sources, counts.IndependentGroups, importance, node.Importance)
		})
	}
}

func TestEvidenceProvenanceMixedLegacyAdoptionGenesisSuffixAndNextTransition(t *testing.T) {
	log := pr37Journal(t)
	live, err := RebuildRuntimeFromCommittedJournal(log, knowledge.NewBrain)
	if err != nil {
		t.Fatal(err)
	}
	at := time.Date(2026, 10, 20, 9, 1, 0, 0, time.UTC)
	live.SetClock(NewFixedClock(at))
	live.SetEventLog(log)
	importance := live.brain.Registry.GetByID(1).Importance
	one := OutcomeEvent{RequestID: "request-3", Source: "same-camera", Success: true, Observation: []string{"seen"}, Reliability: .9, ObservedAt: at, EpisodeID: "e1", IndependenceGroup: "origin-a", CausalGroup: "cause-a"}
	if _, err = live.ObserveOutcome(one); err != nil {
		t.Fatal(err)
	}
	if got := live.evidence.Counts(1); got.Observations != 1 || got.LegacyObservations != 2 || got.IndependentGroups != 1 || live.brain.Registry.GetByID(1).Importance != importance {
		t.Fatalf("adoption fabricated old provenance: %+v", got)
	}
	prefix := orderedSave(t, live)
	if err = live.RegisterActionBinding(ActionBinding{RequestID: "request-4", TargetNodeIDs: []knowledge.NodeID{1}}); err != nil {
		t.Fatal(err)
	}
	two := one
	two.RequestID = "request-4"
	two.Source = "other-observer"
	two.IndependenceGroup = "origin-b"
	two.CausalGroup = "cause-b"
	two.ObservedAt = at.Add(time.Second)
	if _, err = live.ObserveOutcome(two); err != nil {
		t.Fatal(err)
	}
	if _, err = live.ObserveOutcome(two); err != nil {
		t.Fatal(err)
	}
	genesis, err := RebuildRuntimeFromCommittedJournal(log, knowledge.NewBrain)
	if err != nil {
		t.Fatal(err)
	}
	suffix, err := RebuildRuntimeFromCheckpointAndJournal(log, prefix)
	if err != nil {
		t.Fatal(err)
	}
	oldPrefix, err := RebuildRuntimeFromCheckpointAndJournal(log, "testdata/stable_projection_v1/prefix-checkpoint.json")
	if err != nil {
		t.Fatal(err)
	}
	standalone := orderedFixtureClone(t, orderedSave(t, live))
	live.SetEventLog(nil)
	for _, other := range []*BrainRuntime{genesis, suffix, oldPrefix, standalone} {
		assertCompleteContinuation(t, live, other)
	}
	for _, r := range []*BrainRuntime{live, genesis, suffix, oldPrefix, standalone} {
		if _, err = r.ObserveOutcome(two); err != nil {
			t.Fatal(err)
		}
		if err = r.RegisterActionBinding(ActionBinding{RequestID: "next", TargetNodeIDs: []knowledge.NodeID{1}}); err != nil {
			t.Fatal(err)
		}
		next := one
		next.RequestID = "next"
		next.ObservedAt = at.Add(2 * time.Second)
		next.EpisodeID = "e2"
		if _, err = r.ObserveOutcome(next); err != nil {
			t.Fatal(err)
		}
		r.SetClock(NewFixedClock(at.Add(3 * time.Second)))
		if _, err = r.CognitiveThink(1); err != nil {
			t.Fatal(err)
		}
	}
	for _, other := range []*BrainRuntime{genesis, suffix, oldPrefix, standalone} {
		assertCompleteContinuation(t, live, other)
	}
	if got := live.evidence.Counts(1); got.Observations != 3 || got.IndependentGroups != 2 || got.LegacyObservations != 2 {
		t.Fatalf("next request counted as another independent source: %+v", got)
	}
}

func TestEvidenceProvenanceInquiryCarriesLineageAndRejectsRetriedOutcomeBeforeWAL(t *testing.T) {
	r := NewBrainRuntime(knowledge.NewBrain())
	log := openKnowledgeJournal(t)
	r.SetEventLog(log)
	at := time.Unix(100, 0).UTC()
	r.SetClock(NewFixedClock(at))
	g, err := r.GroundVectorAt(knowledge.NewNeuralVector([]float64{.2, .8}), at)
	if err != nil {
		t.Fatal(err)
	}
	if err = r.RegisterActionBinding(ActionBinding{RequestID: "inquiry-12-focus", TargetNodeIDs: []knowledge.NodeID{g.NodeID}}); err != nil {
		t.Fatal(err)
	}
	if _, err = r.LearnExperience(learning.Experience{Sequence: []string{"cue", "expected"}, Weight: .8, Confidence: .8, Source: "teacher", Modality: "text"}, at); err != nil {
		t.Fatal(err)
	}
	if _, err = r.CognitiveProcess(Event{ID: "cue-before-action", Timestamp: at, Stimulus: []string{"cue"}, Cycles: 1}); err != nil {
		t.Fatal(err)
	}
	execution := transactionInquiry(t, r, at)
	result := InquiryResult{Execution: execution, Event: Event{ID: "outcome-one", EpisodeID: "scene-episode", Timestamp: at.Add(time.Second), Cycles: 1}, Observation: ObservationInput{Source: "observer", Modality: "vision", Tokens: []string{"observed"}}, Success: true, Reliability: .9, InformationGain: .5, IndependenceGroup: "declared-origin", CausalGroup: "common-scene"}
	if _, _, _, err = NewCognitiveOrchestrator(r).ProcessInquiryOutcome(result, at.Add(time.Second)); err != nil {
		t.Fatal(err)
	}
	counts := r.evidence.Counts(g.NodeID)
	if counts.Episodes != 1 || counts.IndependentGroups != 1 || counts.CausalGroups != 1 {
		t.Fatalf("inquiry dropped lineage: %+v", counts)
	}
	found := false
	for _, p := range r.brain.Patterns.All() {
		for _, e := range p.Evidence {
			if e.Source == "observer" && e.EpisodeID == "scene-episode" && e.IndependenceGroup == "declared-origin" && e.CausalGroup == "common-scene" {
				found = true
			}
		}
	}
	if !found {
		t.Fatal("neural outcome trace lost supplied provenance")
	}
	before := continuationBytes(t, r)
	data, err := os.ReadFile(log.file.Name())
	if err != nil {
		t.Fatal(err)
	}
	result.Event.ID = "different-event-same-outcome"
	result.Observation.Tokens = []string{"must-not-ground"}
	if _, _, _, err = NewCognitiveOrchestrator(r).ProcessInquiryOutcome(result, at.Add(time.Second)); err == nil {
		t.Fatal("new event ID bypassed outcome idempotency")
	}
	after, err := os.ReadFile(log.file.Name())
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(before, continuationBytes(t, r)) || !bytes.Equal(data, after) || r.RecoveryRequired() {
		t.Fatal("retry changed state/WAL or poisoned runtime")
	}
	prefix := orderedSave(t, r)
	result.Event.ID = "relay-outcome"
	result.Observation.Source = "relay"
	result.Observation.Tokens = []string{"observed"}
	result.IndependenceGroup = "relay-declaration"
	if _, _, _, err = NewCognitiveOrchestrator(r).ProcessInquiryOutcome(result, at.Add(time.Second)); err != nil {
		t.Fatal(err)
	}
	if got := r.evidence.Counts(g.NodeID); got.Observations != 2 || got.Sources != 2 || got.IndependentGroups != 1 {
		t.Fatalf("compound outcome ignored shared cause: %+v", got)
	}
	identities := make(map[string]bool)
	for _, p := range r.brain.Patterns.All() {
		for _, e := range p.Evidence {
			if e.CausalLink == "inquiry:inquiry-12-focus" {
				identities[e.ExperienceID] = true
			}
		}
	}
	if len(identities) != 2 {
		t.Fatal("separate source reports collapsed into one neural evidence ID")
	}
	genesis, err := RebuildRuntimeFromCommittedJournal(log, knowledge.NewBrain)
	if err != nil {
		t.Fatal(err)
	}
	assertCompleteContinuation(t, r, genesis)
	suffix, err := RebuildRuntimeFromCheckpointAndJournal(log, prefix)
	if err != nil {
		t.Fatal(err)
	}
	assertCompleteContinuation(t, r, suffix)
	standalone := orderedFixtureClone(t, orderedSave(t, r))
	assertCompleteContinuation(t, r, standalone)
}

func TestEvidenceProvenanceCheckpointAuthenticatesLineageAndRejectsOldEnvelope(t *testing.T) {
	r := NewBrainRuntime(knowledge.NewBrain())
	at := time.Unix(100, 0).UTC()
	r.SetClock(NewFixedClock(at))
	g, err := r.GroundVectorAt(knowledge.NewNeuralVector([]float64{.2, .8}), at)
	if err != nil {
		t.Fatal(err)
	}
	if err = r.RegisterActionBinding(ActionBinding{RequestID: "r", TargetNodeIDs: []knowledge.NodeID{g.NodeID}}); err != nil {
		t.Fatal(err)
	}
	if _, err = r.ObserveOutcome(OutcomeEvent{RequestID: "r", Observation: []string{"seen"}, Source: "camera", Success: true, ObservedAt: at, EpisodeID: "episode", IndependenceGroup: "group", CausalGroup: "cause"}); err != nil {
		t.Fatal(err)
	}
	path := orderedSave(t, r)
	original, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	for _, field := range []string{"source", "episode", "group", "cause", "old-envelope", "unknown-target"} {
		t.Run(field, func(t *testing.T) {
			var s BrainSnapshot
			if err = json.Unmarshal(original, &s); err != nil {
				t.Fatal(err)
			}
			p := s.EvidenceLedger.Provenance[g.NodeID]
			switch field {
			case "source":
				p.Records[0].Source = "tampered"
			case "episode":
				p.Records[0].EpisodeID = "tampered"
			case "group":
				p.Records[0].IndependenceGroup = "tampered"
			case "cause":
				p.Records[0].CausalGroup = "tampered"
			case "old-envelope":
				s.SchemaVersion = observedFeedbackBrainSnapshotSchemaVersion
			case "unknown-target":
				p.Records[0].NodeID = 999999
			}
			s.EvidenceLedger.Provenance[g.NodeID] = p
			if field == "old-envelope" || field == "unknown-target" {
				s.Checksum, err = checkpointChecksumV7(s)
				if err != nil {
					t.Fatal(err)
				}
			}
			data, err := json.Marshal(s)
			if err != nil {
				t.Fatal(err)
			}
			tampered := filepath.Join(t.TempDir(), "tampered.json")
			if err = os.WriteFile(tampered, data, 0600); err != nil {
				t.Fatal(err)
			}
			candidate := NewBrainRuntime(knowledge.NewBrain())
			before := continuationBytes(t, candidate)
			if err = candidate.RestoreCheckpoint(tampered); err == nil {
				t.Fatal("invalid provenance checkpoint accepted")
			}
			if !bytes.Equal(before, continuationBytes(t, candidate)) {
				t.Fatal("invalid typed continuation mutated runtime")
			}
		})
	}
}

func TestEvidenceProvenanceExperienceKeepsFrequencyWithoutSyntheticIndependence(t *testing.T) {
	r := NewBrainRuntime(knowledge.NewBrain())
	at := time.Unix(100, 0).UTC()
	r.SetClock(NewFixedClock(at))
	var confidence float64
	for i := 0; i < 4; i++ {
		event := Event{ID: fmt.Sprintf("observation-%d", i), EpisodeID: fmt.Sprintf("episode-%d", i), Timestamp: at.Add(time.Duration(i) * time.Second)}
		e := NewExperienceFromObservation(event, ObservationInput{Source: "same-teacher", Modality: "text"}, []string{"cue", "outcome"})
		if e.IndependenceGroup != "" || e.EpisodeID != event.EpisodeID {
			t.Fatal("observation identity became an independence declaration")
		}
		if _, err := r.LearnExperience(e, event.Timestamp); err != nil {
			t.Fatal(err)
		}
		patterns := r.brain.Patterns.All()
		if len(patterns) != 1 {
			t.Fatal("repeated experience created parallel patterns")
		}
		p := patterns[0]
		if i == 0 {
			confidence = p.Confidence
		} else if p.Confidence != confidence {
			t.Fatal("same-teacher repetition inflated calibrated independent confidence")
		}
		counts := knowledge.CountEvidenceProvenance(p.Evidence)
		if counts.Observations != i+1 || counts.Episodes != i+1 || counts.Sources != 1 || counts.IndependentGroups != 0 {
			t.Fatalf("experience provenance: %+v", counts)
		}
	}
	if _, err := r.CognitiveProcess(Event{ID: "after-experience", Timestamp: at.Add(4 * time.Second), Stimulus: []string{"cue"}, Cycles: 1}); err != nil {
		t.Fatal(err)
	}
	standalone := orderedFixtureClone(t, orderedSave(t, r))
	assertCompleteContinuation(t, r, standalone)
}

func TestEvidenceProvenanceEmptyAndUnsupportedMetadataRemainConservative(t *testing.T) {
	r := NewBrainRuntime(knowledge.NewBrain())
	at := time.Unix(100, 0).UTC()
	g, err := r.GroundVectorAt(knowledge.NewNeuralVector([]float64{.2, .8}), at)
	if err != nil {
		t.Fatal(err)
	}
	importance := r.brain.Registry.GetByID(g.NodeID).Importance
	for i := 0; i < 3; i++ {
		id := fmt.Sprint(i)
		if err = r.RegisterActionBinding(ActionBinding{RequestID: id, TargetNodeIDs: []knowledge.NodeID{g.NodeID}}); err != nil {
			t.Fatal(err)
		}
		if _, err = r.ObserveOutcome(OutcomeEvent{RequestID: id, Observation: []string{"seen"}, Success: true, Reliability: .9, ObservedAt: at}); err != nil {
			t.Fatal(err)
		}
	}
	if got := r.evidence.Counts(g.NodeID); got.Observations != 3 || got.IndependentGroups != 0 || r.brain.Registry.GetByID(g.NodeID).Importance != importance {
		t.Fatalf("default provenance invented independent support: %+v", got)
	}
	r.dynamicsEncoding = knowledge.StableProjectionDynamics
	before := continuationBytes(t, r)
	if _, err = r.ObserveOutcome(OutcomeEvent{RequestID: "0", Observation: []string{"seen"}, Source: "camera", Success: true, ObservedAt: at, IndependenceGroup: "unsupported"}); err == nil {
		t.Fatal("old dynamics silently ignored new provenance")
	}
	if !bytes.Equal(before, continuationBytes(t, r)) || r.RecoveryRequired() {
		t.Fatal("unsupported metadata mutated runtime")
	}
}
