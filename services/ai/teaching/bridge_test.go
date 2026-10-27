package teaching

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"testing"
	"time"

	"github.com/project-horizon/horizon-core/services/ai/knowledge"
	brainruntime "github.com/project-horizon/horizon-core/services/ai/runtime"
)

type harness struct {
	bridge     Bridge
	brain      *knowledge.Brain
	log        *brainruntime.EventLog
	dir, start string
}

func newHarness(t *testing.T) *harness {
	t.Helper()
	h := &harness{brain: knowledge.NewBrain(), dir: t.TempDir()}
	r := brainruntime.NewBrainRuntime(h.brain)
	r.SetClock(brainruntime.NewFixedClock(base))
	var e error
	h.log, e = brainruntime.OpenEventLog(filepath.Join(h.dir, "events.jsonl"))
	if e != nil {
		t.Fatal(e)
	}
	t.Cleanup(func() { _ = h.log.Close() })
	r.SetEventLog(h.log)
	w, e := r.PlanNeuralWrittenGrounding()
	if e != nil {
		t.Fatal(e)
	}
	if e = r.ApplyNeuralWrittenGrounding(w, filepath.Join(h.dir, "before-written.json"), base); e != nil {
		t.Fatal(e)
	}
	c := knowledge.DefaultCompositionConfig()
	c.Version = knowledge.NeuralCompositionResolutionV5
	c.Initialization = "random"
	p, e := r.PlanNeuralComposition(c)
	if e != nil {
		t.Fatal(e)
	}
	if e = r.ApplyNeuralComposition(p, filepath.Join(h.dir, "before-composition.json"), base); e != nil {
		t.Fatal(e)
	}
	h.start = filepath.Join(h.dir, "start.json")
	if e = r.Checkpoint(h.start); e != nil {
		t.Fatal(e)
	}
	h.bridge = Bridge{Runtime: r, Clock: brainruntime.NewFixedClock(base.Add(48 * time.Hour))}
	return h
}
func exchange(t *testing.T, b Bridge, p Packet) Packet {
	t.Helper()
	raw, e := b.Exchange(mustWire(t, p))
	if e != nil {
		t.Fatal(e)
	}
	reply, e := Unmarshal(raw)
	if e != nil {
		t.Fatal(e)
	}
	if reply.ID != p.ID || reply.Question != p.Question.Canonical() || reply.Reply == nil {
		t.Fatal("reply lost request binding")
	}
	return reply
}
func snapshot(t *testing.T, h *harness) []byte {
	t.Helper()
	p := filepath.Join(h.dir, "probe.json")
	if e := h.bridge.Runtime.Checkpoint(p); e != nil {
		t.Fatal(e)
	}
	b, e := os.ReadFile(p)
	if e != nil {
		t.Fatal(e)
	}
	return b
}

type pair struct{ x, y float64 }

func grid(xs, ys []float64) []pair {
	v := []pair{}
	for _, x := range xs {
		for _, y := range ys {
			v = append(v, pair{x, y})
		}
	}
	return v
}
func sign(x float64) float64 {
	if x > 0 {
		return 1
	}
	return -1
}
func lesson(q Question, id uint64, fit, check []pair, truth func(pair) float64) Packet {
	p := Packet{Version: Version, Kind: Teach, ID: id, Source: 1, Stream: 1, At: base.Add(time.Duration(id) * time.Minute).UnixNano(), Question: q}
	for set, rows := range [][]pair{fit, check} {
		for i, x := range rows {
			e := askFixture(q, x.x, x.y).Examples[0]
			e.Role = Role(set + 1)
			e.Episode = uint64(set*100+i+1) + id*1000
			e.ObservedAt = base.Add(-time.Second).UnixNano()
			e.Outcome = &Evidence{Value: truth(x), ObservedAt: base.UnixNano()}
			p.Examples = append(p.Examples, e)
		}
	}
	return p
}
func evidence(t *testing.T, name string, v any) {
	t.Helper()
	root := os.Getenv("HORIZON_HNL_RESULTS")
	if root == "" {
		return
	}
	if e := os.MkdirAll(root, 0700); e != nil {
		t.Fatal(e)
	}
	b, e := json.MarshalIndent(v, "", "  ")
	if e != nil {
		t.Fatal(e)
	}
	if e = os.WriteFile(filepath.Join(root, name), append(b, '\n'), 0600); e != nil {
		t.Fatal(e)
	}
}

