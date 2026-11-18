//go:build horizon_s2r6 && horizon_s2r5 && horizon_s2r4 && horizon_s2r3 && horizon_s2r2

package runtime

import (
	"bytes"
	"fmt"
	k "github.com/project-horizon/horizon-core/services/ai/knowledge"
	"os"
	"path/filepath"
	"reflect"
	"testing"
	"time"
)

func TestS2R6CoveragePrepare(t *testing.T) {
	root, work, pop := s2r6Paths(t)
	cp, wal := s2r3Source(root)
	source := orderedFixtureClone(t, cp)
	thresholds := map[string]any{}
	for i, name := range []string{"normal", "shifted"} {
		dir := filepath.Join(work, "coverage", name)
		r, w := s2r2Fork(t, cp, wal, dir)
		t.Setenv("HORIZON_S2_WORK", dir)
		rows := s2r5OldCollect(t, r, pop, 181810001+int64(1000*i), 64, i == 1, nil, -1)
		s1Write(t, filepath.Join(dir, "rows.json"), rows)
		s2r3SaveCP(t, r, filepath.Join(dir, "final.json"))
		s1Check(t, w.Close())
		dd := filepath.Join(work, "retention-development", name)
		dr, dw := s2r2Fork(t, cp, wal, dd)
		t.Setenv("HORIZON_S2_WORK", dd)
		dev := s2r5OldCollect(t, dr, pop, 181820003+int64(1000*i), 128, i == 1, nil, -1)
		s1Write(t, filepath.Join(dd, "rows.json"), dev)
		s2r3SaveCP(t, dr, filepath.Join(dd, "final.json"))
		s1Check(t, dw.Close())
		ret := s2r4Retention(t, source, source, dev, pop)
		metric := ret["S2_source"].(s2Metric)
		thresholds[name] = map[string]any{"source_RMSE": metric.RMSE, "limit": metric.RMSE + 1e-7, "development_N": len(dev), "source_predictions": s2Predictions(t, source, dev, pop)}
	}
	s1Write(t, filepath.Join(work, "RETENTION_LOCK.json"), thresholds)
}

type s2r6Arm struct {
	Name                string
	Mask, IID, Coverage bool
}

func s2r6Arms() []s2r6Arm {
	return []s2r6Arm{{Name: "A"}, {Name: "A0", Mask: true}, {Name: "B", Coverage: true}, {Name: "B0", Mask: true, Coverage: true}, {Name: "BI", IID: true, Coverage: true}, {Name: "BI0", Mask: true, IID: true, Coverage: true}}
}
func s2r6AllArms() []string { return []string{"A", "A0", "B", "B0", "BI", "BI0", "C", "F"} }

func s2r6HistoryLesson(t *testing.T, r *BrainRuntime, arm s2r6Arm, id string, fit, check []s2r4Row, pop [][]k.NodeID, work string) (*CompositionLearningReceipt, map[string]any) {
	e := s2Lesson(1, s2r4Trials(fit), s2r4Trials(check), pop, s2r4Now(r))
	e.ID = id
	e.EpisodeID = id
	e.IndependenceGroup = id
	e.Source = "r6-fresh-actual-history-consequence"
	e.Resolution.Proposal = &k.NMetricProposalConfig{Protocol: k.NMetricConstrainedV1, Strategy: "full-refit", PreserveShared: true, FeatureLimit: 32, Update: "readout", RefineLeafDirections: true, BlockSparse: true}
	convert := func(rows []s2Trial) []k.NObservation {
		out := []k.NObservation{}
		for _, row := range rows {
			out = append(out, k.NObservation{Input: s2Input(row.Pre, pop), Outcome: row.Error, ObservedAt: row.AfterAt, EpisodeID: row.ID})
		}
		return out
	}
	var original []s2Trial
	s1Read(t, filepath.Join(work, "anchors/rows.json"), &original)
	e.Resolution.Retention = convert(original)
	if arm.Coverage {
		for _, name := range []string{"normal", "shifted"} {
			var rows []s2Trial
			s1Read(t, filepath.Join(work, "coverage", name, "rows.json"), &rows)
			e.Resolution.RetentionBanks = append(e.Resolution.RetentionBanks, convert(rows))
		}
	}
	if arm.Mask {
		for _, set := range [][]k.NObservation{e.Resolution.Fit, e.Resolution.Check} {
			for i := range set {
				for j := 3; j < 6; j++ {
					set[i].Input.Sequence[1][j] = 0
				}
			}
		}
		e.Input = e.Resolution.Fit[0].Input
	}
	start := time.Now()
	_, rec, err := r.LearnCompositionWithDiagnosis(e, s2r4Now(r))
	s1Check(t, err)
	return rec, map[string]any{"learning_nanoseconds": time.Since(start).Nanoseconds(), "trials": len(rec.Diagnosis.Trials), "memory": s2r3Memory(), "inventory": s2r3Inventory(t, r)}
}

