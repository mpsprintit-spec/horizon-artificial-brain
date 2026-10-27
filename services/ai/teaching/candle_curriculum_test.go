package teaching

// Explicit opt-in research lesson. Teacher truth/contrast construction lives
// only in this experiment, never in the HNL inference bridge.
import (
	"bytes"
	"encoding/json"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"sort"
	"testing"
	"time"

	"github.com/project-horizon/horizon-core/services/ai/knowledge"
	brainruntime "github.com/project-horizon/horizon-core/services/ai/runtime"
)

type chartBar struct{ O, H, L, C, V float64 }
type chartCase struct {
	ID      string `json:"id"`
	Present struct {
		Asset   string     `json:"asset"`
		At      int64      `json:"observed_ms"`
		Candles []chartBar `json:"candles"`
	} `json:"present"`
}
type chartTask struct {
	Name string
	Q    Question
}

func chartTasks() []chartTask {
	r := func(l, c uint8) Ref { return Ref{l, c} }
	return []chartTask{
		{"body", Question{Domain: 3, Relation: Greater, A: r(0, 3), B: r(0, 0)}},
		{"wick", Question{Domain: 3, Relation: SumGreater, A: r(0, 1), B: r(0, 2), C: r(0, 0), D: r(0, 3)}},
		{"recent12h", Question{Domain: 3, Relation: Greater, A: r(0, 3), B: r(3, 3)}},
		{"previous12h", Question{Domain: 3, Relation: Greater, A: r(3, 3), B: r(6, 3)}},
		{"body32hAgo", Question{Domain: 3, Relation: Greater, A: r(8, 3), B: r(8, 0)}},
		{"middle16h", Question{Domain: 3, Relation: Greater, A: r(8, 3), B: r(12, 3)}},
		{"intervalAlignment", Question{Domain: 3, Relation: SameChange, A: r(0, 3), B: r(3, 3), C: r(3, 3), D: r(6, 3)}},
	}
}

// No answer-derived feature: raw O/H/L/C share one positive affine price
// transform. Volume has its own bounded transform. Channel 5 is reserved zero.
func chartExample(c chartCase, id uint64) Example {
	lo, hi := c.Present.Candles[0].L, c.Present.Candles[0].H
	for _, b := range c.Present.Candles {
		lo = math.Min(lo, b.L)
		hi = math.Max(hi, b.H)
	}
	scale := hi - lo
	center := (hi + lo) / 2
	vol := []float64{}
	for _, b := range c.Present.Candles[len(c.Present.Candles)-12:] {
		vol = append(vol, b.V)
	}
	sort.Float64s(vol)
	ref := (vol[5] + vol[6]) / 2
	e := Example{Role: Probe, Episode: id, ObservedAt: c.Present.At * 1e6}
	for i, b := range c.Present.Candles {
		e.Samples = append(e.Samples, Sample{Lag: uint8(len(c.Present.Candles) - 1 - i), Present: true, Values: [6]float64{(b.O - center) / scale, (b.H - center) / scale, (b.L - center) / scale, (b.C - center) / scale, math.Tanh(math.Log(math.Max(b.V, 1e-12)/math.Max(ref, 1e-12)) / 2), 0}})
	}
	return e
}
func cloneExample(e Example) Example {
	e.Samples = append([]Sample(nil), e.Samples...)
	e.Outcome = nil
	return e
}
func atRef(e Example, r Ref) float64 {
	for _, s := range e.Samples {
		if s.Lag == r.Lag {
			return s.Values[r.Channel]
		}
	}
	panic("missing teacher reference")
}
func setRef(e *Example, r Ref, v float64) {
	for i := range e.Samples {
		if e.Samples[i].Lag == r.Lag {
			e.Samples[i].Values[r.Channel] = v
			return
		}
	}
	panic("missing teacher reference")
}
func teacherMargin(q Question, e Example) float64 {
	a, b := atRef(e, q.A), atRef(e, q.B)
	switch q.Relation {
	case Sign:
		return a
	case Greater:
		return a - b
	case SameSign:
		return a * b
	case SumGreater:
		return a + b - atRef(e, q.C) - atRef(e, q.D)
	case SameChange:
		return (a - b) * (atRef(e, q.C) - atRef(e, q.D))
	}
	panic("unknown teacher question")
}
func askChart(q Question, e Example) Packet {
	return Packet{Version: ExtendedVersion, Kind: Ask, ID: e.Episode, Source: 16, Stream: 1, At: e.ObservedAt, Question: q, Examples: []Example{e}}
}

