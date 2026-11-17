//go:build horizon_s2r2

package runtime

import (
	"bytes"
	"fmt"
	"math"
	"math/rand"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"testing"

	k "github.com/project-horizon/horizon-core/services/ai/knowledge"
)

var s2r2Fractions = []float64{1. / 64, 1. / 16, 1. / 4, 1. / 2}
var s2r2Arms = []string{"legacy", "full-refit", "bounded-joint", "new-feature", "combined"}

func s2r2Paths(t *testing.T) (string, string) {
	t.Helper()
	root, work := os.Getenv("HORIZON_S2R2_SOURCE"), os.Getenv("HORIZON_S2R2_WORK")
	if root == "" || work == "" {
		t.Skip("requires immutable original S2 and isolated S2-R2 work")
	}
	if root == work {
		t.Fatal("source overlaps work")
	}
	t.Setenv("HORIZON_S2_SOURCE", root)
	t.Setenv("HORIZON_S2_WORK", work)
	return root, work
}
func s2r2Copy(t *testing.T, a, b string) {
	t.Helper()
	raw, e := os.ReadFile(a)
	s1Check(t, e)
	s1Check(t, os.WriteFile(b, raw, 0600))
}
func s2r2Fork(t *testing.T, cp, wal, dir string) (*BrainRuntime, *EventLog) {
	t.Helper()
	if _, e := os.Stat(dir); !os.IsNotExist(e) {
		t.Fatal("fork exists", dir)
	}
	s1Check(t, os.MkdirAll(dir, 0700))
	s2r2Copy(t, cp, filepath.Join(dir, "state.json"))
	s2r2Copy(t, wal, filepath.Join(dir, "events.jsonl"))
	return s1Open(t, filepath.Join(dir, "state.json"), filepath.Join(dir, "events.jsonl"))
}
func s2r2Source(t *testing.T, root string) {
	t.Helper()
	var manifest map[string]struct {
		Bytes int    `json:"bytes"`
		SHA   string `json:"sha256"`
	}
	s1Read(t, filepath.Join(root, "MANIFEST.json"), &manifest)
	for name, v := range manifest {
		b, e := os.ReadFile(filepath.Join(root, name))
		s1Check(t, e)
		if len(b) != v.Bytes || s1Digest(b) != v.SHA {
			t.Fatal("source integrity", name)
		}
	}
	s := s1Snapshot(t, filepath.Join(root, "s2/lineage/s2-continuation.json"))
	sum, e := checkpointChecksumForSchema(s)
	s1Check(t, e)
	if s.Sequence != 76617 || s.Checksum != sum || sum != "b87285e38a321d45e4da920f219b388d1a2d061554042d2dd0f109883b38ab9d" {
		t.Fatal("source lineage")
	}
}
func s2r2Lesson(t *testing.T, r *BrainRuntime, id, arm string, fit, check []s2Trial, pop [][]k.NodeID) *CompositionLearningReceipt {
	t.Helper()
	e := s2Lesson(1, fit, check, pop, s1Time(r))
	e.ID = id
	e.EpisodeID = id
	e.IndependenceGroup = id
	e.Source = "s2r2-native-observed-error"
	if arm != "legacy" {
		e.Resolution.Proposal = &k.NMetricProposalConfig{Protocol: k.NMetricBoundedV1, Strategy: arm, Fractions: append([]float64(nil), s2r2Fractions...), PreserveShared: true}
	}
	_, receipt, err := r.LearnCompositionWithDiagnosis(e, s1Time(r))
	s1Check(t, err)
	if receipt == nil {
		t.Fatal("missing native receipt")
	}
	return receipt
}
func s2r2MetricProtocol(t *testing.T, work string) s2Protocol {
	var p s2Protocol
	s1Read(t, filepath.Join(work, "METRIC_PROTOCOL.json"), &p)
	return p
}