func TestHNLNativeCurriculumOrderAndFineResolution(t *testing.T) {
	staged, reverse := newHarness(t), newHarness(t)
	fit := grid([]float64{-.7, -.5, .5, .7}, []float64{-.8, .8})
	check := grid([]float64{-.9, -.3, .3, .9}, []float64{-.6, .6})
	fine := grid([]float64{-.00015, -.0001, .0001, .00015}, []float64{-.8, .8})
	fineCheck := grid([]float64{-.00012, -.00008, .00008, .00012}, []float64{-.6, .6})
	lessons := []Packet{
		lesson(qX, 1, fit, check, func(p pair) float64 { return sign(p.x) }),
		lesson(qX, 2, fine, fineCheck, func(p pair) float64 { return sign(p.x) }),
		lesson(qY, 3, fit, check, func(p pair) float64 { return sign(p.y) }),
		lesson(qRelation, 4, fit, check, func(p pair) float64 { return sign(p.x * p.y) }),
	}
	initial := exchange(t, staged.bridge, askFixture(qRelation, .2, -.3)).Reply
	if initial.Status == 3 || staged.brain.BrainState.NResolution != nil && len(staged.brain.BrainState.NResolution.Features) != 0 {
		t.Fatal("untrained relation was answered from its opcode")
	}
	receipts := []*Response{}
	for _, p := range lessons {
		receipts = append(receipts, exchange(t, staged.bridge, p).Reply)
	}
	if receipts[1].Cause != 3 || receipts[1].Grown != 0 || receipts[1].LearnedN != 1 || receipts[3].Cause != 6 || receipts[3].LearnedN != 3 {
		t.Fatalf("lost diagnosed fine resolution or learned composition: %+v %+v", receipts[1], receipts[3])
	}
	reverseReceipts := []*Response{}
	for step, index := range []int{3, 0, 1, 2} {
		p := lessons[index]
		p.At = base.Add(time.Duration(step+1) * time.Minute).UnixNano()
		reverseReceipts = append(reverseReceipts, exchange(t, reverse.bridge, p).Reply)
	}
	// The exact same four lesson payloads and supervision are presented once;
	// only their order and delivery timestamps differ.
	held := grid([]float64{-.43, -.21, .17, .61}, []float64{-.49, .37})
	stagedCorrect, reverseCorrect, stagedAccepted, reverseAccepted := 0, 0, 0, 0
	perQuestion := [3]int{}
	records := []map[string]any{}
	for _, x := range held {
		for q, query := range []Question{qX, qY, qRelation} {
			out := exchange(t, staged.bridge, askFixture(query, x.x, x.y)).Reply
			truth := []float64{sign(x.x), sign(x.y), sign(x.x * x.y)}[q]
			if out.Status == 3 && sign(out.Value) == truth {
				perQuestion[q]++
			}
		}
		a := exchange(t, staged.bridge, askFixture(qRelation, x.x, x.y)).Reply
		b := exchange(t, reverse.bridge, askFixture(qRelation, x.x, x.y)).Reply
		if a.Status == 3 {
			stagedAccepted++
			if sign(a.Value) == sign(x.x*x.y) {
				stagedCorrect++
			}
		}
		if b.Status == 3 {
			reverseAccepted++
			if sign(b.Value) == sign(x.x*x.y) {
				reverseCorrect++
			}
		}
		records = append(records, map[string]any{"input": []float64{x.x, x.y}, "staged": a, "reverse": b})
	}
	if stagedCorrect != len(held) || reverseAccepted != 0 || reverseReceipts[0].Committed {
		t.Fatalf("unexpected order comparison: staged %d, reverse %d/%d", stagedCorrect, reverseCorrect, reverseAccepted)
	}
	for q, n := range perQuestion {
		if n != len(held) {
			t.Fatalf("prerequisite/held-out q%d lost: %d", q, n)
		}
	}
	rescue := lessons[3]
	rescue.ID = 5
	rescue.At = base.Add(5 * time.Minute).UnixNano()
	rescued := exchange(t, reverse.bridge, rescue).Reply
	rescuedCorrect := 0
	for _, x := range held {
		out := exchange(t, reverse.bridge, askFixture(qRelation, x.x, x.y)).Reply
		if out.Status == 3 && sign(out.Value) == sign(x.x*x.y) {
			rescuedCorrect++
		}
	}
	if rescuedCorrect != len(held) || !rescued.Committed {
		t.Fatal("revisiting composition after prerequisites failed")
	}
	// Unseen nuisance appearances and reversed commutative syntax reuse the
	// learned relation, while an intervention on a relevant input flips it.
	before := snapshot(t, staged)
	invariant := 0
	for i := 0; i < 128; i++ {
		p := askFixture(qRelation, .31, -.27)
		p.ID = uint64(i + 50)
		p.Source = 99
		p.Stream = 77
		p.Question.A, p.Question.B = p.Question.B, p.Question.A
		p.Examples[0].Samples[0].Values[2] = float64(i-64) / 65
		out := exchange(t, staged.bridge, p).Reply
		if out.Status == 3 && out.Value < 0 {
			invariant++
		}
	}
	if invariant != 128 || !bytes.Equal(before, snapshot(t, staged)) {
		t.Fatal("inference changed Brain or nuisance changed relation")
	}
	finePositive := exchange(t, staged.bridge, askFixture(qX, .0001, .8)).Reply
	fineNegative := exchange(t, staged.bridge, askFixture(qX, -.0001, .8)).Reply
	if finePositive.Status != 3 || fineNegative.Status != 3 || finePositive.Value <= 0 || fineNegative.Value >= 0 || finePositive.Units[0].Code != 8 || fineNegative.Units[0].Code != -8 {
		t.Fatal("binary teaching lost the fine numerical distinction")
	}
	intervened := exchange(t, staged.bridge, askFixture(qRelation, .31, .27)).Reply
	if intervened.Status != 3 || intervened.Value <= 0 {
		t.Fatal("relevant input intervention did not change the relation")
	}
	evidence(t, "fine_resolution_results.json", map[string]any{"positive": finePositive, "negative": fineNegative, "semantic_intervention_flip": true})
	probe := askFixture(qRelation, .31, -.27)
	wire := mustWire(t, probe)
	out := exchange(t, staged.bridge, probe).Reply
	input, e := CompileAsk(probe)
	if e != nil {
		t.Fatal(e)
	}
	native, e := staged.bridge.Runtime.ReadComposition(input, time.Unix(0, probe.At))
	if e != nil {
		t.Fatal(e)
	}
	if native.NResolution.Value != out.Value || len(native.NResolution.Units) != len(out.Units) {
		t.Fatal("bridge answered differently from native Brain")
	}
	final := filepath.Join(staged.dir, "final.json")
	if e = staged.bridge.Runtime.Checkpoint(final); e != nil {
		t.Fatal(e)
	}
	r := brainruntime.NewBrainRuntime(knowledge.NewBrain())
	if e = r.RestoreCheckpoint(final); e != nil {
		t.Fatal(e)
	}
	replay, e := brainruntime.RebuildRuntimeFromCheckpointAndJournal(staged.log, staged.start)
	if e != nil {
		t.Fatal(e)
	}
	for _, runtime := range []*brainruntime.BrainRuntime{r, replay} {
		got := exchange(t, Bridge{Runtime: runtime, Clock: staged.bridge.Clock}, probe).Reply
		if !reflect.DeepEqual(got, out) {
			t.Fatal("numeric reply changed after checkpoint/full native replay")
		}
	}
	if e = staged.log.Close(); e != nil {
		t.Fatal(e)
	}
	root := os.Getenv("HORIZON_HNL_RESULTS")
	if root != "" {
		for _, name := range []string{"start.json", "final.json", "events.jsonl"} {
			b, e := os.ReadFile(filepath.Join(staged.dir, name))
			if e != nil {
				t.Fatal(e)
			}
			if e = os.WriteFile(filepath.Join(root, name), b, 0600); e != nil {
				t.Fatal(e)
			}
		}
		for name, p := range map[string]Packet{"ask.hnl": probe, "lesson-sign.hnl": lessons[0], "lesson-fine.hnl": lessons[1], "lesson-second.hnl": lessons[2], "lesson-relation.hnl": lessons[3]} {
			if e = os.WriteFile(filepath.Join(root, name), mustWire(t, p), 0600); e != nil {
				t.Fatal(e)
			}
		}
	}
	// Timing is codec + native read in this shared process, no WAL write.
	latencies := []float64{}
	direct := []float64{}
	for i := 0; i < 200; i++ {
		begin := time.Now()
		if _, e = staged.bridge.Exchange(wire); e != nil {
			t.Fatal(e)
		}
		latencies = append(latencies, time.Since(begin).Seconds()*1000)
		begin = time.Now()
		if _, e = staged.bridge.Runtime.ReadComposition(input, time.Unix(0, probe.At)); e != nil {
			t.Fatal(e)
		}
		direct = append(direct, time.Since(begin).Seconds()*1000)
	}
	sort.Float64s(latencies)
	sort.Float64s(direct)
	evidence(t, "curriculum_results.json", map[string]any{"synthetic_only": true, "same_four_lessons_different_order": true, "staged_order": []int{0, 1, 2, 3}, "reverse_order": []int{3, 0, 1, 2}, "staged_receipts": receipts, "reverse_receipts": reverseReceipts, "held_out_per_question_correct": perQuestion, "held_out_per_question_total": len(held), "composition_staged_correct": stagedCorrect, "composition_staged_accepted": stagedAccepted, "composition_reverse_correct": reverseCorrect, "composition_reverse_accepted": reverseAccepted, "rescue_correct": rescuedCorrect, "rescue_receipt": rescued, "nuisance_invariance": invariant, "nuisance_total": 128, "records": records, "learned_n_units": len(staged.brain.BrainState.NResolution.Features), "total_neural_units": len(staged.brain.Registry.Nodes()), "reply_native_exact": true, "checkpoint_reply_equal": true, "full_journal_replay_reply_equal": true, "read_only_inference": true, "ask_wire_bytes": len(wire), "ask_codec_native_p50_ms": latencies[100], "ask_codec_native_p95_ms": latencies[190], "native_read_p50_ms": direct[100], "native_read_p95_ms": direct[190], "timing_iterations": 200})
}

