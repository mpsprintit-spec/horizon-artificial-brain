package knowledge

import (
	"encoding/json"
	"sort"
	"time"
)

// Frozen PR #40 encoder, retained only to verify byte compatibility.
func marshalCanonicalReference(k *KnowledgeBase) ([]byte, error) {
	k.projectionMu.RLock()
	defer k.projectionMu.RUnlock()
	nodes := k.Registry.Nodes()
	sort.Slice(nodes, func(i, j int) bool { return nodes[i].ID < nodes[j].ID })
	neuralUnits := make([]canonicalNeuralUnit, 0, len(nodes))
	synapses := make([]canonicalSynapse, 0)
	for _, node := range nodes {
		if node == nil {
			continue
		}
		neuralUnits = append(neuralUnits, canonicalNeuralUnit{ID: node.ID, Representation: append([]float64(nil), node.Representation...), Activation: node.Activation, RestingActivation: node.RestingActivation, Threshold: node.Threshold, Frequency: node.Frequency, Importance: node.Importance, Plasticity: node.Plasticity, UsageHistory: append([]time.Time(nil), node.UsageHistory...), LastActivation: node.LastActivation})
		targets := make([]NodeID, 0, len(node.Synapses))
		for targetID := range node.Synapses {
			targets = append(targets, targetID)
		}
		sort.Slice(targets, func(i, j int) bool { return targets[i] < targets[j] })
		for _, targetID := range targets {
			for _, synapse := range node.Synapses[targetID] {
				if synapse == nil {
					continue
				}
				synapses = append(synapses, canonicalSynapse{SourceID: node.ID, TargetID: synapse.TargetID, Kind: synapse.Kind, Weight: synapse.Weight, Activation: synapse.Activation, Frequency: synapse.Frequency, Confidence: synapse.Confidence, Inhibitory: synapse.Inhibitory, LastActivation: synapse.LastActivation, Dynamic: synapse.Dynamic})
			}
		}
	}
	sort.Slice(synapses, func(i, j int) bool {
		if synapses[i].SourceID != synapses[j].SourceID {
			return synapses[i].SourceID < synapses[j].SourceID
		}
		if synapses[i].TargetID != synapses[j].TargetID {
			return synapses[i].TargetID < synapses[j].TargetID
		}
		if synapses[i].Inhibitory != synapses[j].Inhibitory {
			return !synapses[i].Inhibitory
		}
		return synapses[i].Kind < synapses[j].Kind
	})
	provenance := make([]canonicalProvenance, 0, len(k.ProjectionPopulations))
	for i, population := range k.ProjectionPopulations {
		experienceID := ""
		if i < len(BootstrapNetworkDomains) {
			experienceID = "bootstrap-" + BootstrapNetworkDomains[i].Name
		}
		if population.Scope != nil {
			provenance = append(provenance, canonicalProvenance{Domain: population.Domain, Provenance: population.Provenance})
			continue
		}
		provenance = append(provenance, canonicalProvenance{Domain: population.Domain, ExperienceID: experienceID, Provenance: BootstrapProvenance{Origin: "developmental_bootstrap", DirectExperience: false, Status: "initial_hypothesis"}})
	}
	state := k.BrainState
	errorState := map[string]float64{}
	for key, value := range state.PredictionState {
		if len(key) >= 6 && key[:6] == "error_" {
			errorState[key[6:]] = value
		}
	}
	payload := canonicalBrainFile{MemoryDecay: state.MemoryDecay, SensoryTraces: state.SensoryTraces, SchemaVersion: 2, BrainIdentity: "horizon", NeuralUnits: neuralUnits, Populations: append([]ProjectionPopulation(nil), k.ProjectionPopulations...), Synapses: synapses, TemporalPatterns: append([]*PatternSynapse(nil), k.Patterns.All()...), Episodes: append([]BootstrapExperience(nil), state.Episodes...), BootstrapExperiences: append([]BootstrapExperience(nil), state.BootstrapExperiences...), AttentionState: cloneFloatMap(state.AttentionState), PredictionState: cloneFloatMap(state.PredictionState), ErrorState: errorState, MemoryState: cloneFloatMap(state.MemoryState), CuriosityState: cloneFloatMap(state.CuriosityState), LearningPolicyState: state.LearningPolicyState, SelfModelState: cloneFloatMap(state.SelfModelState), SocialModelState: cloneFloatMap(state.SocialModelState), ValueState: cloneFloatMap(state.ValueState), PlasticityState: cloneFloatMap(state.PlasticityState), InquiryState: state.InquiryState, Provenance: provenance, DevelopmentalExperiences: append([]DevelopmentalExperience(nil), state.DevelopmentalExperiences...), ExperienceTraces: append([]ExperienceTrace(nil), state.ExperienceTraces...)}
	if state.MemoryDecay != nil {
		payload.SchemaVersion = 3
	}
	return json.MarshalIndent(payload, "", "  ")
}
