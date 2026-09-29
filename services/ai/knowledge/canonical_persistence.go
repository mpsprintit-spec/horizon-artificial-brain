package knowledge

import (
	"encoding/json"
	"sort"
	"time"
)

// canonicalNeuralUnit persists one neural unit without embedding connectivity.
type canonicalNeuralUnit struct {
	ID NodeID `json:"id"`
	Representation []float64 `json:"representation,omitempty"`
	Activation float64 `json:"activation"`
	RestingActivation float64 `json:"resting_activation"`
	Threshold float64 `json:"threshold"`
	Frequency int64 `json:"frequency"`
	Importance float64 `json:"importance"`
	Plasticity float64 `json:"plasticity"`
	UsageHistory []time.Time `json:"usage_history,omitempty"`
	LastActivation time.Time `json:"last_activation,omitempty"`
}

type canonicalSynapse struct {
	SourceID NodeID `json:"source_id"`
	TargetID NodeID `json:"target_id"`
	Kind RelationKind `json:"kind,omitempty"`
	Weight float64 `json:"weight"`
	Activation float64 `json:"activation"`
	Frequency int64 `json:"frequency"`
	Confidence float64 `json:"confidence"`
	Inhibitory bool `json:"inhibitory"`
	LastActivation time.Time `json:"last_activation,omitempty"`
	Dynamic DynamicState `json:"dynamic"`
}

type canonicalProvenance struct {
	Domain string `json:"domain"`
	ExperienceID string `json:"experience_id"`
	Provenance BootstrapProvenance `json:"provenance"`
}

type canonicalBrainFile struct {
	SchemaVersion int `json:"schema_version"`
	BrainIdentity string `json:"brain_identity"`
	NeuralUnits []canonicalNeuralUnit `json:"neural_units"`
	Populations []ProjectionPopulation `json:"populations"`
	Synapses []canonicalSynapse `json:"synapses"`
	TemporalPatterns []*PatternSynapse `json:"temporal_patterns"`
	Episodes []BootstrapExperience `json:"episodes"`
	BootstrapExperiences []BootstrapExperience `json:"bootstrap_experiences"`
	AttentionState map[string]float64 `json:"attention_state"`
	PredictionState map[string]float64 `json:"prediction_state"`
	ErrorState map[string]float64 `json:"error_state"`
	MemoryState map[string]float64 `json:"memory_state"`
	CuriosityState map[string]float64 `json:"curiosity_state"`
	LearningPolicyState LearningPolicyState `json:"learning_policy_state"`
	SelfModelState map[string]float64 `json:"self_model_state"`
	SocialModelState map[string]float64 `json:"social_model_state"`
	ValueState map[string]float64 `json:"value_state"`
	PlasticityState map[string]float64 `json:"plasticity_state"`
	InquiryState InquiryState `json:"inquiry_state"`
	Provenance []canonicalProvenance `json:"provenance"`
}

func marshalCanonicalBrain(k *KnowledgeBase) ([]byte, error) {
	k.projectionMu.RLock()
	defer k.projectionMu.RUnlock()
	nodes := k.Registry.Nodes()
	sort.Slice(nodes, func(i, j int) bool { return nodes[i].ID < nodes[j].ID })
	neuralUnits := make([]canonicalNeuralUnit, 0, len(nodes))
	synapses := make([]canonicalSynapse, 0)
	for _, node := range nodes {
		if node == nil { continue }
		neuralUnits = append(neuralUnits, canonicalNeuralUnit{ID: node.ID, Representation: append([]float64(nil), node.Representation...), Activation: node.Activation, RestingActivation: node.RestingActivation, Threshold: node.Threshold, Frequency: node.Frequency, Importance: node.Importance, Plasticity: node.Plasticity, UsageHistory: append([]time.Time(nil), node.UsageHistory...), LastActivation: node.LastActivation})
		targets := make([]NodeID, 0, len(node.Synapses))
		for targetID := range node.Synapses { targets = append(targets, targetID) }
		sort.Slice(targets, func(i, j int) bool { return targets[i] < targets[j] })
		for _, targetID := range targets {
			for _, synapse := range node.Synapses[targetID] {
				if synapse == nil { continue }
				synapses = append(synapses, canonicalSynapse{SourceID: node.ID, TargetID: synapse.TargetID, Kind: synapse.Kind, Weight: synapse.Weight, Activation: synapse.Activation, Frequency: synapse.Frequency, Confidence: synapse.Confidence, Inhibitory: synapse.Inhibitory, LastActivation: synapse.LastActivation, Dynamic: synapse.Dynamic})
			}
		}
	}
	sort.Slice(synapses, func(i, j int) bool {
		if synapses[i].SourceID != synapses[j].SourceID { return synapses[i].SourceID < synapses[j].SourceID }
		if synapses[i].TargetID != synapses[j].TargetID { return synapses[i].TargetID < synapses[j].TargetID }
		if synapses[i].Inhibitory != synapses[j].Inhibitory { return !synapses[i].Inhibitory }
		return synapses[i].Kind < synapses[j].Kind
	})
	provenance := make([]canonicalProvenance, 0, len(k.ProjectionPopulations))
	for i, population := range k.ProjectionPopulations {
		experienceID := ""
		if i < len(BootstrapNetworkDomains) { experienceID = "bootstrap-" + BootstrapNetworkDomains[i].Name }
		provenance = append(provenance, canonicalProvenance{Domain: population.Domain, ExperienceID: experienceID, Provenance: BootstrapProvenance{Origin: "developmental_bootstrap", DirectExperience: false, Status: "initial_hypothesis"}})
	}
	state := k.BrainState
	errorState := map[string]float64{}
	for key, value := range state.PredictionState { if len(key) >= 6 && key[:6] == "error_" { errorState[key[6:]] = value } }
	payload := canonicalBrainFile{SchemaVersion: 2, BrainIdentity: "horizon", NeuralUnits: neuralUnits, Populations: append([]ProjectionPopulation(nil), k.ProjectionPopulations...), Synapses: synapses, TemporalPatterns: append([]*PatternSynapse(nil), k.Patterns.All()...), Episodes: append([]BootstrapExperience(nil), state.Episodes...), BootstrapExperiences: append([]BootstrapExperience(nil), state.BootstrapExperiences...), AttentionState: cloneFloatMap(state.AttentionState), PredictionState: cloneFloatMap(state.PredictionState), ErrorState: errorState, MemoryState: cloneFloatMap(state.MemoryState), CuriosityState: cloneFloatMap(state.CuriosityState), LearningPolicyState: state.LearningPolicyState, SelfModelState: cloneFloatMap(state.SelfModelState), SocialModelState: cloneFloatMap(state.SocialModelState), ValueState: cloneFloatMap(state.ValueState), PlasticityState: cloneFloatMap(state.PlasticityState), InquiryState: state.InquiryState, Provenance: provenance}
	return json.MarshalIndent(payload, "", "  ")
}