func TestS2R6Train(t *testing.T) {
	root, work, pop := s2r6Paths(t)
	cp, wal := s2r3Source(root)
	seed := s2r6Seed(t)
	var decision struct {
		H1 bool `json:"H1_pass"`
	}
	s1Read(t, filepath.Join(work, "DEVELOPMENT_DECISION.json"), &decision)
	if decision.H1 {
		t.Fatal("H1 prerequisite passed: implement and preregister H2 before this restricted final protocol")
	}
	if _, err := os.Stat(filepath.Join(work, "TRAINING_LOCK.json")); err != nil {
		t.Fatal("training lock absent")
	}
	base := filepath.Join(work, fmt.Sprintf("training/seed-%d", seed))
	observed := map[bool][]s2r4Row{}
	for _, iid := range []bool{false, true} {
		name := "persistent"
		if iid {
			name = "independent"
		}
		dir := filepath.Join(base, "observer-"+name)
		r, w := s2r2Fork(t, cp, wal, dir)
		observed[iid] = s2r4Collect(t, r, pop, dir, name, 181000000+int64(seed*1000), 32, 16, false)
		s1Check(t, w.Close())
	}
	for _, arm := range s2r6Arms() {
		dir := filepath.Join(base, arm.Name)
		r, w := s2r2Fork(t, cp, wal, dir)
		name := "persistent"
		if arm.IID {
			name = "independent"
		}
		s2r5ReplayObserver(t, r, filepath.Join(base, "observer-"+name, "events.jsonl"))
		rows := observed[arm.IID]
		rec := []*CompositionLearningReceipt{}
		cost := []any{}
		for b := 0; b < 4; b++ {
			fit, check := s2r4Subset(rows, 4*b, 4*b+2), s2r4Subset(rows, 4*b+2, 4*b+4)
			s1Write(t, filepath.Join(dir, fmt.Sprintf("block-%d-fit.json", b)), fit)
			s1Write(t, filepath.Join(dir, fmt.Sprintf("block-%d-check.json", b)), check)
			receipt, c := s2r6HistoryLesson(t, r, arm, fmt.Sprintf("r6-s%d-%s-b%d", seed, arm.Name, b), fit, check, pop, work)
			rec = append(rec, receipt)
			cost = append(cost, c)
			s1Write(t, filepath.Join(dir, "receipts.json"), rec)
			s1Write(t, filepath.Join(dir, "costs.json"), cost)
			s2r3SaveCP(t, r, filepath.Join(dir, fmt.Sprintf("block-%d.json", b)))
			t.Log(seed, arm.Name, b, receipt.Diagnosis.Committed, receipt.Diagnosis.Cause)
		}
		s2r3SaveCP(t, r, filepath.Join(dir, "final.json"))
		s1Check(t, w.Close())
	}
	bd := filepath.Join(base, "B")
	br := orderedFixtureClone(t, filepath.Join(bd, "final.json"))
	utility := s2r6CausalRows(t, br, s2r4Subset(observed[false], 16, 32), pop, filepath.Join(base, "utility-acquisition"), 181300000+int64(seed*1000), false)
	for _, arm := range []string{"C", "F"} {
		dir := filepath.Join(base, arm)
		r, w := s2r2Fork(t, filepath.Join(bd, "final.json"), filepath.Join(bd, "events.jsonl"), dir)
		receipts := []*CompositionLearningReceipt{}
		costs := []any{}
		for b := 0; b < 4; b++ {
			fit, check := s2r6Selected(utility, 128*b, 128*b+64), s2r6Selected(utility, 128*b+64, 128*b+128)
			s1Write(t, filepath.Join(dir, fmt.Sprintf("utility-block-%d-fit.json", b)), fit)
			s1Write(t, filepath.Join(dir, fmt.Sprintf("utility-block-%d-check.json", b)), check)
			start := time.Now()
			rec := s2r6UtilityLesson(t, r, fmt.Sprintf("r6-s%d-%s-utility-%d", seed, arm, b), fit, check, arm == "F", int64(181400000+seed*100+b*10))
			receipts = append(receipts, rec)
			costs = append(costs, map[string]any{"learning_nanoseconds": time.Since(start).Nanoseconds(), "trials": len(rec.Diagnosis.Trials), "memory": s2r3Memory(), "inventory": s2r3Inventory(t, r)})
			s1Write(t, filepath.Join(dir, "receipts.json"), receipts)
			s1Write(t, filepath.Join(dir, "costs.json"), costs)
			s2r3SaveCP(t, r, filepath.Join(dir, fmt.Sprintf("block-%d.json", b)))
			t.Log(seed, arm, b, rec.Diagnosis.Committed, rec.Diagnosis.Cause)
		}
		before := s2Predictions(t, br, s2r4Trials(observed[false]), pop)
		after := s2Predictions(t, r, s2r4Trials(observed[false]), pop)
		if !reflect.DeepEqual(before, after) {
			t.Fatal("utility-only lesson changed S2 function")
		}
		s1Write(t, filepath.Join(dir, "fixed_basis_audit.json"), map[string]any{"S2_predictions_exact": true, "cases": len(before), "features": len(r.brain.BrainState.NResolution.Features), "native_policy_exact": reflect.DeepEqual(br.brain.NativeControlSnapshot(), r.brain.NativeControlSnapshot())})
		s2r3SaveCP(t, r, filepath.Join(dir, "final.json"))
		s1Check(t, w.Close())
	}
	if s2r3FileHash(t, cp) != s2r3FileHash(t, filepath.Join(work, "00-source.json")) {
		t.Fatal("source mutation")
	}
}

