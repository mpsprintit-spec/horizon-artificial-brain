package activation

import (
	"testing"
	"time"

	"github.com/project-horizon/horizon-core/services/ai/knowledge"
)

func TestPredictOutcomeFromNodesReflectsContradictoryEvidenceStrength(t *testing.T) {
	brain := knowledge.NewBrain()
	target := brain.Store("target")
	outcomeB := brain.Store("outcome-b")
	outcomeC := brain.Store("outcome-c")
	now := time.Date(2026, 9, 30, 12, 0, 0, 0, time.UTC)

	brain.ConnectAt(target, outcomeB, 0.8, 0.9, false, now)
	brain.ConnectAt(target, outcomeB, 0.8, 0.9, false, now.Add(time.Second))
	brain.ConnectAt(target, outcomeC, 0.8, 0.9, false, now.Add(2*time.Second))

	engine := NewEngine(brain)
	prediction := engine.PredictOutcomeFromNodes([]knowledge.NodeID{target.ID}, now.Add(3*time.Second), 1)

	b := prediction.State[outcomeB.ID]
	c := prediction.State[outcomeC.ID]
	if b <= 0 || c <= 0 {
		t.Fatalf("expected both contradictory outcomes in prediction: B=%v C=%v", b, c)
	}
	if b <= c {
		t.Fatalf("repeated evidence should increase relative predictive strength: B=%v C=%v", b, c)
	}
}

func TestPredictOutcomeFromNodesKeepsWeakerContradictoryOutcomeRepresentable(t *testing.T) {
	brain := knowledge.NewBrain()
	target := brain.Store("target")
	outcomeB := brain.Store("outcome-b")
	outcomeC := brain.Store("outcome-c")
	now := time.Date(2026, 9, 30, 12, 0, 0, 0, time.UTC)

	brain.ConnectAt(target, outcomeB, 0.8, 0.9, false, now)
	brain.ConnectAt(target, outcomeB, 0.8, 0.9, false, now.Add(time.Second))
	brain.ConnectAt(target, outcomeC, 0.8, 0.9, false, now.Add(2*time.Second))

	engine := NewEngine(brain)
	prediction := engine.PredictOutcomeFromNodes([]knowledge.NodeID{target.ID}, now.Add(3*time.Second), 1)

	if prediction.State[outcomeC.ID] <= 0 {
		t.Fatal("weaker contradictory outcome was erased from prediction")
	}
}
