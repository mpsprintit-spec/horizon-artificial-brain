//go:build horizon_s2r8 && horizon_s2r4 && horizon_s2r3 && horizon_s2r2

package runtime

import (
	"encoding/json"
	"fmt"
	k "github.com/project-horizon/horizon-core/services/ai/knowledge"
	"math"
	"os"
	"path/filepath"
	"testing"
)

type s2r8Contrast struct {
	Kind                               string
	A, B                               s2Trial
	CurrentStateEqual, TargetDifferent bool
}

func s2r8Observed(t *testing.T, r *BrainRuntime, pop [][]k.NodeID, id string, state map[k.NodeID]float64, action int, gain float64, history []float64) s2Trial {
	s2r4Command(t, r, NativeControlCommand{ID: id + "-begin", Begin: &NativeBegin{EpisodeID: id, Observation: &NativeObservation{State: state, Source: "r8-contrast-observation", SampleID: id}}})
	p := s2Pre{Before: r.activation.CurrentState(), Action: action, PreviousAction: -1, At: s2r4Now(r), History: history, HistoryAction: 0}
	p.Forecast = s1Predict(t, r, s1Sample{Before: p.Before, Action: action, PreviousAction: -1, BeforeAt: p.At}, pop)
	encoded, err := json.Marshal(s2Input(p, pop))
	s1Check(t, err)
	ledger, err := os.OpenFile(filepath.Join(os.Getenv("R8_WORK"), "distinctness/preoperation.jsonl"), os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0600)
	s1Check(t, err)
	s1Check(t, json.NewEncoder(ledger).Encode(map[string]any{"id": id, "input_sha256": s1Digest(encoded), "before_sequence": r.LastSequence(), "forecast": p.Forecast, "estimate": s2Predict(t, r, p, pop)}))
	s1Check(t, ledger.Sync())
	s1Check(t, ledger.Close())
	command := NativeControlCommand{ID: id + "-target", Step: &NativeStep{EpisodeID: id, ActionOverride: &action}}
	if action == 0 {
		command.Step.EnvironmentGain = &gain
	}
	step := s2r4Command(t, r, command)
	actual := s1Target(step.State, pop)
	residual := make([]float64, 3)
	magnitude := 0.
	for i := range residual {
		residual[i] = actual[i] - p.Forecast[i]
		magnitude += math.Abs(residual[i]) / 3
	}
	if !step.Stopped {
		stop := 1
		step = s2r4Command(t, r, NativeControlCommand{ID: id + "-stop", Step: &NativeStep{EpisodeID: id, ActionOverride: &stop}})
	}
	s2r4Command(t, r, NativeControlCommand{ID: id + "-close", Outcome: &NativeOutcome{EpisodeID: id, TraceDigest: step.TraceDigest, Source: "r8-contrast-observation", OutcomeID: id, Provenance: "observed-environment", Learn: false}})
	return s2Trial{ID: id, Pre: p, Actual: actual, Residual: residual, Error: magnitude, AfterAt: s2r4Now(r), InputSHA256: s1Digest(encoded)}
}

func TestS2R8DistinctnessAcquire(t *testing.T) {
	root, work, _, pop := s2r8Paths(t)
	cp, wal := s2r3Source(root)
	dir := filepath.Join(work, "distinctness")
	r, w := s2r2Fork(t, cp, wal, dir)
	defer w.Close()
	pairs := []s2r8Contrast{}
	for _, kind := range []string{"similar-input-different-relation", "equal-mean-activity-weight-relation", "different-operation", "available-history-context", "unobserved-future-dynamics"} {
		for n := 0; n < 16; n++ {
			a, b := map[k.NodeID]float64{}, map[k.NodeID]float64{}
			for group, p := range pop {
				for _, id := range p {
					v := .04 + .02*float64(group) + .001*float64(n)
					a[id] = v
					b[id] = v
				}
			}
			aa, ba := 0, 0
			ag, bg := .7, .7
			var ah, bh []float64
			switch kind {
			case "similar-input-different-relation":
				for _, id := range pop[5] {
					a[id] = .6
					b[id] = .59
				}
				for _, id := range pop[6] {
					a[id] = .03
					b[id] = .04
				}
			case "equal-mean-activity-weight-relation":
				for i, id := range pop[5] {
					if i%2 == 0 {
						a[id] = .2
						b[id] = .7
					} else {
						a[id] = .7
						b[id] = .2
					}
				}
			case "different-operation":
				ba = 1
			case "available-history-context":
				ag, bg = .1, .95
				priorA := s2r8Observed(t, r, pop, fmt.Sprintf("r8-context-prior-a-%d", n), a, 0, ag, nil)
				priorB := s2r8Observed(t, r, pop, fmt.Sprintf("r8-context-prior-b-%d", n), b, 0, bg, nil)
				for i := 0; i < 3; i++ {
					ah = append(ah, math.Abs(priorA.Residual[i]))
					bh = append(bh, math.Abs(priorB.Residual[i]))
				}
			case "unobserved-future-dynamics":
				ag, bg = .1, .95
			}
			x := s2r8Observed(t, r, pop, fmt.Sprintf("r8-contrast-%s-%d-a", kind, n), a, aa, ag, ah)
			y := s2r8Observed(t, r, pop, fmt.Sprintf("r8-contrast-%s-%d-b", kind, n), b, ba, bg, bh)
			pairs = append(pairs, s2r8Contrast{Kind: kind, A: x, B: y, CurrentStateEqual: k.NativeStateDigest(x.Pre.Before) == k.NativeStateDigest(y.Pre.Before), TargetDifferent: math.Abs(x.Error-y.Error) > 1e-10})
		}
	}
	s2r8Write(t, filepath.Join(dir, "CASES.json"), pairs)
	s1Check(t, r.Checkpoint(filepath.Join(dir, "final.json")))
	s2r8Write(t, filepath.Join(dir, "LOCK.json"), map[string]any{"sha256": s2r3FileHash(t, filepath.Join(dir, "CASES.json")), "pairs": 80, "future_gain_not_encoded": true, "training_on_cases": false})
}

func TestS2R8DistinctnessEvaluate(t *testing.T) {
	root, work, _, pop := s2r8Paths(t)
	var pairs []s2r8Contrast
	s1Read(t, filepath.Join(work, "distinctness/CASES.json"), &pairs)
	inputs := []k.CompositionInput{}
	for _, pair := range pairs {
		inputs = append(inputs, s2Input(pair.A.Pre, pop), s2Input(pair.B.Pre, pop))
	}
	roots := s2r8Roots(t, root)
	cp, _ := s2r3Source(root)
	paths := map[string]string{"source": cp}
	for _, seed := range s2r8Seeds {
		paths[fmt.Sprintf("B-%d", seed)] = filepath.Join(work, fmt.Sprintf("seed-%d/B/final.json", seed))
	}
	for name, path := range paths {
		if name != "source" {
			s2r8Unzip(t, path)
		}
		r := orderedFixtureClone(t, path)
		out, err := r.brain.S2R8Inspect(inputs, roots)
		s1Check(t, err)
		s2r8Write(t, filepath.Join(work, "distinctness", name+"-trace.json"), out)
		if name != "source" {
			s1Check(t, os.Remove(path))
		}
	}
}
