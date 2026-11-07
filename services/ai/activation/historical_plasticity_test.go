package activation

// Frozen reference laws from d7cb140; compare complete canonical bytes.
import (
	"bytes"
	"fmt"
	"github.com/project-horizon/horizon-core/services/ai/knowledge"
	"math/rand"
	"testing"
	"time"
)

func (e *Engine) referenceActivity(preState, postState map[knowledge.NodeID]float64, now time.Time, local bool) {
	if e == nil || e.Memory == nil {
		return
	}
	if now.IsZero() {
		now = time.Now().UTC()
	}
	e.Memory.Lock()
	defer e.Memory.Unlock()

	for _, source := range e.Memory.Registry.Nodes() {
		pre := clamp01(preState[source.ID])
		for _, synapse := range source.OutboundAll() {
			if synapse == nil {
				continue
			}

			target := clamp01(postState[synapse.TargetID])
			coactivity := pre * target
			eligibility := decayEligibility(synapse.Dynamic.Eligibility, synapse.Dynamic.LastEligibilityUpdate, now)

			if local {
				// Only elapsed time forgets eligibility. Actual co-activity
				// adds bounded evidence; empty calls cannot multiply it away.
				eligibility = clamp01(eligibility + .15*coactivity*(1-eligibility))
			} else {
				eligibility = clamp01(eligibility*0.85 + coactivity*0.15)
			}
			synapse.Dynamic.Eligibility = eligibility
			if !local || synapse.Dynamic.LastEligibilityUpdate.IsZero() || now.After(synapse.Dynamic.LastEligibilityUpdate) {
				synapse.Dynamic.LastEligibilityUpdate = now
			}

			if synapse.Inhibitory {
				if coactivity >= activityPlasticityMinActive {
					delta := activityPlasticityLearningRate * coactivity * eligibility
					synapse.Dynamic.Weight = clamp(synapse.Dynamic.Weight+delta, activityPlasticityMinWeight, activityPlasticityMaxWeight)
					synapse.Dynamic.LastModification = now
				}
				synapse.Weight = synapse.Dynamic.Weight
				continue
			}

			if coactivity >= activityPlasticityMinActive {
				delta := activityPlasticityLearningRate * coactivity * eligibility
				synapse.Dynamic.Weight = clamp(synapse.Dynamic.Weight+delta, activityPlasticityMinWeight, activityPlasticityMaxWeight)
				synapse.Dynamic.Confidence = clamp01(synapse.Dynamic.Confidence + delta*0.5)
				synapse.Dynamic.Frequency++
				synapse.Dynamic.LastActivation = now
				synapse.Dynamic.LastModification = now
			} else if !local {
				decay := activityPlasticityDecayRate * (1 - eligibility)
				synapse.Dynamic.Weight = clamp(synapse.Dynamic.Weight-decay, activityPlasticityMinWeight, activityPlasticityMaxWeight)
				synapse.Dynamic.Confidence = clamp01(synapse.Dynamic.Confidence - decay*0.25)
			}

			synapse.Weight = synapse.Dynamic.Weight
			synapse.Confidence = synapse.Dynamic.Confidence
		}
	}
}

func (e *Engine) referencePrediction(predicted, actual map[knowledge.NodeID]float64, errorSignal float64, now time.Time, local bool) {
	if e == nil || e.Memory == nil || errorSignal <= 0 {
		return
	}
	if now.IsZero() {
		now = time.Now().UTC()
	}
	errorSignal = clamp01(errorSignal)

	// Canonical brain mutation boundary. Memory optimization, checkpointing,
	// activation and all plasticity paths must serialize against the same
	// KnowledgeBase lock when they mutate canonical synapse state.
	e.Memory.Lock()
	defer e.Memory.Unlock()

	for _, source := range e.Memory.Registry.Nodes() {
		if local && max(clamp01(predicted[source.ID]), clamp01(actual[source.ID])) < activityPlasticityMinActive {
			continue
		}
		for _, synapse := range source.OutboundAll() {
			if synapse == nil || synapse.Inhibitory {
				continue
			}
			targetActual := actual[synapse.TargetID]
			targetPredicted := predicted[synapse.TargetID]
			direction := targetActual - targetPredicted
			if abs(direction) < 0.001 {
				continue
			}

			eligibility := decayPredictionEligibility(synapse.Dynamic.Eligibility, synapse.Dynamic.LastEligibilityUpdate, now)
			if eligibility <= 0 {
				eligibility = clamp01(source.Activation * synapse.Activation)
			}
			if eligibility <= 0 {
				continue
			}

			synapse.Dynamic.Eligibility = eligibility
			if !local || synapse.Dynamic.LastEligibilityUpdate.IsZero() || now.After(synapse.Dynamic.LastEligibilityUpdate) {
				synapse.Dynamic.LastEligibilityUpdate = now
			}
			magnitude := predictionLearningRate * errorSignal * clamp01(abs(direction)) * eligibility
			if direction > 0 {
				synapse.Dynamic.Weight = clamp(synapse.Dynamic.Weight+magnitude, predictionMinWeight, predictionMaxWeight)
				synapse.Dynamic.Confidence = clamp01(synapse.Dynamic.Confidence + magnitude*0.5)
			} else {
				synapse.Dynamic.Weight = clamp(synapse.Dynamic.Weight-magnitude, predictionMinWeight, predictionMaxWeight)
				synapse.Dynamic.Confidence = clamp01(synapse.Dynamic.Confidence - magnitude*0.5)
			}
			synapse.Dynamic.LastModification = now

			synapse.Weight = synapse.Dynamic.Weight
			synapse.Confidence = synapse.Dynamic.Confidence
		}
	}

	// Prediction error also informs structural competition. This is a bounded
	// reconsolidation signal, not a semantic contradiction detector: competing
	// traces remain in the same Brain and retain their evidence history.
	e.Memory.Patterns.ReconsolidateFromState(predicted, actual, errorSignal)
}

