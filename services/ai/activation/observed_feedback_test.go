package activation

import (
	"reflect"
	"testing"
	"time"

	"github.com/project-horizon/horizon-core/services/ai/knowledge"
)

func TestObservedFeedbackDoesNotCountInferredActivityAsEvidence(t *testing.T) {
	for _, dynamics := range []knowledge.DynamicsEncoding{knowledge.LocalActivityDynamics, knowledge.ObservedFeedbackDynamics} {
		e, s, source, target, at := localActivityFixture(t, false)
		before := s.Dynamic
		result := e.ActivateWith(Request{DynamicsEncoding: &dynamics, StimulusNodeIDs: []knowledge.NodeID{source}, Cycles: 1, Now: at})
		if result.Activations[target] <= 0 {
			t.Fatal("control did not produce an inferred target")
		}
		if dynamics == knowledge.ObservedFeedbackDynamics {
			if s.Dynamic.Weight != before.Weight || s.Dynamic.Confidence != before.Confidence || s.Dynamic.Frequency != before.Frequency {
				t.Fatal("inferred target became empirical coactivity")
			}
			if e.LastObservedPopulation()[target] != 0 || e.LastObservedPopulation()[source] != 1 {
				t.Fatal("observation provenance includes inferred neurons")
			}
		} else if s.Dynamic.Frequency <= before.Frequency {
			t.Fatal("legacy activity ablation did not reproduce self reinforcement")
		}
	}
}

func TestObservedFeedbackThoughtKeepsEvidenceBoundaryAndLearningSignals(t *testing.T) {
	e, s, source, target, at := localActivityFixture(t, false)
	e.Memory.Patterns.LearnOrderedTraceWithEvidence(&knowledge.ObservationTrace{Stimulus: [][]knowledge.NodeID{{source}, {target}}}, target, .7, .8, knowledge.ExperienceEvidence{Source: "test", Timestamp: at})
	dynamics := knowledge.ObservedFeedbackDynamics
	e.ActivateWith(Request{DynamicsEncoding: &dynamics, StimulusNodeIDs: []knowledge.NodeID{source}, Cycles: 1, Now: at})
	before := s.Dynamic
	boundary := e.SnapshotState().ObservationPrediction
	canonicalBefore, _ := e.Memory.CanonicalJSON()
	for i := 0; i < 20; i++ {
		e.ThinkWithContextDynamicsAt(2, nil, at, dynamics)
	}
	if s.Dynamic.Weight != before.Weight || s.Dynamic.Confidence != before.Confidence || s.Dynamic.Frequency != before.Frequency {
		t.Fatal("simulation created observed confidence/frequency/weight evidence")
	}
	if !reflect.DeepEqual(boundary, e.SnapshotState().ObservationPrediction) {
		t.Fatal("thought rewrote the pending observed forecast")
	}
	canonicalAfter, _ := e.Memory.CanonicalJSON()
	if reflect.DeepEqual(canonicalBefore, canonicalAfter) {
		t.Fatal("internal thought stopped evolving the existing substrate")
	}
	detached := e.SnapshotState()
	detached.ObservationPrediction.Source[source] = .01
	detached.ObservationPrediction.Prediction[source] = 999
	for id := range detached.ObservationPrediction.Patterns {
		detached.ObservationPrediction.Patterns[id] = .01
	}
	if reflect.DeepEqual(detached.ObservationPrediction, e.SnapshotState().ObservationPrediction) {
		t.Fatal("observation snapshot aliases the live engine")
	}
}

func TestObservedFeedbackBoundariesAndSingleConsumption(t *testing.T) {
	for _, boundary := range []string{"episode", "reset", "rollback", "expiry"} {
		e, s, source, _, at := localActivityFixture(t, false)
		other := e.Memory.StoreAt("other", at).ID
		dynamics := knowledge.ObservedFeedbackDynamics
		e.ActivateWith(Request{DynamicsEncoding: &dynamics, StimulusNodeIDs: []knowledge.NodeID{source}, EpisodeID: "one", Cycles: 1, Now: at})
		now := at
		episode := "one"
		reset := false
		switch boundary {
		case "episode":
			episode = "two"
		case "reset":
			reset = true
		case "rollback":
			now = at.Add(-time.Second)
		case "expiry":
			now = at.Add(knowledge.ContextMaxAge + time.Second)
		}
		e.Memory.ApplyMemoryDynamicsWithEncoding(now, dynamics)
		before := s.Weight
		result := e.ActivateWith(Request{DynamicsEncoding: &dynamics, StimulusNodeIDs: []knowledge.NodeID{other}, EpisodeID: episode, ResetContext: reset, Cycles: 1, Now: now})
		if s.Weight != before || result.PredictionError != 0 {
			t.Fatalf("%s boundary reused invalid source feedback", boundary)
		}
	}
	e, s, source, _, at := localActivityFixture(t, false)
	other := e.Memory.StoreAt("other", at).ID
	dynamics := knowledge.ObservedFeedbackDynamics
	e.ActivateWith(Request{DynamicsEncoding: &dynamics, StimulusNodeIDs: []knowledge.NodeID{source}, Cycles: 1, Now: at})
	e.ActivateWith(Request{DynamicsEncoding: &dynamics, StimulusNodeIDs: []knowledge.NodeID{other}, Cycles: 1, Now: at})
	before := s.Weight
	for i := 0; i < 10; i++ {
		e.ActivateWith(Request{DynamicsEncoding: &dynamics, StimulusNodeIDs: []knowledge.NodeID{other}, Cycles: 1, Now: at})
	}
	if s.Weight != before {
		t.Fatal("one old forecast received outcome credit more than once")
	}
}

func TestObservedFeedbackInternalDiscrepancyCannotEraseRealCueCredit(t *testing.T) {
	e, s, source, target, at := localActivityFixture(t, false)
	other := e.Memory.StoreAt("other", at).ID
	dynamics := knowledge.ObservedFeedbackDynamics
	e.ActivateWith(Request{DynamicsEncoding: &dynamics, StimulusNodeIDs: []knowledge.NodeID{source}, EpisodeID: "one", Cycles: 1, Now: at})
	before := s.Weight
	for i := 0; i < 10; i++ {
		e.ThinkWithContextDynamicsAt(1, map[knowledge.NodeID]float64{other: 1}, at, dynamics)
	}
	e.ActivateWith(Request{DynamicsEncoding: &dynamics, StimulusNodeIDs: []knowledge.NodeID{other}, EpisodeID: "one", Cycles: 1, Now: at.Add(time.Second)})
	if s.Weight >= before {
		t.Fatal("intervening thought replaced the observed forecast or blocked real negative feedback")
	}
	if e.LastObservedPopulation()[target] != 0 || e.LastObservedPopulation()[other] != 1 {
		t.Fatal("outcome provenance contains the earlier prediction")
	}
}
