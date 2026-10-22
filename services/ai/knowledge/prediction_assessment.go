package knowledge

import (
	"encoding/json"
	"math"
	"slices"
	"sort"
)

// PopulationAlternative describes an existing learned outcome population.
// Probability is an evidence-weighted estimate, not normalized activation.
type PopulationAlternative struct {
	NodeIDs     []NodeID `json:"node_ids"`
	Probability float64  `json:"probability"`
}

// PredictionAssessment is a detached readout of existing learned channels.
// It adds no answer memory and assigns no semantic meaning to neurons.
type PredictionAssessment struct {
	Scope              string                  `json:"scope"`
	NodeIDs            []NodeID                `json:"node_ids,omitempty"`
	Confidence         float64                 `json:"confidence"`
	UnknownProbability float64                 `json:"unknown_probability"`
	EvidenceMass       float64                 `json:"evidence_mass"`
	ContextSupport     float64                 `json:"context_support"`
	Ambiguity          float64                 `json:"ambiguity"`
	Alternatives       []PopulationAlternative `json:"alternatives,omitempty"`
}

func UnknownPredictionAssessment() *PredictionAssessment {
	return &PredictionAssessment{Scope: "learned-next-population", UnknownProbability: 1}
}

func (a *PredictionAssessment) Clone() *PredictionAssessment {
	if a == nil {
		return nil
	}
	out := *a
	out.NodeIDs = slices.Clone(a.NodeIDs)
	out.Alternatives = slices.Clone(a.Alternatives)
	for i := range out.Alternatives {
		out.Alternatives[i].NodeIDs = slices.Clone(a.Alternatives[i].NodeIDs)
	}
	return &out
}

// AssessObservedPrediction uses only the best cue/context channels already
// credited by PredictRecurrent. Population members are averaged, not counted as
// independent evidence or competing outcomes. Duplicate paths use their maximum
// support. One unit of unknown mass prevents a lone observation implying certainty.
func (p *PatternIndex) AssessObservedPrediction(credit map[PatternID]float64, predicted map[NodeID]float64) *PredictionAssessment {
	out := UnknownPredictionAssessment()
	if p == nil || len(credit) == 0 {
		return out
	}
	type group struct {
		population []NodeID
		mass       map[NodeID]float64
		quality    map[NodeID]float64
	}
	groups := map[string]*group{}
	ids := make([]PatternID, 0, len(credit))
	for id := range credit {
		ids = append(ids, id)
	}
	slices.Sort(ids)
	p.mu.RLock()
	defer p.mu.RUnlock()
	for _, id := range ids {
		pattern := p.patterns[id]
		quality := finiteUnit(credit[id])
		if pattern == nil || quality == 0 || pattern.Dynamics != OrderedDynamics || pattern.Observation == nil || len(pattern.Observation.Stimulus) < 2 {
			continue
		}
		population := slices.Clone(pattern.Observation.Stimulus[len(pattern.Observation.Stimulus)-1])
		slices.Sort(population)
		population = slices.Compact(population)
		if len(population) == 0 || !slices.Contains(population, pattern.Result) {
			continue
		}
		key, _ := json.Marshal(population)
		g := groups[string(key)]
		if g == nil {
			g = &group{population: population, mass: map[NodeID]float64{}, quality: map[NodeID]float64{}}
			groups[string(key)] = g
		}
		// Frequency counts experiences, not independent sources. The cap is
		// numerical protection; it is not an independence or accuracy claim.
		frequency := min(1e6, float64(max(int64(0), pattern.Frequency)))
		mass := frequency * finiteUnit(pattern.Weight) * finiteUnit(pattern.Confidence) * quality
		g.mass[pattern.Result] = max(g.mass[pattern.Result], mass)
		g.quality[pattern.Result] = max(g.quality[pattern.Result], quality)
	}
	var evidence []populationEvidence
	for _, g := range groups {
		mass, quality := 0.0, 1.0
		for _, id := range g.population {
			mass += g.mass[id]
			quality = min(quality, g.quality[id])
		}
		mass /= float64(len(g.population))
		evidence = append(evidence, populationEvidence{g.population, mass, quality})
	}
	return assessPopulationEvidence(evidence, predicted)
}

type populationEvidence struct {
	population    []NodeID
	mass, quality float64
}

