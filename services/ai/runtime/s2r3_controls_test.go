//go:build horizon_s2r3 && horizon_s2r2

package runtime

import (
	"bytes"
	"encoding/json"
	"fmt"
	"math"
	"math/rand"
	"os"
	"path/filepath"
	"runtime"
	"runtime/pprof"
	"sort"
	"strconv"
	"testing"
	"time"

	k "github.com/project-horizon/horizon-core/services/ai/knowledge"
)

func TestS2R3HistoricalReplay(t *testing.T) {
	root, work, _ := s2r3Paths(t)
	dir := filepath.Join(work, "historical-replay")
	r, w := s2r2Fork(t, filepath.Join(root, "s2r2/00-source.json"), filepath.Join(root, "s2r2/s2r2-final-events.jsonl"), dir)
	defer w.Close()
	expected := orderedFixtureClone(t, filepath.Join(root, "s2r2/s2r2-final.json"))
	if !bytes.Equal(continuationBytes(t, r), continuationBytes(t, expected)) {
		t.Fatal("historical default R2 replay changed")
	}
	s1Write(t, filepath.Join(work, "historical-compatibility.json"), map[string]any{"source_sequence": 76617, "recovered_sequence": r.LastSequence(), "complete_state_exact": true, "new_experiences": 0, "new_learning_commands": 0})
}

func s2r3ControlRows(template s2Trial, seed int64, n int, positive bool) []s2Trial {
	u := rand.New(rand.NewSource(seed))
	v := rand.New(rand.NewSource(seed + 9000000))
	out := []s2Trial{}
	for i := 0; i < n; i++ {
		p := template.Pre
		signal := .01 + .08*u.Float64()
		p.History = []float64{signal, .5 * signal, .25 * signal}
		p.HistoryAction = 0
		y := .01 + .5*signal
		if !positive {
			y = .01 + .5*(.01+.08*v.Float64())
		}
		out = append(out, s2Trial{ID: fmt.Sprintf("control-%v-%d-%d", positive, seed, i), Pre: p, AfterAt: p.At, Error: y})
	}
	return out
}
func s2r3ControlInput(row s2Trial, pop [][]k.NodeID, positive bool) k.CompositionInput {
	in := s2Input(row.Pre, pop)
	word := make([]float64, 16)
	q := 4
	if !positive {
		q = 5
	}
	word[q] = 1
	word[15] = .125
	in.Words = [][]float64{word}
	return in
}
func s2r3ControlPredictions(t *testing.T, r *BrainRuntime, rows []s2Trial, pop [][]k.NodeID, positive bool, mode string) []float64 {
	rng := rand.New(rand.NewSource(1300007))
	donors := rng.Perm(len(rows))
	out := []float64{}
	for i, row := range rows {
		in := s2r3ControlInput(row, pop, positive)
		if mode == "no-history" {
			for j := 3; j < 6; j++ {
				in.Sequence[1][j] = 0
			}
		}
		if mode == "permute-history" {
			copy(in.Sequence[1][3:6], s2r3ControlInput(rows[donors[i]], pop, positive).Sequence[1][3:6])
		}
		o, e := r.ReadComposition(in, in.Scene.ObservedAt)
		s1Check(t, e)
		value := 0.
		if o.NResolution != nil {
			value = o.NResolution.Value
		}
		out = append(out, value)
	}
	return out
}
func TestS2R3Controls(t *testing.T) {
	root, work, pop := s2r3Paths(t)
	cp, wal := s2r3Source(root)
	var dev []s2Trial
	s1Read(t, filepath.Join(work, "development-0-fit.json"), &dev)
	dir := filepath.Join(work, "information-controls")
	s1Check(t, os.Mkdir(dir, 0700))
	results := map[string]any{}
	for _, seed := range s2r3Seeds {
		for _, positive := range []bool{true, false} {
			for _, cap := range []int{32, 64} {
				name := fmt.Sprintf("s%d-positive%v-c%d", seed, positive, cap)
				arm := filepath.Join(dir, name)
				r, w := s2r2Fork(t, cp, wal, arm)
				receipts := []*CompositionLearningReceipt{}
				progress := []map[string]any{}
				mean, count := 0., 0
				for block := 0; block < 4; block++ {
					fit := s2r3ControlRows(dev[0], int64(1200000+1000*seed+100*block+1), 64, positive)
					check := s2r3ControlRows(dev[0], int64(1200000+1000*seed+100*block+7), 64, positive)
					for _, row := range fit {
						mean += row.Error
						count++
					}
					s1Write(t, filepath.Join(arm, fmt.Sprintf("block-%d-fit.json", block)), fit)
					s1Write(t, filepath.Join(arm, fmt.Sprintf("block-%d-check.json", block)), check)
					e := s2Lesson(1, fit, check, pop, s1Time(r))
					id := fmt.Sprintf("s2r3-control-%d-%v-%d", seed, positive, block)
					e.ID = id
					e.EpisodeID = id
					e.IndependenceGroup = id
					e.Source = "isolated-synthetic-temporal-control"
					for i, row := range fit {
						e.Resolution.Fit[i].Input = s2r3ControlInput(row, pop, positive)
					}
					for i, row := range check {
						e.Resolution.Check[i].Input = s2r3ControlInput(row, pop, positive)
					}
					e.Input = e.Resolution.Fit[0].Input
					e.Resolution.Proposal = &k.NMetricProposalConfig{Protocol: k.NMetricBoundedV1, Strategy: "combined", Fractions: append([]float64(nil), s2r2Fractions...), PreserveShared: true, FeatureLimit: cap}
					start := time.Now()
					_, receipt, err := r.LearnCompositionWithDiagnosis(e, s1Time(r))
					elapsed := time.Since(start).Nanoseconds()
					s1Check(t, err)
					receipts = append(receipts, receipt)
					progress = append(progress, map[string]any{"block": block, "nanoseconds": elapsed, "committed": receipt.Diagnosis.Committed, "features": len(r.brain.BrainState.NResolution.Features)})
					s2r3SaveCP(t, r, filepath.Join(arm, fmt.Sprintf("block-%d.json", block)))
				}
				s1Write(t, filepath.Join(arm, "receipts.json"), receipts)
				s1Write(t, filepath.Join(arm, "progress.json"), progress)
				rows := s2r3ControlRows(dev[0], int64(1200000+1000*seed+901), 512, positive)
				s1Write(t, filepath.Join(arm, "test.json"), rows)
				pred := map[string][]float64{"full": s2r3ControlPredictions(t, r, rows, pop, positive, "full"), "no-history": s2r3ControlPredictions(t, r, rows, pop, positive, "no-history"), "permute-history": s2r3ControlPredictions(t, r, rows, pop, positive, "permute-history"), "constant": s2Constant(rows, mean/float64(count))}
				s1Write(t, filepath.Join(arm, "predictions.json"), pred)
				metrics := map[string]any{}
				p := s2r3Metric(t, work)
				for name, v := range pred {
					metrics[name] = s2MetricOf(rows, v, p)
				}
				s2r3SaveCP(t, r, filepath.Join(arm, "final.json"))
				s1Check(t, w.Close())
				results[name] = map[string]any{"synthetic_only": true, "capacity": cap, "seed": seed, "positive": positive, "metrics": metrics, "features": len(r.brain.BrainState.NResolution.Features)}
				s1Write(t, filepath.Join(work, "information-controls.json"), results)
				t.Log("isolated control", name, metrics)
			}
		}
	}
}

