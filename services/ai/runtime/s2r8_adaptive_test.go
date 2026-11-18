//go:build horizon_s2r8 && horizon_s2r4 && horizon_s2r3 && horizon_s2r2

package runtime

import (
	"bytes"
	"encoding/json"
	"fmt"
	"math"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strconv"
	"syscall"
	"testing"
	"time"

	k "github.com/project-horizon/horizon-core/services/ai/knowledge"
)

var s2r8Seeds = []int{37, 83, 151, 211, 307, 419}

const s2r8ComputeBudgetNS int64 = 25000000000

func s2r8CPU() int64 {
	var v syscall.Rusage
	_ = syscall.Getrusage(syscall.RUSAGE_SELF, &v)
	return (v.Utime.Sec+v.Stime.Sec)*1000000000 + (v.Utime.Usec+v.Stime.Usec)*1000
}

type s2r8Corpus struct {
	Seed              int
	Fit, Check        [][]s2Trial
	Monitor           []s2Trial
	TrainingFinalTime time.Time
}

func s2r8Paths(t *testing.T) (string, string, int, [][]k.NodeID) {
	t.Helper()
	root, work := os.Getenv("R8_SOURCE"), os.Getenv("R8_WORK")
	if root == "" || work == "" {
		t.Skip("isolated R8 paths required")
	}
	if root == work || filepath.Dir(work) == root {
		t.Fatal("source/work overlap")
	}
	seed, err := strconv.Atoi(os.Getenv("R8_SEED"))
	s1Check(t, err)
	s1Check(t, os.MkdirAll(work, 0700))
	t.Setenv("HORIZON_S2_SOURCE", root)
	t.Setenv("HORIZON_S2_WORK", work)
	return root, work, seed, s1Pop(t, filepath.Join(root, "source/source/source"))
}
func s2r8Write(t *testing.T, p string, v any) {
	t.Helper()
	s1Check(t, os.MkdirAll(filepath.Dir(p), 0700))
	s1Write(t, p, v)
}
func s2r8Zip(t *testing.T, p string) {
	t.Helper()
	b, e := exec.Command("zstd", "-q", "-T1", "--rm", p).CombinedOutput()
	if e != nil {
		t.Fatal(e, string(b))
	}
}
func s2r8Unzip(t *testing.T, p string) {
	t.Helper()
	if _, e := os.Stat(p); e == nil {
		return
	}
	b, e := exec.Command("zstd", "-q", "-d", "-k", p+".zst").CombinedOutput()
	if e != nil {
		t.Fatal(e, string(b))
	}
}
func s2r8Roots(t *testing.T, root string) []k.NodeID {
	r := orderedFixtureClone(t, filepath.Join(root, "source/source/s1/lineage/s1-continuation.json"))
	ids := []k.NodeID{}
	for _, f := range r.brain.BrainState.NResolution.Features {
		ids = append(ids, f.Node)
	}
	return ids
}
func s2r8Config(arm string, seed, b int, roots []k.NodeID) *k.NMetricProposalConfig {
	p := &k.NMetricProposalConfig{Protocol: k.NMetricAdaptiveV1, Strategy: "combined", Fractions: append([]float64(nil), s2r2Fractions...), PreserveShared: true, FeatureLimit: 96, Ridge: .001, RefineLeafDirections: true}
	if arm == "A" {
		p.FixedBasis = true
		p.RefineLeafDirections = false
		p.FeatureLimit = 32
	}
	if arm == "C" {
		p.ExcludedFeatures = append([]k.NodeID(nil), roots...)
	}
	if arm == "D" {
		p.PermuteGrowth = true
		p.GrowthSeed = int64(813000000 + seed*100 + b)
	}
	return p
}
func s2r8Lesson(t *testing.T, r *BrainRuntime, c s2r8Corpus, arm string, b int, roots []k.NodeID, pop [][]k.NodeID) *CompositionLearningReceipt {
	at := c.TrainingFinalTime.Add(time.Duration(b+1) * time.Nanosecond)
	e := s2Lesson(b+1, c.Fit[b], c.Check[b], pop, at)
	id := fmt.Sprintf("r8-%d-%s-block-%d-%d", c.Seed, arm, b, at.UnixNano())
	e.ID = id
	e.EpisodeID = id
	e.IndependenceGroup = id
	e.Source = "r8-authenticated-actual-neural-error"
	e.Resolution.Proposal = s2r8Config(arm, c.Seed, b, roots)
	// Historical R2 training observations are retention-only evidence. New
	// holdout labels never enter fitting or admission. Keep contexts in banks.
	for _, index := range []int{0, 1} {
		var historical []s2Trial
		s1Read(t, filepath.Join(os.Getenv("R8_SOURCE"), fmt.Sprintf("s2r2/train-%d-fit.json", index)), &historical)
		bank := []k.NObservation{}
		for _, row := range historical[:32] {
			bank = append(bank, k.NObservation{Input: s2Input(row.Pre, pop), Outcome: row.Error, ObservedAt: row.AfterAt, EpisodeID: row.ID})
		}
		e.Resolution.RetentionBanks = append(e.Resolution.RetentionBanks, bank)
	}
	r.SetClock(NewFixedClock(at))
	_, rec, err := r.LearnCompositionWithDiagnosis(e, at)
	s1Check(t, err)
	if rec == nil {
		t.Fatal("missing native receipt")
	}
	return rec
}
func s2r8Score(rows []s2Trial, pred []float64) map[string]any {
	mse, mae := 0., 0.
	se := make([]float64, len(rows))
	target := make([]float64, len(rows))
	for i, r := range rows {
		d := pred[i] - r.Error
		se[i] = d * d
		target[i] = r.Error
		mse += d * d
		mae += math.Abs(d)
	}
	return map[string]any{"n": len(rows), "RMSE": math.Sqrt(mse / float64(len(rows))), "MAE": mae / float64(len(rows)), "squared_errors": se, "predictions": pred, "targets": target}
}

