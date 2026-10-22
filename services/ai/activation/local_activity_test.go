package activation

import (
	"math"
	"reflect"
	"testing"
	"time"

	"github.com/project-horizon/horizon-core/services/ai/knowledge"
)

func localActivityFixture(t *testing.T, inhibitory bool) (*Engine, *knowledge.Synapse, knowledge.NodeID, knowledge.NodeID, time.Time) {
	t.Helper()
	brain := knowledge.NewBrain()
	at := time.Date(2026, 10, 20, 3, 0, 0, 123, time.UTC)
	source, target := brain.StoreAt("source", at), brain.StoreAt("target", at)
	brain.ConnectAt(source, target, .7, .8, inhibitory, at)
	return NewEngine(brain), source.FindDynamicSynapse(target.ID, inhibitory), source.ID, target.ID, at
}

func TestLocalActivityInactivePathsDoNotForgetByCallCount(t *testing.T) {
	for _, inhibitory := range []bool{false, true} {
		for _, participation := range []string{"empty", "source-only", "target-only", "subthreshold"} {
			t.Run(participation+map[bool]string{false: "/excitatory", true: "/inhibitory"}[inhibitory], func(t *testing.T) {
				e, s, source, target, at := localActivityFixture(t, inhibitory)
				s.Dynamic.Eligibility = .5
				before := s.Dynamic
				pre, post := map[knowledge.NodeID]float64{}, map[knowledge.NodeID]float64{}
				switch participation {
				case "source-only":
					pre[source] = 1
				case "target-only":
					post[target] = 1
				case "subthreshold":
					pre[source] = .1
					post[target] = .1
				}
				for i := 0; i < 96; i++ {
					e.ApplyActivityPlasticityWithDynamics(pre, post, at, knowledge.LocalActivityDynamics)
				}
				if s.Weight != before.Weight || s.Confidence != before.Confidence || s.Dynamic.Frequency != before.Frequency || !s.Dynamic.LastModification.Equal(before.LastModification) || s.Dynamic.Eligibility < before.Eligibility {
					t.Fatal("inactive/subthreshold activity changed learned strength or forgot eligibility")
				}
				if participation != "subthreshold" && !reflect.DeepEqual(before, s.Dynamic) {
					t.Fatal("zero coactivity changed adaptive state")
				}
			})
		}
	}
}

func TestLocalActivityEligibilityUsesOnlyElapsedTimeWhenInactive(t *testing.T) {
	for _, steps := range []int{1, 24, 96} {
		e, s, _, _, at := localActivityFixture(t, false)
		before := s.Dynamic
		for i := 1; i <= steps; i++ {
			e.ApplyActivityPlasticityWithDynamics(nil, nil, at.Add(time.Duration(i)*2*time.Second/time.Duration(steps)), knowledge.LocalActivityDynamics)
		}
		if math.Abs(s.Dynamic.Eligibility-.25) > 1e-12 || s.Dynamic.Weight != before.Weight || s.Dynamic.Confidence != before.Confidence || s.Dynamic.Frequency != before.Frequency || !s.Dynamic.LastModification.Equal(before.LastModification) {
			t.Fatal("empty updates charged call-count decay or changed learned state")
		}
		settled := s.Dynamic
		for i := 0; i < 96; i++ {
			e.ApplyActivityPlasticityWithDynamics(nil, nil, at.Add(2*time.Second), knowledge.LocalActivityDynamics)
		}
		e.ApplyActivityPlasticityWithDynamics(nil, nil, at, knowledge.LocalActivityDynamics)
		if !reflect.DeepEqual(settled, s.Dynamic) {
			t.Fatal("duplicate/rollback changed eligibility or rewound its clock")
		}
	}
}

func TestLocalActivityCoactivityStillLearnsAndAccumulatesEvidence(t *testing.T) {
	for _, inhibitory := range []bool{false, true} {
		e, s, source, target, at := localActivityFixture(t, inhibitory)
		s.Dynamic.Eligibility = 0
		initialWeight := s.Dynamic.Weight
		lastEligibility := 0.0
		for i := 0; i < 3; i++ {
			e.ApplyActivityPlasticityWithDynamics(map[knowledge.NodeID]float64{source: .9}, map[knowledge.NodeID]float64{target: .9}, at, knowledge.LocalActivityDynamics)
			if s.Dynamic.Eligibility <= lastEligibility || s.Dynamic.Eligibility > 1 {
				t.Fatal("observed coactivity did not add bounded eligibility")
			}
			lastEligibility = s.Dynamic.Eligibility
		}
		if s.Dynamic.Weight <= initialWeight || s.Weight != s.Dynamic.Weight || s.Dynamic.Weight > activityPlasticityMaxWeight {
			t.Fatal("local rule froze or unboundedly strengthened a coactive path")
		}
	}
}