func saveCandleLesson(t *testing.T, p Packet, name string) {
	t.Helper()
	if out := os.Getenv("HORIZON_HNL_RESULTS"); out != "" {
		if err := os.MkdirAll(out, 0700); err != nil {
			t.Fatal(err)
		}
		wire, err := Marshal(p)
		if err != nil {
			t.Fatal(err)
		}
		if err = os.WriteFile(filepath.Join(out, name), wire, 0600); err != nil {
			t.Fatal(err)
		}
	}
}
func coefficients(q Question) map[Ref]float64 {
	w := map[Ref]float64{}
	w[q.A]++
	if q.Relation == Greater {
		w[q.B]--
	} else {
		w[q.B]++
		w[q.C]--
		w[q.D]--
	}
	return w
}
func validChart(e Example) bool {
	for _, s := range e.Samples {
		v := s.Values
		for _, x := range v {
			if !finite(x) || math.Abs(x) > 1 {
				return false
			}
		}
		if v[1] < math.Max(v[0], v[3]) || v[2] > math.Min(v[0], v[3]) {
			return false
		}
	}
	return true
}

// Center a hypothetical chart on a question's decision boundary, then vary
// only its referenced measurements in opposite directions. Other predicate
// states are held stable. These are teacher-controlled valid OHLC experiences,
// derived from past charts; they are labelled as hypothetical, not actual ticks.
func contrastPair(q Question, source Example, delta float64, previous []Question) (Example, Example, bool) {
	w := coefficients(q)
	e := cloneExample(source)
	den := 0.
	for _, v := range w {
		den += v * v
	}
	m := teacherMargin(q, e)
	for r, v := range w {
		setRef(&e, r, atRef(e, r)-m*v/den)
	}
	if q.Relation != SumGreater {
		for i := range e.Samples {
			s := &e.Samples[i]
			o, c := s.Values[0], s.Values[3]
			padO := math.Abs(w[Ref{s.Lag, 0}]) * delta * 1.4
			padC := math.Abs(w[Ref{s.Lag, 3}]) * delta * 1.4
			s.Values[1] = math.Max(s.Values[1], math.Max(o+padO, c+padC)+delta)
			s.Values[2] = math.Min(s.Values[2], math.Min(o-padO, c-padC)-delta)
		}
	}
	a, b := cloneExample(e), cloneExample(e)
	for r, v := range w {
		setRef(&a, r, atRef(a, r)+delta*v)
		setRef(&b, r, atRef(b, r)-delta*v)
	}
	if !validChart(a) || !validChart(b) || teacherMargin(q, a) <= 0 || teacherMargin(q, b) >= 0 {
		return a, b, false
	}
	for _, old := range previous {
		x, y := teacherMargin(old, a), teacherMargin(old, b)
		if x != y && (sign(x) != sign(y) || math.Min(math.Abs(x), math.Abs(y)) < 16*math.Abs(x-y)) {
			return a, b, false
		}
	}
	return a, b, true
}

func chartLesson(t *testing.T, task chartTask, id uint64, delta float64, pools [2][]Example, previous []Question) (Packet, []uint64) {
	t.Helper()
	p := Packet{Version: ExtendedVersion, Kind: Teach, ID: 100 + id, Source: 16, Stream: 2, At: base.Add(2*time.Hour + time.Duration(id)*time.Second).UnixNano(), Question: task.Q}
	origins := []uint64{}
	for set, pool := range pools {
		n := 0
		for _, e := range pool {
			d := delta * (1 + float64(n%3)*.2)
			a, b, ok := contrastPair(task.Q, e, d, previous)
			if !ok {
				continue
			}
			origins = append(origins, e.Episode)
			for j, v := range []Example{a, b} {
				v.Role = Role(set + 1)
				v.Episode = (100+id)*100000 + uint64(set*1000+n*2+j+1)
				v.ObservedAt = p.At - 1000000
				v.Outcome = &Evidence{Value: sign(teacherMargin(task.Q, v)), ObservedAt: p.At - 1}
				p.Examples = append(p.Examples, v)
			}
			n++
			if n == 8 {
				break
			}
		}
		if n != 8 {
			t.Fatalf("not enough independent prototype charts for %s delta %.12g set%d: %d", task.Name, delta, set, n)
		}
	}
	return p, origins
}

