package core

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/project-horizon/horizon-core/services/ai/knowledge"
	"github.com/project-horizon/horizon-core/services/ai/learning"
	"github.com/project-horizon/horizon-core/services/ai/perception"
	"github.com/project-horizon/horizon-core/services/ai/runtime"
)

func TestNativePulseRetainsLearnedContextAcrossSignalsAndHonorsReset(t *testing.T) {
	r := runtime.NewBrainRuntime(knowledge.NewBrain())
	at := time.Unix(100, 0).UTC()
	r.SetClock(runtime.NewFixedClock(at))
	for _, sequence := range [][]string{{"xa", "cue", "left"}, {"ya", "cue", "right"}} {
		if _, err := r.LearnExperience(learning.Experience{Sequence: sequence, Weight: .9, Confidence: .9}, at); err != nil {
			t.Fatal(err)
		}
	}
	graph, err := r.InspectBrain()
	if err != nil {
		t.Fatal(err)
	}
	a, _ := graph.Graph.NodeForSurface("left")
	b, _ := graph.Graph.NodeForSurface("right")
	prefix := filepath.Join(t.TempDir(), "trained.json")
	if err = r.Checkpoint(prefix); err != nil {
		t.Fatal(err)
	}
	for _, reset := range []bool{false, true} {
		clone := runtime.NewBrainRuntime(knowledge.NewBrain())
		if err = clone.RestoreCheckpoint(prefix); err != nil {
			t.Fatal(err)
		}
		owner, err := runtime.NewRuntimeSupervisor(clone)
		if err != nil {
			t.Fatal(err)
		}
		defer owner.Close()
		h := &HorizonEngine{supervisor: owner}
		if _, err = h.PulseSignals([]perception.PerceptionSignal{
			{SessionID: "one", Source: "user", Modality: "text", Tokens: []string{"ya"}, ObservedAt: at.Add(time.Second)},
			{SessionID: "one", ResetContext: reset, Source: "user", Modality: "text", Tokens: []string{"cue"}, ObservedAt: at.Add(2 * time.Second)},
		}); err != nil {
			t.Fatal(err)
		}
		path := filepath.Join(t.TempDir(), "after.json")
		if err = owner.WithMutation(func(active *runtime.BrainRuntime) error { return active.Checkpoint(path) }); err != nil {
			t.Fatal(err)
		}
		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		var snapshot runtime.BrainSnapshot
		if err = json.Unmarshal(data, &snapshot); err != nil {
			t.Fatal(err)
		}
		margin := snapshot.Activation.LastPrediction[a] - snapshot.Activation.LastPrediction[b]
		if !reset && margin > -.2 || reset && (margin < -.03 || margin > .03) {
			t.Fatalf("native context/reset failed: reset=%v margin=%g", reset, margin)
		}
	}
}