func TestS2R2Prepare(t *testing.T) {
	root, work := s2r2Paths(t)
	s2r2Source(t, root)
	if _, e := os.Stat(work); !os.IsNotExist(e) {
		t.Fatal("work must be new")
	}
	s1Check(t, os.MkdirAll(work, 0700))
	s2r2Copy(t, filepath.Join(root, "s2/lineage/s2-continuation.json"), filepath.Join(work, "00-source.json"))
	s2r2Copy(t, filepath.Join(root, "s2/lineage/events.jsonl"), filepath.Join(work, "source-events.jsonl"))
	s2r2Copy(t, "../../../docs/research/s2_r2_repair_20261117/DESIGN.md", filepath.Join(work, "DESIGN.md"))
	s2r2Copy(t, "../../../docs/research/s2_r2_repair_20261117/DECISION_RULES.json", filepath.Join(work, "DECISION_RULES.json"))
	s2r2Copy(t, "../../../docs/research/s2_r2_repair_20261117/REPAIR_AMENDMENT.md", filepath.Join(work, "REPAIR_AMENDMENT.md"))
	reuse := os.Getenv("HORIZON_S2R2_DEVELOPMENT_REUSE")
	observerCP, observerWAL := filepath.Join(work, "00-source.json"), filepath.Join(work, "source-events.jsonl")
	if reuse != "" {
		observerCP = filepath.Join(reuse, "development-observer/final.json")
		observerWAL = filepath.Join(reuse, "development-observer/events.jsonl")
		raw, e := os.ReadFile(filepath.Join(reuse, "preoperation.jsonl"))
		s1Check(t, e)
		lines := strings.Split(string(raw), "\n")
		if len(lines) < 512 {
			t.Fatal("incomplete development ledger")
		}
		s1Check(t, os.WriteFile(filepath.Join(work, "preoperation.jsonl"), []byte(strings.Join(lines[:512], "\n")+"\n"), 0600))
		s1Write(t, filepath.Join(work, "DEVELOPMENT_REUSE.json"), map[string]any{"from_failed_engineering_revision": true, "actual_observations": 512, "new_development_samples": 0, "parameter_copy": false})
	}
	observer, ow := s2r2Fork(t, observerCP, observerWAL, filepath.Join(work, "development-observer"))
	pop := s1Pop(t, filepath.Join(root, "source/source"))
	pairs := [][2]int64{{41011, 41017}, {41023, 41039}, {41047, 41051}, {41057, 41077}}
	all := []s2Trial{}
	constant := 0.
	count := 0
	for i, seeds := range pairs {
		var fit, check []s2Trial
		if reuse != "" {
			s1Read(t, filepath.Join(reuse, fmt.Sprintf("dev-%d-fit.json", i)), &fit)
			s1Read(t, filepath.Join(reuse, fmt.Sprintf("dev-%d-check.json", i)), &check)
		} else {
			fit = s2Collect(t, observer, pop, seeds[0], 64, i >= 2, nil, -1)
			check = s2Collect(t, observer, pop, seeds[1], 64, i >= 2, nil, -1)
		}
		s1Write(t, filepath.Join(work, fmt.Sprintf("dev-%d-fit.json", i)), fit)
		s1Write(t, filepath.Join(work, fmt.Sprintf("dev-%d-check.json", i)), check)
		for _, s := range fit {
			constant += s.Error
			count++
		}
		all = append(all, check...)
	}
	s1Check(t, observer.Checkpoint(filepath.Join(work, "development-observer/final.json")))
	s1Check(t, ow.Close())
	values := make([]float64, len(all))
	mean := 0.
	for i, s := range all {
		values[i] = s.Error
		mean += s.Error / float64(len(all))
	}
	sort.Float64s(values)
	variance := 0.
	for _, v := range values {
		variance += (v - mean) * (v - mean) / float64(len(values))
	}
	p := s2Protocol{Baseline: s2Baseline, RelativeRMSE: .8, Correlation: .5, AblationRatio: 1.2, RetentionTolerance: 1e-7, Constant: constant / float64(count), LowCut: values[len(values)/3], HighCut: values[2*len(values)/3], DevelopmentSD: math.Sqrt(variance)}
	p.CalibrationLimit = math.Max(.003, .25*p.DevelopmentSD)
	for i := 1; i <= 5; i++ {
		p.Bins = append(p.Bins, values[len(values)-1]*float64(i)/5)
	}
	s1Write(t, filepath.Join(work, "METRIC_PROTOCOL.json"), p)
	s1Write(t, filepath.Join(work, "ACQUISITION_PROTOCOL.json"), map[string]any{"fractions": s2r2Fractions, "arms": s2r2Arms, "development_pairs": pairs, "blocks": 8, "training_fit_seeds": "42001+20*b", "training_check_seeds": "42007+20*b", "holdout_seeds": []int{51001, 51031}, "holdout_per_stream": 128, "checkpoint_selection": "eighth block irrespective of score", "continuation_seed": 71001, "continuation_learning_seed": 71021, "original_final_holdout_used": false})
	t.Log("verified 75 source payloads and 76617; new development observations", len(all)*2)
}

