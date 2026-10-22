package runtime

import (
	"math/rand"
	"testing"
	"time"

	"github.com/project-horizon/horizon-core/services/ai/knowledge"
	"github.com/project-horizon/horizon-core/services/ai/learning"
)

func TestRecurrentContextUsesNumericPopulationHistoryWithoutSurfaceLabels(t *testing.T) {
	at := time.Unix(100, 0).UTC()
	for _, seed := range []int64{7, 17, 47} {
		random := rand.New(rand.NewSource(seed))
		axes := random.Perm(8)
		r := NewBrainRuntime(knowledge.NewBrain())
		var populations [][]knowledge.NodeID
		for i := 0; i < 5; i++ {
			values := make([]float64, 8)
			values[axes[i]] = float64(random.Intn(2)*2 - 1)
			grounded, err := r.GroundVectorAt(knowledge.NewNeuralVector(values), at)
			if err != nil {
				t.Fatal(err)
			}
			populations = append(populations, grounded.Population)
		}
		for exposure := 0; exposure < 5; exposure++ {
			for side := 0; side < 2; side++ {
				_, err := r.LearnExperience(learning.Experience{NeuralSequence: [][]knowledge.NodeID{populations[side], populations[2], populations[3+side]}, Weight: .9, Confidence: .9}, at)
				if err != nil {
					t.Fatal(err)
				}
			}
		}
		checkpoint := orderedSave(t, r)
		for side := 0; side < 2; side++ {
			clone := orderedFixtureClone(t, checkpoint)
			if _, err := clone.CognitiveProcess(Event{ID: "context", EpisodeID: "numeric", Timestamp: at.Add(time.Second), StimulusNodeIDs: populations[side], Cycles: 1}); err != nil {
				t.Fatal(err)
			}
			clone = orderedFixtureClone(t, orderedSave(t, clone))
			if _, err := clone.CognitiveProcess(Event{ID: "cue", EpisodeID: "numeric", Timestamp: at.Add(2 * time.Second), StimulusNodeIDs: populations[2], Cycles: 1}); err != nil {
				t.Fatal(err)
			}
			prediction := clone.activation.PredictionSnapshot().State
			strength := func(pop []knowledge.NodeID) float64 {
				v := 0.0
				for _, id := range pop {
					v += prediction[id]
				}
				return v / float64(len(pop))
			}
			if strength(populations[3+side]) <= strength(populations[4-side])+.1 {
				t.Fatalf("numeric context failed: seed=%d side=%d", seed, side)
			}
			if len(clone.brain.SurfaceAnnotationsSnapshot()) != 0 {
				t.Fatal("numeric eligibility created lexical identity")
			}
		}
	}
}
