package knowledge

import "time"

const visualBootstrapSourceHash = "sha256:89220a3626733c1c31840d30ef7d97a854a24a0d7052523c69b493411e137bfd"
const visualBootstrapExperienceID = "bootstrap-visual-face-89220a36"

type visualBootstrapRegion struct {
	name string
	position string
	representation []float64
}

var visualBootstrapRegions = []visualBootstrapRegion{
	{name: "whole", position: "whole", representation: []float64{
		0.24995577, -0.34876299, -0.25128084, 0.24575305, 0.57118511, 0.05266428, 0.13222718, 0.52654755,
		0.59730768, 0.26357126, 0.35985863, 0.81157196, 0.22998989, 0.02976930, 0.14715827, 0.77690363,
		0.95930483, 0.11332774, 0.24637360, 1.00000000, 0.34009939, -0.36330481, -0.12065310, -0.11327234,
		0.37401962, 0.23921585, 0.19705880, -0.70419803, -0.67024994, -0.64913601, 0.27465117, -0.52764404,
	}},
	{name: "upper", position: "upper_region", representation: []float64{
		0.55132842, -0.42794180, -0.45690465, 0.38504565, 0.01183534, -0.34275848, -0.11747873, 0.13962197,
		0.37001157, -0.00368333, 0.17619860, 0.22878897, 0.64369106, -0.01063532, 0.00911236, 0.66643918,
		0.62669528, -0.01463876, -0.00989739, 1.00000000, 0.76074262, -0.01887755, 0.11651319, 0.65939848,
		0.18921590, 0.08774507, 0.05490196, -0.67631167, -0.64931309, -0.63722840, 0.11391687, -0.46643072,
	}},
	{name: "middle", position: "middle_region", representation: []float64{
		0.53806961, -0.01507378, 0.12558365, 0.33937097, 0.60602629, 0.08093023, 0.10681200, 0.69635355,
		0.58970094, 0.37312150, 0.45089316, 0.73506880, 0.59880137, 0.23023272, 0.32817352, 0.84195304,
		0.96853461, 0.17234397, 0.31306866, 1.00000000, 0.13104590, -0.33523306, -0.31823895, 0.08152081,
		0.53970587, 0.37009823, 0.30980396, -0.79525584, -0.72881967, -0.69838318, 0.41412616, -0.61371949,
	}},
	{name: "lower", position: "lower_region", representation: []float64{
		0.60225713, 0.31257010, 0.39941716, 0.78569746, 0.59925854, 0.09644735, 0.20879555, 0.89958882,
		0.63033962, 0.08124244, 0.28415918, 0.92047179, -0.25714755, -0.01435411, 0.02424121, 0.59943318,
		0.27951300, -0.01666700, 0.10178377, 0.13164556, 1.00000000, 0.09276030, 0.73292439, 0.24352792,
		0.49656868, 0.34509802, 0.29999995, -0.70635045, -0.64071071, -0.60862336, 0.38577628, -0.59684426,
	}},
	{name: "left", position: "left_region", representation: []float64{
		0.61418116, -0.11426950, -0.39137650, -0.30614954, 0.96034527, 0.18202519, -0.04113519, 0.14646339,
		0.82182562, 0.37278950, 0.30879629, 0.21834612, 0.39269400, 0.06728590, -0.04703188, 0.10657048,
		0.36537692, 1.00000000, -0.10081110, 0.07964604, 0.14779742, 0.39279582, -0.29309314, -0.35191315,
		0.30637264, 0.17009795, 0.12303936, -0.66565356, -0.62477520, -0.60716510, 0.20570993, -0.54716638,
	}},
	{name: "right", position: "right_region", representation: []float64{
		-0.01902735, -0.48353434, -0.34492695, 0.83643317, 0.28402865, -0.01957440, 0.12038255, 0.93271232,
		0.29562604, 0.42409110, 0.64586782, 0.97727680, 0.12073529, 0.17358124, 0.68695569, 0.86685193,
		0.63391090, 0.25442830, 1.00000000, 0.43004471, -0.26137316, 0.31501891, 0.38645509, -0.69418789,
		0.44166672, 0.30784309, 0.26960790, -0.58995485, -0.55125448, -0.53326848, 0.34359241, -0.50062087,
	}},
}

