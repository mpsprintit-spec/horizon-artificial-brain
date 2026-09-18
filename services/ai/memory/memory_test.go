package memory

import (
	"testing"
	"time"

	"github.com/project-horizon/horizon-core/services/ai/knowledge"
)

func TestOptimizeUpdatesCanonicalDynamicState(t *testing.T) {
	brain := knowledge.NewBrain()
	source := brain.Store("source")
	target := brain.Store("target")
	brain.Connect(source, target, 0.8, 0.9, false)
	synapse := source.FindDynamicSynapse(target.ID, false)
	if synapse == nil {
		t.Fatal("expected dynamic synapse")
	}
	stale := time.Unix(100, 0).UTC()
	synapse.Dynamic.LastActivation = stale
	synapse.LastActivation = stale

	NewEngine(brain).Optimize(stale.Add(31*24*time.Hour), 30*24*time.Hour, 0.10)

	if synapse.Dynamic.Weight >= 0.8 {
		t.Fatalf("expected dynamic weight to decay: %v", synapse.Dynamic.Weight)
	}
	if synapse.Weight != synapse.Dynamic.Weight {
		t.Fatalf("legacy weight mirror diverged: weight=%v dynamic=%v", synapse.Weight, synapse.Dynamic.Weight)
	}
	if synapse.Confidence != synapse.Dynamic.Confidence {
		t.Fatalf("legacy confidence mirror diverged: confidence=%v dynamic=%v", synapse.Confidence, synapse.Dynamic.Confidence)
	}
	if !synapse.Dynamic.LastModification.Equal(stale.Add(31 * 24 * time.Hour)) {
		t.Fatalf("expected modification timestamp to advance: %v", synapse.Dynamic.LastModification)
	}
}