// AssessObservedPrediction first reads matching ordered channels. In their
// absence it reads actual population transitions retained by the same Brain.
// Synaptic conductance weights those empirical counts. Inferred targets and
// pure thought are never counted as additional outcome observations.
func (k *KnowledgeBase) AssessObservedPrediction(source []NodeID, credit map[PatternID]float64, predicted map[NodeID]float64) *PredictionAssessment {
	if k == nil {
		return UnknownPredictionAssessment()
	}
	k.mu.RLock()
	defer k.mu.RUnlock()
	if len(credit) > 0 {
		return k.Patterns.AssessObservedPrediction(credit, predicted)
	}
	source = slices.Clone(source)
	slices.Sort(source)
	source = slices.Compact(source)
	if len(source) == 0 {
		return UnknownPredictionAssessment()
	}
	type counted struct {
		population []NodeID
		count      float64
	}
	groups := map[string]*counted{}
	for _, trace := range k.BrainState.ExperienceTraces {
		previous := slices.Clone(trace.PreviousPopulation)
		slices.Sort(previous)
		previous = slices.Compact(previous)
		if !slices.Equal(source, previous) {
			continue
		}
		population := slices.Clone(trace.CurrentPopulation)
		slices.Sort(population)
		population = slices.Compact(population)
		if len(population) == 0 {
			continue
		}
		key, _ := json.Marshal(population)
		g := groups[string(key)]
		if g == nil {
			g = &counted{population: population}
			groups[string(key)] = g
		}
		g.count++
	}
	var evidence []populationEvidence
	for _, g := range groups {
		conductance := 0.0
		for _, sourceID := range source {
			node := k.Registry.GetByID(sourceID)
			if node == nil {
				continue
			}
			for _, targetID := range g.population {
				strength := 0.0
				for _, synapse := range node.Synapses[targetID] {
					if synapse != nil && !synapse.Inhibitory {
						strength = max(strength, finiteUnit(synapse.Dynamic.Weight)*finiteUnit(synapse.Dynamic.Confidence))
					}
				}
				conductance += strength
			}
		}
		conductance /= float64(len(source) * len(g.population))
		evidence = append(evidence, populationEvidence{g.population, g.count * conductance, 1})
	}
	return assessPopulationEvidence(evidence, predicted)
}

func assessPopulationEvidence(evidence []populationEvidence, predicted map[NodeID]float64) *PredictionAssessment {
	out := UnknownPredictionAssessment()
	sort.Slice(evidence, func(i, j int) bool { return slices.Compare(evidence[i].population, evidence[j].population) < 0 })
	var masses, qualities []float64
	bestScore := 0.0
	selected := -1
	for _, g := range evidence {
		if g.mass <= 0 || g.quality <= 0 {
			continue
		}
		score := 0.0
		for _, id := range g.population {
			score += finiteUnit(predicted[id])
		}
		score /= float64(len(g.population))
		out.EvidenceMass += g.mass
		out.ContextSupport = max(out.ContextSupport, g.quality)
		if score > bestScore {
			bestScore, selected = score, len(masses)
		}
		masses = append(masses, g.mass)
		qualities = append(qualities, g.quality)
		out.Alternatives = append(out.Alternatives, PopulationAlternative{NodeIDs: slices.Clone(g.population)})
	}
	if out.EvidenceMass <= 0 {
		return out
	}
	totalKnown := 0.0
	for i, mass := range masses {
		probability := qualities[i] * mass / (1 + out.EvidenceMass)
		out.Alternatives[i].Probability = probability
		totalKnown += probability
		conditional := mass / out.EvidenceMass
		out.Ambiguity -= conditional * math.Log(conditional)
	}
	if len(masses) > 1 {
		out.Ambiguity = finiteUnit(out.Ambiguity / math.Log(float64(len(masses))))
	}
	out.UnknownProbability = finiteUnit(1 - totalKnown)
	if selected >= 0 {
		out.NodeIDs = slices.Clone(out.Alternatives[selected].NodeIDs)
		out.Confidence = out.Alternatives[selected].Probability
		out.ContextSupport = qualities[selected]
	}
	return out
}

func finiteUnit(value float64) float64 {
	if math.IsNaN(value) || math.IsInf(value, 0) {
		return 0
	}
	return max(0, min(1, value))
}
