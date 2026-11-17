//go:build horizon_s2r5 && horizon_s2r4 && horizon_s2r3 && horizon_s2r2

package runtime

import (
	"bytes"
	"fmt"
	k "github.com/project-horizon/horizon-core/services/ai/knowledge"
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"strconv"
	"testing"
	"time"
)

var s2r5Seeds = []int{37, 83, 151, 211, 307, 419}

func s2r5Paths(t *testing.T) (string, string, [][]k.NodeID) {
	root, work := os.Getenv("HORIZON_S2R5_SOURCE"), os.Getenv("HORIZON_S2R5_WORK")
	if root == "" || work == "" {
		t.Skip("isolated authenticated R5 research only")
	}
	if root == work || filepath.Dir(work) == root {
		t.Fatal("overlapping source")
	}
	return root, work, s1Pop(t, filepath.Join(root, "source/source/source"))
}
func TestS2R5Prepare(t *testing.T) {
	root, work, pop := s2r5Paths(t)
	cp, wal := s2r3Source(root)
	s1Check(t, os.MkdirAll(work, 0700))
	snap := s1Snapshot(t, cp)
	sum, e := checkpointChecksumForSchema(snap)
	s1Check(t, e)
	if snap.Sequence != 80735 || sum != s2r3Checksum || snap.Checksum != sum {
		t.Fatal("BLOCKED source checksum")
	}
	r, w := s2r2Fork(t, cp, wal, filepath.Join(work, "source-verification"))
	defer w.Close()
	inv := s2r3Inventory(t, r)
	if inv["neurons"] != 274 || inv["synapses"] != 6408 || inv["features"] != 32 || inv["native_policy_updates"] != uint64(10752) {
		t.Logf("inventory %#v", inv)
		if fmt.Sprint(inv["native_policy_updates"]) != "10752" {
			t.Fatal("inventory mismatch")
		}
	}
	if r.LastSequence() != 80735 || r.activation.Memory != r.brain || r.learning.Kb != r.brain || r.dnf.Brain() != r.brain {
		t.Fatal("source invariants")
	}
	earlier := filepath.Join(root, "s2r2/s2r2-final.json")
	replay, e := RebuildRuntimeFromCheckpointAndJournal(w, earlier)
	s1Check(t, e)
	if !bytes.Equal(continuationBytes(t, r), continuationBytes(t, replay)) {
		t.Fatal("historical 80681 to 80735 replay")
	}
	s2r2Copy(t, cp, filepath.Join(work, "00-source.json"))
	s2r2Copy(t, wal, filepath.Join(work, "source-events.jsonl"))
	s1Write(t, filepath.Join(work, "SOURCE_VERIFICATION.json"), map[string]any{"sequence": 80735, "native_checksum": sum, "checkpoint_sha256": s2r3FileHash(t, cp), "journal_sha256": s2r3FileHash(t, wal), "inventory": inv, "native_restore_exact": true, "historical_suffix_replay_exact": true, "single_owner": true})
	// Development evidence, never R4 final holdout, for independent calibration audit.
	r4 := os.Getenv("HORIZON_S2R5_R4")
	for _, condition := range []string{"persistent", "independent"} {
		var rows []s2r4Row
		s1Read(t, filepath.Join(r4, "s2r4/development", condition, "rows.json"), &rows)
		out := map[string]any{"rows": rows, "source": s2Predictions(t, r, s2r4Trials(rows), pop)}
		for _, seed := range []int{37, 83, 151} {
			a := orderedFixtureClone(t, filepath.Join(r4, fmt.Sprintf("s2r4/training/seed-%d/A32/final.json", seed)))
			out[fmt.Sprint(seed)] = s2Predictions(t, a, s2r4Trials(rows), pop)
		}
		s1Write(t, filepath.Join(work, "calibration-r4-"+condition+".json"), out)
	}
	// Snapshot all original anchors and feature/synapse ownership.
	s1Write(t, filepath.Join(work, "ORIGINAL_WITNESSES.json"), r.brain.BrainState.NResolution.Witnesses)
	if !reflect.DeepEqual(r.brain.NativeControlSnapshot(), replay.brain.NativeControlSnapshot()) {
		t.Fatal("policy replay")
	}
}

type s2r5Arm struct {
	OnlyBlocks                          [][]int
	Name                                string
	Method                              string
	Capacity                            int
	Mask, IID, Refine, Blocks, Coverage bool
}