func TestHNLLabelIsolationAndFailBeforeMutation(t *testing.T) {
	h := newHarness(t)
	fit := grid([]float64{-.7, -.5, .5, .7}, []float64{-.8, .8})
	check := grid([]float64{-.9, -.3, .3, .9}, []float64{-.6, .6})
	p := lesson(qX, 1, fit, check, func(p pair) float64 { return sign(p.x) })
	a, e := CompileLesson(p)
	if e != nil {
		t.Fatal(e)
	}
	for i := range p.Examples {
		p.Examples[i].Outcome.Value *= -1
	}
	b, e := CompileLesson(p)
	if e != nil {
		t.Fatal(e)
	}
	if !reflect.DeepEqual(a.Input, b.Input) || reflect.DeepEqual(a.Resolution.Fit[0].Outcome, b.Resolution.Fit[0].Outcome) {
		t.Fatal("target leaked into input or was not transmitted as feedback")
	}
	before := snapshot(t, h)
	mutations := []func(*Packet){func(p *Packet) { p.Examples[len(fit)].Episode = p.Examples[0].Episode }, func(p *Packet) { p.Examples[0].Outcome.ObservedAt = p.At + 1 }, func(p *Packet) { p.Examples[0].Samples[0].Values[1] = 2 }}
	for i, change := range mutations {
		q := lesson(qX, 1, fit, check, func(p pair) float64 { return sign(p.x) })
		change(&q)
		if _, e := CompileLesson(q); e == nil {
			t.Fatalf("accepted malformed lesson %d", i)
		}
	}
	future := askFixture(qX, .2, .3)
	future.At = base.Add(72 * time.Hour).UnixNano()
	if _, e = h.bridge.Exchange(mustWire(t, future)); e == nil {
		t.Fatal("packet advanced trusted observation clock")
	}
	bad := mustWire(t, askFixture(qX, .2, .3))
	bad[len(bad)-1] ^= 1
	if _, e = h.bridge.Exchange(bad); e == nil {
		t.Fatal("accepted corrupt wire")
	}
	if !bytes.Equal(before, snapshot(t, h)) {
		t.Fatal("invalid request mutated Brain")
	}
	p = lesson(qX, 1, fit, check, func(p pair) float64 { return sign(p.x) })
	exchange(t, h.bridge, p)
	before = snapshot(t, h)
	if _, e = h.bridge.Exchange(mustWire(t, p)); e == nil {
		t.Fatal("duplicate teaching ID was not rejected")
	}
	if !bytes.Equal(before, snapshot(t, h)) {
		t.Fatal("duplicate teaching request mutated Brain")
	}
	evidence(t, "boundary_results.json", map[string]any{"label_not_in_input": true, "invalid_requests_leave_brain_equal": true, "duplicate_lesson_rejected_without_mutation": true, "trusted_clock_enforced": true, "fit_check_overlap_rejected": true})
}

