package runtime

import (
	"bytes"
	"fmt"
	"math"
	"math/rand"
	"reflect"
	"slices"
	"testing"
	"time"

	"github.com/project-horizon/horizon-core/services/ai/knowledge"
)

func stableSensor(id string, at time.Time, v []float64, tolerance float64) SensorObservation {
	return SensorObservation{SampleID: id, SessionID: "stream", Source: "camera", Modality: "vision", Timestamp: at, ChannelStates: slices.Clone(v), Confidence: .9, NoiseTolerance: tolerance, Provenance: SensorProvenance{Adapter: "h14-numeric-probe"}}
}

func TestStableProjectionRepeatedAlternatingAndMultipleSensors(t *testing.T) {
	for seed := int64(1); seed <= 20; seed++ {
		for _, patterns := range []int{1, 2, 4} {
			r := NewBrainRuntime(knowledge.NewBootstrapBrain())
			at := time.Unix(100, 0).UTC()
			random := rand.New(rand.NewSource(seed))
			axes := random.Perm(8)
			var vectors [][]float64
			for i := 0; i < patterns; i++ {
				v := make([]float64, 8)
				v[axes[i]] = float64(2*random.Intn(2) - 1)
				vectors = append(vectors, v)
			}
			first := make([][]knowledge.NodeID, patterns)
			before := map[knowledge.NodeID][]float64{}
			warm, reuse := 0, 0
			for i := 0; i < 12*patterns; i++ {
				got, err := r.ProcessSensorObservation(stableSensor(fmt.Sprintf("p-%d", i), at.Add(time.Duration(i)*time.Second), vectors[i%patterns], 0))
				if err != nil {
					t.Fatal(err)
				}
				population := slices.Clone(got.GroundedRepresentations[0].Population)
				slices.Sort(population)
				if i < patterns {
					first[i] = population
					if i == patterns-1 {
						warm = len(r.brain.Registry.Nodes())
						for _, node := range r.brain.Registry.Nodes() {
							before[node.ID] = slices.Clone(node.Representation)
						}
					}
				} else if slices.Equal(population, first[i%patterns]) {
					reuse++
				}
				if seed == 17 {
					r.SetClock(NewFixedClock(at.Add(time.Duration(i) * time.Second)))
					if _, err := r.CognitiveThink(1); err != nil {
						t.Fatal(err)
					}
				}
			}
			if reuse != 11*patterns || len(r.brain.Registry.Nodes()) != warm {
				t.Fatalf("seed=%d patterns=%d reuse=%d/%d growth=%d", seed, patterns, reuse, 11*patterns, len(r.brain.Registry.Nodes())-warm)
			}
			moved := false
			for _, node := range r.brain.Registry.Nodes() {
				if !slices.Equal(before[node.ID], node.Representation) {
					moved = true
					break
				}
			}
			if patterns > 1 && !moved {
				t.Fatal("stable retrieval was obtained by freezing representation learning")
			}
		}
	}
}

func TestStableProjectionNoiseRequiresDeclaredUniqueMeasurementBasin(t *testing.T) {
	for _, tolerance := range []float64{0, .02} {
		r := NewBrainRuntime(knowledge.NewBrain())
		at := time.Unix(100, 0).UTC()
		first := make([][]knowledge.NodeID, 2)
		random := rand.New(rand.NewSource(17))
		warm, reuse := 0, 0
		for i := 0; i < 50; i++ {
			index := i % 2
			v := []float64{.2 + .6*float64(index), 0, 0, 0}
			if i >= 2 {
				for j := range v {
					v[j] += .01 * (2*random.Float64() - 1)
				}
			}
			got, err := r.ProcessSensorObservation(stableSensor(fmt.Sprintf("noise-%d", i), at.Add(time.Duration(i)*time.Second), v, tolerance))
			if err != nil {
				t.Fatal(err)
			}
			population := slices.Clone(got.GroundedRepresentations[0].Population)
			slices.Sort(population)
			if i < 2 {
				first[index] = population
				if i == 1 {
					warm = len(r.brain.Registry.Nodes())
				}
			} else if slices.Equal(population, first[index]) {
				reuse++
			}
		}
		growth := len(r.brain.Registry.Nodes()) - warm
		t.Logf("tolerance=%f noisy_reuse=%d/48 growth=%d", tolerance, reuse, growth)
		if tolerance > 0 && (reuse != 48 || growth != 0) {
			t.Fatal("bounded noise grew duplicate substrate despite one declared basin")
		}
		if tolerance == 0 && (reuse == 48 || growth == 0) {
			t.Fatal("zero-tolerance ablation silently equated distinct measurements")
		}
		if slices.Equal(first[0], first[1]) {
			t.Fatal("different sensor factors merged")
		}
	}
}

