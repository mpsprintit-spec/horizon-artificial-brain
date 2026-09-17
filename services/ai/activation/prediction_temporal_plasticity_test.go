package activation

import (
	"testing"
	"time"

	"github.com/project-horizon/horizon-core/services/ai/knowledge"
)

func TestPredictionErrorUsesElapsedEligibility(t *testing.T) {
	brain := knowledge.NewBrain()
	source := brain.Store("source")
	target := brain.Store("target")
	brain.Connect(source, target, 0.40, 0.50, false)

	e := NewEngine(brain)
	synapses := source.OutboundAll()
	if len(synapses) != 1 {
		t.Fatal("expected one synapse")
	}
	synapse := synapses[0]
	synapse.Dynamic.Eligibility = 1
	t0 := time.Unix(300, 0).UTC()
	synapse.Dynamic.LastEligibilityUpdate = t0

	predicted := map[knowledge.NodeID]float64{target.ID: 0.10}
	actual := map[knowledge.NodeID]float64{target.ID: 0.90}

	e.ApplyPredictionErrorPlasticity(predicted, actual, 1, t0.Add(time.Second))
	oneSecond := synapse.Dynamic.Weight
	oneSecondEligibility := synapse.Dynamic.Eligibility
	if oneSecondEligibility >= 0.60 || oneSecondEligibility <= 0.40 {
		t.Fatalf("expected approximately half-life eligibility after one second, got %v", oneSecondEligibility)
	}
	if oneSecond <= 0.40 {
		t.Fatalf("expected prediction error to strengthen eligible pathway, got %v", oneSecond)
	}

	beforeSecond := synapse.Dynamic.Weight
	e.ApplyPredictionErrorPlasticity(predicted, actual, 1, t0.Add(2*time.Second))
	twoSecondEligibility := synapse.Dynamic.Eligibility
	if twoSecondEligibility >= oneSecondEligibility {
		t.Fatalf("expected eligibility to continue decaying: one=%v two=%v", oneSecondEligibility, twoSecondEligibility)
	}
	if synapse.Dynamic.Weight-beforeSecond >= oneSecond-0.40 {
		t.Fatalf("later prediction-error update was not reduced by temporal eligibility: firstDelta=%v secondDelta=%v", oneSecond-0.40, synapse.Dynamic.Weight-beforeSecond)
	}
	if !synapse.Dynamic.LastEligibilityUpdate.Equal(t0.Add(2 * time.Second)) {
		t.Fatalf("unexpected eligibility timestamp: %v", synapse.Dynamic.LastEligibilityUpdate)
	}
}