func cloneFloatMap(in map[string]float64) map[string]float64 { out := make(map[string]float64, len(in)); for key, value := range in { out[key] = value }; return out }

func loadCanonicalBrain(data []byte) ([]*ConceptNode, []*PatternSynapse, []ProjectionPopulation, BrainState, error) {
	var raw struct {
		NeuralUnits []canonicalNeuralUnit `json:"neural_units"`
		Populations []ProjectionPopulation `json:"populations"`
		Synapses []canonicalSynapse `json:"synapses"`
		TemporalPatterns []*PatternSynapse `json:"temporal_patterns"`
		Episodes []BootstrapExperience `json:"episodes"`
		BootstrapExperiences []BootstrapExperience `json:"bootstrap_experiences"`
		AttentionState map[string]float64 `json:"attention_state"`
		PredictionState map[string]float64 `json:"prediction_state"`
		ErrorState map[string]float64 `json:"error_state"`
		MemoryState map[string]float64 `json:"memory_state"`
		CuriosityState map[string]float64 `json:"curiosity_state"`
		LearningPolicyState LearningPolicyState `json:"learning_policy_state"`
		SelfModelState map[string]float64 `json:"self_model_state"`
		SocialModelState map[string]float64 `json:"social_model_state"`
		ValueState map[string]float64 `json:"value_state"`
		PlasticityState map[string]float64 `json:"plasticity_state"`
		InquiryState InquiryState `json:"inquiry_state"`
	}
	if err := json.Unmarshal(data, &raw); err != nil { return nil, nil, nil, BrainState{}, err }
	if len(raw.NeuralUnits) == 0 {
		var legacy persistedGraph
		if err := json.Unmarshal(data, &legacy); err != nil { return nil, nil, nil, BrainState{}, err }
		return legacy.Nodes, legacy.Patterns, legacy.ProjectionPopulations, legacy.BrainState, nil
	}
	registry := make(map[NodeID]*ConceptNode, len(raw.NeuralUnits))
	for _, unit := range raw.NeuralUnits { registry[unit.ID] = &ConceptNode{ID: unit.ID, Representation: append([]float64(nil), unit.Representation...), Activation: unit.Activation, RestingActivation: unit.RestingActivation, Threshold: unit.Threshold, Frequency: unit.Frequency, Importance: unit.Importance, Plasticity: unit.Plasticity, UsageHistory: append([]time.Time(nil), unit.UsageHistory...), LastActivation: unit.LastActivation, Synapses: make(map[NodeID]SynapseList)} }
	for _, item := range raw.Synapses { source := registry[item.SourceID]; if source == nil { continue }; synapse := &Synapse{TargetID: item.TargetID, Kind: item.Kind, Weight: item.Weight, Activation: item.Activation, Frequency: item.Frequency, Confidence: item.Confidence, Inhibitory: item.Inhibitory, LastActivation: item.LastActivation, Dynamic: item.Dynamic}; hydrateSynapseDynamicState(synapse); source.Synapses[item.TargetID] = append(source.Synapses[item.TargetID], synapse) }
	nodes := make([]*ConceptNode, 0, len(registry)); for _, node := range registry { nodes = append(nodes, node) }; sort.Slice(nodes, func(i, j int) bool { return nodes[i].ID < nodes[j].ID })
	state := BrainState{AttentionState: raw.AttentionState, CuriosityState: raw.CuriosityState, PredictionState: raw.PredictionState, MemoryState: raw.MemoryState, SelfModelState: raw.SelfModelState, SocialModelState: raw.SocialModelState, ValueState: raw.ValueState, PlasticityState: raw.PlasticityState, LearningPolicyState: raw.LearningPolicyState, InquiryState: raw.InquiryState, BootstrapExperiences: raw.BootstrapExperiences, Episodes: raw.Episodes}
	if state.PredictionState == nil { state.PredictionState = map[string]float64{} }; for key, value := range raw.ErrorState { state.PredictionState["error_"+key] = value }
	return nodes, raw.TemporalPatterns, raw.Populations, state, nil
}