func alignmentLesson(t *testing.T, task chartTask, id uint64, pools [2][]Example) (Packet, []uint64) {
	t.Helper()
	p := Packet{Version: ExtendedVersion, Kind: Teach, ID: 100 + id, Source: 16, Stream: 2, At: base.Add(2*time.Hour + time.Duration(id)*time.Second).UnixNano(), Question: task.Q}
	origins := []uint64{}
	for set, pool := range pools {
		n := 0
		for _, source := range pool {
			e := cloneExample(source)
			m := atRef(e, Ref{3, 3})
			eps := .001 * (1 + float64(n%3)*.2)
			if math.Abs(m-atRef(e, Ref{0, 0})) < 64*eps {
				continue
			}
			for _, lag := range []uint8{0, 6} {
				o := atRef(e, Ref{lag, 0})
				setRef(&e, Ref{lag, 1}, math.Max(o, m)+.08)
				setRef(&e, Ref{lag, 2}, math.Min(o, m)-.03)
			}
			if !validChart(e) {
				continue
			}
			variants := []Example{}
			for _, recent := range []float64{-1, 1} {
				for _, prior := range []float64{-1, 1} {
					v := cloneExample(e)
					setRef(&v, Ref{0, 3}, m+recent*eps)
					setRef(&v, Ref{6, 3}, m-prior*eps)
					v.Role = Role(set + 1)
					v.Episode = (100+id)*100000 + uint64(set*1000+n*4+len(variants)+1)
					v.ObservedAt = p.At - 1000000
					v.Outcome = &Evidence{Value: sign(recent * prior), ObservedAt: p.At - 1}
					if !validChart(v) {
						variants = nil
						break
					}
					variants = append(variants, v)
				}
			}
			if len(variants) != 4 {
				continue
			}
			p.Examples = append(p.Examples, variants...)
			origins = append(origins, e.Episode)
			n++
			if n == 8 {
				break
			}
		}
		if n != 8 {
			t.Fatalf("not enough valid alignment prototypes set %d", set)
		}
	}
	return p, origins
}

