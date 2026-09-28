package knowledge

import (
	"errors"
	"math"
	"time"
)

// BootstrapDomain identifies one developmental network field. It is structural
// metadata, not a world-fact or lexical definition.
type BootstrapDomain struct {
	Name string
}

var BootstrapNetworkDomains = []BootstrapDomain{
	{Name: "existence"},
	{Name: "change"},
	{Name: "continuity"},
	{Name: "identity"},
	{Name: "body_state"},
	{Name: "agency"},
	{Name: "time"},
	{Name: "space"},
	{Name: "object_state"},
	{Name: "prediction"},
	{Name: "error"},
	{Name: "attention"},
	{Name: "curiosity"},
	{Name: "memory"},
	{Name: "thought"},
	{Name: "learning_policy"},
	{Name: "self_model"},
	{Name: "social_model"},
	{Name: "value"},
	{Name: "communication"},
}

const bootstrapPopulationSize = 8
const bootstrapSharedUnits = 2

// MaterializeBootstrap builds the developmental substrate required by the
// bootstrap addendum: distributed/overlapping populations, recurrent
// excitation and inhibition, temporal patterns, prediction/error pathways,
// plasticity, bootstrap episodes, provenance, and learning-policy state.
func MaterializeBootstrap(brain *Brain, at time.Time) error {
	if brain == nil || brain.Registry == nil || brain.Patterns == nil {
		return ErrNilBrain
	}
	if at.IsZero() {
		at = time.Unix(1767225600, 0).UTC()
	} else {
		at = at.UTC()
	}
	if len(brain.Registry.Nodes()) > 0 || len(brain.ProjectionPopulations) > 0 ||
		len(brain.BrainState.BootstrapExperiences) > 0 {
		if bootstrapMaterialized(brain) {
			return nil
		}
		return errors.New("brain already contains non-bootstrap substrate")
	}

	pools := make([][]NodeID, len(BootstrapNetworkDomains))
	nextID := brain.Registry.nextID
	if nextID < 1 {
		nextID = 1
	}

	for domainIndex := range BootstrapNetworkDomains {
		pool := make([]NodeID, 0, bootstrapPopulationSize)
		if domainIndex > 0 {
			previous := pools[domainIndex-1]
			pool = append(pool, previous[len(previous)-bootstrapSharedUnits:]...)
		}
		for slot := len(pool); slot < bootstrapPopulationSize; slot++ {
			node := newRepresentationNodeAt(nextID, bootstrapVector(domainIndex, slot), at)
			node.Activation = 0.12
			node.RestingActivation = 0.05
			node.Threshold = 0.25
			node.Frequency = 1
			node.Importance = 0.5 + 0.02*float64(domainIndex%5)
			node.Plasticity = 0.65
			brain.Registry.byID[nextID] = node
			pool = append(pool, nextID)
			nextID++
		}
		pools[domainIndex] = pool
	}
	brain.Registry.nextID = nextID

	for domainIndex, domain := range BootstrapNetworkDomains {
		pool := pools[domainIndex]
		brain.ProjectionPopulations = append(brain.ProjectionPopulations, ProjectionPopulation{
			Domain: domain.Name,
			Units: populationUnits(pool, 0.25),
			LearningTarget: append([]NodeID(nil), pool...),
			CounterEvidenceTargets: append([]NodeID(nil), pools[10]...),
			CounterEvidenceTargets: append([]NodeID(nil), pools[10]...),
			TemporalScale: bootstrapTemporalScale(domainIndex),
		})

		for i, sourceID := range pool {
			source := brain.Registry.GetByID(sourceID)
			if source == nil {
				continue
			}
			for offset := 1; offset <= 2; offset++ {
				target := brain.Registry.GetByID(pool[(i+offset)%len(pool)])
				connectBootstrapAt(source, target, 0.24, 0.55, false, at)
			}
			inhibitoryTarget := brain.Registry.GetByID(pool[(i+3)%len(pool)])
			connectBootstrapAt(source, inhibitoryTarget, 0.12, 0.40, true, at)
		}

		if domainIndex+1 < len(pools) {
			nextPool := pools[domainIndex+1]
			for i := 0; i < bootstrapSharedUnits; i++ {
				a := brain.Registry.GetByID(pool[i])
				b := brain.Registry.GetByID(nextPool[i])
				connectBootstrapAt(a, b, 0.28, 0.60, false, at)
				connectBootstrapAt(b, a, 0.20, 0.50, false, at)
			}
		}

		sequence := make([]PatternStep, 0, 4)
		for position := 0; position < 4; position++ {
			sequence = append(sequence, PatternStep{
				NodeID: pool[position],
				Position: position,
				Delta: time.Duration(position+1) * 25 * time.Millisecond,
				Activation: 0.25 + 0.10*float64(position),
			})
		}
		brain.Patterns.LearnTrace(sequence, []ContextFrame{
			{NodeID: pool[0], Activation: 0.25, Weight: 0.5},
			{NodeID: pool[len(pool)-1], Activation: 0.25, Weight: 0.5},
		}, pool[3], 0.55, 0.35)

		experience := BootstrapExperience{
			ID: "bootstrap-" + domain.Name,
			Populations: append([]NodeID(nil), pool...),
			TemporalTrace: []time.Time{
				at, at.Add(25 * time.Millisecond),
				at.Add(50 * time.Millisecond), at.Add(75 * time.Millisecond),
			},
			Activation: 0.25,
			Confidence: 0.20,
			Provenance: BootstrapProvenance{
				Origin: "developmental_bootstrap",
				DirectExperience: false,
				Status: "initial_hypothesis",
			},
		}
		brain.BrainState.BootstrapExperiences = append(brain.BrainState.BootstrapExperiences, experience)
		brain.BrainState.Episodes = append(brain.BrainState.Episodes, experience)
	}

	// Materialize explicit developmental trajectories for agency, object
	// continuity, internal thinking, and curiosity. These are numeric pathway
	// patterns, not semantic definitions.
	special := []struct {
		id string
		members []NodeID
	}{
		{"bootstrap-agency-outcome", []NodeID{pools[5][0], pools[4][0], pools[7][0], pools[18][0]}},
		{"bootstrap-object-continuity", []NodeID{pools[8][0], pools[2][0], pools[3][0], pools[9][0]}},
		{"bootstrap-thought-simulation", []NodeID{pools[10][0], pools[13][0], pools[14][0], pools[9][0], pools[15][0]}},
		{"bootstrap-curiosity-inquiry", []NodeID{pools[12][0], pools[11][0], pools[7][0], pools[15][0]}},
	}
	for _, item := range special {
		sequence := make([]PatternStep, len(item.members))
		for i, id := range item.members {
			sequence[i] = PatternStep{NodeID: id, Position: i, Delta: time.Duration(i+1)*40*time.Millisecond, Activation: 0.30}
		}
		brain.Patterns.LearnTrace(sequence, nil, item.members[len(item.members)-1], 0.60, 0.40)
		trace := make([]time.Time, len(item.members))
		for i := range trace {
			trace[i] = at.Add(time.Duration(i) * 40 * time.Millisecond)
		}
		experience := BootstrapExperience{
			ID: item.id, Populations: append([]NodeID(nil), item.members...),
			TemporalTrace: trace,
			Activation: 0.30, Confidence: 0.20,
			Provenance: BootstrapProvenance{Origin: "developmental_bootstrap", DirectExperience: false, Status: "initial_hypothesis"},
		}
		brain.BrainState.BootstrapExperiences = append(brain.BrainState.BootstrapExperiences, experience)
		brain.BrainState.Episodes = append(brain.BrainState.Episodes, experience)
	}

	// Error/contradiction pathways explicitly feed back into prediction and
	// learning-policy populations, providing a counterevidence route.
	for _, sourceID := range pools[10] {
		source := brain.Registry.GetByID(sourceID)
		connectBootstrapAt(source, brain.Registry.GetByID(pools[9][0]), 0.30, 0.60, true, at)
		connectBootstrapAt(source, brain.Registry.GetByID(pools[15][0]), 0.28, 0.55, false, at)
	}
	for _, sourceID := range pools[9] {
		connectBootstrapAt(brain.Registry.GetByID(sourceID), brain.Registry.GetByID(pools[10][0]), 0.30, 0.60, false, at)
	}

	// Prediction and error form an explicit reciprocal learning circuit:
	// prediction excites error detection; error supplies inhibitory counterevidence
	// back to prediction. This is a developmental mechanism, not a semantic fact.
	predictionPool := pools[9]
	errorPool := pools[10]
	for i := 0; i < bootstrapSharedUnits; i++ {
		prediction := brain.Registry.GetByID(predictionPool[i])
		errNode := brain.Registry.GetByID(errorPool[i])
		brain.Connect(prediction, errNode, 0.32, 0.65, false)
		brain.Connect(errNode, prediction, 0.18, 0.55, true)
	}

	composites := []struct {
		id       string
		members  []NodeID
		interval time.Duration
	}{
		{id: "bootstrap-agency-outcome", members: []NodeID{pools[5][0], pools[4][0], pools[7][0], pools[18][0]}, interval: 40 * time.Millisecond},
		{id: "bootstrap-object-continuity", members: []NodeID{pools[8][0], pools[2][0], pools[3][0], pools[9][0]}, interval: 40 * time.Millisecond},
		{id: "bootstrap-thought-simulation", members: []NodeID{pools[10][0], pools[13][0], pools[14][0], pools[9][0], pools[15][0]}, interval: 40 * time.Millisecond},
		{id: "bootstrap-curiosity-inquiry", members: []NodeID{pools[10][7], pools[10][6], pools[7][0], pools[14][6]}, interval: 40 * time.Millisecond},
	}
	for _, composite := range composites {
		sequence := make([]PatternStep, len(composite.members))
		trace := make([]time.Time, len(composite.members))
		for i, id := range composite.members {
			sequence[i] = PatternStep{NodeID: id, Position: i, Delta: time.Duration(i+1) * composite.interval, Activation: 0.30}
			trace[i] = at.Add(time.Duration(i) * composite.interval)
		}
		brain.Patterns.LearnTrace(sequence, nil, composite.members[len(composite.members)-1], 0.60, 0.40)
		experience := BootstrapExperience{
			ID: composite.id, Populations: append([]NodeID(nil), composite.members...),
			TemporalTrace: trace, Activation: 0.30, Confidence: 0.20,
			Provenance: BootstrapProvenance{Origin: "developmental_bootstrap", DirectExperience: false, Status: "initial_hypothesis"},
		}
		brain.BrainState.BootstrapExperiences = append(brain.BrainState.BootstrapExperiences, experience)
		brain.BrainState.Episodes = append(brain.BrainState.Episodes, experience)
	}

	brain.BrainState.AttentionState["novelty"] = brain.BrainState.LearningPolicyState.NoveltySensitivity
	brain.BrainState.AttentionState["uncertainty"] = brain.BrainState.LearningPolicyState.UncertaintySensitivity
	brain.BrainState.AttentionState["bootstrap_domains"] = float64(len(BootstrapNetworkDomains))
	brain.BrainState.PredictionState["error_sensitivity"] = brain.BrainState.LearningPolicyState.UncertaintySensitivity
	brain.BrainState.PredictionState["bootstrap_prediction_paths"] = float64(len(BootstrapNetworkDomains))
	brain.BrainState.PredictionState["bootstrap_error_paths"] = float64(len(BootstrapNetworkDomains))
	brain.BrainState.PlasticityState["baseline"] = 0.8
	brain.BrainState.PlasticityState["transition_strength"] = 0.25
	brain.BrainState.PlasticityState["bootstrap_population_paths"] = float64(len(BootstrapNetworkDomains))
	brain.BrainState.CuriosityState["pressure"] = brain.BrainState.LearningPolicyState.CuriosityPressure
	brain.BrainState.CuriosityState["bootstrap_uncertainty_drive"] = 0.65
	brain.BrainState.MemoryState["retention"] = brain.BrainState.LearningPolicyState.MemoryRetention
	brain.BrainState.MemoryState["priority"] = 0.5
	brain.BrainState.MemoryState["bootstrap_temporal_traces"] = float64(len(brain.BrainState.Episodes))
	brain.BrainState.SelfModelState["continuity"] = 0.20
	brain.BrainState.SelfModelState["agency"] = 0.20
	brain.BrainState.SelfModelState["body_continuity"] = 0.20
	brain.BrainState.SocialModelState["agent_model_capacity"] = 0.20
	brain.BrainState.ValueState["consequence_sensitivity"] = 0.20
	return nil
}