// Acquisition is completed and hashed before any confirmatory fitting.
// The training and final holdout files are separate, and Train never opens
// the final holdout file. Each label follows a durable pre-operation ledger.
func TestS2R8Prepare(t *testing.T) {
	root, work, seed, pop := s2r8Paths(t)
	cp, wal := s2r3Source(root)
	dir := filepath.Join(work, fmt.Sprintf("seed-%d", seed))
	s1Check(t, os.Mkdir(dir, 0700))
	snap := s1Snapshot(t, cp)
	sum, err := checkpointChecksumForSchema(snap)
	s1Check(t, err)
	if snap.Sequence != 80735 || sum != s2r3Checksum || s2r3FileHash(t, cp) != "445991d177dc955ec58e7f9a3244a3f27da98fb90de584b09d4cc05c5c82d29d" || s2r3FileHash(t, wal) != "05659f8f3ea4bbe29fbcdd90a4f1c00ee8fbbe488234ee4af762730926c083d8" {
		t.Fatal("unauthenticated R2 source")
	}
	r, w := s2r2Fork(t, cp, wal, filepath.Join(dir, "training-observer"))
	t.Setenv("HORIZON_S2_WORK", filepath.Join(dir, "training-observer"))
	c := s2r8Corpus{Seed: seed}
	for b := 0; b < 4; b++ {
		base := int64(818000000 + seed*1000 + b*20)
		var fit, check []s2Trial
		if b%2 == 0 {
			fit = s2Collect(t, r, pop, base+1, 32, b == 2, nil, -1)
			check = s2Collect(t, r, pop, base+7, 32, b == 2, nil, -1)
		} else {
			fit = s2r4Trials(s2r4Collect(t, r, pop, filepath.Join(dir, fmt.Sprintf("training-observer/persistent-%d-fit", b)), "persistent", base+1, 4, 8, b == 3))
			check = s2r4Trials(s2r4Collect(t, r, pop, filepath.Join(dir, fmt.Sprintf("training-observer/persistent-%d-check", b)), "persistent", base+7, 4, 8, b == 3))
		}
		c.Fit = append(c.Fit, fit)
		c.Check = append(c.Check, check)
	}
	c.Monitor = s2Collect(t, r, pop, int64(819000000+seed*1000), 32, true, nil, -1)
	c.TrainingFinalTime = s2r4Now(r)
	s2r8Write(t, filepath.Join(dir, "TRAINING.json"), c)
	s1Check(t, r.Checkpoint(filepath.Join(dir, "training-observer/final.json")))
	s1Check(t, w.Close())
	h, hwal := s2r2Fork(t, cp, wal, filepath.Join(dir, "holdout-observer"))
	t.Setenv("HORIZON_S2_WORK", filepath.Join(dir, "holdout-observer"))
	hold := map[string][][]s2Trial{}
	for condition, name := range []string{"normal", "shifted", "persistent"} {
		for stream := 0; stream < 4; stream++ {
			base := int64(828000000 + seed*1000 + condition*100 + stream*10 + 1)
			var rows []s2Trial
			if name == "persistent" {
				rows = s2r4Trials(s2r4Collect(t, h, pop, filepath.Join(dir, fmt.Sprintf("holdout-observer/persistent-%d", stream)), "persistent", base, 1, 24, true))
			} else {
				rows = s2Collect(t, h, pop, base, 24, condition == 1, nil, -1)
			}
			hold[name] = append(hold[name], rows)
		}
	}
	s2r8Write(t, filepath.Join(dir, "FINAL_HOLDOUT.json"), hold)
	s1Check(t, h.Checkpoint(filepath.Join(dir, "holdout-observer/final.json")))
	s1Check(t, hwal.Close())
	s2r8Write(t, filepath.Join(dir, "DATA_LOCK.json"), map[string]any{"training_sha256": s2r3FileHash(t, filepath.Join(dir, "TRAINING.json")), "holdout_sha256": s2r3FileHash(t, filepath.Join(dir, "FINAL_HOLDOUT.json")), "source_checksum": sum, "source_sequence": 80735, "inventory": s2r3Inventory(t, orderedFixtureClone(t, cp)), "training_observations": 256, "monitor_observations": 32, "final_holdout_observations": 288, "actual_operations": 576, "fit_check_independent": true, "preoperation_ledgers": true})
	t.Log("locked actual data", seed, "256 training, 32 monitor, 288 independent holdout")
}