func TestHNLCandleCurriculumFromPreviousBrain(t *testing.T) {
	dataset, previous := os.Getenv("HORIZON_HNL_CURRICULUM"), os.Getenv("HORIZON_HNL_PREVIOUS")
	if dataset == "" || previous == "" {
		t.Skip("explicit sealed historical data and previous HNL checkpoint required")
	}
	raw, e := os.ReadFile(dataset)
	if e != nil {
		t.Fatal(e)
	}
	var data struct {
		Groups []struct {
			Split string      `json:"split"`
			Cases []chartCase `json:"cases"`
		} `json:"groups"`
	}
	if e = json.Unmarshal(raw, &data); e != nil {
		t.Fatal(e)
	}
	sets := map[string][]chartCase{}
	for _, g := range data.Groups {
		sets[g.Split] = append(sets[g.Split], g.Cases...)
	}
	if len(sets["train"]) != 1462 || len(sets["test2025"]) != 2187 || len(sets["test2026"]) != 546 {
		t.Fatal("sealed dataset population changed")
	}
	for key := range sets {
		sort.Slice(sets[key], func(i, j int) bool {
			a, b := sets[key][i], sets[key][j]
			if a.Present.At == b.Present.At {
				return a.Present.Asset < b.Present.Asset
			}
			return a.Present.At < b.Present.At
		})
	}
	dir := t.TempDir()
	for _, n := range []string{"final.json", "events.jsonl"} {
		b, e := os.ReadFile(filepath.Join(previous, n))
		if e != nil {
			t.Fatal(e)
		}
		if e = os.WriteFile(filepath.Join(dir, n), b, 0600); e != nil {
			t.Fatal(e)
		}
	}
	log, e := brainruntime.OpenEventLog(filepath.Join(dir, "events.jsonl"))
	if e != nil {
		t.Fatal(e)
	}
	defer log.Close()
	r, e := brainruntime.RebuildRuntimeFromCheckpointAndJournal(log, filepath.Join(dir, "final.json"))
	if e != nil {
		t.Fatal(e)
	}
	r.SetEventLog(log)
	r.SetClock(brainruntime.NewFixedClock(base.Add(48 * time.Hour)))
	b := Bridge{Runtime: r, Clock: brainruntime.NewFixedClock(base.Add(48 * time.Hour))}
	start := filepath.Join(dir, "start.json")
	if e = r.Checkpoint(start); e != nil {
		t.Fatal(e)
	}
	retention := func() int {
		correct := 0
		for _, v := range grid([]float64{-.43, -.21, .17, .61}, []float64{-.49, .37}) {
			for i, q := range []Question{qX, qY, qRelation} {
				out := exchange(t, b, askFixture(q, v.x, v.y)).Reply
				truth := []float64{sign(v.x), sign(v.y), sign(v.x * v.y)}[i]
				if out.Status == 3 && sign(out.Value) == truth {
					correct++
				}
			}
		}
		return correct
	}
	if retention() != 24 {
		t.Fatal("previous learned Brain was not restored")
	}
	pools := [2][]Example{}
	allTrain := []Example{}
	for i, c := range sets["train"] {
		v := chartExample(c, uint64(i+1))
		if !validChart(v) {
			t.Fatal("invalid historical OHLC normalization")
		}
		allTrain = append(allTrain, v)
		date := time.UnixMilli(c.Present.At).UTC()
		if date.Before(time.Date(2024, 6, 21, 0, 0, 0, 0, time.UTC)) {
			pools[0] = append(pools[0], v)
		}
		if !date.Before(time.Date(2024, 7, 10, 0, 0, 0, 0, time.UTC)) {
			pools[1] = append(pools[1], v)
		}
	}
	tasks := chartTasks()
	old := []Question{qX, qY, qRelation}
	receipts := []map[string]any{}
	step := uint64(0)
	minGaps := map[string]float64{}
	startLearn := time.Now()
	for i, task := range tasks {
		if task.Q.Relation == SameChange {
			step++
			p, origins := alignmentLesson(t, task, step, pools)
			out := exchange(t, b, p).Reply
			saveCandleLesson(t, p, fmt.Sprintf("lesson-%02d-%s.hnl", step, task.Name))
			receipts = append(receipts, map[string]any{"task": task.Name, "phase": "composition", "origins": origins, "reply": out})
			if !out.Committed {
				t.Logf("alignment correction not committed: %+v", out)
			}
		} else {
			gap := 1.
			for _, v := range allTrain {
				m := math.Abs(teacherMargin(task.Q, v))
				if m > 1e-14 && m < gap {
					gap = m
				}
			}
			minGaps[task.Name] = gap
			for phase, delta := range []float64{.001, math.Max(1e-8, math.Min(.0001, gap/16))} {
				step++
				p, origins := chartLesson(t, task, step, delta, pools, old)
				out := exchange(t, b, p).Reply
				saveCandleLesson(t, p, fmt.Sprintf("lesson-%02d-%s.hnl", step, task.Name))
				receipts = append(receipts, map[string]any{"task": task.Name, "phase": phase, "contrast_delta": delta, "origins": origins, "reply": out})
				if !out.Committed {
					t.Logf("%s phase%d not committed cause=%d error %.8g", task.Name, phase, out.Cause, out.AfterError)
				}
			}
		}
		old = append(old, task.Q)
		if retention() != 24 {
			t.Fatalf("earlier knowledge lost after stage %s", task.Name)
		}
		t.Logf("stage %d/%d %s retained 24/24", i+1, len(tasks), task.Name)
	}
	learnSeconds := time.Since(startLearn).Seconds()
	final := filepath.Join(dir, "trained.json")
	if e = r.Checkpoint(final); e != nil {
		t.Fatal(e)
	}
	before := filepath.Join(dir, "readonly-before.json")
	if e = r.Checkpoint(before); e != nil {
		t.Fatal(e)
	}
	results := map[string]any{}
	for _, split := range []string{"test2025", "test2026"} {
		accepted, correct, rawCorrect, majority := make([]int, len(tasks)), make([]int, len(tasks)), make([]int, len(tasks)), make([]int, len(tasks))
		positive := make([]int, len(tasks))
		for _, v := range allTrain {
			for q, task := range tasks {
				if teacherMargin(task.Q, v) > 0 {
					positive[q]++
				}
			}
		}
		records := []map[string]any{}
		latencies := []float64{}
		for j, c := range sets[split] {
			v := chartExample(c, uint64(100000+j))
			values := []float64{}
			status := []uint8{}
			truths := []float64{}
			begin := time.Now()
			for q, task := range tasks {
				out := exchange(t, b, askChart(task.Q, v)).Reply
				truth := sign(teacherMargin(task.Q, v))
				values = append(values, out.Value)
				status = append(status, out.Status)
				truths = append(truths, truth)
				if sign(out.Value) == truth {
					rawCorrect[q]++
				}
				if out.Status == 3 {
					accepted[q]++
					if sign(out.Value) == truth {
						correct[q]++
					}
				}
				if (positive[q]*2 >= len(allTrain)) == (truth > 0) {
					majority[q]++
				}
			}
			latencies = append(latencies, time.Since(begin).Seconds()*1000)
			records = append(records, map[string]any{"id": c.ID, "values": values, "statuses": status, "truth": truths})
		}
		sort.Float64s(latencies)
		results[split] = map[string]any{"cases": len(sets[split]), "accepted": accepted, "correct_accepted": correct, "raw_sign_correct": rawCorrect, "majority_correct": majority, "records": records, "seven_question_p50_ms": latencies[len(latencies)/2], "seven_question_p95_ms": latencies[int(float64(len(latencies))*.95)]}
		t.Logf("%s accepted=%v correct=%v / %d", split, accepted, correct, len(sets[split]))
	}
	// Keep the latest candle exactly identical while changing earlier context.
	contextPairs, contextCorrect := 0, 0
	for j := len(pools[1]) - 1; j >= 0 && contextPairs < 32; j-- {
		a := cloneExample(pools[1][j])
		c := atRef(a, Ref{0, 3})
		eps := .01
		setRef(&a, Ref{3, 3}, c-eps)
		setRef(&a, Ref{6, 3}, c-2*eps)
		for _, lag := range []uint8{3, 6} {
			o := atRef(a, Ref{lag, 0})
			setRef(&a, Ref{lag, 1}, math.Max(o, c+eps)+.02)
			setRef(&a, Ref{lag, 2}, math.Min(o, c-2*eps)-.02)
		}
		z := cloneExample(a)
		setRef(&z, Ref{3, 3}, c+eps)
		if !validChart(a) || !validChart(z) {
			continue
		}
		contextPairs++
		qa, qz := exchange(t, b, askChart(tasks[6].Q, a)).Reply, exchange(t, b, askChart(tasks[6].Q, z)).Reply
		if qa.Status == 3 && qz.Status == 3 && qa.Value > 0 && qz.Value < 0 {
			contextCorrect++
		}
	}
	after := filepath.Join(dir, "readonly-after.json")
	if e = r.Checkpoint(after); e != nil {
		t.Fatal(e)
	}
	ba, _ := os.ReadFile(before)
	bb, _ := os.ReadFile(after)
	if !bytes.Equal(ba, bb) {
		t.Fatal("historical inference changed Brain")
	}
	restored := brainruntime.NewBrainRuntime(knowledge.NewBrain())
	if e = restored.RestoreCheckpoint(final); e != nil {
		t.Fatal(e)
	}
	replayed, e := brainruntime.RebuildRuntimeFromCheckpointAndJournal(log, start)
	if e != nil {
		t.Fatal(e)
	}
	probe := askChart(tasks[6].Q, chartExample(sets["test2026"][0], 900000))
	expected := exchange(t, b, probe)
	for _, runtime := range []*brainruntime.BrainRuntime{restored, replayed} {
		got := exchange(t, Bridge{Runtime: runtime, Clock: b.Clock}, probe)
		x, _ := Marshal(expected)
		y, _ := Marshal(got)
		if !bytes.Equal(x, y) {
			t.Fatal("HNL reply changed after restore/full suffix replay")
		}
	}
	last := exchange(t, b, askFixture(qX, .43, .37)).Reply
	evidence(t, "candle_curriculum_results.json", map[string]any{"started_from_previous_trained_brain": true, "previous_knowledge_retained": retention(), "previous_knowledge_total": 24, "teacher_supplied_grammar_contrasts_labels": true, "historical_sets_previously_examined": true, "actual_market_forecast_test": false, "lessons": step, "learning_seconds": learnSeconds, "tasks": tasks, "minimum_training_gaps": minGaps, "receipts": receipts, "evaluation": results, "context_pairs": contextPairs, "context_both_correct": contextCorrect, "learned_n_units": len(last.Units), "read_only_inference": true, "restore_reply_equal": true, "journal_suffix_replay_reply_equal": true})
	if e = log.Close(); e != nil {
		t.Fatal(e)
	}
	if out := os.Getenv("HORIZON_HNL_RESULTS"); out != "" {
		for _, n := range []string{"start.json", "trained.json", "events.jsonl"} {
			v, e := os.ReadFile(filepath.Join(dir, n))
			if e != nil {
				t.Fatal(e)
			}
			if e = os.WriteFile(filepath.Join(out, n), v, 0600); e != nil {
				t.Fatal(e)
			}
		}
		// These are unlabelled real historical questions for a production-built
		// CLI smoke check; none of this file's teacher helpers enter that binary.
		v := chartExample(sets["test2026"][20], 900020)
		for _, task := range []chartTask{tasks[0], tasks[1], tasks[6]} {
			wire, err := Marshal(askChart(task.Q, v))
			if err != nil {
				t.Fatal(err)
			}
			if err = os.WriteFile(filepath.Join(out, "sample-"+task.Name+".hnl"), wire, 0600); err != nil {
				t.Fatal(err)
			}
		}
	}
	fmt.Printf("HNL candle curriculum finished: %d lessons, %d learned N, context %d/%d\n", step, len(last.Units), contextCorrect, contextPairs)
}