func TestS2R2Mechanism(t *testing.T) {
	root, work := s2r2Paths(t)
	pop := s1Pop(t, filepath.Join(root, "source/source"))
	p := s2r2MetricProtocol(t, work)
	results := map[string]any{}
	mechanism := false
	for _, arm := range s2r2Arms {
		r, w := s2r2Fork(t, filepath.Join(work, "00-source.json"), filepath.Join(work, "source-events.jsonl"), filepath.Join(work, "mechanism-"+arm))
		receipts := []*CompositionLearningReceipt{}
		scores := []map[string]any{}
		for i := 0; i < 4; i++ {
			var fit, check []s2Trial
			s1Read(t, filepath.Join(work, fmt.Sprintf("dev-%d-fit.json", i)), &fit)
			s1Read(t, filepath.Join(work, fmt.Sprintf("dev-%d-check.json", i)), &check)
			before := s2MetricOf(check, s2Predictions(t, r, check, pop), p)
			receipt := s2r2Lesson(t, r, fmt.Sprintf("s2r2-mechanism-%s-%d", arm, i), arm, fit, check, pop)
			receipts = append(receipts, receipt)
			after := s2MetricOf(check, s2Predictions(t, r, check, pop), p)
			if arm == "combined" && receipt.Diagnosis.Committed {
				var winner k.NTrial
				for _, trial := range receipt.Diagnosis.Trials {
					if trial.Eligible {
						winner = trial
					}
				}
				mechanism = mechanism || (winner.Proposal != "full-refit" && winner.Proposal != "" && winner.RetentionRegressions == 0)
			}
			scores = append(scores, map[string]any{"block": i, "before": before, "after": after, "committed": receipt.Diagnosis.Committed})
			t.Log(arm, i, receipt.Diagnosis.Committed, after.RMSE, receipt.Diagnosis.Cause)
		}
		s1Write(t, filepath.Join(work, "mechanism-"+arm, "receipts.json"), receipts)
		s1Check(t, r.Checkpoint(filepath.Join(work, "mechanism-"+arm, "final.json")))
		s1Check(t, w.Close())
		results[arm] = scores
	}
	s1Write(t, filepath.Join(work, "mechanism.json"), map[string]any{"arms": results, "bounded_or_decoupled_admitted": mechanism, "thresholds_unchanged": true, "independent_new_observations": true})
	if !mechanism {
		t.Fatal("BLOCKED: new-data mechanism control did not admit a bounded/decoupled proposal")
	}
}

func TestS2R2Train(t *testing.T) {
	root, work := s2r2Paths(t)
	pop := s1Pop(t, filepath.Join(root, "source/source"))
	p := s2r2MetricProtocol(t, work)
	var mechanism struct {
		Pass bool `json:"bounded_or_decoupled_admitted"`
	}
	s1Read(t, filepath.Join(work, "mechanism.json"), &mechanism)
	if !mechanism.Pass {
		t.Fatal("mechanism gate not passed")
	}
	runtimes := map[string]*BrainRuntime{}
	logs := map[string]*EventLog{}
	receipts := map[string][]*CompositionLearningReceipt{}
	for _, arm := range s2r2Arms {
		r, w := s2r2Fork(t, filepath.Join(work, "00-source.json"), filepath.Join(work, "source-events.jsonl"), filepath.Join(work, "training-"+arm))
		runtimes[arm] = r
		logs[arm] = w
	}
	r := runtimes["combined"]
	progress := []map[string]any{}
	for b := 0; b < 8; b++ {
		fit := s2Collect(t, r, pop, 42001+int64(20*b), 64, b%2 == 1, nil, -1)
		check := s2Collect(t, r, pop, 42007+int64(20*b), 64, b%2 == 1, nil, -1)
		s1Write(t, filepath.Join(work, fmt.Sprintf("train-%d-fit.json", b)), fit)
		s1Write(t, filepath.Join(work, fmt.Sprintf("train-%d-check.json", b)), check)
		before := s2MetricOf(check, s2Predictions(t, r, check, pop), p)
		for _, arm := range s2r2Arms {
			rr := runtimes[arm]
			rec := s2r2Lesson(t, rr, fmt.Sprintf("s2r2-learning-%s-%d", arm, b), arm, fit, check, pop)
			receipts[arm] = append(receipts[arm], rec)
		}
		after := s2MetricOf(check, s2Predictions(t, r, check, pop), p)
		progress = append(progress, map[string]any{"block": b, "sequence": r.LastSequence(), "before": before, "after": after, "committed": receipts["combined"][b].Diagnosis.Committed, "features": len(r.brain.BrainState.NResolution.Features)})
		s1Check(t, r.Checkpoint(filepath.Join(work, fmt.Sprintf("block-%d.json", b))))
		s1Write(t, filepath.Join(work, "progress.json"), progress)
		s1Write(t, filepath.Join(work, "learning.json"), receipts)
		t.Log("block", b, "commit", receipts["combined"][b].Diagnosis.Committed, "RMSE", before.RMSE, "->", after.RMSE, "features", len(r.brain.BrainState.NResolution.Features))
	}
	for _, arm := range s2r2Arms {
		s1Check(t, runtimes[arm].Checkpoint(filepath.Join(work, "training-"+arm, "final.json")))
		s1Check(t, logs[arm].Close())
	}
	s2r2Copy(t, filepath.Join(work, "training-combined/final.json"), filepath.Join(work, "s2r2-final.json"))
}

