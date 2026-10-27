package runtime

// Optional historical integration check. Its data is explicitly supplied;
// ordinary package tests do not download prices or contact an exchange.
import (
	"encoding/json"
	"fmt"
	"github.com/project-horizon/horizon-core/services/ai/knowledge"
	"math"
	"os"
	"path/filepath"
	"sort"
	"testing"
	"time"
)

type nHistBar struct {
	O float64 `json:"o"`
	H float64 `json:"h"`
	L float64 `json:"l"`
	C float64 `json:"c"`
	V float64 `json:"v"`
}
type nHistCase struct {
	ID      string `json:"id"`
	Present struct {
		At      int64      `json:"observed_ms"`
		Asset   string     `json:"asset"`
		Candles []nHistBar `json:"candles"`
		ATR     float64    `json:"atr"`
	} `json:"present"`
}

func nHistInput(c nHistCase, q int) knowledge.CompositionInput {
	p := c.Present
	vol := []float64{}
	for _, b := range p.Candles[37:] {
		vol = append(vol, b.V)
	}
	sort.Float64s(vol)
	ref := (vol[5] + vol[6]) / 2
	in := nInputFixture(q, nXY{}, time.UnixMilli(p.At).UTC())
	in.Sequence = nil
	for i, b := range p.Candles {
		prev := b.O
		if i > 0 {
			prev = p.Candles[i-1].C
		}
		raw := []float64{(b.C - b.O) / p.ATR, (b.H - math.Max(b.O, b.C)) / p.ATR, (math.Min(b.O, b.C) - b.L) / p.ATR, (b.C - prev) / p.ATR, (b.O - prev) / p.ATR, math.Log(math.Max(b.V, 1e-12) / math.Max(ref, 1e-12))}
		v := make([]float64, 7)
		for j, x := range raw {
			v[j] = math.Tanh(x / 2)
		}
		v[6] = 1
		in.Sequence = append(in.Sequence, v)
	}
	return in
}
func nHistTruth(c nHistCase, q int) float64 {
	v := c.Present.Candles
	b := v[48]
	answer := false
	switch q {
	case 0:
		answer = b.C > b.O
	case 1:
		answer = b.H-math.Max(b.C, b.O) > math.Min(b.C, b.O)-b.L
	case 2:
		answer = v[48].C > v[45].C
	case 3:
		answer = v[40].C > v[40].O
	case 4:
		answer = v[40].C > v[36].C
	case 5:
		answer = (v[48].C > v[45].C) != (v[45].C > v[42].C)
	}
	if answer {
		return 1
	}
	return -1
}
func TestNResolutionHistoricalSemanticIntegration(t *testing.T) {
	path := os.Getenv("HORIZON_N_DATASET")
	if path == "" {
		t.Skip("explicit sealed historical dataset required")
	}
	raw, e := os.ReadFile(path)
	compositionCheck(t, e)
	var data struct {
		Groups []struct {
			Split string      `json:"split"`
			Cases []nHistCase `json:"cases"`
		} `json:"groups"`
	}
	compositionCheck(t, json.Unmarshal(raw, &data))
	train, test := []nHistCase{}, []nHistCase{}
	for _, g := range data.Groups {
		if g.Split == "train" {
			train = append(train, g.Cases...)
		}
		if g.Split == "test2026" {
			test = append(test, g.Cases...)
		}
	}
	if len(train) != 1462 || len(test) != 546 {
		t.Fatal("unexpected sealed dataset population")
	}
	r, log, dir, start := nNewRuntime(t)
	steps := 0
	diagnoses := []knowledge.NDiagnosis{}
	startLearn := time.Now()
	for offset := 0; offset < len(train); offset += 128 {
		chunk := train[offset:min(offset+128, len(train))]
		mid := len(chunk) / 2
		if mid < 4 {
			continue
		}
		for q := 0; q < 6; q++ {
			steps++
			at := nBase.Add(time.Duration(steps) * time.Millisecond)
			batch := &knowledge.NResolutionLesson{}
			for i, c := range chunk {
				in := nHistInput(c, q)
				o := knowledge.NObservation{Input: in, Outcome: nHistTruth(c, q), ObservedAt: in.Scene.ObservedAt, EpisodeID: c.ID}
				if i < mid {
					batch.Fit = append(batch.Fit, o)
				} else {
					batch.Check = append(batch.Check, o)
				}
			}
			side := batch.Fit[0].Outcome
			exp := knowledge.CompositionExposure{ID: fmt.Sprintf("historical-N-%d", steps), Source: "sealed-historical-chart-observations", EpisodeID: fmt.Sprintf("batch-%d-q%d", offset/128, q), IndependenceGroup: "historical-cases-have-overlapping-windows", CausalGroup: fmt.Sprintf("batch-%d", offset/128), Reliability: 1, Input: batch.Fit[0].Input, Resolution: batch, Motor: []float64{0, 0, side, 1}, After: nScene(at)}
			exp.After.Patches[0].Values[3] = side
			d := nTeach(t, r, exp)
			diagnoses = append(diagnoses, d)
		}
		t.Logf("historical batch %d/%d features=%d", offset/128+1, (len(train)+127)/128, len(r.brain.BrainState.NResolution.Features))
	}
	learnSeconds := time.Since(startLearn).Seconds()
	correct, available, majorityCorrect := [6]int{}, [6]int{}, [6]int{}
	priors := [6]int{}
	for _, c := range train {
		for q := 0; q < 6; q++ {
			if nHistTruth(c, q) > 0 {
				priors[q]++
			}
		}
	}
	latency := []float64{}
	records := []map[string]any{}
	for _, c := range test {
		values := []float64{}
		statuses := []string{}
		begin := time.Now()
		for q := 0; q < 6; q++ {
			in := nHistInput(c, q)
			out, e := r.ReadComposition(in, in.Scene.ObservedAt)
			compositionCheck(t, e)
			v := 0.
			if out.NResolution != nil {
				v = out.NResolution.Value
			}
			values = append(values, v)
			statuses = append(statuses, out.Status)
			if out.Accepted {
				available[q]++
				if nSign(v) == nHistTruth(c, q) {
					correct[q]++
				}
			}
			if (priors[q]*2 >= len(train)) == (nHistTruth(c, q) > 0) {
				majorityCorrect[q]++
			}
		}
		latency = append(latency, time.Since(begin).Seconds()*1000)
		records = append(records, map[string]any{"id": c.ID, "values": values, "statuses": statuses})
	}
	final := filepath.Join(dir, "historical-final.json")
	compositionCheck(t, r.Checkpoint(final))
	expected := continuationBytes(t, r)
	restored := NewBrainRuntime(knowledge.NewBrain())
	compositionCheck(t, restored.RestoreCheckpoint(final))
	restored.SetClock(NewFixedClock(nBase.Add(time.Duration(steps) * time.Millisecond)))
	replayed, e := RebuildRuntimeFromCheckpointAndJournal(log, start)
	compositionCheck(t, e)
	replayed.SetClock(NewFixedClock(nBase.Add(time.Duration(steps) * time.Millisecond)))
	if !reflectNBytes(expected, continuationBytes(t, restored)) || !reflectNBytes(expected, continuationBytes(t, replayed)) {
		t.Fatal("historical full state restore/replay differs")
	}
	compositionCheck(t, log.Close())
	root := os.Getenv("HORIZON_N_RESULTS")
	if root != "" {
		compositionCheck(t, os.MkdirAll(root, 0700))
		for _, name := range []string{"historical-final.json", "events.jsonl", "start.json"} {
			b, e := os.ReadFile(filepath.Join(dir, name))
			compositionCheck(t, e)
			compositionCheck(t, os.WriteFile(filepath.Join(root, "historical-"+name), b, 0600))
		}
	}
	sort.Float64s(latency)
	mean := 0.
	for _, x := range latency {
		mean += x
	}
	mean /= float64(len(latency))
	nSaveResult(t, "historical_results.json", map[string]any{"train_cases": len(train), "test_cases": len(test), "lessons": steps, "diagnoses": diagnoses, "correct_accepted": correct, "available": available, "training_majority_baseline_correct": majorityCorrect, "learned_n_units": len(r.brain.BrainState.NResolution.Features), "total_neural_units": len(r.brain.Registry.Nodes()), "retained_witnesses": len(r.brain.BrainState.NResolution.Witnesses), "learning_seconds": learnSeconds, "six_question_read_mean_ms": mean, "six_question_read_p50_ms": latency[len(latency)/2], "six_question_read_p95_ms": latency[int(float64(len(latency))*.95)], "records": records, "full_restore_equal": true, "full_journal_replay_equal": true, "already_examined_development_period": true, "price_forecast_or_trading_test": false})
	t.Logf("historical semantics correct accepted=%v available=%v of %d, N=%d", correct, available, len(test), len(r.brain.BrainState.NResolution.Features))
}
func reflectNBytes(a, b []byte) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
