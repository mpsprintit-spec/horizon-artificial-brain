package knowledge

import (
	"bufio"
	"crypto/sha256"
	"encoding/hex"
	"io"
	"sort"

	"github.com/project-horizon/horizon-core/services/ai/internal/jsonstream"
)

type canonicalEdgeRef struct {
	source NodeID
	edge   *Synapse
}

func canonicalUnit(node *ConceptNode) canonicalNeuralUnit {
	return canonicalNeuralUnit{ID: node.ID, Representation: node.Representation, Activation: node.Activation, RestingActivation: node.RestingActivation, Threshold: node.Threshold, Frequency: node.Frequency, Importance: node.Importance, Plasticity: node.Plasticity, UsageHistory: node.UsageHistory, LastActivation: node.LastActivation}
}
func canonicalEdge(source NodeID, edge *Synapse) canonicalSynapse {
	return canonicalSynapse{SourceID: source, TargetID: edge.TargetID, Kind: edge.Kind, Weight: edge.Weight, Activation: edge.Activation, Frequency: edge.Frequency, Confidence: edge.Confidence, Inhibitory: edge.Inhibitory, LastActivation: edge.LastActivation, Dynamic: edge.Dynamic}
}
func canonicalProvenanceAt(index int, population ProjectionPopulation) canonicalProvenance {
	if population.Scope != nil {
		return canonicalProvenance{Domain: population.Domain, Provenance: population.Provenance}
	}
	id := ""
	if index < len(BootstrapNetworkDomains) {
		id = "bootstrap-" + BootstrapNetworkDomains[index].Name
	}
	return canonicalProvenance{Domain: population.Domain, ExperienceID: id, Provenance: BootstrapProvenance{Origin: "developmental_bootstrap", DirectExperience: false, Status: "initial_hypothesis"}}
}

// Caller holds k.RLock. Pointer indexes preserve the exact historical global
// sort (including parallel-edge ties) without copying unit vectors or edges.
func writeCanonicalBrain(w io.Writer, k *KnowledgeBase, pretty bool, prefix string) error {
	k.projectionMu.RLock()
	defer k.projectionMu.RUnlock()
	nodes := k.Registry.Nodes()
	edgeCount := 0
	for _, node := range nodes {
		for _, list := range node.Synapses {
			for _, edge := range list {
				if edge != nil {
					edgeCount++
				}
			}
		}
	}
	edges := make([]canonicalEdgeRef, 0, edgeCount)
	for _, node := range nodes {
		targets := make([]NodeID, 0, len(node.Synapses))
		for target := range node.Synapses {
			targets = append(targets, target)
		}
		sort.Slice(targets, func(i, j int) bool { return targets[i] < targets[j] })
		for _, target := range targets {
			for _, edge := range node.Synapses[target] {
				if edge != nil {
					edges = append(edges, canonicalEdgeRef{node.ID, edge})
				}
			}
		}
	}
	sort.Slice(edges, func(i, j int) bool {
		a, b := edges[i], edges[j]
		if a.source != b.source {
			return a.source < b.source
		}
		if a.edge.TargetID != b.edge.TargetID {
			return a.edge.TargetID < b.edge.TargetID
		}
		if a.edge.Inhibitory != b.edge.Inhibitory {
			return !a.edge.Inhibitory
		}
		return a.edge.Kind < b.edge.Kind
	})
	state := k.BrainState
	errorState := map[string]float64{}
	for key, value := range state.PredictionState {
		if len(key) >= 6 && key[:6] == "error_" {
			errorState[key[6:]] = value
		}
	}
	// Empty slices historically passed through append(nil,...), hence null.
	populations := k.ProjectionPopulations
	if len(populations) == 0 {
		populations = nil
	}
	patterns := k.Patterns.All()
	if len(patterns) == 0 {
		patterns = nil
	}
	episodes := state.Episodes
	if len(episodes) == 0 {
		episodes = nil
	}
	bootstrap := state.BootstrapExperiences
	if len(bootstrap) == 0 {
		bootstrap = nil
	}
	developmental := state.DevelopmentalExperiences
	if len(developmental) == 0 {
		developmental = nil
	}
	traces := state.ExperienceTraces
	if len(traces) == 0 {
		traces = nil
	}
	payload := canonicalBrainFile{NResolution: state.NResolution, SchemaVersion: 2, BrainIdentity: "horizon", SensoryTraces: state.SensoryTraces, Populations: populations, TemporalPatterns: patterns, Episodes: episodes, BootstrapExperiences: bootstrap, DevelopmentalExperiences: developmental, ExperienceTraces: traces, AttentionState: canonicalMap(state.AttentionState), PredictionState: canonicalMap(state.PredictionState), ErrorState: errorState, MemoryState: canonicalMap(state.MemoryState), MemoryDecay: state.MemoryDecay, CuriosityState: canonicalMap(state.CuriosityState), LearningPolicyState: state.LearningPolicyState, SelfModelState: canonicalMap(state.SelfModelState), SocialModelState: canonicalMap(state.SocialModelState), ValueState: canonicalMap(state.ValueState), PlasticityState: canonicalMap(state.PlasticityState), InquiryState: state.InquiryState}
	if state.MemoryDecay != nil {
		payload.SchemaVersion = 3
	}
	// Provenance was a nonnil empty array even without populations.
	overrides := map[string]jsonstream.ValueWriter{
		"neural_units": func(w io.Writer, p bool, base string) error {
			return jsonstream.Array(w, len(nodes), false, p, base, func(i int) any { return canonicalUnit(nodes[i]) })
		},
		"synapses": func(w io.Writer, p bool, base string) error {
			return jsonstream.Array(w, len(edges), false, p, base, func(i int) any { return canonicalEdge(edges[i].source, edges[i].edge) })
		},
		"provenance": func(w io.Writer, p bool, base string) error {
			return jsonstream.Array(w, len(populations), false, p, base, func(i int) any { return canonicalProvenanceAt(i, populations[i]) })
		},
	}
	buffer := bufio.NewWriterSize(w, 64*1024)
	if err := jsonstream.Object(buffer, payload, pretty, prefix, overrides); err != nil {
		return err
	}
	return buffer.Flush()
}

// WriteCanonicalJSON writes the exact existing canonical bytes without retaining
// the whole JSON document. Writers must not call back into a Brain mutator.
func (k *KnowledgeBase) WriteCanonicalJSON(w io.Writer) error { return k.WriteCanonical(w, true, "") }

// WriteCanonical also supports compact embedding in complete checkpoints. The
// public CanonicalJSON/CanonicalHash contract always uses pretty=true,prefix="".
func (k *KnowledgeBase) WriteCanonical(w io.Writer, pretty bool, prefix string) error {
	if k == nil {
		return ErrNilBrain
	}
	k.RLock()
	defer k.RUnlock()
	return writeCanonicalBrain(w, k, pretty, prefix)
}
func (k *KnowledgeBase) CanonicalHash() (string, error) {
	h := sha256.New()
	if err := k.WriteCanonicalJSON(h); err != nil {
		return "", err
	}
	return "sha256:" + hex.EncodeToString(h.Sum(nil)), nil
}

func canonicalMap(values map[string]float64) map[string]float64 {
	if values == nil {
		return map[string]float64{}
	}
	return values
}
