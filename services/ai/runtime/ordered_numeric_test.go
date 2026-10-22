package runtime

import (
	"fmt"
	"reflect"
	"slices"
	"testing"
	"time"

	"github.com/project-horizon/horizon-core/services/ai/knowledge"
	"github.com/project-horizon/horizon-core/services/ai/learning"
)

func TestOrderedNumericSensorPopulationsLearnWithoutLexicalLabels(t *testing.T) {
	r := NewBrainRuntime(knowledge.NewBrain())
	at := time.Unix(100, 0).UTC()
	populations := make([][]knowledge.NodeID, 0, 5)
	// Obtain populations through the physical sensor boundary, not token IDs.
	for i, values := range [][]float64{{-1, -1, -1}, {1, -1, -1}, {-1, 1, -1}, {-1, -1, 1}, {1, 1, 1}} {
		out, err := r.ProcessSensorObservation(SensorObservation{SampleID: fmt.Sprintf("sample-%d", i), Source: "probe", Modality: "imu", Timestamp: at.Add(time.Duration(i) * time.Second), ChannelStates: values, Confidence: .9})
		if err != nil {
			t.Fatal(err)
		}
		populations = append(populations, slices.Clone(out.GroundedRepresentations[0].Population))
	}
	for i := range populations {
		for j := 0; j < i; j++ {
			if slices.Equal(populations[i], populations[j]) {
				t.Fatal("probe did not obtain distinct sensory populations")
			}
		}
	}
	for exposure := 0; exposure < 5; exposure++ {
		for side := 0; side < 2; side++ {
			sequence := [][]knowledge.NodeID{populations[side], populations[1-side], populations[2], populations[3+side]}
			if _, err := r.LearnExperience(learning.Experience{NeuralSequence: sequence, Weight: .9, Confidence: .9, Source: "probe"}, at.Add(5*time.Second)); err != nil {
				t.Fatal(err)
			}
		}
	}
	if len(r.brain.SurfaceAnnotationsSnapshot()) != 0 {
		t.Fatal("numeric learning created lexical labels")
	}
	checkpoint := orderedSave(t, r)
	for side := 0; side < 2; side++ {
		var reference *BrainRuntime
		for permutation := 0; permutation < 2; permutation++ {
			clone := orderedFixtureClone(t, checkpoint)
			trace := (&knowledge.ObservationTrace{Stimulus: [][]knowledge.NodeID{populations[side], populations[1-side], populations[2]}}).Clone()
			if permutation == 1 {
				for _, frame := range trace.Stimulus {
					slices.Reverse(frame)
				}
			}
			if _, err := clone.CognitiveProcess(Event{ID: "numeric-cue", Timestamp: at.Add(6 * time.Second), Cycles: 1, NeuralTrace: trace}); err != nil {
				t.Fatal(err)
			}
			prediction := clone.activation.PredictionSnapshot().State
			strength := func(frame []knowledge.NodeID) float64 {
				v := 0.0
				for _, id := range frame {
					v += prediction[id]
				}
				return v / float64(len(frame))
			}
			if strength(populations[3+side]) <= strength(populations[4-side])+.1 {
				t.Fatalf("numeric order did not condition prediction: side=%d populations=%v strengths=%v/%v channels=%+v", side, populations, strength(populations[3+side]), strength(populations[4-side]), clone.brain.Patterns.PredictOrdered(trace))
			}
			if reference != nil && (!reflect.DeepEqual(reference.activation.SnapshotState(), clone.activation.SnapshotState()) || !reflect.DeepEqual(continuationBytes(t, reference), continuationBytes(t, clone))) {
				t.Fatal("incidental member order changed simultaneous population semantics")
			}
			reference = clone
		}
	}
}
