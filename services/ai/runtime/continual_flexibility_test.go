package runtime

// A data-driven research harness, not a new serving API or cognitive pipeline.
// Evaluation copies never replace or train the one main A->B->C->D lineage.
import (
	"bytes"
	"encoding/json"
	"fmt"
	"math"
	"math/bits"
	"os"
	"path/filepath"
	"slices"
	"testing"
	"time"

	k "github.com/project-horizon/horizon-core/services/ai/knowledge"
	"github.com/project-horizon/horizon-core/services/ai/learning"
)

type flexExposure struct{ Context, Sequence []int }
type flexCurriculum struct {
	Seed            int64
	Repetitions     int
	A, B, D         []flexExposure
	TrainAmplitudes []float64 `json:"train_amplitudes"`
	EvalAmplitudes  []float64 `json:"eval_amplitudes"`
	NativeRounds    int       `json:"native_rounds"`
}
type flexPolicyProbe struct {
	Amplitude                             float64
	Condition                             int
	Values, Probabilities, FactualReturns []float64
	Greedy, Optimal                       int
}
type flexPolicyMetrics struct {
	OptimalProbability, Accuracy, AdvantageError float64
	Probes                                       []flexPolicyProbe
}
type flexRecord struct {
	Stage, Identity, CanonicalHash, ContinuationHash, CheckpointChecksum string
	Sequence                                                             uint64
	Neurons, Synapses, Patterns                                          int
	ChangedSynapses                                                      int
	NativeUpdates                                                        uint64
	A, B, D, MissingIntermediate                                         []float64
	C                                                                    *flexPolicyMetrics `json:",omitempty"`
	GenesisReplay, SuffixReplay                                          bool
}

func flexAt(r *BrainRuntime) time.Time {
	return time.Date(2026, 11, 14, 0, 0, 0, 0, time.UTC).Add(time.Duration(r.LastSequence()+1) * time.Millisecond)
}
func flexJSON(t *testing.T, path string, value any) {
	t.Helper()
	raw, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	if err = os.WriteFile(path, append(raw, '\n'), 0600); err != nil {
		t.Fatal(err)
	}
}
func flexFrames(pop [][]k.NodeID, indices []int, reverse bool) [][]k.NodeID {
	frames := make([][]k.NodeID, 0, len(indices))
	for _, i := range indices {
		p := slices.Clone(pop[i])
		if reverse {
			slices.Reverse(p)
		}
		frames = append(frames, p)
	}
	return frames
}
func flexMean(state map[k.NodeID]float64, frame []k.NodeID) float64 {
	v := 0.0
	for _, id := range frame {
		v += state[id]
	}
	return v / float64(len(frame))
}
func flexQuery(t *testing.T, r *BrainRuntime, pop [][]k.NodeID, stimulus, context []int, reset, reverse bool) {
	t.Helper()
	_, err := r.CognitiveProcess(Event{ID: fmt.Sprintf("eval-%d", r.LastSequence()+1), Timestamp: flexAt(r), Cycles: 1, EpisodeID: "flex-evaluation", ResetContext: reset,
		NeuralTrace: &k.ObservationTrace{Stimulus: flexFrames(pop, stimulus, reverse), Context: flexFrames(pop, context, reverse)}})
	if err != nil {
		t.Fatal(err)
	}
}
func flexMargins(t *testing.T, checkpoint string, pop [][]k.NodeID, curriculum []flexExposure, ablate ...bool) []float64 {
	t.Helper()
	margins := []float64{}
	for variant := 0; variant < 4; variant++ {
		for side, e := range curriculum {
			r := orderedFixtureClone(t, checkpoint)
			if len(ablate) > 0 && ablate[0] {
				r.brain.Patterns = k.NewPatternIndex()
			}
			cue := slices.Clone(e.Sequence[:len(e.Sequence)-1])
			if variant >= 2 {
				cue = append([]int{8}, cue...)
			}
			flexQuery(t, r, pop, cue, e.Context, true, variant%2 == 1)
			prediction := r.activation.PredictionSnapshot().State
			target := e.Sequence[len(e.Sequence)-1]
			other := curriculum[1-side].Sequence[len(curriculum[1-side].Sequence)-1]
			margins = append(margins, flexMean(prediction, pop[target])-flexMean(prediction, pop[other]))
		}
	}
	return margins
}
func flexReuse(t *testing.T, checkpoint string, pop [][]k.NodeID, observed bool) []float64 {
	t.Helper()
	result := []float64{}
	for variant := 0; variant < 2; variant++ {
		for side := 0; side < 2; side++ {
			r := orderedFixtureClone(t, checkpoint)
			cue := [][]int{{0, 1, 2}, {1, 0, 2}}[side]
			if variant == 1 {
				cue = append([]int{8}, cue...)
			}
			flexQuery(t, r, pop, cue, nil, true, variant == 1)
			first := r.activation.PredictionSnapshot().State
			result = append(result, flexMean(first, pop[3+side])-flexMean(first, pop[4-side]))
			if observed {
				flexQuery(t, r, pop, []int{3 + side}, nil, false, variant == 1)
			} else {
				if _, _, err := r.ThinkAt(1, flexAt(r)); err != nil {
					t.Fatal(err)
				}
			}
			flexQuery(t, r, pop, []int{5}, nil, false, variant == 1)
			second := r.activation.PredictionSnapshot().State
			result = append(result, flexMean(second, pop[6+side])-flexMean(second, pop[7-side]))
		}
	}
	return result
}
func flexCState(pop [][]k.NodeID, condition int, amplitude float64) map[k.NodeID]float64 {
	frame := pop[5]
	if condition == 1 {
		frame = pop[6]
	}
	state := map[k.NodeID]float64{}
	for _, id := range frame {
		state[id] = amplitude
	}
	return state
}
func flexApply(t *testing.T, r *BrainRuntime, c NativeControlCommand) NativeControlResult {
	t.Helper()
	out, err := r.ApplyNativeControl(c, flexAt(r))
	if err != nil {
		t.Fatalf("%s: %v", c.ID, err)
	}
	return out
}