func s2r5Lesson(t *testing.T, r *BrainRuntime, arm s2r5Arm, id string, fit, check []s2r4Row, pop [][]k.NodeID) (*CompositionLearningReceipt, map[string]any) {
	if arm.Method == "original" && !arm.Blocks {
		return s2r4Lesson(t, r, s2r4Arm{Name: arm.Name, Capacity: 32, NoHistory: arm.Mask, Repair: arm.Refine}, id, fit, check, pop)
	}
	e := s2Lesson(1, s2r4Trials(fit), s2r4Trials(check), pop, s2r4Now(r))
	e.ID = id
	e.EpisodeID = id
	e.IndependenceGroup = id
	e.Source = "s2r5-actual-native-residual"
	e.Resolution.Proposal = &k.NMetricProposalConfig{CoordinateBlocks: arm.OnlyBlocks, Protocol: k.NMetricConstrainedV1, Strategy: "full-refit", PreserveShared: true, FeatureLimit: arm.Capacity, Update: arm.Method, FixedBasis: !arm.Refine && arm.Capacity == 32, RefineLeafDirections: arm.Refine, BlockSparse: arm.Blocks}
	if arm.Method == "original" {
		e.Resolution.Proposal.Protocol = k.NMetricBlockV1
		e.Resolution.Proposal.Update = ""
		e.Resolution.Proposal.FixedBasis = false
		e.Resolution.Proposal.Strategy = "combined"
		e.Resolution.Proposal.Fractions = append([]float64(nil), s2r2Fractions...)
	}
	if arm.Coverage {
		_, work, _ := s2r5Paths(t)
		var anchors []s2Trial
		s1Read(t, filepath.Join(work, "anchors/rows.json"), &anchors)
		for _, row := range anchors {
			e.Resolution.Retention = append(e.Resolution.Retention, k.NObservation{Input: s2Input(row.Pre, pop), Outcome: row.Error, ObservedAt: row.AfterAt, EpisodeID: row.ID})
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
	before := s2r3Memory()
	start := time.Now()
	_, receipt, err := r.LearnCompositionWithDiagnosis(e, s2r4Now(r))
	s1Check(t, err)
	return receipt, map[string]any{"wall_nanoseconds": time.Since(start).Nanoseconds(), "memory_before": before, "memory_after": s2r3Memory(), "inventory": s2r3Inventory(t, r), "trials": len(receipt.Diagnosis.Trials)}
}
func s2r5ReplayObserver(t *testing.T, r *BrainRuntime, path string) {
	events, e := ReadEventLog(path)
	s1Check(t, e)
	for _, ev := range events {
		if ev.Sequence > 80735 {
			if ev.NativeControl == nil {
				t.Fatal("non native observer")
			}
			_, e = r.ApplyNativeControl(*ev.NativeControl, ev.Timestamp)
			s1Check(t, e)
		}
	}
}
func TestS2R5Development(t *testing.T) {
	root, work, pop := s2r5Paths(t)
	cp, wal := s2r3Source(root)
	for _, cond := range []string{"persistent", "independent"} {
		dir := filepath.Join(work, "development", cond)
		r, w := s2r2Fork(t, cp, wal, dir)
		s2r4Collect(t, r, pop, dir, cond, 5100011, 24, 16, false)
		s1Check(t, w.Close())
	}
}
func TestS2R5FixedDevelopment(t *testing.T) {
	root, work, pop := s2r5Paths(t)
	cp, wal := s2r3Source(root)
	var rows []s2r4Row
	s1Read(t, filepath.Join(work, "development/persistent/rows.json"), &rows)
	arms := []s2r5Arm{{Name: "A", Method: "original", Capacity: 32}, {Name: "B", Method: "bias", Capacity: 32}, {Name: "Baff", Method: "affine", Capacity: 32}, {Name: "C", Method: "readout", Capacity: 32}, {Name: "C0", Method: "readout", Capacity: 32, Mask: true}}
	source := orderedFixtureClone(t, cp)
	var old []s2Trial
	s1Read(t, filepath.Join(root, "s2r2/holdout.json"), &old)
	for _, arm := range arms {
		dir := filepath.Join(work, "fixed-development-v2", arm.Name)
		r, w := s2r2Fork(t, cp, wal, dir)
		s2r5ReplayObserver(t, r, filepath.Join(work, "development/persistent/events.jsonl"))
		receipts := []*CompositionLearningReceipt{}
		costs := []any{}
		for b := 0; b < 4; b++ {
			fit, check := s2r4Subset(rows, 2*b, 2*b+2), s2r4Subset(rows, 8+2*b, 10+2*b)
			rec, cost := s2r5Lesson(t, r, arm, fmt.Sprintf("r5-dev-%s-%d", arm.Name, b), fit, check, pop)
			receipts = append(receipts, rec)
			costs = append(costs, cost)
			s1Write(t, filepath.Join(dir, "receipts.json"), receipts)
			s1Write(t, filepath.Join(dir, "costs.json"), costs)
			t.Log(arm.Name, b, rec.Diagnosis.Committed, rec.Diagnosis.Cause)
		}
		s2r3SaveCP(t, r, filepath.Join(dir, "final.json"))
		eval := s2r4Subset(rows, 16, 24)
		pred, _ := s2r4Predict(t, r, eval, pop, s2r4Arm{NoHistory: arm.Mask}, "full", nil)
		mask, _ := s2r4Predict(t, r, eval, pop, s2r4Arm{}, "no-history", nil)
		s1Write(t, filepath.Join(dir, "development-evaluation.json"), map[string]any{"rows": eval, "predictions": pred, "masked": mask, "retention": s2r4Retention(t, source, r, old, pop)})
		s1Check(t, w.Close())
	}
}

func TestS2R5Anchors(t *testing.T) {
	root, work, pop := s2r5Paths(t)
	cp, wal := s2r3Source(root)
	dir := filepath.Join(work, "anchors")
	r, w := s2r2Fork(t, cp, wal, dir)
	defer w.Close()
	t.Setenv("HORIZON_S2_SOURCE", root)
	t.Setenv("HORIZON_S2_WORK", dir)
	rows := s2r5OldCollect(t, r, pop, 5200011, 64, false, nil, -1)
	s1Write(t, filepath.Join(dir, "rows.json"), rows)
	s2r3SaveCP(t, r, filepath.Join(dir, "observer-final.json"))
}
func TestS2R5CoverageDevelopment(t *testing.T) {
	root, work, pop := s2r5Paths(t)
	cp, wal := s2r3Source(root)
	var rows []s2r4Row
	s1Read(t, filepath.Join(work, "development/persistent/rows.json"), &rows)
	arms := []s2r5Arm{{Name: "Cw", Method: "readout", Capacity: 32, Coverage: true}, {Name: "Cw0", Method: "readout", Capacity: 32, Coverage: true, Mask: true}}
	s2r5DevelopArms(t, root, work, cp, wal, pop, rows, arms, "coverage-development")
}
func s2r5DevelopArms(t *testing.T, root, work, cp, wal string, pop [][]k.NodeID, rows []s2r4Row, arms []s2r5Arm, stage string) {
	source := orderedFixtureClone(t, cp)
	var old []s2Trial
	s1Read(t, filepath.Join(root, "s2r2/holdout.json"), &old)
	for _, arm := range arms {
		dir := filepath.Join(work, stage, arm.Name)
		r, w := s2r2Fork(t, cp, wal, dir)
		s2r5ReplayObserver(t, r, filepath.Join(work, "development/persistent/events.jsonl"))
		receipts := []*CompositionLearningReceipt{}
		costs := []any{}
		for b := 0; b < 4; b++ {
			rec, cost := s2r5Lesson(t, r, arm, fmt.Sprintf("r5-%s-%s-%d", stage, arm.Name, b), s2r4Subset(rows, 2*b, 2*b+2), s2r4Subset(rows, 8+2*b, 10+2*b), pop)
			receipts = append(receipts, rec)
			costs = append(costs, cost)
			s1Write(t, filepath.Join(dir, "receipts.json"), receipts)
			s1Write(t, filepath.Join(dir, "costs.json"), costs)
			t.Log(stage, arm.Name, b, rec.Diagnosis.Committed, rec.Diagnosis.Cause)
		}
		s2r3SaveCP(t, r, filepath.Join(dir, "final.json"))
		eval := s2r4Subset(rows, 16, 24)
		pred, _ := s2r4Predict(t, r, eval, pop, s2r4Arm{NoHistory: arm.Mask}, "full", nil)
		mask, _ := s2r4Predict(t, r, eval, pop, s2r4Arm{}, "no-history", nil)
		s1Write(t, filepath.Join(dir, "development-evaluation.json"), map[string]any{"rows": eval, "predictions": pred, "masked": mask, "retention": s2r4Retention(t, source, r, old, pop)})
		s1Check(t, w.Close())
	}
}
func TestS2R5StructureDevelopment(t *testing.T) {
	root, work, pop := s2r5Paths(t)
	cp, wal := s2r3Source(root)
	var rows []s2r4Row
	s1Read(t, filepath.Join(work, "development/persistent/rows.json"), &rows)
	arms := []s2r5Arm{{Name: "Full32", Method: "original", Capacity: 32, Refine: true}, {Name: "Block32", Method: "original", Capacity: 32, Refine: true, Blocks: true}, {Name: "D", Method: "readout", Capacity: 32, Refine: true, Blocks: true, Coverage: true}, {Name: "D0", Method: "readout", Capacity: 32, Refine: true, Blocks: true, Coverage: true, Mask: true}}
	s2r5DevelopArms(t, root, work, cp, wal, pop, rows, arms, "structure-development")
}
func TestS2R5CapacityDevelopment(t *testing.T) {
	root, work, pop := s2r5Paths(t)
	cp, wal := s2r3Source(root)
	var rows []s2r4Row
	s1Read(t, filepath.Join(work, "development/persistent/rows.json"), &rows)
	arms := []s2r5Arm{{Name: "E", Method: "readout", Capacity: 64, Blocks: true, Coverage: true}, {Name: "E0", Method: "readout", Capacity: 64, Blocks: true, Coverage: true, Mask: true}}
	s2r5DevelopArms(t, root, work, cp, wal, pop, rows, arms, "capacity-development")
}

func TestS2R5NegativeBlockDevelopment(t *testing.T) {
	root, work, pop := s2r5Paths(t)
	cp, wal := s2r3Source(root)
	var rows []s2r4Row
	s1Read(t, filepath.Join(work, "development/persistent/rows.json"), &rows)
	arms := []s2r5Arm{{Name: "ZeroBlock", Method: "readout", Capacity: 32, Refine: true, Blocks: true, Coverage: true, OnlyBlocks: [][]int{{65, 66, 67}}}}
	s2r5DevelopArms(t, root, work, cp, wal, pop, rows, arms, "negative-block-development")
}

func s2r5Arms() []s2r5Arm {
	base := []s2r5Arm{{Name: "A", Method: "original", Capacity: 32}, {Name: "B", Method: "bias", Capacity: 32}, {Name: "C", Method: "readout", Capacity: 32}, {Name: "Cw", Method: "readout", Capacity: 32, Coverage: true}, {Name: "D", Method: "readout", Capacity: 32, Refine: true, Blocks: true, Coverage: true}, {Name: "E", Method: "readout", Capacity: 64, Blocks: true, Coverage: true}}
	out := []s2r5Arm{}
	for _, arm := range base {
		out = append(out, arm)
		arm.Name += "0"
		arm.Mask = true
		out = append(out, arm)
	}
	for _, arm := range base {
		if arm.Name != "A" && arm.Name != "D" && arm.Name != "E" {
			continue
		}
		arm.IID = true
		arm.Name += "I"
		out = append(out, arm)
		arm.Name += "0"
		arm.Mask = true
		out = append(out, arm)
	}
	return out
}
func s2r5CompatibilityArms() []s2r4Arm {
	out := []s2r4Arm{}
	for _, a := range s2r5Arms() {
		out = append(out, s2r4Arm{Name: a.Name, Capacity: a.Capacity, NoHistory: a.Mask, Independent: a.IID})
	}
	return out
}
func TestS2R5Train(t *testing.T) {
	root, work, pop := s2r5Paths(t)
	cp, wal := s2r3Source(root)
	seed, e := strconv.Atoi(os.Getenv("HORIZON_S2R5_SEED"))
	s1Check(t, e)
	if !slices.Contains(s2r5Seeds, seed) {
		t.Fatal("unregistered seed")
	}
	if _, e := os.Stat(filepath.Join(work, "TRAINING_LOCK.json")); e != nil {
		t.Fatal("training lock missing")
	}
	dir := filepath.Join(work, fmt.Sprintf("training/seed-%d", seed))
	s1Check(t, os.MkdirAll(dir, 0700))
	observed := map[bool][]s2r4Row{}
	for _, iid := range []bool{false, true} {
		cond := "persistent"
		if iid {
			cond = "independent"
		}
		od := filepath.Join(dir, "observer-"+cond)
		r, w := s2r2Fork(t, cp, wal, od)
		observed[iid] = s2r4Collect(t, r, pop, od, cond, int64(6100000+1000*seed), 16, 16, false)
		s1Check(t, w.Close())
	}
	for _, arm := range s2r5Arms() {
		ad := filepath.Join(dir, arm.Name)
		r, w := s2r2Fork(t, cp, wal, ad)
		cond := "persistent"
		if arm.IID {
			cond = "independent"
		}
		s2r5ReplayObserver(t, r, filepath.Join(dir, "observer-"+cond+"/events.jsonl"))
		rows := observed[arm.IID]
		receipts := []*CompositionLearningReceipt{}
		costs := []any{}
		for b := 0; b < 4; b++ {
			fit, check := s2r4Subset(rows, 4*b, 4*b+2), s2r4Subset(rows, 4*b+2, 4*b+4)
			s1Write(t, filepath.Join(ad, fmt.Sprintf("block-%d-fit.json", b)), fit)
			s1Write(t, filepath.Join(ad, fmt.Sprintf("block-%d-check.json", b)), check)
			rec, cost := s2r5Lesson(t, r, arm, fmt.Sprintf("r5-s%d-%s-b%d", seed, arm.Name, b), fit, check, pop)
			receipts = append(receipts, rec)
			costs = append(costs, cost)
			s1Write(t, filepath.Join(ad, "receipts.json"), receipts)
			s1Write(t, filepath.Join(ad, "costs.json"), costs)
			t.Log(seed, arm.Name, b, rec.Diagnosis.Committed, rec.Diagnosis.Cause)
		}
		s2r3SaveCP(t, r, filepath.Join(ad, "final.json"))
		s1Check(t, w.Close())
	}
}
func TestS2R5AcquireFinal(t *testing.T) {
	root, work, pop := s2r5Paths(t)
	s2r4CheckLock(t, work)
	cp, wal := s2r3Source(root)
	f, e := os.OpenFile(filepath.Join(work, "FINAL_STARTED"), os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
	s1Check(t, e)
	s1Check(t, f.Close())
	for i, condition := range []string{"persistent", "independent", "shifted"} {
		dir := filepath.Join(work, "holdout", condition)
		r, w := s2r2Fork(t, cp, wal, dir)
		cond := condition
		if cond == "shifted" {
			cond = "persistent"
		}
		seed := []int64{8100017, 8200031, 8300043}[i]
		s2r4Collect(t, r, pop, dir, cond, seed, 32, 24, condition == "shifted")
		s1Check(t, w.Close())
		t.Log("acquired", condition)
	}
	for i, seed := range []int64{8900003, 9000007} {
		dir := filepath.Join(work, fmt.Sprintf("new-retention-%d", i))
		r, w := s2r2Fork(t, cp, wal, dir)
		t.Setenv("HORIZON_S2_SOURCE", root)
		t.Setenv("HORIZON_S2_WORK", dir)
		rows := s2r5OldCollect(t, r, pop, seed, 128, i == 1, nil, -1)
		s1Write(t, filepath.Join(dir, "rows.json"), rows)
		s2r3SaveCP(t, r, filepath.Join(dir, "observer-final.json"))
		s1Check(t, w.Close())
	}
}
func TestS2R5RecoveryPreflight(t *testing.T) {
	_, work, _ := s2r5Paths(t)
	out := map[string]any{}
	for _, seed := range s2r5Seeds {
		for _, a := range s2r5Arms() {
			path := filepath.Join(work, fmt.Sprintf("training/seed-%d/%s/final.json", seed, a.Name))
			r := orderedFixtureClone(t, path)
			s := s1Snapshot(t, path)
			sum, e := checkpointChecksumForSchema(s)
			s1Check(t, e)
			if sum != s.Checksum {
				t.Fatal("checkpoint checksum")
			}
			out[fmt.Sprintf("s%d-%s", seed, a.Name)] = s2r3Inventory(t, r)
		}
	}
	s1Write(t, filepath.Join(work, "RECOVERY_PREFLIGHT.json"), out)
}
