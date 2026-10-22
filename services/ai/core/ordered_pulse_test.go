package core

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/project-horizon/horizon-core/services/ai/knowledge"
	"github.com/project-horizon/horizon-core/services/ai/learning"
	"github.com/project-horizon/horizon-core/services/ai/perception"
	"github.com/project-horizon/horizon-core/services/ai/runtime"
)

func TestNativePulsePredictsDifferentHeldOutTargetsForReversedSequence(t *testing.T) {
	r := runtime.NewBrainRuntime(knowledge.NewBrain())
	at := time.Unix(100, 0).UTC()
	r.SetClock(runtime.NewFixedClock(at))
	for _, sequence := range []string{"naku melo cue tiva", "melo naku cue luma"} {
		if _, err := r.LearnExperience(learning.Experience{Sequence: strings.Fields(sequence), Weight: .9, Confidence: .9}, at); err != nil {
			t.Fatal(err)
		}
	}
	graph, err := r.InspectBrain()
	if err != nil {
		t.Fatal(err)
	}
	a, _ := graph.Graph.NodeForSurface("tiva")
	b, _ := graph.Graph.NodeForSurface("luma")
	prefix := filepath.Join(t.TempDir(), "trained.json")
	if err = r.Checkpoint(prefix); err != nil {
		t.Fatal(err)
	}
	for side, prompt := range []string{"naku melo cue", "melo naku cue"} {
		clone := runtime.NewBrainRuntime(knowledge.NewBrain())
		if err = clone.RestoreCheckpoint(prefix); err != nil {
			t.Fatal(err)
		}
		owner, err := runtime.NewRuntimeSupervisor(clone)
		if err != nil {
			t.Fatal(err)
		}
		defer owner.Close()
		h := &HorizonEngine{supervisor: owner, Perception: fixedPerception{signal: perception.PerceptionSignal{Source: "user", Modality: "text", Tokens: strings.Fields(prompt), ObservedAt: at.Add(time.Second)}}}
		if _, err = h.Pulse(prompt); err != nil {
			t.Fatal(err)
		}
		path := filepath.Join(t.TempDir(), "after-pulse.json")
		if err = owner.WithMutation(func(active *runtime.BrainRuntime) error { return active.Checkpoint(path) }); err != nil {
			t.Fatal(err)
		}
		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		var s runtime.BrainSnapshot
		if err = json.Unmarshal(data, &s); err != nil {
			t.Fatal(err)
		}
		margin := s.Activation.LastPrediction[a] - s.Activation.LastPrediction[b]
		if side == 0 && margin < .2 || side == 1 && margin > -.2 {
			t.Fatalf("native Pulse collapsed ordered cue %q: margin=%g", prompt, margin)
		}
	}
}