func NewBootstrapBrain() *Brain {
	brain := NewBrain()
	_ = MaterializeBootstrap(brain, time.Time{})
	return brain
}

func bootstrapMaterialized(brain *Brain) bool {
	return len(brain.Registry.Nodes()) > 0 &&
		len(brain.ProjectionPopulations) >= len(BootstrapNetworkDomains) &&
		len(brain.BrainState.BootstrapExperiences) >= len(BootstrapNetworkDomains) &&
		len(brain.Patterns.All()) >= len(BootstrapNetworkDomains)+4
}

func bootstrapVector(domainIndex, slot int) []float64 {
	values := make([]float64, 8)
	seed := float64((domainIndex+1)*17 + (slot+1)*7)
	for i := range values {
		values[i] = math.Sin(seed+float64(i)*0.73)*0.5 +
			math.Cos(seed*0.37+float64(i)*0.41)*0.5
	}
	return values
}


// connectBootstrapAt is the deterministic bootstrap equivalent of Connect.
// It keeps the generated substrate reproducible for replay/checkpoint tests.
func connectBootstrapAt(source, target *ConceptNode, weight, confidence float64, inhibitory bool, at time.Time) {
	if source == nil || target == nil {
		return
	}
	if source.Synapses == nil {
		source.Synapses = make(map[NodeID]SynapseList)
	}
	list := source.Synapses[target.ID]
	synapse := list.FindDynamic(inhibitory)
	if synapse == nil {
		synapse = &Synapse{TargetID: target.ID, Inhibitory: inhibitory}
		source.Synapses[target.ID] = append(list, synapse)
	}
	synapse.Dynamic.Reinforce(weight, confidence, 1, at)
	syncSynapseLegacyState(synapse)
	source.LastActivation = at
}

func populationUnits(ids []NodeID, activation float64) []PopulationUnit {
	units := make([]PopulationUnit, len(ids))
	for i, id := range ids {
		units[i] = PopulationUnit{NodeID: id, Activation: activation}
	}
	return units
}

func bootstrapTemporalScale(index int) string {
	switch index % 4 {
	case 0:
		return "fast"
	case 1:
		return "medium"
	case 2:
		return "slow"
	default:
		return "multi"
	}
}
