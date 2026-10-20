package knowledge

import (
	"slices"
	"time"
)

// ApplyPredictionErrorRepresentationPlasticityWithDynamics preserves the old
// reconsolidation law for historical encodings. Sensory prototypes in the new
// law retain observed geometry; internal prediction discrepancy changes each
// directly observed neuron's learned representation once, never inactive members
// through a shared anchor or a population's cached activation.
func (k *KnowledgeBase) ApplyPredictionErrorRepresentationPlasticityWithDynamics(predicted, actual map[NodeID]float64, errorSignal, learningRate float64, now time.Time, dynamics DynamicsEncoding) {
	if !dynamics.HasStableProjection() {
		k.ApplyPredictionErrorRepresentationPlasticity(predicted, actual, errorSignal, learningRate, now)
		return
	}
	if k == nil || len(actual) == 0 || errorSignal <= 0 || learningRate <= 0 {
		return
	}
	if now.IsZero() {
		now = time.Now().UTC()
	} else {
		now = now.UTC()
	}
	k.mu.Lock()
	defer k.mu.Unlock()
	a, ok := weightedRepresentationVector(k.Registry, actual)
	if !ok {
		return
	}
	p, ok := weightedRepresentationVector(k.Registry, predicted)
	if !ok || len(a.Values) != len(p.Values) {
		return
	}
	errorVector := make([]float64, len(a.Values))
	magnitude := 0.0
	for i := range errorVector {
		errorVector[i] = a.Values[i] - p.Values[i]
		magnitude += errorVector[i] * errorVector[i]
	}
	if magnitude <= 1e-12 {
		return
	}
	k.projectionMu.Lock()
	defer k.projectionMu.Unlock()
	eligible := map[NodeID]bool{}
	for _, population := range k.ProjectionPopulations {
		for _, unit := range population.Units {
			eligible[unit.NodeID] = true
		}
	}
	updated := map[NodeID]bool{}
	ids := make([]NodeID, 0, len(actual))
	for id := range actual {
		ids = append(ids, id)
	}
	slices.Sort(ids)
	for _, id := range ids {
		if !eligible[id] {
			continue
		}
		n := k.Registry.GetByID(id)
		if n == nil || len(n.Representation) != len(errorVector) {
			continue
		}
		rate := clamp01(learningRate) * clamp01(errorSignal) * clamp01(n.Plasticity) * clamp01(actual[id])
		if rate <= 0 {
			continue
		}
		for dimension, delta := range errorVector {
			n.Representation[dimension] = clamp(n.Representation[dimension]+rate*delta, -1, 1)
		}
		n.Activation = clamp01(NewNeuralVector(n.Representation).Similarity(a))
		n.LastActivation = now
		n.Frequency++
		n.UsageHistory = append(n.UsageHistory, now)
		updated[id] = true
	}
	for i := range k.ProjectionPopulations {
		for j := range k.ProjectionPopulations[i].Units {
			u := &k.ProjectionPopulations[i].Units[j]
			if updated[u.NodeID] {
				u.Activation = k.Registry.GetByID(u.NodeID).Activation
			}
		}
	}
}