func s2r8NumericRetention(t *testing.T, source, candidate *BrainRuntime, rows []s2Trial, pop [][]k.NodeID) map[string]any {
	a := s1Predictions(t, source, s2S1Rows(rows), pop)
	b := s1Predictions(t, candidate, s2S1Rows(rows), pop)
	d := s1AuditMax(s1AuditValue(t, a), s1AuditValue(t, b))
	sp, cp := s2Predictions(t, source, rows, pop), s2Predictions(t, candidate, rows, pop)
	s, cs := s2r8Score(rows, sp), s2r8Score(rows, cp)
	policyExact := reflect.DeepEqual(source.brain.NativeControlSnapshot(), candidate.brain.NativeControlSnapshot())
	pass := d <= 1e-7 && cs["RMSE"].(float64) <= s["RMSE"].(float64)*1.05+1e-7 && policyExact
	return map[string]any{"S1_max_prediction_shift": d, "S2_source": s, "S2_candidate": cs, "S2_RMSE_limit": s["RMSE"].(float64)*1.05 + 1e-7, "native_policy_exact": policyExact, "pass": pass}
}

// Behavioral probe timestamps depend on revision in the historical oracle.
// Match revision and physical memory time only on a disposable evaluation
// control. This does not alter source bytes, parameters, or replay evidence.
func s2r8AE1(t *testing.T, controlPath, candidatePath, dir string, pop [][]k.NodeID) map[string]any {
	control := orderedFixtureClone(t, controlPath)
	candidate := orderedFixtureClone(t, candidatePath)
	control.mu.Lock()
	control.seq = candidate.LastSequence()
	control.settleMemoryTimeLocked(candidate.brain.BrainState.MemoryDecay.AppliedAt)
	control.mu.Unlock()
	s1Check(t, os.MkdirAll(dir, 0700))
	cp := filepath.Join(dir, "control.json")
	s1Check(t, control.Checkpoint(cp))
	ci := s2r4OracleImage(t, cp, filepath.Join(dir, "control-oracle.json"))
	ca := s2r4OracleImage(t, candidatePath, filepath.Join(dir, "candidate-oracle.json"))
	out := s2r3AE1(t, ci, ca, pop)
	out["evaluation_only_revision_match"] = true
	out["control_sequence"] = control.LastSequence()
	out["candidate_sequence"] = candidate.LastSequence()
	out["physical_time"] = candidate.brain.BrainState.MemoryDecay.AppliedAt
	return out
}

func s2r8CheckLock(t *testing.T, work string) {
	var v struct{ Files map[string]string }
	s1Read(t, filepath.Join(work, "CONFIRMATORY_LOCK.json"), &v)
	if len(v.Files) < 10 {
		t.Fatal("missing scientific lock")
	}
	for p, sha := range v.Files {
		if s2r3FileHash(t, p) != sha {
			t.Fatal("frozen scientific input changed", p)
		}
	}
}