// Both alternatives really execute; the reward meter reads the resulting
// canonical activation. The stop arm does not borrow a forecast of propagation.
func flexTrial(t *testing.T, r *BrainRuntime, pop [][]k.NodeID, state map[k.NodeID]float64, id, pair string, action int, train bool) float64 {
	t.Helper()
	flexApply(t, r, NativeControlCommand{ID: id, Begin: &NativeBegin{EpisodeID: id, Training: train, Intervention: train, MatchKey: pair,
		Observation: &NativeObservation{Source: "numeric-flex-world", SampleID: id, State: state}}})
	step := func(a int) NativeControlResult {
		c := NativeStep{EpisodeID: id}
		if train {
			c.ExperimentAction = &a
		} else {
			c.ActionOverride = &a
		}
		return flexApply(t, r, NativeControlCommand{ID: fmt.Sprintf("%s-%d", id, r.LastSequence()), Step: &c})
	}
	out := step(action)
	if !out.Stopped {
		out = step(1)
	}
	if !out.Stopped {
		t.Fatal("trial did not stop")
	}
	reward := math.Min(1, flexMean(out.State, pop[6])+flexMean(out.State, pop[7]))
	mode := ""
	if train {
		mode = "action"
	}
	flexApply(t, r, NativeControlCommand{ID: id + "-outcome", Outcome: &NativeOutcome{EpisodeID: id, TraceDigest: out.TraceDigest, Source: "numeric-flex-world", OutcomeID: id,
		Provenance: "observed-environment", Reward: reward, Learn: train, ContrastID: pair, ContrastMode: mode}})
	if action == 0 {
		reward -= .05
	}
	return reward
}
func flexC(t *testing.T, checkpoint string, pop [][]k.NodeID, amplitudes []float64, ablate ...bool) *flexPolicyMetrics {
	t.Helper()
	metrics := &flexPolicyMetrics{}
	for _, amplitude := range amplitudes {
		for condition := 0; condition < 2; condition++ {
			r := orderedFixtureClone(t, checkpoint)
			if len(ablate) > 0 && ablate[0] {
				s := r.brain.NativeControlSnapshot()
				for _, from := range s.ParameterSources() {
					for _, group := range s.Outputs {
						for _, to := range group {
							e := r.brain.Registry.GetByID(from).Synapses[to][0]
							e.Weight = 0
							e.Dynamic.Weight = 0
							e.Inhibitory = false
						}
					}
				}
			}
			state := flexCState(pop, condition, amplitude)
			p, err := r.brain.NativePolicy(state, true, 0)
			if err != nil {
				t.Fatal(err)
			}
			returns := []float64{}
			for action := 0; action < 2; action++ {
				fork := orderedFixtureClone(t, checkpoint)
				returns = append(returns, flexTrial(t, fork, pop, state, fmt.Sprintf("eval-%d", action), "", action, false))
			}
			best := 0
			if returns[1] > returns[0] {
				best = 1
			}
			metrics.OptimalProbability += p.Probabilities[best]
			if p.Action == best {
				metrics.Accuracy++
			}
			metrics.AdvantageError += math.Abs((p.Values[0] - p.Values[1]) - (returns[0] - returns[1]))
			metrics.Probes = append(metrics.Probes, flexPolicyProbe{amplitude, condition, p.Values, p.Probabilities, returns, p.Action, best})
		}
	}
	n := float64(len(metrics.Probes))
	metrics.OptimalProbability /= n
	metrics.Accuracy /= n
	metrics.AdvantageError /= n
	return metrics
}
func flexEdges(t *testing.T, r *BrainRuntime) map[string]string {
	t.Helper()
	raw, err := r.brain.CanonicalJSON()
	if err != nil {
		t.Fatal(err)
	}
	var data struct{ Synapses []json.RawMessage }
	if err = json.Unmarshal(raw, &data); err != nil {
		t.Fatal(err)
	}
	out := map[string]string{}
	for _, edge := range data.Synapses {
		var e struct {
			From k.NodeID `json:"source_id"`
			To   k.NodeID `json:"target_id"`
			Kind string   `json:"kind"`
		}
		if err = json.Unmarshal(edge, &e); err != nil {
			t.Fatal(err)
		}
		key := fmt.Sprintf("%d-%d-%s", e.From, e.To, e.Kind)
		out[key] = string(edge)
	}
	return out
}