func TestHNLConcurrentReceiptsAreDetached(t *testing.T) {
	h := newHarness(t)
	fit := grid([]float64{-.7, -.5, .5, .7}, []float64{-.8, .8})
	check := grid([]float64{-.9, -.3, .3, .9}, []float64{-.6, .6})
	beforeSequence := h.bridge.Runtime.LastSequence()
	type result struct {
		seq     uint64
		receipt *brainruntime.CompositionLearningReceipt
		err     error
	}
	results := make(chan result, 2)
	for i, q := range []Question{qX, qY} {
		truth := func(p pair) float64 { return sign(p.x) }
		if i == 1 {
			truth = func(p pair) float64 { return sign(p.y) }
		}
		p := lesson(q, uint64(i+1), fit, check, truth)
		e, err := CompileLesson(p)
		if err != nil {
			t.Fatal(err)
		}
		go func(e knowledge.CompositionExposure, at int64) {
			s, r, err := h.bridge.Runtime.LearnCompositionWithDiagnosis(e, time.Unix(0, at))
			results <- result{s, r, err}
		}(e, p.At)
	}
	all := []result{<-results, <-results}
	for _, x := range all {
		if x.err != nil {
			t.Fatal(x.err)
		}
		if x.receipt == nil || uint64(x.receipt.LearnedNUnits) != x.seq-beforeSequence || !x.receipt.Diagnosis.Committed {
			t.Fatal("teaching receipt did not describe its atomic transaction")
		}
	}
	before := snapshot(t, h)
	for _, x := range all {
		x.receipt.Diagnosis.Query[0] = 0
		if len(x.receipt.Diagnosis.Trials) > 0 {
			x.receipt.Diagnosis.Trials[0].CheckError = 123
		}
	}
	if !bytes.Equal(before, snapshot(t, h)) {
		t.Fatal("detached receipt mutation changed canonical Brain")
	}
}