func TestS2R8Train(t *testing.T) {
	root, work, seed, pop := s2r8Paths(t)
	arm := os.Getenv("R8_ARM")
	if arm != "A" && arm != "B" && arm != "C" && arm != "D" {
		t.Fatal("unknown arm")
	}
	if seed != 13 {
		s2r8CheckLock(t, work)
	}
	base := filepath.Join(work, fmt.Sprintf("seed-%d", seed))
	var c s2r8Corpus
	s1Read(t, filepath.Join(base, "TRAINING.json"), &c)
	cp, wal := filepath.Join(base, "training-observer/final.json"), filepath.Join(base, "training-observer/events.jsonl")
	s2r8Unzip(t, cp)
	s2r8Unzip(t, wal)
	dir := filepath.Join(base, arm)
	r, w := s2r2Fork(t, cp, wal, dir)
	defer w.Close()
	roots := s2r8Roots(t, root)
	source := orderedFixtureClone(t, cp)
	var old []s2Trial
	s1Read(t, filepath.Join(root, "s2r2/holdout.json"), &old)
	receipts := []*CompositionLearningReceipt{}
	curves := []any{map[string]any{"block": 0, "exposures": 0, "inventory": s2r3Inventory(t, r), "monitor": s2r8Score(c.Monitor, s2Predictions(t, r, c.Monitor, pop))}}
	latency := []int64{}
	retention := []any{}
	for b := 0; b < 4; b++ {
		before := s2r3Memory()
		cpuStarted := s2r8CPU()
		started := time.Now()
		rec := s2r8Lesson(t, r, c, arm, b, roots, pop)
		ns := time.Since(started).Nanoseconds()
		cpuNS := s2r8CPU() - cpuStarted
		after := s2r3Memory()
		latency = append(latency, ns)
		receipts = append(receipts, rec)
		path := filepath.Join(dir, fmt.Sprintf("block-%d.json", b+1))
		s1Check(t, r.Checkpoint(path))
		scratch := t.TempDir()
		ae1 := s2r8AE1(t, cp, path, scratch, pop)
		retention = append(retention, map[string]any{"block": b + 1, "AE1": ae1, "historical_S1_S2": s2r8NumericRetention(t, source, r, old, pop)})
		s1Check(t, os.RemoveAll(scratch))
		curves = append(curves, map[string]any{"block": b + 1, "exposures": (b + 1) * 64, "inventory": s2r3Inventory(t, r), "monitor": s2r8Score(c.Monitor, s2Predictions(t, r, c.Monitor, pop)), "learning_nanoseconds": ns, "learning_cpu_nanoseconds": cpuNS, "memory_before": before, "memory_after": after, "committed": rec.Diagnosis.Committed, "cause": rec.Diagnosis.Cause, "trials": len(rec.Diagnosis.Trials)})
		s2r8Write(t, filepath.Join(dir, "receipts.json"), receipts)
		s2r8Write(t, filepath.Join(dir, "curves.json"), curves)
		s2r8Write(t, filepath.Join(dir, "retention.json"), retention)
		s2r8Zip(t, path)
		t.Log(seed, arm, b+1, rec.Diagnosis.Committed, rec.Diagnosis.Cause, "features", len(r.brain.BrainState.NResolution.Features), "ns", ns, "AE1", ae1["pass"])
	}
	final := filepath.Join(dir, "final.json")
	s1Check(t, r.Checkpoint(final))
	original := continuationBytes(t, r)
	restart, err := RebuildRuntimeFromCheckpointAndJournal(w, final)
	s1Check(t, err)
	replayStart := time.Now()
	replay, err := RebuildRuntimeFromCheckpointAndJournal(w, filepath.Join(root, "s2r2/s2r2-continuation.json"))
	s1Check(t, err)
	replayNS := time.Since(replayStart).Nanoseconds()
	raw, err := os.ReadFile(filepath.Join(dir, "events.jsonl"))
	s1Check(t, err)
	prefix, err := os.ReadFile(filepath.Join(root, "s2r2/training-combined/events.jsonl"))
	s1Check(t, err)
	persist := map[string]any{"restart_exact": bytes.Equal(original, continuationBytes(t, restart)), "full_suffix_replay_exact": bytes.Equal(original, continuationBytes(t, replay)), "source_prefix_exact": bytes.HasPrefix(raw, prefix), "replay_nanoseconds": replayNS, "inventory": s2r3Inventory(t, r), "final_native_checksum": s1Snapshot(t, final).Checksum, "learning_nanoseconds": latency, "single_N_owner": r.activation.Memory == r.brain && r.learning.Kb == r.brain && r.dnf.Brain() == r.brain}
	// Identical additional learning after restart and on the live runtime.
	continuedDir := filepath.Join(dir, "restart-continuation")
	resumed, resumedWAL := s2r2Fork(t, final, filepath.Join(dir, "events.jsonl"), continuedDir)
	c2 := c
	c2.TrainingFinalTime = c.TrainingFinalTime.Add(10 * time.Nanosecond)
	x := s2r8Lesson(t, r, c2, arm, 0, roots, pop)
	y := s2r8Lesson(t, resumed, c2, arm, 0, roots, pop)
	persist["learning_after_restart_exact"] = bytes.Equal(continuationBytes(t, r), continuationBytes(t, resumed)) && reflect.DeepEqual(x, y)
	s1Check(t, r.Checkpoint(filepath.Join(dir, "continued.json")))
	s1Check(t, resumedWAL.Close())
	s1Check(t, os.RemoveAll(continuedDir))
	continuedReplay, err := RebuildRuntimeFromCheckpointAndJournal(w, filepath.Join(root, "s2r2/s2r2-continuation.json"))
	s1Check(t, err)
	persist["continued_full_replay_exact"] = bytes.Equal(continuationBytes(t, r), continuationBytes(t, continuedReplay))
	persist["continued_native_checksum"] = s1Snapshot(t, filepath.Join(dir, "continued.json")).Checksum
	// Roll back a detached disposable copy; the live experiment WAL is retained.
	continuedReplay.SetEventLog(nil)
	s1Check(t, continuedReplay.RestoreCheckpoint(filepath.Join(root, "s2r2/s2r2-continuation.json")))
	persist["source_rollback_exact"] = bytes.Equal(continuationBytes(t, continuedReplay), continuationBytes(t, orderedFixtureClone(t, filepath.Join(root, "s2r2/s2r2-continuation.json"))))
	s2r8Write(t, filepath.Join(dir, "persistence.json"), persist)
	s1Check(t, w.Close())
	for _, p := range []string{filepath.Join(dir, "state.json"), final, filepath.Join(dir, "continued.json"), filepath.Join(dir, "events.jsonl")} {
		s2r8Zip(t, p)
	}
	for _, name := range []string{"restart_exact", "full_suffix_replay_exact", "source_prefix_exact", "learning_after_restart_exact", "continued_full_replay_exact", "source_rollback_exact", "single_N_owner"} {
		if persist[name] != true {
			t.Error("persistence failure retained", name)
		}
	}
}