func TestStableProjectionRetrievesFirstOutcomePopulations(t *testing.T) {
	for _, seed := range []int64{7, 17, 47} {
		r := NewBrainRuntime(knowledge.NewBrain())
		at := time.Unix(100, 0).UTC()
		axes := rand.New(rand.NewSource(seed)).Perm(8)
		vectors := make([][]float64, 4)
		for i := range vectors {
			vectors[i] = make([]float64, 8)
			vectors[i][axes[i]] = 1
		}
		first := make([][]knowledge.NodeID, 4)
		for episode := 0; episode < 16; episode++ {
			for side := 0; side < 2; side++ {
				for step, index := range []int{2 * side, 2*side + 1} {
					sensor := stableSensor(fmt.Sprintf("train-%d-%d-%d", episode, side, step), at.Add(time.Duration(4*episode+2*side+step)*time.Second), vectors[index], 0)
					sensor.SessionID = fmt.Sprintf("episode-%d-%d", episode, side)
					got, err := r.ProcessSensorObservation(sensor)
					if err != nil {
						t.Fatal(err)
					}
					if episode == 0 {
						first[index] = slices.Clone(got.GroundedRepresentations[0].Population)
					}
				}
			}
		}
		path := orderedSave(t, r)
		for side := 0; side < 2; side++ {
			probe := orderedFixtureClone(t, path)
			sensor := stableSensor("heldout", at.Add(65*time.Second), vectors[2*side], 0)
			sensor.SessionID = "heldout"
			if _, err := probe.ProcessSensorObservation(sensor); err != nil {
				t.Fatal(err)
			}
			prediction := probe.activation.PredictionSnapshot().State
			mean := func(pop []knowledge.NodeID) float64 {
				sum := 0.0
				for _, id := range pop {
					sum += prediction[id]
				}
				return sum / float64(len(pop))
			}
			margin := mean(first[2*side+1]) - mean(first[2*(1-side)+1])
			t.Logf("seed=%d side=%d first_population_margin=%f", seed, side, margin)
			if margin < .02 {
				t.Fatal("retrieval requires a historical union or lost the learned successor")
			}
			ablated := orderedFixtureClone(t, path)
			ablated.brain.Lock()
			for _, source := range first[2*side] {
				for _, targetPopulation := range [][]knowledge.NodeID{first[1], first[3]} {
					for _, target := range targetPopulation {
						delete(ablated.brain.Registry.GetByID(source).Synapses, target)
					}
				}
			}
			ablated.brain.Unlock()
			if _, err := ablated.ProcessSensorObservation(sensor); err != nil {
				t.Fatal(err)
			}
			ablatedState := ablated.activation.PredictionSnapshot().State
			ablatedMargin := 0.0
			for _, id := range first[2*side+1] {
				ablatedMargin += ablatedState[id] / float64(len(first[2*side+1]))
			}
			for _, id := range first[2*(1-side)+1] {
				ablatedMargin -= ablatedState[id] / float64(len(first[2*(1-side)+1]))
			}
			if margin < ablatedMargin+.02 {
				t.Fatal("successor advantage survived removal of the learned source/outcome channels")
			}
			if len(probe.brain.SurfaceAnnotationsSnapshot()) != 0 {
				t.Fatal("numeric recovery invented lexical identity")
			}
		}
	}
}

func TestStableProjectionRejectsInvalidToleranceBeforeJournalOrMutation(t *testing.T) {
	for _, tolerance := range []float64{-.01, math.NaN(), math.Inf(1), 1.01} {
		r := NewBrainRuntime(knowledge.NewBrain())
		log := openKnowledgeJournal(t)
		r.SetEventLog(log)
		before := continuationBytes(t, r)
		sensor := stableSensor("invalid", time.Unix(100, 0), []float64{.2, .4}, tolerance)
		if _, err := r.ProcessSensorObservation(sensor); err == nil {
			t.Fatal("invalid tolerance accepted")
		}
		after := continuationBytes(t, r)
		events, err := ReadEventLog(log.file.Name())
		if err != nil {
			t.Fatal(err)
		}
		if !bytes.Equal(before, after) || len(events) != 0 || r.RecoveryRequired() {
			t.Fatal("invalid tolerance mutated state or appended intent")
		}
	}
	r := NewBrainRuntime(knowledge.NewBrain())
	r.dynamicsEncoding = knowledge.EvidenceUncertaintyDynamics
	if _, err := r.ProcessSensorObservation(stableSensor("old-mode", time.Unix(100, 0), []float64{.2, .4}, .02)); err == nil {
		t.Fatal("old dynamics silently accepted a new measurement contract")
	}
}

func TestStableProjectionToleranceCannotBlurSensorShape(t *testing.T) {
	sensor := stableSensor("a", time.Unix(100, 0), []float64{.2, .4}, .02)
	sensor.SpatialCoordinates = []float64{.1}
	sensor.TemporalDelta = []float64{.3}
	vector, err := sensor.NeuralVector()
	if err != nil {
		t.Fatal(err)
	}
	tolerance := sensor.projectionTolerance()
	if len(tolerance) != len(vector.Values) || !reflect.DeepEqual(tolerance, []float64{0, 0, .02, .02, 0, 0, .02, 0, 0, .02, 0}) {
		t.Fatal("measurement tolerance blurred layout markers or confidence")
	}
}