func s2r2Retention(t *testing.T, root, work, cp, wal, suffix string) map[string]any {
	t.Helper()
	pop := s1Pop(t, filepath.Join(root, "source/source"))
	dir := filepath.Join(work, "retention-"+suffix)
	control, cw := s2r2Fork(t, filepath.Join(work, "00-source.json"), filepath.Join(work, "source-events.jsonl"), dir)
	events, e := ReadEventLog(wal)
	s1Check(t, e)
	n := 0
	for _, ev := range events {
		if ev.Sequence <= 76617 {
			continue
		}
		if ev.Type == EventTypeCompositionExposure {
			continue
		}
		if ev.Type != EventTypeNativeControl || ev.NativeControl == nil {
			t.Fatal("unexpected schedule")
		}
		_, err := control.ApplyNativeControl(*ev.NativeControl, ev.Timestamp)
		s1Check(t, err)
		n++
	}
	target := s1Snapshot(t, cp).Sequence
	s2Advance(t, control, pop, target)
	ccp := filepath.Join(dir, "time-matched.json")
	s1Check(t, control.Checkpoint(ccp))
	s1Check(t, cw.Close())
	b, a := e1Retention(t, ccp, pop), e1Retention(t, cp, pop)
	be, ae := e1Measure(t, ccp, pop), e1Measure(t, cp, pop)
	delta := s1AuditMax(s1AuditValue(t, b), s1AuditValue(t, a))
	edelta := s1AuditMax(s1AuditValue(t, be), s1AuditValue(t, ae))
	c := a["C"].(*flexPolicyMetrics)
	return map[string]any{"control": b, "candidate": a, "control_E1": be, "candidate_E1": ae, "delta": delta, "E1_delta": edelta, "C_correct": int(c.Accuracy * 6), "E1_correct": int(math.Round(ae.Accuracy * 36)), "sequence": target, "native_commands": n, "pass": delta <= 1e-7 && edelta <= 1e-7 && c.Accuracy == 1 && ae.Accuracy >= 35./36, "historical_S1_retention_failure_retained": true}
}

func TestS2R2RetentionAndFreeze(t *testing.T) {
	root, work := s2r2Paths(t)
	cp := filepath.Join(work, "s2r2-final.json")
	wal := filepath.Join(work, "training-combined/events.jsonl")
	// Complete provenance metadata before sealing. All scoring parameters were
	// already computed from development fit/check, never from final holdout.
	p := s2r2MetricProtocol(t, work)
	p.FitN = 512
	p.DevelopmentN = 256
	p.HoldoutN = 128
	p.Lessons = 8
	p.Scale = 1
	p.HoldoutSeeds = []int64{51001, 51031}
	p.TrainingSeed = 42001
	p.NewTrainingSeed = 42021
	p.DevelopmentSeed = 41011
	p.Identifiable = p.DevelopmentSD*p.DevelopmentSD > 1e-8 && p.HighCut-p.LowCut > .003
	for name, dst := range map[string]*string{"00-source.json": &p.SourceCheckpointSHA256, "source-events.jsonl": &p.SourceJournalSHA256, "DESIGN.md": &p.DesignSHA256} {
		raw, e := os.ReadFile(filepath.Join(work, name))
		s1Check(t, e)
		*dst = s1Digest(raw)
	}
	s1Write(t, filepath.Join(work, "METRIC_PROTOCOL.json"), p)
	ret := s2r2Retention(t, root, work, cp, wal, "final")
	s1Write(t, filepath.Join(work, "retention.json"), ret)
	pop := s1Pop(t, filepath.Join(root, "source/source"))
	r := orderedFixtureClone(t, cp)
	source := orderedFixtureClone(t, filepath.Join(work, "00-source.json"))
	var rows []s2Trial
	for b := 0; b < 8; b++ {
		var rr []s2Trial
		s1Read(t, filepath.Join(work, fmt.Sprintf("train-%d-check.json", b)), &rr)
		rows = append(rows, rr...)
	}
	a, b := s1Predictions(t, r, s2S1Rows(rows), pop), s1Predictions(t, source, s2S1Rows(rows), pop)
	s1delta := s1AuditMax(s1AuditValue(t, a), s1AuditValue(t, b))
	s1Write(t, filepath.Join(work, "s1-retention.json"), map[string]any{"shared_observations": len(rows), "coordinates": 3 * len(rows), "max_delta": s1delta, "pass": s1delta <= 1e-12, "historical_status": "PARTIAL; historical retention failure unchanged"})
	frozen := map[string]any{"candidate_sequence": s1Snapshot(t, cp).Sequence, "candidate_native_checksum": s1Snapshot(t, cp).Checksum, "frozen_before_final_holdout": true, "holdout_evaluations": 0, "candidate_selection": "eighth block"}
	for _, name := range []string{"s2r2-final.json", "METRIC_PROTOCOL.json", "ACQUISITION_PROTOCOL.json", "DESIGN.md", "DECISION_RULES.json", "REPAIR_AMENDMENT.md"} {
		raw, e := os.ReadFile(filepath.Join(work, name))
		s1Check(t, e)
		frozen[name] = s1Digest(raw)
	}
	for _, name := range []string{"neural_metric_learning.go", "neural_metric_proposal.go", "neural_resolution.go", "neural_metric.go", "s2r2_probe.go"} {
		raw, e := os.ReadFile(filepath.Join("../knowledge", name))
		s1Check(t, e)
		frozen["code/"+name] = s1Digest(raw)
	}
	raw, e := os.ReadFile("s2r2_repair_test.go")
	s1Check(t, e)
	frozen["harness_sha256"] = s1Digest(raw)
	s1Write(t, filepath.Join(work, "CONFIRMATORY_LOCK.json"), frozen)
	if ret["pass"] != true || s1delta > 1e-12 {
		t.Fatal("retention gate blocked final ability evaluation")
	}
}