func TestS2R8Evaluate(t *testing.T) {
	root, work, seed, pop := s2r8Paths(t)
	s2r8CheckLock(t, work)
	base := filepath.Join(work, fmt.Sprintf("seed-%d", seed))
	var hold map[string][][]s2Trial
	s1Read(t, filepath.Join(base, "FINAL_HOLDOUT.json"), &hold)
	results := map[string]any{}
	for _, arm := range []string{"A", "B", "C", "D"} {
		dir := filepath.Join(base, arm)
		path := filepath.Join(dir, "final.json")
		s2r8Unzip(t, path)
		r := orderedFixtureClone(t, path)
		result := map[string]any{}
		times := []int64{}
		for _, name := range []string{"normal", "shifted", "persistent"} {
			scores := []any{}
			for _, rows := range hold[name] {
				started := time.Now()
				pred := s2Predictions(t, r, rows, pop)
				ns := time.Since(started).Nanoseconds()
				times = append(times, ns/int64(len(rows)))
				scores = append(scores, s2r8Score(rows, pred))
			}
			result[name] = scores
		}
		result["inventory"] = s2r3Inventory(t, r)
		result["inference_stream_mean_nanoseconds"] = times
		roots := s2r8Roots(t, root)
		in := []k.CompositionInput{}
		for _, name := range []string{"normal", "shifted", "persistent"} {
			for _, rows := range hold[name] {
				for _, row := range rows {
					in = append(in, s2Input(row.Pre, pop))
				}
			}
		}
		trace, err := r.brain.S2R8Inspect(in, roots)
		s1Check(t, err)
		s2r8Write(t, filepath.Join(dir, "final-neural-trace.json"), trace)
		results[arm] = result
		var curves []struct {
			CPU int64 `json:"learning_cpu_nanoseconds"`
		}
		s1Read(t, filepath.Join(dir, "curves.json"), &curves)
		spent := int64(0)
		block := 0
		for i := 1; i < len(curves); i++ {
			if spent+curves[i].CPU > s2r8ComputeBudgetNS {
				break
			}
			spent += curves[i].CPU
			block = i
		}
		budgetPath := filepath.Join(base, "training-observer/final.json")
		if block > 0 {
			budgetPath = filepath.Join(dir, fmt.Sprintf("block-%d.json", block))
		}
		s2r8Unzip(t, budgetPath)
		budgetModel := orderedFixtureClone(t, budgetPath)
		budgetResult := map[string]any{"budget_cpu_nanoseconds": s2r8ComputeBudgetNS, "used_cpu_nanoseconds": spent, "block": block, "exposures": 64 * block, "protocol": "prelocked completed-command prefix under CPU budget; acquisition, retention, persistence and unused command costs excluded and reported separately"}
		for _, name := range []string{"normal", "shifted", "persistent"} {
			scores := []any{}
			for _, rows := range hold[name] {
				scores = append(scores, s2r8Score(rows, s2Predictions(t, budgetModel, rows, pop)))
			}
			budgetResult[name] = scores
		}
		result["compute_budget"] = budgetResult
		if block > 0 {
			s1Check(t, os.Remove(budgetPath))
		}
		s2r8Write(t, filepath.Join(base, "FINAL_RESULTS.json"), results)
		// final.json already has its immutable compressed copy.
		s1Check(t, os.Remove(path))
	}
	t.Log("final evaluated", seed)
}