func TestHNLTemporalComparisonAndInverseReuse(t *testing.T) {
	h := newHarness(t)
	q := Question{Domain: 2, Relation: Greater, A: Ref{0, 0}, B: Ref{2, 0}}
	inverse := q
	inverse.A, inverse.B = inverse.B, inverse.A
	pairs := func(values []float64) []pair {
		rows := []pair{}
		for _, a := range values {
			for _, b := range values {
				if a != b {
					rows = append(rows, pair{a, b})
				}
			}
		}
		return rows
	}
	fit, check := pairs([]float64{-.75, -.25, .25, .75}), pairs([]float64{-.9, -.3, .3, .9})
	temporal := func(p *Packet) {
		for i := range p.Examples {
			x := p.Examples[i].Samples[0].Values
			p.Examples[i].Samples = []Sample{{Lag: 2, Present: true, Values: [6]float64{x[1]}}, {Lag: 0, Present: true, Values: [6]float64{x[0]}}}
		}
	}
	p := lesson(q, 1, fit, check, func(p pair) float64 { return sign(p.x - p.y) })
	temporal(&p)
	first := exchange(t, h.bridge, p).Reply
	if !first.Committed || first.LearnedN != 1 {
		t.Fatal("temporal comparison was not learned")
	}
	probe := askFixture(inverse, .58, .34)
	temporal(&probe)
	if exchange(t, h.bridge, probe).Reply.Status == 3 {
		t.Fatal("untrained reversed question was answered by the compiler")
	}
	p = lesson(inverse, 2, fit, check, func(p pair) float64 { return sign(p.y - p.x) })
	temporal(&p)
	reused := exchange(t, h.bridge, p).Reply
	if !reused.Committed || reused.Cause != 2 || reused.Grown != 0 || reused.LearnedN != 1 {
		t.Fatal("inverse comparison did not reuse the learned information")
	}
	held := grid([]float64{-.82, -.42, .18, .58}, []float64{-.66, -.26, .34, .74})
	correct := [2]int{}
	for _, v := range held {
		for i, query := range []Question{q, inverse} {
			p := askFixture(query, v.x, v.y)
			temporal(&p)
			out := exchange(t, h.bridge, p).Reply
			truth := sign(v.x - v.y)
			if i == 1 {
				truth = -truth
			}
			if out.Status == 3 && sign(out.Value) == truth {
				correct[i]++
			}
		}
	}
	if correct != [2]int{len(held), len(held)} {
		t.Fatalf("temporal comparison generalization: %v/%d", correct, len(held))
	}
	evidence(t, "temporal_comparison_results.json", map[string]any{"synthetic_observed_state_only": true, "correct": correct, "per_query_total": len(held), "first_receipt": first, "inverse_receipt": reused, "learned_n_units": len(h.brain.BrainState.NResolution.Features), "zero_untrained_opcode_execution": true})
}

func ExampleQuestionCode() {
	v, _ := QuestionCode(Question{Domain: 1, Relation: SameSign, A: Ref{Lag: 0, Channel: 0}, B: Ref{Lag: 0, Channel: 1}})
	fmt.Println(len(v), v[2], v[6])
	// Output: 16 0.375 0.125
}
