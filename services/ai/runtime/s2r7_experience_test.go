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
	"slices"
	"testing"
	"time"
)

func s2r7Paths(t *testing.T) (string, string, [][]k.NodeID) {
	root, work := os.Getenv("HORIZON_S2R7_SOURCE"), os.Getenv("HORIZON_S2R7_WORK")
	if root == "" || work == "" {
		t.Skip("authenticated isolated R7 research only")
	}
	if root == work || filepath.Dir(work) == root {
		t.Fatal("source/work overlap")
	}
	for _, name := range []string{"HORIZON_S2R6_SOURCE", "HORIZON_S2R5_SOURCE", "HORIZON_S2_SOURCE"} {
		t.Setenv(name, root)
	}
	for _, name := range []string{"HORIZON_S2R6_WORK", "HORIZON_S2R5_WORK", "HORIZON_S2_WORK"} {
		t.Setenv(name, work)
	}
	return root, work, s1Pop(t, filepath.Join(root, "source/source/source"))
}
func s2r7Write(t *testing.T, p string, v any) {
	t.Helper()
	s1Check(t, os.MkdirAll(filepath.Dir(p), 0700))
	s1Write(t, p, v)
}
func s2r7Squash(v float64) float64 { return v / (1 + math.Abs(v)) }

// Six scalar channels per existing sequence row; the codec neither multiplies
// activity by a delta nor evaluates a forecast error or utility equation.
func s2r7Input(ctx k.S2R7Context, original k.CompositionInput, state []float64, descriptor string, retrospective *float64) k.CompositionInput {
	if descriptor == "D0" && retrospective == nil {
		return s2r6UtilityInput(ctx.Original, original.Scene.ObservedAt, 4)
	}
	paths := slices.Clone(ctx.Pathways)
	k.SortS2R7Pathways(paths)
	seq := [][]float64{}
	if descriptor == "D4" {
		for i := 0; i < len(state); i += 6 {
			v := make([]float64, 7)
			copy(v, state[i:min(i+6, len(state))])
			v[6] = 1
			seq = append(seq, v)
		}
	}
	if descriptor == "D2" || descriptor == "D3" || descriptor == "D4" {
		for i, row := range original.Sequence {
			v := slices.Clone(row)
			if descriptor == "D2" {
				if i == 0 {
					v[0], v[1] = 0, 0
				}
				if i == 1 {
					for j := 3; j < 6; j++ {
						v[j] = 0
					}
				}
			}
			seq = append(seq, v)
		}
	}
	for i := 3; i >= 0; i-- {
		a, b := make([]float64, 7), make([]float64, 7)
		if i < len(paths) {
			p := paths[i]
			a = []float64{s2r7Squash(p.Value), p.Weight / 8, p.Delta / 8, p.Activation, s2r7Squash(p.Eligibility), float64(p.Fanout) / 32, 1}
			b[4] = float64(len(p.ParentValues)) / 2
			b[6] = 1
			for j, v := range p.ParentValues {
				if j < 2 {
					b[j] = s2r7Squash(v)
				}
			}
			for j, v := range p.ParentActivations {
				if j < 2 {
					b[2+j] = v
				}
			}
		}
		seq = append(seq, b, a)
	}
	seq = append(seq, []float64{ctx.Before, float64(len(paths)) / 4, 0, 0, 0, 0, 1})
	word := make([]float64, 16)
	word[4] = 1
	word[15] = .125
	if retrospective != nil {
		word[4] = 0
		word[5] = 1
		seq[len(seq)-1][2] = *retrospective
		seq[len(seq)-1][3] = 1
	}
	precision := .00001
	return k.CompositionInput{Words: [][]float64{word}, Scene: original.Scene, Sequence: seq, RequiredPrecision: &precision}
}

type s2r7Row struct {
	ID, OriginID, Stream, SourceChecksum, Descriptor         string
	Stage, Step                                              int
	BeforeSequence, OperationSequence                        uint64
	Intervention                                             k.S2R6Intervention
	Context                                                  k.S2R7Context
	Original                                                 k.CompositionInput
	State                                                    []float64
	Input                                                    k.CompositionInput
	Representation                                           k.S2R7Representation
	Before, After, Actual, Utility, Contribution, Prediction float64
	DescriptorNanoseconds, PredictionNanoseconds             int64
	ObservedAt                                               time.Time
}