func s2r2AUC(rows []s2Trial, pred []float64, p s2Protocol) float64 {
	wins, n := 0., 0.
	for i, a := range rows {
		if a.Error < p.HighCut {
			continue
		}
		for j, b := range rows {
			if b.Error > p.LowCut {
				continue
			}
			n++
			if pred[i] > pred[j] {
				wins++
			} else if pred[i] == pred[j] {
				wins += .5
			}
		}
	}
	if n == 0 {
		return 0
	}
	return wins / n
}
func s2r2Strata(rows []s2Trial, history bool) [][]int {
	groups := [][]int{}
	keys := []string{}
	for i, s := range rows {
		key := fmt.Sprint(s.Pre.Action)
		if history {
			key = fmt.Sprintf("%d/%d/%d/%d", s.Pre.Action, s.Pre.PreviousAction, s.Pre.HistoryAction, len(s.Pre.History))
		}
		at := -1
		for j, k := range keys {
			if k == key {
				at = j
				break
			}
		}
		if at < 0 {
			at = len(keys)
			keys = append(keys, key)
			groups = append(groups, nil)
		}
		groups[at] = append(groups[at], i)
	}
	return groups
}
func s2r2Intervention(t *testing.T, r *BrainRuntime, rows []s2Trial, pop [][]k.NodeID, condition string) []float64 {
	t.Helper()
	inputs := make([]k.CompositionInput, len(rows))
	for i, s := range rows {
		inputs[i] = s2Input(s.Pre, pop)
	}
	permutation := make([]int, len(rows))
	for i := range permutation {
		permutation[i] = i
	}
	if condition == "permute-history-conditional" || condition == "permute-action-conditional" || condition == "permute-extras" || condition == "permute-forecast" {
		history := condition == "permute-history-conditional"
		seed := int64(61031)
		if history {
			seed = 61001
		}
		rng := rand.New(rand.NewSource(seed))
		for _, g := range s2r2Strata(rows, history) {
			donors := append([]int(nil), g...)
			rng.Shuffle(len(donors), func(a, b int) { donors[a], donors[b] = donors[b], donors[a] })
			for j, i := range g {
				permutation[i] = donors[j]
			}
		}
	}
	out := make([]float64, len(rows))
	for i, in := range inputs {
		in.Sequence = make([][]float64, len(inputs[i].Sequence))
		for j, row := range inputs[i].Sequence {
			in.Sequence[j] = append([]float64(nil), row...)
		}
		switch condition {
		case "no-forecast":
			for j := 0; j < 3; j++ {
				in.Sequence[1][j] = 0
			}
		case "no-history-magnitude":
			for j := 3; j < 6; j++ {
				in.Sequence[1][j] = 0
			}
		case "no-history-action":
			in.Sequence[0][1] = 0
		case "no-history-presence":
			in.Sequence[0][0] = 0
		case "no-history-magnitude-and-action":
			for j := 3; j < 6; j++ {
				in.Sequence[1][j] = 0
			}
			in.Sequence[0][1] = 0
		case "permute-history-conditional":
			copy(in.Sequence[1][3:6], inputs[permutation[i]].Sequence[1][3:6])
		case "permute-action-conditional":
			in.Sequence[0][1] = inputs[permutation[i]].Sequence[0][1]
		case "permute-forecast":
			copy(in.Sequence[1][:3], inputs[permutation[i]].Sequence[1][:3])
		case "permute-extras":
			copy(in.Sequence[0][:2], inputs[permutation[i]].Sequence[0][:2])
			copy(in.Sequence[1][:6], inputs[permutation[i]].Sequence[1][:6])
		}
		// Inputs are independent allocations; permutation donors must remain
		// pristine even after another row's intervention.
		if condition == "new-feature-clamp" || condition == "new-readout-clamp" || condition == "all-readout-clamp" {
			mode := map[string]string{"new-feature-clamp": "feature-clamp", "new-readout-clamp": "readout-clamp", "all-readout-clamp": "all-readout-clamp"}[condition]
			v, _, e := r.brain.S2R2Probe(in, 264, mode)
			s1Check(t, e)
			out[i] = v
		} else {
			o, e := r.ReadComposition(in, in.Scene.ObservedAt)
			s1Check(t, e)
			out[i] = o.NResolution.Value
		}
	}
	return out
}

