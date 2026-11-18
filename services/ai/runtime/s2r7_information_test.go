//go:build horizon_s2r7 && horizon_s2r6 && horizon_s2r5 && horizon_s2r4 && horizon_s2r3 && horizon_s2r2

package runtime

import (
	"encoding/json"
	"fmt"
	k "github.com/project-horizon/horizon-core/services/ai/knowledge"
	"math"
	"math/rand"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestS2R7Information(t *testing.T) {
	root, work, pop := s2r7Paths(t)
	cp, wal := s2r3Source(root)
	dir := filepath.Join(work, "information")
	r, w := s2r2Fork(t, cp, wal, filepath.Join(dir, "alias-observer"))
	defer w.Close()
	reader := orderedFixtureClone(t, cp)
	f, e := os.OpenFile(filepath.Join(dir, "alias-preutility.jsonl"), os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
	s1Check(t, e)
	defer f.Close()
	pending := []s2r7Row{}
	results := []any{}
	pre := func(p s2Pre, id, stream string, step int, seq uint64) {
		for trial := 0; trial < 2; trial++ {
			sign := 1.
			if trial == 1 {
				sign = -1
			}
			change := k.S2R6Intervention{Ordinals: []int{step % 32, (step + 1) % 32}, Delta: []float64{sign * .08, -sign * .08}}
			in := s2Input(p, pop)
			ctx, err := reader.brain.S2R7Describe(in, change)
			s1Check(t, err)
			row := s2r7Row{ID: fmt.Sprintf("%s-%d", id, trial), OriginID: id, Stream: stream, BeforeSequence: seq, Intervention: change, Context: ctx, Original: in}
			pending = append(pending, row)
			s1Check(t, json.NewEncoder(f).Encode(map[string]any{"ID": row.ID, "BeforeSequence": seq, "D0": s2r7Input(ctx, in, nil, "D0", nil), "D1": s2r7Input(ctx, in, nil, "D1", nil)}))
		}
		s1Check(t, f.Sync())
	}
	post := func(measured s2r4Row) {
		for i := range pending {
			row := &pending[i]
			before, after, err := reader.brain.S2R6Counterfactual(row.Original, row.Intervention)
			s1Check(t, err)
			y := measured.Trial.Error
			row.Before, row.After, row.Actual = before, after, y
			row.Utility = ((y-before)*(y-before) - (y-after)*(y-after)) / 4
			row.OperationSequence = measured.OperationSequence
		}
		a, b := pending[0], pending[1]
		a0, b0 := s2r7Input(a.Context, a.Original, nil, "D0", nil), s2r7Input(b.Context, b.Original, nil, "D0", nil)
		a1, b1 := s2r7Input(a.Context, a.Original, nil, "D1", nil), s2r7Input(b.Context, b.Original, nil, "D1", nil)
		results = append(results, map[string]any{"A": a, "B": b, "D0_exact_alias": reflect.DeepEqual(a0.Sequence, b0.Sequence), "D1_exact_alias": reflect.DeepEqual(a1.Sequence, b1.Sequence), "utility_difference": a.Utility - b.Utility})
		pending = nil
	}
	s2r4CollectHooks(t, r, pop, filepath.Join(dir, "alias-acquisition"), "persistent", 272100037, 2, 32, false, pre, post)
	s2r7Write(t, filepath.Join(dir, "alias-pairs.json"), results)
	// Matched initial observations, no history, distinct unobserved gains. This
	// is a controlled uncertainty diagnostic, never a native utility answer.
	a, aw := s2r2Fork(t, cp, wal, filepath.Join(dir, "hidden-low"))
	defer aw.Close()
	b, bw := s2r2Fork(t, cp, wal, filepath.Join(dir, "hidden-high"))
	defer bw.Close()
	twins := []any{}
	rng := rand.New(rand.NewSource(272200037))
	for i := 0; i < 32; i++ {
		state := map[k.NodeID]float64{}
		for _, group := range pop {
			for _, id := range group {
				state[id] = .03 + .6*rng.Float64()
			}
		}
		pair := []s2r7Row{}
		digests := []string{}
		for j, runtime := range []*BrainRuntime{a, b} {
			id := fmt.Sprintf("r7-hidden-%d", i)
			s2r4Command(t, runtime, NativeControlCommand{ID: id + "-begin", Begin: &NativeBegin{EpisodeID: id, Observation: &NativeObservation{State: state, Source: "r7-controlled-twin", SampleID: id}}})
			p := s2Pre{Before: runtime.activation.CurrentState(), Action: 0, PreviousAction: -1, HistoryAction: -1, At: s2r4Now(runtime)}
			p.Forecast = s1Predict(t, runtime, s1Sample{Before: p.Before, Action: 0, PreviousAction: -1, BeforeAt: p.At}, pop)
			in := s2Input(p, pop)
			change := k.S2R6Intervention{Ordinals: []int{i}, Delta: []float64{.08}}
			ctx, err := runtime.brain.S2R7Describe(in, change)
			s1Check(t, err)
			forecast := s2r6Read(t, runtime, s2r7Input(ctx, in, nil, "D3", nil))
			beforeSequence := runtime.LastSequence()
			digests = append(digests, k.NativeStateDigest(p.Before))
			s1Check(t, json.NewEncoder(f).Encode(map[string]any{"ID": id, "Twin": j, "BeforeSequence": beforeSequence, "Input": in, "Prediction": forecast}))
			s1Check(t, f.Sync())
			gain := .10
			if j == 1 {
				gain = .97
			}
			action := 0
			op := s2r4Command(t, runtime, NativeControlCommand{ID: id + "-operation", Step: &NativeStep{EpisodeID: id, ActionOverride: &action, EnvironmentGain: &gain}})
			actual := s1Target(op.State, pop)
			y := 0.
			for q := range actual {
				y += math.Abs(actual[q]-p.Forecast[q]) / 3
			}
			before, after, err := runtime.brain.S2R6Counterfactual(in, change)
			s1Check(t, err)
			utility := ((y-before)*(y-before) - (y-after)*(y-after)) / 4
			pair = append(pair, s2r7Row{ID: id, BeforeSequence: beforeSequence, OperationSequence: op.Sequence, Original: in, Context: ctx, Actual: y, Before: before, After: after, Utility: utility, Prediction: forecast})
			stop := 1
			last := s2r4Command(t, runtime, NativeControlCommand{ID: id + "-stop", Step: &NativeStep{EpisodeID: id, ActionOverride: &stop}})
			s2r4Command(t, runtime, NativeControlCommand{ID: id + "-outcome", Outcome: &NativeOutcome{EpisodeID: id, TraceDigest: last.TraceDigest, Source: "r7-controlled-twin", OutcomeID: id, Provenance: "observed-environment", Learn: false}})
		}
		z := pair[0].Utility - pair[1].Utility
		twins = append(twins, map[string]any{"low": pair[0], "high": pair[1], "full_prestate_exact": digests[0] == digests[1], "preinput_exact": reflect.DeepEqual(pair[0].Original, pair[1].Original), "conditional_utility_variance_equal_mixture": z * z / 4})
	}
	s2r7Write(t, filepath.Join(dir, "hidden-twins.json"), twins)
}

func TestS2R7InformationAccess(t *testing.T) {
	root, work, _ := s2r7Paths(t)
	cp, _ := s2r3Source(root)
	r := orderedFixtureClone(t, cp)
	var pairs []struct{ A, B s2r7Row }
	s1Read(t, filepath.Join(work, "information/alias-pairs.json"), &pairs)
	out := []any{}
	for _, pair := range pairs {
		record := map[string]any{"OriginID": pair.A.OriginID, "UtilityA": pair.A.Utility, "UtilityB": pair.B.Utility}
		for _, d := range []string{"D0", "D1", "D2", "D3"} {
			a, e := r.brain.S2R7Observe(s2r7Input(pair.A.Context, pair.A.Original, nil, d, nil))
			s1Check(t, e)
			b, e := r.brain.S2R7Observe(s2r7Input(pair.B.Context, pair.B.Original, nil, d, nil))
			s1Check(t, e)
			record[d] = map[string]any{"A": a, "B": b, "input_exact_alias": reflect.DeepEqual(a.Input, b.Input), "activation_exact_alias": reflect.DeepEqual(a.Activation, b.Activation), "metric_exact_alias": reflect.DeepEqual(a.Metric, b.Metric)}
		}
		out = append(out, record)
	}
	s2r7Write(t, filepath.Join(work, "information/controlled-access.json"), out)
}