// Every invocation logs each learner's forecast durably before the observer's
// factual native Step. Only the post hook receives the measured target.
func s2r7Acquire(t *testing.T, observer *BrainRuntime, learners map[string]*BrainRuntime, pop [][]k.NodeID, dir string, seed int64, streams, steps int, shift, transfer bool, descriptors map[string]string, training bool) (map[string][]s2r7Row, []s2r4Row) {
	t.Helper()
	s1Check(t, os.MkdirAll(dir, 0700))
	f, err := os.OpenFile(filepath.Join(dir, "preutility.jsonl"), os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
	s1Check(t, err)
	defer f.Close()
	keys := []string{}
	digests := map[string]string{}
	for name, r := range learners {
		keys = append(keys, name)
		raw, e := r.brain.CanonicalJSON()
		s1Check(t, e)
		digests[name] = s1Digest(raw)
	}
	slices.Sort(keys)
	out := map[string][]s2r7Row{}
	pending := map[string][]s2r7Row{}
	rng := rand.New(rand.NewSource(seed + 7711))
	features := []int{}
	for i := 0; i < 32; i++ {
		if (i%3 == 0) == transfer {
			features = append(features, i)
		}
	}
	pre := func(p s2Pre, id, stream string, step int, seq uint64) {
		state := []float64{}
		for _, group := range pop {
			for _, node := range group {
				state = append(state, p.Before[node])
			}
		}
		stage := 1 + (len(out[keys[0]])/128)%7
		for trial := 0; trial < 2; trial++ {
			ids := []int{features[rng.Intn(len(features))]}
			if (!training || stage >= 4) && (step+trial)%2 == 0 {
				other := features[rng.Intn(len(features))]
				if other != ids[0] {
					ids = append(ids, other)
				}
			}
			deltas := make([]float64, len(ids))
			magnitudes := []float64{-.08, -.02, 0, .02, .08}
			if training && stage == 1 {
				magnitudes = []float64{-.02, 0, .02}
			}
			if !training && shift {
				magnitudes = []float64{-.11, -.035, 0, .035, .11}
			}
			for j := range deltas {
				deltas[j] = magnitudes[rng.Intn(len(magnitudes))]
			}
			change := k.S2R6Intervention{Ordinals: ids, Delta: deltas, Clamp: (step+trial)%5 == 0}
			for _, name := range keys {
				r := learners[name]
				source := s2Input(p, pop)
				start := time.Now()
				ctx, e := r.brain.S2R7Describe(source, change)
				s1Check(t, e)
				in := s2r7Input(ctx, source, state, descriptors[name], nil)
				descriptorCost := time.Since(start).Nanoseconds()
				start = time.Now()
				prediction := s2r6Read(t, r, in)
				cost := time.Since(start).Nanoseconds()
				rep, e := r.brain.S2R7Observe(in)
				s1Check(t, e)
				uid := fmt.Sprintf("r7-%d-%s-%d-%d", seed, id, trial, seq)
				row := s2r7Row{ID: uid, OriginID: id, Stream: stream, SourceChecksum: digests[name], Descriptor: descriptors[name], Stage: stage, Step: step, BeforeSequence: seq, Intervention: change, Context: ctx, Original: source, State: state, Input: in, Representation: rep, Prediction: prediction, DescriptorNanoseconds: descriptorCost, PredictionNanoseconds: cost}
				pending[name] = append(pending[name], row)
				s1Check(t, json.NewEncoder(f).Encode(map[string]any{"ID": uid, "Arm": name, "OriginID": id, "BeforeSequence": seq, "Input": in, "InputSHA256": s2r3JSONHash(t, in), "Prediction": prediction, "Intervention": change}))
			}
		}
		s1Check(t, f.Sync())
	}
	post := func(measured s2r4Row) {
		for _, name := range keys {
			for _, row := range pending[name] {
				before, after, e := learners[name].brain.S2R6Counterfactual(row.Original, row.Intervention)
				s1Check(t, e)
				y := measured.Trial.Error
				row.Before, row.After, row.Actual = before, after, y
				row.Utility = ((y-before)*(y-before) - (y-after)*(y-after)) / 4
				row.Contribution = after - before
				row.OperationSequence = measured.OperationSequence
				row.ObservedAt = s2r4Now(observer)
				if row.BeforeSequence >= row.OperationSequence {
					t.Fatal("forecast after operation")
				}
				out[name] = append(out[name], row)
			}
			pending[name] = nil
		}
	}
	rows := s2r4CollectHooks(t, observer, pop, filepath.Join(dir, "observer"), "persistent", seed, streams, steps, shift, pre, post)
	for _, name := range keys {
		s2r7Write(t, filepath.Join(dir, name+"-rows.json"), out[name])
	}
	return out, rows
}

func TestS2R7CodecBoundary(t *testing.T) {
	ctx := k.S2R7Context{Before: .1, Pathways: []k.S2R7Pathway{{Value: .8, Weight: .2, Delta: .08}, {Value: .2, Weight: .2, Delta: -.08}}}
	in := s2r6UtilityInput(k.S2R6Context{Measurements: make([]float64, 18)}, s1Base, 3)
	a := s2r7Input(ctx, in, nil, "D1", nil)
	slices.Reverse(ctx.Pathways)
	b := s2r7Input(ctx, in, nil, "D1", nil)
	if !reflect.DeepEqual(a, b) {
		t.Fatal("pathway permutation changed descriptor")
	}
	ctx.Pathways[0].Delta, ctx.Pathways[1].Delta = ctx.Pathways[1].Delta, ctx.Pathways[0].Delta
	c := s2r7Input(ctx, in, nil, "D1", nil)
	if reflect.DeepEqual(a, c) {
		t.Fatal("association destroyed")
	}
	if err := a.Validate(); err != nil {
		t.Fatal(err)
	}
}