var _ = json.Marshal

// An externally taught positive control on an unused measured coordinate.
// It is not included in S2 scores and never becomes a canonical successor.
func TestS2R8ControlledGrowthBeyond32(t *testing.T) {
	root, work, _, _ := s2r8Paths(t)
	cp, wal := s2r3Source(root)
	dir := filepath.Join(work, "controlled-growth-beyond-32")
	r, w := s2r2Fork(t, cp, wal, dir)
	defer w.Close()
	makeInput := func(x float64) k.CompositionInput {
		q := make([]float64, 16)
		q[8] = 1
		q[15] = .125
		seq := make([][]float64, 20)
		for i := range seq {
			seq[i] = make([]float64, 7)
			seq[i][6] = 1
		}
		seq[0][0] = x
		precision := .003
		return k.CompositionInput{Words: [][]float64{q}, Sequence: seq, Scene: s1Scene(s1Base), RequiredPrecision: &precision}
	}
	results := []any{}
	for b := 0; b < 4; b++ {
		lesson := &k.NResolutionLesson{Proposal: s2r8Config("B", 13, b, nil)}
		for set, xs := range [][]float64{{-.8, -.4, .4, .8}, {-.9, -.2, .2, .9}} {
			for i, x := range xs {
				o := k.NObservation{Input: makeInput(x), Outcome: .3 * x, ObservedAt: s1Base, EpisodeID: fmt.Sprintf("r8-positive-%d-%d-%d", b, set, i)}
				if set == 0 {
					lesson.Fit = append(lesson.Fit, o)
				} else {
					lesson.Check = append(lesson.Check, o)
				}
			}
		}
		id := fmt.Sprintf("r8-positive-%d", b)
		e := k.CompositionExposure{ID: id, Source: "external-positive-control", EpisodeID: id, IndependenceGroup: id, CausalGroup: id, Reliability: 1, Input: makeInput(.1), Motor: []float64{0, 0, 0, 0}, After: s1Scene(s1Base), Resolution: lesson}
		_, rec, err := r.LearnCompositionWithDiagnosis(e, s1Base)
		s1Check(t, err)
		results = append(results, rec)
	}
	out, err := r.ReadComposition(makeInput(.3), s1Base)
	s1Check(t, err)
	features := len(r.brain.BrainState.NResolution.Features)
	pass := features > 32 && out.NResolution != nil && math.Abs(out.NResolution.Value-.09) < .01
	s2r8Write(t, filepath.Join(dir, "RESULT.json"), map[string]any{"pass": pass, "features": features, "prediction": out, "receipts": results, "source_sequence": 80735, "scope": "new independently measured coordinate; positive mechanism control only"})
	s1Check(t, r.Checkpoint(filepath.Join(dir, "final.json")))
	replay, err := RebuildRuntimeFromCheckpointAndJournal(w, cp)
	s1Check(t, err)
	if !bytes.Equal(continuationBytes(t, r), continuationBytes(t, replay)) {
		t.Fatal("positive growth replay differs")
	}
	if !pass {
		t.Fatal("positive growth control failed")
	}
}
