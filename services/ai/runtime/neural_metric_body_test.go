package runtime

import (
	"fmt"
	"github.com/project-horizon/horizon-core/services/ai/knowledge"
	"math"
	"math/rand"
	"os"
	"path/filepath"
	"sort"
	"testing"
	"time"
)

func mBodyInput(q int, x, v, target float64, at time.Time, tol float64) knowledge.CompositionInput {
	in := nInputFixture(q, nXY{x, v}, at)
	in.Sequence[0][2] = target
	if tol > 0 {
		in.RequiredPrecision = &tol
	}
	return in
}
func mBodyLesson(phase int, rng *rand.Rand, staged bool) knowledge.CompositionExposure {
	at := nBase.Add(time.Duration(phase+1) * time.Hour)
	b := &knowledge.NResolutionLesson{}
	q := 8
	if staged && phase < 4 {
		q = 9 + phase%2
	}
	for set := 0; set < 2; set++ {
		for i := 0; i < 32; i++ {
			x, v, target := rng.Float64()*1.6-.8, rng.Float64()*1.4-.7, rng.Float64()*1.6-.8
			if staged && phase < 2 {
				xs, ts, vs := []float64{-.6, -.2, .2, .6}, []float64{-.6, -.2, .2, .6}, []float64{-.4, .4}
				if set == 1 {
					xs = []float64{-.65, -.25, .25, .65}
					ts = []float64{-.55, -.15, .15, .55}
					vs = []float64{-.35, .35}
				}
				x, target, v = xs[(i/8)%4], ts[(i/2)%4], vs[i%2]
			}
			fine := staged && (phase == 2 || phase == 3 || phase == 5 || phase == 7)
			if fine {
				target = rng.Float64()*1.2 - .6
				x = target + (rng.Float64()*4-2)*.001
				v = (rng.Float64()*4 - 2) * .001
			}
			y := math.Max(-1, math.Min(1, .8*(target-x)-1.8*v))
			if q == 9 {
				y = .5 * (target - x)
			}
			if q == 10 {
				y = v
			}
			in := mBodyInput(q, x, v, target, at.Add(-time.Duration(i+1)*time.Millisecond), 0)
			o := knowledge.NObservation{Input: in, Outcome: y, ObservedAt: in.Scene.ObservedAt, EpisodeID: fmt.Sprintf("body-%d-%d-%d", phase, set, i)}
			if set == 0 {
				b.Fit = append(b.Fit, o)
			} else {
				b.Check = append(b.Check, o)
			}
		}
	}
	return knowledge.CompositionExposure{ID: fmt.Sprintf("body-%d", phase), Source: "synthetic-position-velocity-teacher", EpisodeID: fmt.Sprint(phase), IndependenceGroup: fmt.Sprint(phase), CausalGroup: "controlled-reference-action", Reliability: 1, Input: b.Fit[0].Input, Resolution: b, Motor: []float64{0, 0, 0, 0}, After: nScene(at)}
}
func mBodyRead(t *testing.T, r *BrainRuntime, x, v, target float64, tol float64) knowledge.CompositionReadout {
	t.Helper()
	in := mBodyInput(8, math.Max(-1, math.Min(1, x)), math.Max(-1, math.Min(1, v)), target, nBase.Add(24*time.Hour), tol)
	o, e := r.ReadComposition(in, in.Scene.ObservedAt)
	compositionCheck(t, e)
	return o
}
func mPct(v []float64, p float64) float64 {
	a := append([]float64(nil), v...)
	sort.Float64s(a)
	return a[int(float64(len(a)-1)*p)]
}
func TestMetricBodyCurriculumComparison(t *testing.T) {
	if os.Getenv("HORIZON_M_BODY") == "" {
		t.Skip("explicit numerical body benchmark required")
	}
	results := []any{}
	for _, seed := range []int64{7, 17, 47} {
		for _, arm := range []string{"v5-direct", "v6-direct", "v6-staged"} {
			version := knowledge.NeuralCompositionMetricV6
			tol := 1e-5
			if arm == "v5-direct" {
				version = knowledge.NeuralCompositionResolutionV5
				tol = 0
			}
			r, log, dir, _ := mNew(t, version)
			rng := rand.New(rand.NewSource(seed))
			ds := []knowledge.NDiagnosis{}
			learnStart := time.Now()
			for phase := 0; phase < 8; phase++ {
				ds = append(ds, mTeach(t, r, mBodyLesson(phase, rng, arm == "v6-staged"), tol))
			}
			learningMS := float64(time.Since(learnStart).Nanoseconds()) / 1e6
			eval := rand.New(rand.NewSource(104729 + seed))
			sse, mae := 0., 0.
			accepted := 0
			states := []any{}
			lat := []float64{}
			for i := 0; i < 512; i++ {
				x, v, target := eval.Float64()*1.6-.8, eval.Float64()*1.4-.7, eval.Float64()*1.6-.8
				start := time.Now()
				o := mBodyRead(t, r, x, v, target, tol)
				lat = append(lat, float64(time.Since(start).Nanoseconds())/1e6)
				value := 0.
				if o.NResolution != nil {
					value = o.NResolution.Value
				}
				want := math.Max(-1, math.Min(1, .8*(target-x)-1.8*v))
				sse += (value - want) * (value - want)
				mae += math.Abs(value - want)
				if o.Accepted {
					accepted++
				}
				states = append(states, map[string]any{"x": x, "v": v, "target": target, "teacher": want, "prediction": value, "accepted": o.Accepted, "readout": o.NResolution})
			}
			episodes := []any{}
			for _, policy := range []string{"native-gated", "native-raw", "teacher-reference"} {
				for _, target := range []float64{-.5, -.1, -.01, -.001, .001, .01, .1, .5} {
					x, v := 0., 0.
					over := 0.
					lastOutside := -1
					abstain := 0
					trace := []any{}
					for step := 0; step < 500; step++ {
						u := 0.
						if policy == "teacher-reference" {
							u = math.Max(-1, math.Min(1, .8*(target-x)-1.8*v))
						} else {
							o := mBodyRead(t, r, x, v, target, tol)
							if o.NResolution != nil {
								u = o.NResolution.Value
							}
							if !o.Accepted {
								abstain++
								if policy == "native-gated" {
									u = 0
								}
							}
						}
						v += .02 * (u - .05*v)
						x += .02 * v
						if math.Abs(target-x) > .001 {
							lastOutside = step
						}
						over = math.Max(over, (x-target)*math.Copysign(1, target))
						if step%25 == 0 || step == 499 {
							trace = append(trace, []float64{float64(step+1) * .02, x, v, u})
						}
					}
					episodes = append(episodes, map[string]any{"policy": policy, "target_m": target, "terminal_error_m": math.Abs(x - target), "overshoot_m": over, "settled_last_second_1mm": lastOutside < 450, "last_outside_step": lastOutside, "abstentions": abstain, "trace": trace})
				}
			}
			file := filepath.Join(dir, "body.json")
			compositionCheck(t, r.Checkpoint(file))
			root := os.Getenv("HORIZON_M_RESULTS")
			if root != "" {
				compositionCheck(t, os.MkdirAll(root, 0700))
				b, e := os.ReadFile(file)
				compositionCheck(t, e)
				compositionCheck(t, os.WriteFile(filepath.Join(root, fmt.Sprintf("body-%s-%d.json", arm, seed)), b, 0600))
			}
			compositionCheck(t, log.Close())
			results = append(results, map[string]any{"seed": seed, "arm": arm, "diagnoses": ds, "learned_n": len(r.brain.BrainState.NResolution.Features), "total_units": len(r.brain.Registry.Nodes()), "action_rmse": math.Sqrt(sse / 512), "action_mae": mae / 512, "accepted": accepted, "test_states": 512, "learning_ms": learningMS, "inference_ms_p50": mPct(lat, .5), "inference_ms_p95": mPct(lat, .95), "states": states, "episodes": episodes})
			mSave(t, "body_comparison.json", results)
			t.Logf("body %s seed %d RMSE %.8g accepted %d N %d", arm, seed, math.Sqrt(sse/512), accepted, len(r.brain.BrainState.NResolution.Features))
		}
	}
}