func materializeVisualBootstrap(brain *Brain, at time.Time) error {
	if brain == nil || brain.Registry == nil {
		return ErrNilBrain
	}
	for _, population := range brain.ProjectionPopulations {
		if population.Domain == "visual_form" {
			return nil
		}
	}
	if at.IsZero() {
		at = time.Unix(1767225600, 0).UTC()
	} else {
		at = at.UTC()
	}

	ids := make([]NodeID, len(visualBootstrapRegions))
	nextID := brain.Registry.nextID
	if nextID < 1 {
		nextID = 1
	}
	for i, region := range visualBootstrapRegions {
		id := nextID
		nextID++
		node := newRepresentationNodeAt(id, region.representation, at)
		node.Activation = 0.30
		node.RestingActivation = 0.05
		node.Threshold = 0.25
		node.Frequency = 1
		node.Importance = 0.72
		node.Plasticity = 0.65
		brain.Registry.byID[id] = node
		ids[i] = id
	}
	brain.Registry.nextID = nextID

	for i := range ids {
		source := brain.Registry.GetByID(ids[i])
		if i+1 < len(ids) {
			connectBootstrapAt(source, brain.Registry.GetByID(ids[i+1]), 0.30, 0.35, false, at)
		}
		if i > 0 {
			connectBootstrapAt(source, brain.Registry.GetByID(ids[i-1]), 0.24, 0.30, false, at)
		}
	}
	for _, sourceID := range ids {
		connectBootstrapAt(brain.Registry.GetByID(sourceID), brain.Registry.GetByID(19), 0.16, 0.22, false, at)
		connectBootstrapAt(brain.Registry.GetByID(sourceID), brain.Registry.GetByID(25), 0.12, 0.20, false, at)
	}
	brain.ProjectionPopulations = append(brain.ProjectionPopulations, ProjectionPopulation{
		Domain: "visual_form",
		Units: populationUnits(ids, 0.30),
		LearningTarget: append([]NodeID(nil), ids...),
		PredictionTargets: append([]NodeID(nil), ids...),
		ErrorTargets: []NodeID{61, 62, 63, 64, 65, 66, 67, 68},
		PlasticityTargets: append([]NodeID(nil), ids...),
		CounterEvidenceTargets: []NodeID{61, 62, 63, 64, 65, 66, 67, 68},
		TemporalScale: "multi",
		BootstrapExperienceID: visualBootstrapExperienceID,
		Provenance: BootstrapProvenance{
			Origin: "human_provided_visual_bootstrap",
			DirectExperience: false,
			Status: "initial_hypothesis",
			SourceHash: visualBootstrapSourceHash,
		},
	})

	sequence := make([]PatternStep, len(ids))
	trace := make([]time.Time, len(ids))
	for i, id := range ids {
		sequence[i] = PatternStep{
			NodeID: id,
			Position: i,
			Delta: time.Duration(i+1) * 40 * time.Millisecond,
			Activation: 0.30,
		}
		trace[i] = at.Add(time.Duration(i) * 40 * time.Millisecond)
	}
	brain.Patterns.LearnTrace(sequence, nil, ids[0], 0.30, 0.25)

	experience := DevelopmentalExperience{
		ID: visualBootstrapExperienceID,
		BeforeState: DevelopmentalStateFrame{
			ActiveUnits: append([]NodeID(nil), ids...),
			ActivePopulations: []NodeID{ids[0]},
			Representation: append([]float64(nil), visualBootstrapRegions[0].representation...),
			Confidence: 0.20,
		},
		Transition: DevelopmentalTransition{
			ChangeRepresentation: make([]float64, len(visualBootstrapRegions[0].representation)),
			TemporalDeltasNanos: []int64{0, 40000000, 80000000, 120000000, 160000000, 200000000},
			Confidence: 0.20,
		},
		AfterState: DevelopmentalStateFrame{
			ActiveUnits: append([]NodeID(nil), ids...),
			ActivePopulations: []NodeID{ids[0]},
			Representation: append([]float64(nil), visualBootstrapRegions[0].representation...),
			Confidence: 0.20,
		},
		LearningTrace: DevelopmentalLearningTrace{
			ActiveUnits: append([]NodeID(nil), ids...),
			ActivePopulations: []NodeID{ids[0]},
			TemporalTrace: trace,
			Prediction: append([]float64(nil), visualBootstrapRegions[0].representation...),
			Outcome: append([]float64(nil), visualBootstrapRegions[0].representation...),
			PredictionError: 0.20,
			RepresentationDelta: make([]float64, len(visualBootstrapRegions[0].representation)),
			SynapticDeltas: []DevelopmentalSynapticDelta{
				{SourceNodeID: ids[0], TargetNodeID: ids[1], Delta: 0.03},
				{SourceNodeID: ids[2], TargetNodeID: ids[3], Delta: 0.03},
			},
			Plasticity: 0.65,
			Confidence: 0.20,
		},
		Provenance: BootstrapProvenance{
			Origin: "human_provided_visual_bootstrap",
			DirectExperience: false,
			Status: "initial_hypothesis",
			SourceHash: visualBootstrapSourceHash,
		},
	}
	for i, region := range visualBootstrapRegions {
		experience.SymbolExposures = append(experience.SymbolExposures, SymbolExposure{
			Modality: "vision",
			Representation: append([]float64(nil), region.representation...),
			RelativePosition: region.position,
			SequencePosition: i,
			Confidence: 0.20,
		})
	}
	brain.BrainState.DevelopmentalExperiences = append(brain.BrainState.DevelopmentalExperiences, experience)
	brain.BrainState.MemoryState["visual_bootstrap_experiences"] += 1
	brain.BrainState.MemoryState["visual_bootstrap_units"] += float64(len(ids))
	brain.BrainState.PredictionState["visual_bootstrap_initial_hypothesis"] = 0.20
	return nil
}
