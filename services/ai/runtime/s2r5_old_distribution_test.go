//go:build horizon_s2r5 && horizon_s2r4 && horizon_s2r3 && horizon_s2r2

package runtime

import (
	"bytes"
	"encoding/json"
	"fmt"
	k "github.com/project-horizon/horizon-core/services/ai/knowledge"
	"math"
	"math/rand"
	"os"
	"path/filepath"
	"testing"
)

func s2r5OldCollect(t *testing.T, r *BrainRuntime, pop [][]k.NodeID, seed int64, n int, shift bool, history []float64, historyAction int) []s2Trial {
	t.Helper()
	_, work := s2Paths(t)
	ledger, err := os.OpenFile(filepath.Join(work, "preoperation.jsonl"), os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0600)
	s1Check(t, err)
	defer ledger.Close()
	rng := rand.New(rand.NewSource(seed))
	rows := make([]s2Trial, 0, n)
	for i := 0; i < n; i++ {
		id := fmt.Sprintf("s2-%d-%03d", seed, i)
		state := map[k.NodeID]float64{}
		lo, hi := .05, .65
		if shift {
			lo, hi = .07, .73
		}
		for _, p := range pop {
			amp := lo + (hi-lo)*rng.Float64()
			for _, v := range p {
				state[v] = amp
			}
		}
		prior := -1
		var prev []float64
		if i%2 == 1 {
			prior = 0
			prev = s1Previous(state, pop)
		}
		if shift && i%6 == 3 {
			prior = 1
			prev = s1Previous(state, pop)
			pid := id + "-prior-stop"
			s2r4Command(t, r, NativeControlCommand{ID: pid + "-begin", Begin: &NativeBegin{EpisodeID: pid, Observation: &NativeObservation{State: state, Source: "s2-neural-observer", SampleID: pid}}})
			a := 1
			step := s2r4Command(t, r, NativeControlCommand{ID: pid + "-step", Step: &NativeStep{EpisodeID: pid, ActionOverride: &a}})
			state = step.State
			s2r4Command(t, r, NativeControlCommand{ID: pid + "-close", Outcome: &NativeOutcome{EpisodeID: pid, TraceDigest: step.TraceDigest, Source: "s2-neural-observer", OutcomeID: pid, Provenance: "observed-sensor", Learn: false}})
		}
		s2r4Command(t, r, NativeControlCommand{ID: id + "-begin", Begin: &NativeBegin{EpisodeID: id, Observation: &NativeObservation{State: state, Source: "s2-controlled-neural-stimulus", SampleID: id}}})
		if prior == 0 {
			a := 0
			s2r4Command(t, r, NativeControlCommand{ID: id + "-prior", Step: &NativeStep{EpisodeID: id, ActionOverride: &a}})
		}
		p := s2Pre{Before: r.activation.CurrentState(), Previous: prev, Action: (i / 2) % 2, PreviousAction: prior, At: s2r4Now(r), History: append([]float64(nil), history...), HistoryAction: historyAction}
		s := s1Sample{Before: p.Before, Previous: p.Previous, Action: p.Action, PreviousAction: p.PreviousAction, BeforeAt: p.At}
		before := continuationBytes(t, r)
		p.Forecast = s1Predict(t, r, s, pop)
		estimate := s2Predict(t, r, p, pop)
		if !bytes.Equal(before, continuationBytes(t, r)) {
			t.Fatal("pre-operation forecast mutated state")
		}
		encoded, e := json.Marshal(s2Input(p, pop))
		s1Check(t, e)
		// Durable evidence is written before the command that produces the label.
		s1Check(t, json.NewEncoder(ledger).Encode(map[string]any{"id": id, "before_sequence": r.LastSequence(), "input_sha256": s1Digest(encoded), "forecast": p.Forecast, "estimate": estimate}))
		s1Check(t, ledger.Sync())
		a := p.Action
		step := s2r4Command(t, r, NativeControlCommand{ID: id + "-target", Step: &NativeStep{EpisodeID: id, ActionOverride: &a}})
		actual := s1Target(step.State, pop)
		residual := make([]float64, 3)
		history = make([]float64, 3)
		magnitude := 0.
		for q := range actual {
			residual[q] = actual[q] - p.Forecast[q]
			history[q] = math.Abs(residual[q])
			magnitude += history[q] / 3
		}
		historyAction = p.Action
		if !step.Stopped {
			stop := 1
			step = s2r4Command(t, r, NativeControlCommand{ID: id + "-stop", Step: &NativeStep{EpisodeID: id, ActionOverride: &stop}})
		}
		s2r4Command(t, r, NativeControlCommand{ID: id + "-close", Outcome: &NativeOutcome{EpisodeID: id, TraceDigest: step.TraceDigest, Source: "s2-error-observation", OutcomeID: id, Provenance: "observed-sensor", Learn: false}})
		rows = append(rows, s2Trial{ID: id, Pre: p, Actual: actual, Residual: residual, Error: magnitude, Estimate: estimate, AfterAt: s2r4Now(r), InputSHA256: s1Digest(encoded)})
	}
	return rows
}