func TestS2R6AcquireFinal(t *testing.T) {
	root, work, pop := s2r6Paths(t)
	cp, wal := s2r3Source(root)
	s2r4CheckLock(t, work)
	f, err := os.OpenFile(filepath.Join(work, "FINAL_STARTED"), os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
	s1Check(t, err)
	s1Check(t, f.Close())
	for i, name := range []string{"persistent", "independent", "shifted"} {
		dir := filepath.Join(work, "holdout", name)
		r, w := s2r2Fork(t, cp, wal, dir)
		condition := name
		if name == "shifted" {
			condition = "persistent"
		}
		s2r4Collect(t, r, pop, dir, condition, 181600017+int64(i*10000), 32, 24, name == "shifted")
		s1Check(t, w.Close())
	}
	for i, name := range []string{"normal", "shifted"} {
		dir := filepath.Join(work, "old-holdout", name)
		r, w := s2r2Fork(t, cp, wal, dir)
		t.Setenv("HORIZON_S2_WORK", dir)
		rows := s2r5OldCollect(t, r, pop, 181700031+int64(i*10000), 128, i == 1, nil, -1)
		s1Write(t, filepath.Join(dir, "rows.json"), rows)
		s2r3SaveCP(t, r, filepath.Join(dir, "final.json"))
		s1Check(t, w.Close())
	}
}

func TestS2R6EvaluateFinal(t *testing.T) {
	root, work, pop := s2r6Paths(t)
	cp, _ := s2r3Source(root)
	s2r4CheckLock(t, work)
	seed := s2r6Seed(t)
	source := orderedFixtureClone(t, cp)
	var historical []s2Trial
	s1Read(t, filepath.Join(root, "s2r2/holdout.json"), &historical)
	for _, name := range s2r6AllArms() {
		dir := filepath.Join(work, fmt.Sprintf("training/seed-%d/%s", seed, name))
		start := time.Now()
		r := orderedFixtureClone(t, filepath.Join(dir, "final.json"))
		recovery := time.Since(start).Nanoseconds()
		ret := s2r4Retention(t, source, r, historical, pop)
		ret["strict_S1_pass"] = ret["S1_max_delta"].(float64) == 0
		ret["fixed_S2_limit"] = .01766457
		ret["fixed_S2_pass"] = ret["S2_candidate"].(s2Metric).RMSE <= .01766457
		s1Write(t, filepath.Join(dir, "historical-retention.json"), ret)
		for _, condition := range []string{"normal", "shifted"} {
			var rows []s2Trial
			s1Read(t, filepath.Join(work, "old-holdout", condition, "rows.json"), &rows)
			s1Write(t, filepath.Join(dir, "old-retention-"+condition+".json"), s2r4Retention(t, source, r, rows, pop))
		}
		mask := name == "A0" || name == "B0" || name == "BI0"
		resources := map[string]any{}
		for _, condition := range []string{"persistent", "independent", "shifted"} {
			var rows []s2r4Row
			s1Read(t, filepath.Join(work, "holdout", condition, "rows.json"), &rows)
			var intervention s2r4Interventions
			s1Read(t, filepath.Join(work, "holdout", condition, "input-interventions.json"), &intervention)
			predictions := map[string][]float64{}
			costs := []int64{}
			for _, mode := range []string{"full", "no-history"} {
				p, c := s2r4Predict(t, r, rows, pop, s2r4Arm{NoHistory: mask}, mode, nil)
				predictions[mode] = p
				if mode == "full" {
					costs = c
				}
			}
			for mode, donors := range intervention.Donors {
				p, _ := s2r4Predict(t, r, rows, pop, s2r4Arm{NoHistory: mask}, mode, donors)
				predictions[mode] = p
			}
			s1Write(t, filepath.Join(dir, "predictions-"+condition+".json"), predictions)
			resources[condition] = map[string]any{"inference_nanoseconds": costs, "checkpoint_recovery_nanoseconds": recovery, "memory": s2r3Memory(), "inventory": s2r3Inventory(t, r)}
			if name == "C" || name == "F" {
				utilityRows := []s2r4Row{}
				for _, row := range rows {
					if row.Step < 16 {
						utilityRows = append(utilityRows, row)
					}
				}
				if condition == "persistent" {
					s2r6CausalRows(t, r, utilityRows, pop, filepath.Join(dir, "eval-seen"), 181900037, false)
					s2r6CausalRows(t, r, utilityRows, pop, filepath.Join(dir, "eval-unseen"), 181900037, true)
				}
				if condition == "shifted" {
					s2r6CausalRows(t, r, utilityRows, pop, filepath.Join(dir, "eval-shifted"), 181910039, true)
				}
			}
		}
		s1Write(t, filepath.Join(dir, "resources.json"), resources)
		t.Log("evaluated", seed, name)
	}
}

func TestS2R6Persistence(t *testing.T) {
	root, work, pop := s2r6Paths(t)
	cp, wal := s2r3Source(root)
	seed := s2r6Seed(t)
	prefix, err := os.ReadFile(wal)
	s1Check(t, err)
	source := orderedFixtureClone(t, cp)
	rollback, rw := s2r2Fork(t, cp, wal, filepath.Join(work, fmt.Sprintf("rollback-%d", seed)))
	if !bytes.Equal(continuationBytes(t, source), continuationBytes(t, rollback)) {
		t.Fatal("rollback mismatch")
	}
	s1Check(t, rw.Close())
	results := map[string]any{}
	for _, arm := range s2r6AllArms() {
		dir := filepath.Join(work, fmt.Sprintf("training/seed-%d/%s", seed, arm))
		path, journal := filepath.Join(dir, "final.json"), filepath.Join(dir, "events.jsonl")
		r, w := s1Open(t, path, journal)
		expected := continuationBytes(t, r)
		start := time.Now()
		replay, err := RebuildRuntimeFromCheckpointAndJournal(w, cp)
		s1Check(t, err)
		elapsed := time.Since(start).Nanoseconds()
		if !bytes.Equal(expected, continuationBytes(t, replay)) {
			t.Fatal("full suffix mismatch", seed, arm)
		}
		canonical, err := r.brain.CanonicalJSON()
		s1Check(t, err)
		canonical2, err := replay.brain.CanonicalJSON()
		s1Check(t, err)
		if !bytes.Equal(canonical, canonical2) {
			t.Fatal("signed N reconstruction")
		}
		raw, err := os.ReadFile(journal)
		s1Check(t, err)
		if !bytes.HasPrefix(raw, prefix) {
			t.Fatal("WAL truncation")
		}
		var rows []s2r4Row
		s1Read(t, filepath.Join(work, "holdout/persistent/rows.json"), &rows)
		var saved map[string][]float64
		s1Read(t, filepath.Join(dir, "predictions-persistent.json"), &saved)
		mask := arm == "A0" || arm == "B0" || arm == "BI0"
		p, _ := s2r4Predict(t, r, rows, pop, s2r4Arm{NoHistory: mask}, "full", nil)
		if !reflect.DeepEqual(p, saved["full"]) {
			t.Fatal("restart forecast mismatch")
		}
		condition := "persistent"
		if arm == "BI" || arm == "BI0" {
			condition = "independent"
		}
		control := orderedFixtureClone(t, filepath.Join(work, fmt.Sprintf("training/seed-%d/observer-%s/observer-final.json", seed, condition)))
		for control.LastSequence() < r.LastSequence() {
			id := fmt.Sprintf("r6-oracle-control-%d", control.LastSequence())
			s2r4Command(t, control, NativeControlCommand{ID: id + "-begin", Begin: &NativeBegin{EpisodeID: id, Observation: &NativeObservation{State: flexCState(pop, 0, .4), Source: "r6-retention-control", SampleID: id}}})
			action := 0
			s2r4Command(t, control, NativeControlCommand{ID: id + "-propagate", Step: &NativeStep{EpisodeID: id, ActionOverride: &action}})
			action = 1
			step := s2r4Command(t, control, NativeControlCommand{ID: id + "-stop", Step: &NativeStep{EpisodeID: id, ActionOverride: &action}})
			s2r4Command(t, control, NativeControlCommand{ID: id + "-close", Outcome: &NativeOutcome{EpisodeID: id, TraceDigest: step.TraceDigest, Source: "r6-retention-control", OutcomeID: id, Provenance: "observed-sensor", Learn: false}})
		}
		controlPath := filepath.Join(dir, "retention-control.json")
		s2r3SaveCP(t, control, controlPath)
		controlImage := s2r4OracleImage(t, controlPath, filepath.Join(dir, "oracle-control-evaluation-only.json"))
		image := s2r4OracleImage(t, path, filepath.Join(dir, "oracle-candidate-evaluation-only.json"))
		ae := s2r3AE1(t, controlImage, image, pop)
		s2r3SaveCP(t, r, filepath.Join(dir, "restarted.json"))
		s1Check(t, w.Close())
		results[arm] = map[string]any{"restart_predictions_exact": true, "full_suffix_replay_exact": true, "signed_parameter_reconstruction_exact": true, "canonical_state_exact": true, "historical_WAL_prefix_exact": true, "original_rollback_exact": true, "replay_nanoseconds": elapsed, "suffix_bytes": len(raw) - len(prefix), "final_sequence": r.LastSequence(), "AE1": ae, "inventory": s2r3Inventory(t, r)}
		s1Write(t, filepath.Join(work, fmt.Sprintf("persistence-%d.json", seed)), results)
		t.Log("persisted", seed, arm)
	}
}
