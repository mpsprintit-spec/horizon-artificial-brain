package runtime

import (
	"testing"
	"time"

	"github.com/project-horizon/horizon-core/services/ai/knowledge"
)

func TestDNFPlasticityChangesNextCognitiveActivation(t *testing.T) {
	brain := knowledge.NewBrain()
	source := brain.Store("source")
	target := brain.Store("target")
	brain.Connect(source, target, 0.05, 0.20, false)

	runtime := NewBrainRuntime(brain)
	if runtime == nil || runtime.DNF() == nil {
		t.Fatal("expected runtime and DNF fabric")
	}
	if runtime.DNF().Brain() != brain {
		t.Fatal("runtime DNF must use the canonical Brain")
	}

	beforeStructure, err := runtime.DNF().Inspect(time.Unix(100, 0).UTC())
	if err != nil {
		t.Fatal(err)
	}
	if beforeStructure.DynamicSynapses != 1 || beforeStructure.Nodes != 2 {
		t.Fatalf("unexpected initial structure: %+v", beforeStructure)
	}

	before, err := runtime.CognitiveProcess(Event{
		ID: "p2-before",
		Stimulus: []string{"source"},
		Cycles: 1,
		Timestamp: time.Unix(101, 0).UTC(),
	})
	if err != nil {
		t.Fatal(err)
	}
	baseline := before.Activations[target.ID]
	if baseline <= 0 {
		t.Fatalf("expected source activation to reach target before promotion: %v", baseline)
	}

	const requestID = "p2-plasticity-loop"
	if err := runtime.RegisterActionBinding(ActionBinding{
		RequestID: requestID,
		BrainIdentity: BrainIdentity,
		Intent: "reinforce target pathway",
		TargetNodeIDs: []knowledge.NodeID{target.ID},
		Synapses: []SynapseBinding{{
			SourceNodeID: source.ID,
			TargetNodeID: target.ID,
		}},
	}); err != nil {
		t.Fatal(err)
	}

	t0 := time.Unix(102, 0).UTC()
	for _, outcome := range []OutcomeEvent{
		{RequestID: requestID, BrainIdentity: BrainIdentity, Success: true, Observation: []string{"verified"}, Source: "executor", Modality: "execution-outcome", ObservedAt: t0, Reliability: 1},
		{RequestID: requestID, BrainIdentity: BrainIdentity, Success: true, Observation: []string{"independent verification"}, Source: "sensor-verification", Modality: "sensor", ObservedAt: t0.Add(time.Minute), Reliability: 1},
	} {
		if _, err := runtime.ObserveOutcome(outcome); err != nil {
			t.Fatal(err)
		}
	}

	state, err := runtime.DNF().SynapseState(source.ID, target.ID, false)
	if err != nil {
		t.Fatal(err)
	}
	if state.Weight <= 0.05 {
		t.Fatalf("validated promotion did not strengthen dynamic synapse: %v", state.Weight)
	}
	if state.Confidence <= 0.20 {
		t.Fatalf("validated promotion did not strengthen synapse confidence: %v", state.Confidence)
	}

	after, err := runtime.CognitiveProcess(Event{
		ID: "p2-after",
		Stimulus: []string{"source"},
		Cycles: 1,
		Timestamp: t0.Add(2 * time.Minute),
	})
	if err != nil {
		t.Fatal(err)
	}
	afterActivation := after.Activations[target.ID]
	if afterActivation <= baseline {
		t.Fatalf("next cognitive cycle did not reflect strengthened pathway: before=%v after=%v", baseline, afterActivation)
	}
	if after.BrainIdentity != BrainIdentity {
		t.Fatalf("brain identity changed: got %q want %q", after.BrainIdentity, BrainIdentity)
	}

	afterStructure, err := runtime.DNF().Inspect(t0.Add(2 * time.Minute))
	if err != nil {
		t.Fatal(err)
	}
	if afterStructure.DynamicSynapses != beforeStructure.DynamicSynapses || afterStructure.Nodes != beforeStructure.Nodes {
		t.Fatalf("plasticity unexpectedly changed topology: before=%+v after=%+v", beforeStructure, afterStructure)
	}
}
