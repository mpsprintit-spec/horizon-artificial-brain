package knowledge

import (
	"math"
	"reflect"
	"testing"
	"time"
)

func assessmentFixture(width, observations, alternatives int) (*PatternIndex, map[PatternID]float64, map[NodeID]float64) {
	p := NewPatternIndex()
	credit := map[PatternID]float64{}
	predicted := map[NodeID]float64{}
	for side := 0; side < alternatives; side++ {
		population := make([]NodeID, width)
		for i := range population {
			population[i] = NodeID(2 + side*width + i)
			predicted[population[i]] = .4
		}
		trace := &ObservationTrace{Stimulus: [][]NodeID{{1}, population}}
		for n := 0; n < observations; n++ {
			for _, id := range population {
				pattern := p.LearnOrderedTraceWithEvidence(trace, id, 1, 1, ExperienceEvidence{})
				credit[pattern.ID] = 1
			}
		}
	}
	return p, credit, predicted
}

func TestPredictionAssessmentCountsPopulationsNotMembers(t *testing.T) {
	for _, observations := range []int{1, 12} {
		for _, alternatives := range []int{1, 2} {
			var want *PredictionAssessment
			for _, width := range []int{1, 4, 32} {
				p, credit, pred := assessmentFixture(width, observations, alternatives)
				got := p.AssessObservedPrediction(credit, pred)
				total := got.UnknownProbability
				for _, alternative := range got.Alternatives {
					total += alternative.Probability
				}
				if math.Abs(total-1) > 1e-12 {
					t.Fatal("outcome probabilities and unknown mass do not normalize")
				}
				if len(got.Alternatives) != alternatives {
					t.Fatal("population members became competing outcomes")
				}
				if want != nil && (math.Abs(got.Confidence-want.Confidence) > 1e-12 || math.Abs(got.EvidenceMass-want.EvidenceMass) > 1e-12 || math.Abs(got.Ambiguity-want.Ambiguity) > 1e-12) {
					t.Fatal("population replication fabricated evidence or certainty")
				}
				want = got
				if alternatives == 2 && (got.Confidence >= .5 || got.Ambiguity < .99) {
					t.Fatal("balanced alternatives imply certainty")
				}
				got.NodeIDs[0] = 9999
				got.Alternatives[0].NodeIDs[0] = 9999
				if reflect.DeepEqual(got, p.AssessObservedPrediction(credit, pred)) {
					t.Fatal("assessment aliases pattern state")
				}
			}
		}
	}
}

func TestPredictionAssessmentWeakContextAndInvalidEvidenceRemainUncertain(t *testing.T) {
	p, credit, pred := assessmentFixture(4, 12, 1)
	for id := range credit {
		credit[id] = .1
	}
	got := p.AssessObservedPrediction(credit, pred)
	if got.Confidence > .1 || got.UnknownProbability < .9 {
		t.Fatal("repetition overrode weak context eligibility")
	}
	for id := range credit {
		credit[id] = math.NaN()
	}
	got = p.AssessObservedPrediction(credit, pred)
	if got.Confidence != 0 || got.UnknownProbability != 1 {
		t.Fatal("invalid credit manufactured support")
	}
}

func TestPredictionAssessmentOtherStrongContextCannotValidateWeakSelectedContext(t *testing.T) {
	evidence := []populationEvidence{{[]NodeID{1}, 1000, .1}, {[]NodeID{2}, 1, 1}}
	got := assessPopulationEvidence(evidence, map[NodeID]float64{1: 1, 2: .5})
	if got.Confidence > .1 || got.ContextSupport != .1 {
		t.Fatal("another alternative's context supplied false certainty to the selected forecast")
	}
}

func TestPredictionAssessmentIgnoresUnrelatedActivityAndDuplicatePaths(t *testing.T) {
	p, credit, pred := assessmentFixture(1, 12, 2)
	want := p.AssessObservedPrediction(credit, pred)
	for id := NodeID(100); id < 200; id++ {
		pred[id] = 1
	}
	var original *PatternSynapse
	for _, pattern := range p.patterns {
		original = pattern
		break
	}
	duplicate := *original
	duplicate.ID = 999
	p.patterns[duplicate.ID] = &duplicate
	credit[duplicate.ID] = 1
	if got := p.AssessObservedPrediction(credit, pred); !reflect.DeepEqual(got, want) {
		t.Fatal("unrelated activity or redundant path inflated evidence")
	}
}

func TestPredictionAssessmentReadsActualTemporalTracesWithoutMutation(t *testing.T) {
	b := NewBrain()
	at := time.Unix(100, 0).UTC()
	a := b.StoreAt("a", at).ID
	x := b.StoreAt("x", at).ID
	for i := 0; i < 12; i++ {
		if err := b.ReinforceTransitionPopulation([]NodeID{a}, []NodeID{x}, at, at, .5); err != nil {
			t.Fatal(err)
		}
	}
	before, _ := b.CanonicalJSON()
	got := b.AssessObservedPrediction([]NodeID{a}, nil, map[NodeID]float64{x: .5})
	after, _ := b.CanonicalJSON()
	if got.Confidence <= .5 || !reflect.DeepEqual(before, after) {
		t.Fatal("empirical temporal readout missing or mutated Brain")
	}
	if other := b.AssessObservedPrediction([]NodeID{x}, nil, map[NodeID]float64{a: 1}); other.Confidence != 0 {
		t.Fatal("inferred reverse direction invented observed evidence")
	}
}