func (e *Engine) referenceConsequence(valence, informationGain float64, now time.Time, local bool) {
	if e == nil || e.Memory == nil || valence == 0 {
		return
	}
	if now.IsZero() {
		now = time.Now().UTC()
	}

	// Consequence learning mutates the same canonical synapse state as
	// prediction plasticity and memory optimization.
	e.Memory.Lock()
	defer e.Memory.Unlock()

	if valence < -1 {
		valence = -1
	} else if valence > 1 {
		valence = 1
	}
	informationGain = clamp01(informationGain)
	gain := 0.5 + 0.5*informationGain

	for _, source := range e.Memory.Registry.Nodes() {
		for _, synapse := range source.OutboundAll() {
			if synapse == nil || synapse.Inhibitory {
				continue
			}
			eligibility := decayPredictionEligibility(synapse.Dynamic.Eligibility, synapse.Dynamic.LastEligibilityUpdate, now)
			if eligibility <= 0 {
				continue
			}
			magnitude := consequenceLearningRate * abs(valence) * gain * eligibility
			if valence > 0 {
				synapse.Dynamic.Weight = clamp(synapse.Dynamic.Weight+magnitude, predictionMinWeight, predictionMaxWeight)
				synapse.Dynamic.Confidence = clamp01(synapse.Dynamic.Confidence + magnitude*0.5)
			} else {
				synapse.Dynamic.Weight = clamp(synapse.Dynamic.Weight-magnitude, predictionMinWeight, predictionMaxWeight)
				synapse.Dynamic.Confidence = clamp01(synapse.Dynamic.Confidence - magnitude*0.5)
			}
			synapse.Dynamic.LastModification = now
			if !local || synapse.Dynamic.LastEligibilityUpdate.IsZero() || now.After(synapse.Dynamic.LastEligibilityUpdate) {
				synapse.Dynamic.LastEligibilityUpdate = now
			}
			synapse.Dynamic.Eligibility = eligibility
			synapse.Weight = synapse.Dynamic.Weight
			synapse.Confidence = synapse.Dynamic.Confidence
		}
	}
}

func TestIndependentPlasticityMatchesHistoricalBytes(t *testing.T) {
	at := time.Date(2026, 11, 6, 0, 0, 0, 0, time.UTC)
	fixture := func() *Engine {
		b := knowledge.NewBrain()
		nodes := []*knowledge.ConceptNode{}
		for i := 0; i < 32; i++ {
			nodes = append(nodes, b.StoreAt(fmt.Sprintf("unit%d", i), at))
		}
		for i, n := range nodes {
			for j := 1; j <= 4; j++ {
				b.ConnectAt(n, nodes[(i+j)%len(nodes)], .2+.1*float64(j), .6, j%3 == 0, at)
			}
		}
		return NewEngine(b)
	}
	for _, local := range []bool{false, true} {
		old, candidate := fixture(), fixture()
		rng := rand.New(rand.NewSource(811))
		nodes := old.Memory.Registry.Nodes()
		for step := 0; step < 100; step++ {
			pre, post := map[knowledge.NodeID]float64{}, map[knowledge.NodeID]float64{}
			for _, n := range nodes {
				if rng.Float64() < .15 {
					pre[n.ID] = rng.Float64()
				}
				if rng.Float64() < .15 {
					post[n.ID] = rng.Float64()
				}
			}
			now := at.Add(time.Duration(step) * 37 * time.Millisecond)
			valence := rng.Float64()*2 - 1
			old.referenceActivity(pre, post, now, local)
			candidate.applyActivityPlasticity(pre, post, now, local)
			old.referencePrediction(pre, post, .5, now, local)
			candidate.applyPredictionErrorPlasticity(pre, post, .5, now, local)
			old.referenceConsequence(valence, .6, now, local)
			candidate.applyConsequencePlasticity(valence, .6, now, local)
			a, _ := old.Memory.CanonicalJSON()
			b, _ := candidate.Memory.CanonicalJSON()
			if !bytes.Equal(a, b) {
				t.Fatalf("canonical numerical regression local=%v step=%d", local, step)
			}
		}
	}
}