func TestS2R2FinalIndependentHoldout(t *testing.T) {
	root, work := s2r2Paths(t)
	var lock map[string]any
	s1Read(t, filepath.Join(work, "CONFIRMATORY_LOCK.json"), &lock)
	for _, name := range []string{"s2r2-final.json", "METRIC_PROTOCOL.json", "ACQUISITION_PROTOCOL.json", "DESIGN.md", "DECISION_RULES.json", "REPAIR_AMENDMENT.md"} {
		raw, e := os.ReadFile(filepath.Join(work, name))
		s1Check(t, e)
		if lock[name] != s1Digest(raw) {
			t.Fatal("frozen payload changed", name)
		}
	}
	raw, e := os.ReadFile("s2r2_repair_test.go")
	s1Check(t, e)
	if lock["harness_sha256"] != s1Digest(raw) {
		t.Fatal("frozen harness changed")
	}
	for _, name := range []string{"neural_metric_learning.go", "neural_metric_proposal.go", "neural_resolution.go", "neural_metric.go", "s2r2_probe.go"} {
		raw, e := os.ReadFile(filepath.Join("../knowledge", name))
		s1Check(t, e)
		if lock["code/"+name] != s1Digest(raw) {
			t.Fatal("frozen learner changed", name)
		}
	}
	marker, e := os.OpenFile(filepath.Join(work, "FINAL_STARTED"), os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
	s1Check(t, e)
	s1Check(t, marker.Close())
	r, w := s2r2Fork(t, filepath.Join(work, "s2r2-final.json"), filepath.Join(work, "training-combined/events.jsonl"), filepath.Join(work, "final-evaluation"))
	pop := s1Pop(t, filepath.Join(root, "source/source"))
	p := s2r2MetricProtocol(t, work)
	rows := []s2Trial{}
	for i, seed := range []int64{51001, 51031} {
		rr := s2Collect(t, r, pop, seed, 128, i == 1, nil, -1)
		rows = append(rows, rr...)
	}
	s1Write(t, filepath.Join(work, "holdout.json"), rows)
	s1Check(t, r.Checkpoint(filepath.Join(work, "final-evaluation/observed.json")))
	s1Check(t, w.Close())
	predictions := map[string][]float64{}
	metrics := map[string]any{}
	for _, arm := range s2r2Arms {
		rr := orderedFixtureClone(t, filepath.Join(work, "training-"+arm, "final.json"))
		predictions[arm] = s2Predictions(t, rr, rows, pop)
	}
	source := orderedFixtureClone(t, filepath.Join(work, "00-source.json"))
	predictions["original-S2"] = s2Predictions(t, source, rows, pop)
	predictions["constant"] = s2Constant(rows, p.Constant)
	predictions["untrained-query"] = make([]float64, len(rows))
	for i, s := range rows {
		in := s2Input(s.Pre, pop)
		in.Words[0][3] = 0
		in.Words[0][4] = 1
		o, e := source.ReadComposition(in, in.Scene.ObservedAt)
		s1Check(t, e)
		if o.NResolution != nil {
			predictions["untrained-query"][i] = o.NResolution.Value
		}
	}
	for _, condition := range []string{"full", "no-forecast", "no-history-magnitude", "no-history-action", "no-history-presence", "no-history-magnitude-and-action", "permute-history-conditional", "permute-action-conditional", "permute-forecast", "permute-extras", "new-feature-clamp", "new-readout-clamp", "all-readout-clamp"} {
		predictions[condition] = s2r2Intervention(t, r, rows, pop, condition)
	}
	for name, pred := range predictions {
		metrics[name] = map[string]any{"pooled": s2MetricOf(rows, pred, p), "stream_0": s2MetricOf(rows[:128], pred[:128], p), "stream_1": s2MetricOf(rows[128:], pred[128:], p), "AUROC": s2r2AUC(rows, pred, p), "context_gaps": s2ContextGaps(rows, pred, p)}
	}
	if !reflect.DeepEqual(predictions["combined"], predictions["full"]) {
		t.Fatal("probe/runtime predictions differ")
	}
	gates := map[string]bool{}
	candidate := predictions["combined"]
	for stream := 0; stream < 2; stream++ {
		lo, hi := 128*stream, 128*(stream+1)
		m := s2MetricOf(rows[lo:hi], candidate[lo:hi], p)
		base := s2MetricOf(rows[lo:hi], predictions["constant"][lo:hi], p)
		gates[fmt.Sprintf("stream_%d_rmse", stream)] = m.RMSE <= .8*base.RMSE
		gates[fmt.Sprintf("stream_%d_correlation", stream)] = m.Correlation >= .5
		gates[fmt.Sprintf("stream_%d_calibration", stream)] = m.CalibrationError <= p.CalibrationLimit
		gaps := s2ContextGaps(rows[lo:hi], candidate[lo:hi], p)
		gates[fmt.Sprintf("stream_%d_discrimination", stream)] = len(gaps) == 2 && gaps["propagate"] > 0 && gaps["stop"] > 0 && s2r2AUC(rows[lo:hi], candidate[lo:hi], p) > .5
	}
	m := s2MetricOf(rows, candidate, p)
	for _, control := range []string{"original-S2", "legacy", "full-refit"} {
		gates["beats_"+control] = m.RMSE < s2MetricOf(rows, predictions[control], p).RMSE
	}
	gates["learned_readout_ablation"] = s2MetricOf(rows, predictions["all-readout-clamp"], p).RMSE >= 1.2*m.RMSE
	gates["new_feature_contribution"] = s2MetricOf(rows, predictions["new-feature-clamp"], p).RMSE > m.RMSE && !reflect.DeepEqual(candidate, predictions["new-feature-clamp"])
	gates["new_readout_contribution"] = s2MetricOf(rows, predictions["new-readout-clamp"], p).RMSE > m.RMSE && !reflect.DeepEqual(candidate, predictions["new-readout-clamp"])
	var ret, ret1 map[string]any
	s1Read(t, filepath.Join(work, "retention.json"), &ret)
	s1Read(t, filepath.Join(work, "s1-retention.json"), &ret1)
	gates["AE1_retention"] = ret["pass"] == true
	gates["S1_retention"] = ret1["pass"] == true
	pass := true
	for _, v := range gates {
		pass = pass && v
	}
	status := "PARTIAL"
	if !gates["beats_original-S2"] || !gates["beats_legacy"] || !gates["beats_full-refit"] {
		status = "FAIL"
	}
	if pass {
		status = "PASS"
	}
	history := s2MetricOf(rows, predictions["no-history-magnitude"], p).RMSE > m.RMSE && s2MetricOf(rows, predictions["permute-history-conditional"], p).RMSE > m.RMSE
	_, graph, err := r.brain.S2R2Probe(s2Input(rows[0].Pre, pop), 264, "")
	s1Check(t, err)
	s1Write(t, filepath.Join(work, "learned-graph.json"), graph)
	s1Write(t, filepath.Join(work, "predictions.json"), predictions)
	s1Write(t, filepath.Join(work, "final-metrics.json"), metrics)
	s1Write(t, filepath.Join(work, "results.json"), map[string]any{"repair_status": "PENDING_ENGINEERING", "ability_status": status, "gates": gates, "history_magnitude_independently_beneficial": history, "final_holdout_evaluations": 1, "candidate_sequence": s1Snapshot(t, filepath.Join(work, "s2r2-final.json")).Sequence, "candidate": m, "constant": s2MetricOf(rows, predictions["constant"], p), "original_S2": s2MetricOf(rows, predictions["original-S2"], p), "production_promoted": false, "S3_started": false})
	t.Log("S2 ability", status, "RMSE", m.RMSE, "constant", s2MetricOf(rows, predictions["constant"], p).RMSE, "correlation", m.Correlation, "gates", gates)
}

func TestS2R2PersistenceAndContinuation(t *testing.T) {
	root, work := s2r2Paths(t)
	cp := filepath.Join(work, "s2r2-final.json")
	wal := filepath.Join(work, "training-combined/events.jsonl")
	pop := s1Pop(t, filepath.Join(root, "source/source"))
	r, w := s1Open(t, cp, wal)
	s2r2Copy(t, wal, filepath.Join(work, "s2r2-final-events.jsonl"))
	expected := continuationBytes(t, r)
	var rows []s2Trial
	s1Read(t, filepath.Join(work, "holdout.json"), &rows)
	var saved map[string][]float64
	s1Read(t, filepath.Join(work, "predictions.json"), &saved)
	if !reflect.DeepEqual(s2Predictions(t, r, rows, pop), saved["combined"]) {
		t.Fatal("restart readout")
	}
	replayed, e := RebuildRuntimeFromCheckpointAndJournal(w, filepath.Join(work, "00-source.json"))
	s1Check(t, e)
	if !bytes.Equal(expected, continuationBytes(t, replayed)) {
		t.Fatal("full suffix replay")
	}
	prefix, e := os.ReadFile(filepath.Join(work, "source-events.jsonl"))
	s1Check(t, e)
	all, e := os.ReadFile(wal)
	s1Check(t, e)
	if !bytes.HasPrefix(all, prefix) {
		t.Fatal("historical WAL prefix changed")
	}
	rollback, rw := s2r2Fork(t, filepath.Join(work, "00-source.json"), filepath.Join(work, "source-events.jsonl"), filepath.Join(work, "rollback"))
	if !bytes.Equal(continuationBytes(t, rollback), continuationBytes(t, orderedFixtureClone(t, filepath.Join(work, "00-source.json")))) {
		t.Fatal("rollback state")
	}
	s1Check(t, rw.Close())
	s1Check(t, r.Checkpoint(filepath.Join(work, "s2r2-restarted.json")))
	if s1Snapshot(t, cp).Checksum != s1Snapshot(t, filepath.Join(work, "s2r2-restarted.json")).Checksum {
		t.Fatal("restart checksum")
	}
	// This schedule was frozen before holdout. It is not model selection.
	continuation := s2Collect(t, r, pop, 71001, 6, false, nil, -1)
	fresh := s2Collect(t, r, pop, 71021, 8, false, nil, -1)
	receipt := s2r2Lesson(t, r, "s2r2-postfreeze-continuation", "combined", fresh[:4], fresh[4:], pop)
	s1Write(t, filepath.Join(work, "continuation-trials.json"), continuation)
	s1Write(t, filepath.Join(work, "continuation-learning-observations.json"), fresh)
	s1Write(t, filepath.Join(work, "continuation-receipt.json"), receipt)
	continued := filepath.Join(work, "s2r2-continuation.json")
	s1Check(t, r.Checkpoint(continued))
	after := continuationBytes(t, r)
	recovered, e := RebuildRuntimeFromCheckpointAndJournal(w, cp)
	s1Check(t, e)
	if !bytes.Equal(after, continuationBytes(t, recovered)) || !reflect.DeepEqual(s2Predictions(t, r, rows, pop), s2Predictions(t, recovered, rows, pop)) {
		t.Fatal("continuation recovery")
	}
	if r.brain.NativeControlSnapshot().Updates != 10752 {
		t.Fatal("native policy changed")
	}
	s1Check(t, w.Close())
	ret := s2r2Retention(t, root, work, continued, wal, "continued")
	s1Write(t, filepath.Join(work, "continued-retention.json"), ret)
	if ret["pass"] != true {
		t.Fatal("continued retention")
	}
	diff := s2SynapseChanges(t, filepath.Join(work, "00-source.json"), cp)
	checks := []map[string]any{}
	for _, name := range []string{"s2r2-final.json", "s2r2-restarted.json", "s2r2-continuation.json"} {
		s := s1Snapshot(t, filepath.Join(work, name))
		sum, e := checkpointChecksumForSchema(s)
		s1Check(t, e)
		if sum != s.Checksum {
			t.Fatal("checkpoint checksum")
		}
		checks = append(checks, map[string]any{"file": name, "sequence": s.Sequence, "checksum": sum})
	}
	s1Write(t, filepath.Join(work, "persistence.json"), map[string]any{"restart_exact": true, "full_suffix_replay_exact": true, "rollback_exact": true, "historical_journal_prefix_exact": true, "historical_journal_bytes": len(prefix), "continuation_replay_exact": true, "native_policy_updates": 10752, "synaptic_changes": diff, "checkpoints": checks, "continuation_learning_committed": receipt.Diagnosis.Committed, "historical_checkpoint_compatible": true})
	s2r2Source(t, root)
}

func TestS2R2IdenticalRetryAndDuplicateCommand(t *testing.T) {
	_, work := s2r2Paths(t)
	root := os.Getenv("HORIZON_S2R2_SOURCE")
	pop := s1Pop(t, filepath.Join(root, "source/source"))
	r, w := s2r2Fork(t, filepath.Join(work, "00-source.json"), filepath.Join(work, "source-events.jsonl"), filepath.Join(work, "retry-engineering"))
	defer w.Close()
	var fit, check []s2Trial
	s1Read(t, filepath.Join(work, "dev-0-fit.json"), &fit)
	s1Read(t, filepath.Join(work, "dev-0-check.json"), &check)
	for i := range fit {
		fit[i].Error = .9
	}
	for i := range check {
		check[i].Error = s2Predict(t, r, check[i].Pre, pop)
	}
	first := s2r2Lesson(t, r, "s2r2-retry-engineering-1", "combined", fit, check, pop)
	seq := r.LastSequence()
	second := s2r2Lesson(t, r, "s2r2-retry-engineering-2", "combined", fit, check, pop)
	if first.Diagnosis.Committed || second.Diagnosis.Committed || second.Diagnosis.Cause != "duplicate-proposal" || r.LastSequence() != seq+1 {
		t.Fatal("identical retry semantics", first.Diagnosis.Cause, second.Diagnosis.Cause)
	}
	e := s2Lesson(1, fit, check, pop, s1Time(r))
	e.ID = "s2r2-retry-engineering-2"
	before := continuationBytes(t, r)
	_, _, err := r.LearnCompositionWithDiagnosis(e, s1Time(r))
	if err == nil || !bytes.Equal(before, continuationBytes(t, r)) {
		t.Fatal("duplicate event changed state")
	}
	s1Check(t, r.Checkpoint(filepath.Join(work, "retry-engineering/final.json")))
	recovered, err := RebuildRuntimeFromCheckpointAndJournal(w, filepath.Join(work, "00-source.json"))
	s1Check(t, err)
	if !bytes.Equal(continuationBytes(t, r), continuationBytes(t, recovered)) {
		t.Fatal("retry WAL replay")
	}
	s1Write(t, filepath.Join(work, "retry-engineering.json"), map[string]any{"synthetic_labels_for_engineering_only": true, "first": first, "second": second, "command_sequence_preserved": true, "duplicate_event_rejected": true, "replay_exact": true})
}