func TestS2R3Cost(t *testing.T) {
	_, work, pop := s2r3Paths(t)
	seed, e := strconv.Atoi(os.Getenv("HORIZON_S2R3_SEED"))
	s1Check(t, e)
	cap, e := strconv.Atoi(os.Getenv("HORIZON_S2R3_CAP"))
	s1Check(t, e)
	if (seed != 37 && seed != 83 && seed != 151) || (cap != 32 && cap != 64) {
		t.Fatal("unregistered cost arm")
	}
	dir := filepath.Join(work, fmt.Sprintf("seed-%d/arm-%d", seed, cap))
	cp, journal := filepath.Join(dir, "final.json"), filepath.Join(dir, "events.jsonl")
	start := time.Now()
	r, w := s1Open(t, cp, journal)
	defer w.Close()
	recovery := time.Since(start).Nanoseconds()
	var rows []s2Trial
	s1Read(t, filepath.Join(work, "holdout.json"), &rows)
	inputs := s2r3Inputs(rows, pop)
	for i := 0; i < 16; i++ {
		_, e := r.ReadComposition(inputs[i], inputs[i].Scene.ObservedAt)
		s1Check(t, e)
	}
	runtime.GC()
	baseline := s2r3Memory()
	durations := []int64{}
	var before, after runtime.MemStats
	runtime.ReadMemStats(&before)
	for _, in := range inputs {
		start := time.Now()
		o, e := r.ReadComposition(in, in.Scene.ObservedAt)
		s1Check(t, e)
		if o.NResolution == nil || math.IsNaN(o.NResolution.Value) {
			t.Fatal("invalid inference")
		}
		durations = append(durations, time.Since(start).Nanoseconds())
	}
	runtime.ReadMemStats(&after)
	sort.Slice(durations, func(i, j int) bool { return durations[i] < durations[j] })
	result := map[string]any{"seed": seed, "capacity": cap, "inventory": s2r3Inventory(t, r), "recovery_nanoseconds": recovery, "inference_p50_nanoseconds": durations[len(durations)/2], "inference_p95_nanoseconds": durations[95*len(durations)/100], "allocated_bytes_per_inference": float64(after.TotalAlloc-before.TotalAlloc) / float64(len(rows)), "allocations_per_inference": float64(after.Mallocs-before.Mallocs) / float64(len(rows)), "post_gc_baseline": baseline, "after_inference": s2r3Memory(), "sample_count": len(rows), "isolated_process": true, "GOMAXPROCS": runtime.GOMAXPROCS(0)}
	s1Write(t, filepath.Join(dir, "cost.json"), result)
	t.Log("measured inference", seed, cap)
}