func TestContinualLearningFlexibility(t *testing.T) {
	data, err := os.ReadFile("testdata/continual_flexibility_v1.json")
	if err != nil {
		t.Fatal(err)
	}
	var cfg flexCurriculum
	if err = json.Unmarshal(data, &cfg); err != nil {
		t.Fatal(err)
	}
	dir := os.Getenv("HORIZON_FLEX_EVIDENCE_DIR")
	if dir == "" {
		dir = t.TempDir()
	}
	if err = os.MkdirAll(dir, 0755); err != nil {
		t.Fatal(err)
	}
	wal, err := OpenEventLog(filepath.Join(dir, "events.jsonl"))
	if err != nil {
		t.Fatal(err)
	}
	defer wal.Close()
	r := NewBrainRuntime(k.NewBrain())
	owner := r.brain
	r.SetEventLog(wal)
	r.SetClock(NewFixedClock(flexAt(r)))
	empty := filepath.Join(dir, "empty.json")
	if err = r.Checkpoint(empty); err != nil {
		t.Fatal(err)
	}
	pop := [][]k.NodeID{}
	for i := 0; i < 9; i++ {
		v := make([]float64, 16)
		// Dense orthogonal Walsh inputs have normalized L1 separation .5.
		// One-hot inputs are too close under the existing L1 projection metric
		// (.0625 for 16 dimensions); their shared cells are a retained negative.
		for j := range v {
			v[j] = 1
			if bits.OnesCount(uint((i+1)&j))%2 == 1 {
				v[j] = -1
			}
		}
		out, err := r.GroundVectorAt(k.NewNeuralVector(v), flexAt(r))
		if err != nil {
			t.Fatal(err)
		}
		if len(out.Population) < 2 {
			t.Fatal("numeric representation not distributed")
		}
		for _, old := range pop {
			if slices.Equal(old, out.Population) {
				t.Fatal("fixture population collision")
			}
		}
		pop = append(pop, out.Population)
	}
	flexJSON(t, filepath.Join(dir, "populations.json"), pop)
	// A genuine production sensory packet joins this same journal and substrate.
	if _, err = r.ProcessSensorObservation(SensorObservation{SampleID: "foundation-sensor", Source: "numeric-flex-world", Modality: "imu", Timestamp: flexAt(r), ChannelStates: []float64{.2, -.5, .7}, Confidence: .9}); err != nil {
		t.Fatal(err)
	}
	previous := empty
	edges := flexEdges(t, r)
	records := []flexRecord{}
	checkpoint := func(stage string, policy bool) string {
		t.Helper()
		if r.brain != owner || r.activation.Memory != owner || r.learning.Kb != owner {
			t.Fatal("canonical Brain owner changed")
		}
		r.SetClock(NewFixedClock(flexAt(r)))
		path := filepath.Join(dir, stage+".json")
		if err = r.Checkpoint(path); err != nil {
			t.Fatal(err)
		}
		raw, err := r.brain.CanonicalJSON()
		if err != nil {
			t.Fatal(err)
		}
		var cp BrainSnapshot
		b, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		if err = json.Unmarshal(b, &cp); err != nil {
			t.Fatal(err)
		}
		nowEdges := flexEdges(t, r)
		changed := 0
		for key, value := range nowEdges {
			if edges[key] != value {
				changed++
			}
		}
		for key := range edges {
			if _, ok := nowEdges[key]; !ok {
				changed++
			}
		}
		edges = nowEdges
		record := flexRecord{Stage: stage, Identity: cp.BrainIdentity, Sequence: r.LastSequence(), CanonicalHash: k.NativeDigest(json.RawMessage(raw)), ContinuationHash: k.NativeDigest(json.RawMessage(continuationBytes(t, r))), CheckpointChecksum: cp.Checksum, Neurons: len(r.brain.Registry.Nodes()), Synapses: len(nowEdges), Patterns: len(r.brain.Patterns.All()), ChangedSynapses: changed}
		record.A = flexMargins(t, path, pop, cfg.A)
		record.B = flexMargins(t, path, pop, cfg.B)
		record.D = flexReuse(t, path, pop, true)
		record.MissingIntermediate = flexReuse(t, path, pop, false)
		if policy {
			record.C = flexC(t, path, pop, cfg.EvalAmplitudes)
			record.NativeUpdates = uint64(r.brain.NativeControlSnapshot().Updates)
		}
		genesis, err := RebuildRuntimeFromCommittedJournal(wal, k.NewBrain)
		if err != nil {
			t.Fatal(err)
		}
		suffix, err := RebuildRuntimeFromCheckpointAndJournal(wal, previous)
		if err != nil {
			t.Fatal(err)
		}
		for _, restored := range []*BrainRuntime{genesis, suffix} {
			if !bytes.Equal(continuationBytes(t, r), continuationBytes(t, restored)) {
				t.Fatal("complete mixed replay mismatch", stage)
			}
			restored.SetEventLog(wal)
			restored.SetClock(NewFixedClock(flexAt(r)))
			p := filepath.Join(t.TempDir(), "recovered.json")
			if err = restored.Checkpoint(p); err != nil {
				t.Fatal(err)
			}
			body, err := os.ReadFile(p)
			if err != nil {
				t.Fatal(err)
			}
			var snap BrainSnapshot
			if err = json.Unmarshal(body, &snap); err != nil {
				t.Fatal(err)
			}
			if snap.Checksum != cp.Checksum {
				t.Fatal("checkpoint integrity mismatch")
			}
		}
		record.GenesisReplay, record.SuffixReplay = true, true
		records = append(records, record)
		previous = path
		flexJSON(t, filepath.Join(dir, "results.json"), records)
		t.Logf("%s sequence=%d synapses_changed=%d A=%v B=%v C=%+v D=%v", stage, record.Sequence, changed, record.A, record.B, record.C, record.D)
		return path
	}
	train := func(label string, exposures []flexExposure) {
		for round := 0; round < cfg.Repetitions; round++ {
			for side := range exposures {
				e := exposures[(round+side)%len(exposures)]
				_, err := r.LearnExperience(learning.Experience{NeuralSequence: flexFrames(pop, e.Sequence, false), NeuralContext: flexFrames(pop, e.Context, false), Weight: .9, Confidence: .9, ExperienceID: fmt.Sprintf("%s-%d-%d", label, round, side), Source: "numeric-flex-world", Modality: "simulator"}, flexAt(r))
				if err != nil {
					t.Fatal(err)
				}
			}
		}
	}
	checkpoint("00-foundation", false)
	train("A", cfg.A)
	checkpoint("01-A", false)
	train("B", cfg.B)
	checkpoint("02-B", false)
	nativeCfg := k.NativeControlConfig{Version: k.NativeControlCausalVersion, Encoder: k.NativeRelationalEncoderV2, Sources: slices.Clone(pop[5]), Operations: []k.NativeOperation{{Kind: "propagate", Gain: 1, Cost: .05}, {Kind: "stop"}}, Seed: cfg.Seed, LearningRate: .5, Gamma: 1, Lambda: 1, Temperature: .15, Epsilon: .1, MaxSteps: 2, MaxActiveNodes: 4096, MaxSynapseReads: 100000}
	flexApply(t, r, NativeControlCommand{ID: "native-opt-in", Initialize: &nativeCfg})
	checkpoint("02-native-ready", true)
	for round := 0; round < cfg.NativeRounds; round++ {
		for _, amplitude := range cfg.TrainAmplitudes {
			for condition := 0; condition < 2; condition++ {
				state := flexCState(pop, condition, amplitude)
				pair := fmt.Sprintf("C-%d-%g-%d", round, amplitude, condition)
				first := round % 2
				for _, action := range []int{first, 1 - first} {
					flexTrial(t, r, pop, state, fmt.Sprintf("%s-%d", pair, action), pair, action, true)
				}
			}
		}
	}
	checkpoint("03-C", true)
	train("D", cfg.D)
	checkpoint("04-D", true)
	// Demonstrate restart+continuation without replacing the original lineage.
	restarted, err := RebuildRuntimeFromCheckpointAndJournal(wal, previous)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(continuationBytes(t, r), continuationBytes(t, restarted)) {
		t.Fatal("restart changed learning")
	}
	// Same continuation, including recurrent thought and learned native selection,
	// on live and restored state; only the primary lineage appends the new input.
	restarted.SetEventLog(nil)
	for _, rr := range []*BrainRuntime{r, restarted} {
		// The last controlled trial can legitimately leave no recurrent activity.
		// Present a new observed cue through the existing production boundary.
		flexQuery(t, rr, pop, []int{0}, nil, true, false)
		if _, _, err = rr.ThinkAt(1, flexAt(rr)); err != nil {
			t.Fatal(err)
		}
		flexApply(t, rr, NativeControlCommand{ID: "restart-begin", Begin: &NativeBegin{EpisodeID: "restart-episode"}})
		var out NativeControlResult
		for i := 0; i < 2; i++ {
			out = flexApply(t, rr, NativeControlCommand{ID: fmt.Sprintf("restart-step-%d", i), Step: &NativeStep{EpisodeID: "restart-episode", Sample: 0}})
			if out.Stopped {
				break
			}
		}
		flexApply(t, rr, NativeControlCommand{ID: "restart-outcome", Outcome: &NativeOutcome{EpisodeID: "restart-episode", TraceDigest: out.TraceDigest, Source: "numeric-flex-world", OutcomeID: "restart-outcome", Provenance: "observed-environment", Reward: math.Min(1, flexMean(out.State, pop[6])+flexMean(out.State, pop[7]))}})
	}
	if !bytes.Equal(continuationBytes(t, r), continuationBytes(t, restarted)) {
		t.Fatal("restart continuation diverged")
	}
	finalPath := checkpoint("05-restart-continuation", true)
	controls := map[string]any{"A_without_patterns": flexMargins(t, finalPath, pop, cfg.A, true), "B_without_patterns": flexMargins(t, finalPath, pop, cfg.B, true), "C_without_policy_synapses": flexC(t, finalPath, pop, cfg.EvalAmplitudes, true)}
	flexJSON(t, filepath.Join(dir, "controls.json"), controls)
	// Confirmatory decisions are written after all evidence, including failures.
	passMargins := func(values []float64) bool {
		for _, v := range values {
			if v < .1 {
				return false
			}
		}
		return true
	}
	for i := 1; i < len(records); i++ {
		if !passMargins(records[i].A) {
			t.Errorf("retention/training A failed at %s", records[i].Stage)
		}
	}
	for i := 2; i < len(records); i++ {
		if !passMargins(records[i].B) {
			t.Errorf("retention/training B failed at %s", records[i].Stage)
		}
	}
	for i, baseline := range records[0].A {
		if records[1].A[i] <= baseline {
			t.Error("A did not improve over baseline")
		}
	}
	for i, baseline := range records[1].B {
		if records[2].B[i] <= baseline {
			t.Error("B did not improve over baseline")
		}
	}
	before := records[3].C
	for _, i := range []int{4, 5, 6} {
		after := records[i].C
		if after.Accuracy < 5.0/6 || after.OptimalProbability < before.OptimalProbability+.1 || after.AdvantageError > before.AdvantageError*.7 {
			t.Errorf("training/retention C local target failed at %s: before=%+v after=%+v", records[i].Stage, before, after)
		}
	}
	if !passMargins(records[5].D) {
		t.Error("D observed-intermediate reuse target failed")
	}
	for _, i := range []int{4, 5, 6} {
		if records[i].NativeUpdates != uint64(cfg.NativeRounds*len(cfg.TrainAmplitudes)*2) {
			t.Error("outcome learned twice or update lost")
		}
	}
	for _, name := range []string{"A_without_patterns", "B_without_patterns"} {
		values := controls[name].([]float64)
		if passMargins(values) {
			t.Errorf("%s retained full learned discrimination; investigate attribution", name)
		}
	}
	if controls["C_without_policy_synapses"].(*flexPolicyMetrics).Accuracy >= records[6].C.Accuracy {
		t.Error("policy improvement survived canonical parameter ablation")
	}
}