func TestLocalActivityKeepsContradictionAndConsequenceAdaptation(t *testing.T) {
	e, s, source, target, at := localActivityFixture(t, false)
	e.ApplyActivityPlasticityWithDynamics(map[knowledge.NodeID]float64{source: 1}, map[knowledge.NodeID]float64{target: 1}, at, knowledge.LocalActivityDynamics)
	before := s.Weight
	e.ApplyPredictionErrorPlasticityWithDynamics(map[knowledge.NodeID]float64{source: 1, target: 1}, map[knowledge.NodeID]float64{source: 1, target: .1}, 1, at, knowledge.LocalActivityDynamics)
	if s.Weight >= before {
		t.Fatal("protection disabled negative prediction learning")
	}
	negative := s.Weight
	e.ApplyPredictionErrorPlasticityWithDynamics(map[knowledge.NodeID]float64{source: 1, target: .1}, map[knowledge.NodeID]float64{source: 1, target: .9}, 1, at, knowledge.LocalActivityDynamics)
	if s.Weight <= negative {
		t.Fatal("protection disabled positive prediction learning")
	}
	positive := s.Weight
	e.ApplyConsequencePlasticityWithDynamics(-.8, .7, at, knowledge.LocalActivityDynamics)
	if s.Weight >= positive {
		t.Fatal("protection disabled negative consequence learning")
	}
	negative = s.Weight
	e.ApplyConsequencePlasticityWithDynamics(.8, .7, at, knowledge.LocalActivityDynamics)
	if s.Weight <= negative || e.Memory.Registry.GetByID(source).FindDynamicSynapse(target, false) != s {
		t.Fatal("consequence learning failed or replaced the canonical connection")
	}
}

func TestLocalActivityPredictionCreditRequiresTheSource(t *testing.T) {
	for _, dynamics := range []knowledge.DynamicsEncoding{knowledge.LegacyDynamics, knowledge.OrderedDynamics, knowledge.RecurrentDynamics, knowledge.PhysicalMemoryDynamics, knowledge.LocalActivityDynamics} {
		e, s, source, target, at := localActivityFixture(t, false)
		e.Memory.Registry.GetByID(source).Activation = 1 // Deliberately stale legacy node state.
		before := s.Dynamic
		e.ApplyPredictionErrorPlasticityWithDynamics(map[knowledge.NodeID]float64{target: .9}, map[knowledge.NodeID]float64{target: .1}, 1, at, dynamics)
		if dynamics == knowledge.LocalActivityDynamics {
			if !reflect.DeepEqual(before, s.Dynamic) {
				t.Fatal("inactive source received unrelated target error")
			}
		} else if s.Weight >= before.Weight {
			t.Fatal("historical prediction law changed")
		}
	}
	for _, previous := range []bool{false, true} {
		e, s, source, target, at := localActivityFixture(t, false)
		predicted, actual := map[knowledge.NodeID]float64{target: .9}, map[knowledge.NodeID]float64{target: .1}
		if previous {
			predicted[source] = 1
		} else {
			actual[source] = 1
		}
		before := s.Weight
		e.ApplyPredictionErrorPlasticityWithDynamics(predicted, actual, 1, at, knowledge.LocalActivityDynamics)
		if s.Weight >= before {
			t.Fatal("active or delayed source lost error credit")
		}
	}
}

func TestLocalActivityErrorAndConsequenceDoNotRewindEligibility(t *testing.T) {
	e, s, source, target, at := localActivityFixture(t, false)
	e.ApplyActivityPlasticityWithDynamics(nil, nil, at.Add(time.Second), knowledge.LocalActivityDynamics)
	e.ApplyPredictionErrorPlasticityWithDynamics(map[knowledge.NodeID]float64{source: 1, target: .9}, map[knowledge.NodeID]float64{source: 1, target: .1}, 1, at, knowledge.LocalActivityDynamics)
	e.ApplyConsequencePlasticityWithDynamics(-.5, .5, at, knowledge.LocalActivityDynamics)
	if !s.Dynamic.LastEligibilityUpdate.Equal(at.Add(time.Second)) || s.Dynamic.Eligibility != .5 {
		t.Fatal("backdated learning rewound the eligibility clock or charged its decay twice")
	}
}

func TestLocalActivityRoutesObservationEmptyFallbackAndInternalThought(t *testing.T) {
	e, s, _, _, at := localActivityFixture(t, false)
	noise := e.Memory.StoreAt("noise", at).ID
	before := s.Dynamic.Weight
	dynamics := knowledge.LocalActivityDynamics
	for i := 0; i < 96; i++ {
		e.ActivateWith(Request{DynamicsEncoding: &dynamics, StimulusNodeIDs: []knowledge.NodeID{noise}, Cycles: 1, Now: at})
		e.ActivateWith(Request{DynamicsEncoding: &dynamics, NeuralTrace: &knowledge.ObservationTrace{}, ContextBoosts: map[knowledge.NodeID]float64{noise: .5}, Cycles: 1, Now: at})
		e.ThinkWithContextDynamicsAt(1, nil, at, dynamics)
	}
	if s.Dynamic.Weight != before {
		t.Fatal("an observation/fallback/thought path bypassed local activity dynamics")
	}
}