// Inspect the already recorded control cases, with no additional fitting.
func TestS2R3ControlAudit(t *testing.T) {
	root, work, pop := s2r3Paths(t)
	cp, _ := s2r3Source(root)
	old := s2r3Old(t, cp)
	for _, seed := range s2r3Seeds {
		for _, positive := range []bool{true, false} {
			for _, cap := range []int{32, 64} {
				dir := filepath.Join(work, "information-controls", fmt.Sprintf("s%d-positive%v-c%d", seed, positive, cap))
				r := orderedFixtureClone(t, filepath.Join(dir, "final.json"))
				var rows []s2Trial
				s1Read(t, filepath.Join(dir, "test.json"), &rows)
				inputs := []k.CompositionInput{}
				for _, row := range rows {
					inputs = append(inputs, s2r3ControlInput(row, pop, positive))
				}
				audit, e := r.brain.S2R3Inspect(inputs, old)
				s1Check(t, e)
				s1Write(t, filepath.Join(dir, "representation-audit.json"), audit)
			}
		}
	}
}

// Re-execute an already recorded last-block transaction on a disposable copy.
// This measures the same admitted learning, with no new experience or selection.
func TestS2R3LearningCost(t *testing.T) {
	_, work, pop := s2r3Paths(t)
	seed, e := strconv.Atoi(os.Getenv("HORIZON_S2R3_SEED"))
	s1Check(t, e)
	cap, e := strconv.Atoi(os.Getenv("HORIZON_S2R3_CAP"))
	s1Check(t, e)
	if (seed != 37 && seed != 83 && seed != 151) || (cap != 32 && cap != 64) {
		t.Fatal("unregistered arm")
	}
	base := filepath.Join(work, fmt.Sprintf("seed-%d", seed))
	arm := filepath.Join(base, fmt.Sprintf("arm-%d", cap))
	dir := filepath.Join(arm, "learning-cost-reproduction")
	s1Check(t, os.Mkdir(dir, 0700))
	s2r2Copy(t, filepath.Join(arm, "block-6.json"), filepath.Join(dir, "state.json"))
	at := s1Snapshot(t, filepath.Join(dir, "state.json")).Sequence
	raw, e := os.ReadFile(filepath.Join(arm, "events.jsonl"))
	s1Check(t, e)
	prefix := []byte{}
	for _, line := range bytes.SplitAfter(raw, []byte("\n")) {
		if len(bytes.TrimSpace(line)) == 0 {
			continue
		}
		var ev LoggedEvent
		s1Check(t, json.Unmarshal(line, &ev))
		if ev.Sequence > at {
			break
		}
		prefix = append(prefix, line...)
	}
	s1Check(t, os.WriteFile(filepath.Join(dir, "events.jsonl"), prefix, 0600))
	r, w := s1Open(t, filepath.Join(dir, "state.json"), filepath.Join(dir, "events.jsonl"))
	defer w.Close()
	var commands []LoggedEvent
	s1Read(t, filepath.Join(base, "block-7-observed-commands.json"), &commands)
	for _, ev := range commands {
		_, e := r.ApplyNativeControl(*ev.NativeControl, ev.Timestamp)
		s1Check(t, e)
	}
	var fit, check []s2Trial
	s1Read(t, filepath.Join(base, "block-7-fit.json"), &fit)
	s1Read(t, filepath.Join(base, "block-7-check.json"), &check)
	runtime.GC()
	baseline := s2r3Memory()
	var before, after runtime.MemStats
	runtime.ReadMemStats(&before)
	profile, e := os.Create(filepath.Join(dir, "transaction-cpu.pprof"))
	s1Check(t, e)
	s1Check(t, pprof.StartCPUProfile(profile))
	receipt, cost := s2r3Lesson(t, r, fmt.Sprintf("s2r3-seed-%d-block-%d", seed, 7), cap, fit, check, pop)
	pprof.StopCPUProfile()
	profile.Close()
	runtime.ReadMemStats(&after)
	final := orderedFixtureClone(t, filepath.Join(arm, "final.json"))
	exact := bytes.Equal(continuationBytes(t, r), continuationBytes(t, final))
	if !exact {
		t.Fatal("recorded learning reproduction mismatch")
	}
	s2r3SaveCP(t, r, filepath.Join(dir, "final.json"))
	s1Write(t, filepath.Join(dir, "receipt.json"), receipt)
	s1Write(t, filepath.Join(arm, "learning-cost.json"), map[string]any{"seed": seed, "capacity": cap, "transaction": cost, "post_gc_baseline": baseline, "after": s2r3Memory(), "allocated_bytes": after.TotalAlloc - before.TotalAlloc, "allocations": after.Mallocs - before.Mallocs, "full_state_reproduced_exact": exact, "additional_experiences": 0, "transaction_reexecution_only": true, "inclusive_feature_install_cost": receipt.Diagnosis.Grown != 0})
}
