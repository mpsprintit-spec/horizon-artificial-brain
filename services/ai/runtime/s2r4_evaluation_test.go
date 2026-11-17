//go:build horizon_s2r4 && horizon_s2r3 && horizon_s2r2

package runtime

import (
	"bytes"
	"fmt"
	k "github.com/project-horizon/horizon-core/services/ai/knowledge"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"sort"
	"testing"
	"time"
)

func s2r4CheckLock(t *testing.T, work string) {
	t.Helper()
	var lock struct{ Files map[string]string }
	s1Read(t, filepath.Join(work, "CONFIRMATORY_LOCK.json"), &lock)
	if len(lock.Files) < 25 {
		t.Fatal("incomplete confirmatory lock")
	}
	for p, want := range lock.Files {
		if s2r3FileHash(t, p) != want {
			t.Fatal("frozen scientific input changed", p)
		}
	}
}

func TestS2R4RecoveryPreflight(t *testing.T) {
	_, work, _ := s2r4Paths(t)
	results := map[string]any{}
	for _, seed := range s2r3Seeds {
		for _, arm := range s2r4Arms {
			path := filepath.Join(work, fmt.Sprintf("training/seed-%d/%s/final.json", seed, arm.Name))
			r := orderedFixtureClone(t, path)
			s := s1Snapshot(t, path)
			sum, e := checkpointChecksumForSchema(s)
			s1Check(t, e)
			if sum != s.Checksum {
				t.Fatal("final native checksum mismatch")
			}
			results[fmt.Sprintf("s%d-%s", seed, arm.Name)] = map[string]any{"restored": true, "native_checksum": sum, "inventory": s2r3Inventory(t, r)}
		}
	}
	s1Write(t, filepath.Join(work, "RECOVERY_PREFLIGHT.json"), results)
}

// The original S2 generator has fixed graph/gain/time and fully observed
// prestate. Reexecute its operations to audit conditional sufficiency. This
// is a mechanism audit, not new training or an ability evaluation.
func TestS2R4HistoricalMechanism(t *testing.T) {
	root, work, pop := s2r4Paths(t)
	cp, wal := s2r3Source(root)
	dir := filepath.Join(work, "historical-mechanism")
	r, w := s2r2Fork(t, cp, wal, dir)
	defer w.Close()
	var rows []s2Trial
	s1Read(t, filepath.Join(root, "s2r2/holdout.json"), &rows)
	covered := map[k.NodeID]bool{}
	for _, g := range pop {
		for _, id := range g {
			covered[id] = true
		}
	}
	maximumActual, maximumError := 0., 0.
	unexpected := []k.NodeID{}
	results := []any{}
	for i, row := range rows {
		for id, v := range row.Pre.Before {
			if v != 0 && !covered[id] {
				unexpected = append(unexpected, id)
			}
		}
		id := fmt.Sprintf("r4-history-mechanism-%d", i)
		s2r4Command(t, r, NativeControlCommand{ID: id + "-begin", Begin: &NativeBegin{EpisodeID: id, Observation: &NativeObservation{State: row.Pre.Before, Source: "historical-mechanism-reexecution", SampleID: id}}})
		action := row.Pre.Action
		step := s2r4Command(t, r, NativeControlCommand{ID: id + "-target", Step: &NativeStep{EpisodeID: id, ActionOverride: &action}})
		actual := s1Target(step.State, pop)
		err := 0.
		delta := 0.
		for q := range actual {
			err += mathAbs(actual[q]-row.Pre.Forecast[q]) / 3
			delta = max(delta, mathAbs(actual[q]-row.Actual[q]))
		}
		maximumActual = max(maximumActual, delta)
		maximumError = max(maximumError, mathAbs(err-row.Error))
		if !step.Stopped {
			action = 1
			step = s2r4Command(t, r, NativeControlCommand{ID: id + "-stop", Step: &NativeStep{EpisodeID: id, ActionOverride: &action}})
		}
		s2r4Command(t, r, NativeControlCommand{ID: id + "-close", Outcome: &NativeOutcome{EpisodeID: id, TraceDigest: step.TraceDigest, Source: "historical-mechanism-reexecution", OutcomeID: id, Provenance: "observed-sensor", Learn: false}})
		results = append(results, map[string]any{"id": row.ID, "actual": actual, "error": err, "actual_delta": delta})
	}
	s1Write(t, filepath.Join(dir, "reexecution.json"), results)
	s1Write(t, filepath.Join(work, "historical-conditional-sufficiency.json"), map[string]any{"stored_cases": len(rows), "reexecuted_actual_operations": len(rows), "max_actual_projection_delta": maximumActual, "max_error_delta": maximumError, "unobserved_active_neurons": unexpected, "interpretation": "In the original fixed-gain/static-graph generator, current complete prestate + operation + S1 forecast determines actual error; history is conditionally redundant at this raw-input level. A lossy learned representation may still benefit from history as a proxy. This is not learned rejection and does not extend to changed hidden dynamics.", "pass": maximumActual <= 1e-14 && maximumError <= 1e-14 && len(unexpected) == 0})
	s2r3SaveCP(t, r, filepath.Join(dir, "final.json"))
	if maximumActual > 1e-14 || maximumError > 1e-14 || len(unexpected) > 0 {
		t.Fatal("historical conditional sufficiency unverified")
	}
}
func mathAbs(x float64) float64 {
	if x < 0 {
		return -x
	}
	return x
}
func TestS2R4AcquireFinal(t *testing.T) {
	root, work, pop := s2r4Paths(t)
	s2r4CheckLock(t, work)
	cp, wal := s2r3Source(root)
	guard, e := os.OpenFile(filepath.Join(work, "FINAL_STARTED"), os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
	s1Check(t, e)
	s1Check(t, guard.Close())
	for i, condition := range []string{"persistent", "independent", "shifted"} {
		dir := filepath.Join(work, "holdout", condition)
		r, w := s2r2Fork(t, cp, wal, dir)
		c := condition
		if c == "shifted" {
			c = "persistent"
		}
		s2r4Collect(t, r, pop, dir, c, 3500037+int64(i*100000), 32, 24, condition == "shifted")
		s1Check(t, w.Close())
		t.Log("fresh locked holdout acquired", condition)
	}
}

type s2r4Interventions struct{ Donors map[string][]int }

func s2r4Predict(t *testing.T, r *BrainRuntime, rows []s2r4Row, pop [][]k.NodeID, arm s2r4Arm, mode string, donors []int) ([]float64, []int64) {
	t.Helper()
	pred := make([]float64, len(rows))
	cost := make([]int64, len(rows))
	for i, row := range rows {
		p := row.Trial.Pre
		if len(donors) > 0 {
			p.History = append([]float64(nil), rows[donors[i]].Trial.Pre.History...)
		}
		in := s2r4EvaluationInput(p, pop, arm.NoHistory || mode == "no-history")
		if mode == "no-forecast" {
			for j := 0; j < 3; j++ {
				in.Sequence[1][j] = 0
			}
		}
		if mode == "no-history-operation" {
			in.Sequence[0][1] = 0
		}
		start := time.Now()
		o, e := r.ReadComposition(in, in.Scene.ObservedAt)
		cost[i] = time.Since(start).Nanoseconds()
		s1Check(t, e)
		if o.NResolution == nil {
			t.Fatal("native readout missing")
		}
		pred[i] = o.NResolution.Value
	}
	return pred, cost
}

func s2r4EvaluationInput(p s2Pre, pop [][]k.NodeID, noHistory bool) k.CompositionInput {
	in := s2Input(p, pop)
	if noHistory {
		for j := 3; j < 6; j++ {
			in.Sequence[1][j] = 0
		}
	}
	return in
}

func TestS2R4MaskPreservesAbsentHistoryContext(t *testing.T) {
	pop := make([][]k.NodeID, 9)
	for i := range pop {
		for j := 0; j < 4; j++ {
			pop[i] = append(pop[i], k.NodeID(1+4*i+j))
		}
	}
	for _, history := range [][]float64{nil, {.1, .2, .3}} {
		p := s2Pre{Before: map[k.NodeID]float64{}, Forecast: []float64{.1, .2, .3}, History: history, HistoryAction: -1, PreviousAction: -1}
		full := s2r4EvaluationInput(p, pop, false)
		masked := s2r4EvaluationInput(p, pop, true)
		if !reflect.DeepEqual(full.Sequence[0], masked.Sequence[0]) {
			t.Fatal("history presence or operation changed")
		}
		for j := 0; j < 3; j++ {
			if full.Sequence[1][j] != masked.Sequence[1][j] {
				t.Fatal("forecast changed")
			}
		}
		if len(history) == 0 && !reflect.DeepEqual(full.Sequence, masked.Sequence) {
			t.Fatal("mask invented history at a stream boundary")
		}
	}
}
func TestS2R4EvaluateFinal(t *testing.T) {
	root, work, pop := s2r4Paths(t)
	s2r4CheckLock(t, work)
	cp, _ := s2r3Source(root)
	old := s2r3Old(t, cp)
	for _, seed := range s2r3Seeds {
		for _, arm := range s2r4Arms {
			dir := filepath.Join(work, fmt.Sprintf("training/seed-%d/%s", seed, arm.Name))
			path := filepath.Join(dir, "final.json")
			start := time.Now()
			r := orderedFixtureClone(t, path)
			recovery := time.Since(start).Nanoseconds()
			results := map[string]any{}
			var oldRows []s2Trial
			s1Read(t, filepath.Join(root, "s2r2/holdout.json"), &oldRows)
			source := orderedFixtureClone(t, cp)
			ret := s2r4Retention(t, source, r, oldRows, pop)
			s1Write(t, filepath.Join(dir, "final-retention.json"), ret)
			for _, condition := range []string{"persistent", "independent", "shifted"} {
				hd := filepath.Join(work, "holdout", condition)
				var rows []s2r4Row
				s1Read(t, filepath.Join(hd, "rows.json"), &rows)
				var interventions s2r4Interventions
				s1Read(t, filepath.Join(hd, "input-interventions.json"), &interventions)
				preds := map[string][]float64{}
				costs := []int64{}
				modes := []string{"full", "no-history", "no-forecast", "no-history-operation"}
				keys := []string{}
				for name := range interventions.Donors {
					keys = append(keys, name)
				}
				sort.Strings(keys)
				modes = append(modes, keys...)
				var before, after runtime.MemStats
				runtime.GC()
				runtime.ReadMemStats(&before)
				for _, mode := range modes {
					p, d := s2r4Predict(t, r, rows, pop, arm, mode, interventions.Donors[mode])
					preds[mode] = p
					if mode == "full" {
						costs = d
						runtime.ReadMemStats(&after)
					}
				}
				s1Write(t, filepath.Join(dir, "predictions-"+condition+".json"), preds)
				sort.Slice(costs, func(i, j int) bool { return costs[i] < costs[j] })
				results[condition] = map[string]any{"N": len(rows), "inference_p50_nanoseconds": costs[len(costs)/2], "inference_p95_nanoseconds": costs[95*len(costs)/100], "allocated_bytes_per_inference": float64(after.TotalAlloc-before.TotalAlloc) / float64(len(rows)), "allocations_per_inference": float64(after.Mallocs-before.Mallocs) / float64(len(rows)), "inventory": s2r3Inventory(t, r), "recovery_nanoseconds": recovery, "memory": s2r3Memory()}
				// Full neural paths and independent single-feature interventions, using
				// the first eight complete streams only to keep trace files manageable.
				inputRows := s2r4Subset(rows, 0, 8)
				ins := s2r3Inputs(s2r4Trials(inputRows), pop)
				if arm.NoHistory {
					for i := range ins {
						for j := 3; j < 6; j++ {
							ins[i].Sequence[1][j] = 0
						}
					}
				}
				audit, e := r.brain.S2R3Inspect(ins, old)
				s1Check(t, e)
				s1Write(t, filepath.Join(dir, "feature-trace-"+condition+".json"), audit)
			}
			s1Write(t, filepath.Join(dir, "resources.json"), results)
			t.Log("native held-out predictions", seed, arm.Name)
		}
	}
}

func TestS2R4Persistence(t *testing.T) {
	root, work, pop := s2r4Paths(t)
	cp, wal := s2r3Source(root)
	prefix, e := os.ReadFile(wal)
	s1Check(t, e)
	filterSeed, filterArm := os.Getenv("HORIZON_S2R4_SEED"), os.Getenv("HORIZON_S2R4_PERSISTENCE_ARM")
	rollbackName, resultPath := "rollback-audit", filepath.Join(work, "persistence.json")
	if filterSeed != "" && filterArm != "" {
		rollbackName += "-" + filterSeed + "-" + filterArm
		s1Check(t, os.MkdirAll(filepath.Join(work, "persistence-individual"), 0700))
		resultPath = filepath.Join(work, "persistence-individual", filterSeed+"-"+filterArm+".json")
	}
	rollback, rw := s2r2Fork(t, cp, wal, filepath.Join(work, rollbackName))
	defer rw.Close()
	source := orderedFixtureClone(t, cp)
	if !bytes.Equal(continuationBytes(t, rollback), continuationBytes(t, source)) {
		t.Fatal("rollback mismatch")
	}
	results := map[string]any{}
	var historical []s2Trial
	s1Read(t, filepath.Join(root, "s2r2/holdout.json"), &historical)
	for _, seed := range s2r3Seeds {
		for _, arm := range s2r4Arms {
			if filterSeed != "" && filterSeed != fmt.Sprint(seed) {
				continue
			}
			if filterArm != "" && filterArm != arm.Name {
				continue
			}
			dir := filepath.Join(work, fmt.Sprintf("training/seed-%d/%s", seed, arm.Name))
			path, journal := filepath.Join(dir, "final.json"), filepath.Join(dir, "events.jsonl")
			r, w := s1Open(t, path, journal)
			expected := continuationBytes(t, r)
			start := time.Now()
			replay, e := RebuildRuntimeFromCheckpointAndJournal(w, cp)
			s1Check(t, e)
			replayNanos := time.Since(start).Nanoseconds()
			exact := bytes.Equal(expected, continuationBytes(t, replay))
			if !exact {
				t.Fatal("full 80735 suffix differs", seed, arm.Name)
			}
			a, e := r.brain.CanonicalJSON()
			s1Check(t, e)
			b, e := replay.brain.CanonicalJSON()
			s1Check(t, e)
			raw, e := os.ReadFile(journal)
			s1Check(t, e)
			prefixExact := bytes.HasPrefix(raw, prefix)
			if !prefixExact {
				t.Fatal("historical WAL prefix changed")
			}
			var rows []s2r4Row
			s1Read(t, filepath.Join(work, "holdout/persistent/rows.json"), &rows)
			var saved map[string][]float64
			s1Read(t, filepath.Join(dir, "predictions-persistent.json"), &saved)
			pred, _ := s2r4Predict(t, r, rows, pop, arm, "full", nil)
			if !reflect.DeepEqual(pred, saved["full"]) {
				t.Fatal("restart predictions differ")
			}
			// Match the evaluation control's logical revision without learning.
			cond := "persistent"
			if arm.Independent {
				cond = "independent"
			}
			controlCP := filepath.Join(work, fmt.Sprintf("training/seed-%d/observer-%s/observer-final.json", seed, cond))
			control := orderedFixtureClone(t, controlCP)
			for control.LastSequence() < r.LastSequence() {
				id := fmt.Sprintf("r4-retention-control-%d", control.LastSequence())
				s2r4Command(t, control, NativeControlCommand{ID: id + "-begin", Begin: &NativeBegin{EpisodeID: id, Observation: &NativeObservation{State: flexCState(pop, 0, .4), Source: "r4-retention-control", SampleID: id}}})
				action := 0
				step := s2r4Command(t, control, NativeControlCommand{ID: id + "-propagate", Step: &NativeStep{EpisodeID: id, ActionOverride: &action}})
				action = 1
				step = s2r4Command(t, control, NativeControlCommand{ID: id + "-stop", Step: &NativeStep{EpisodeID: id, ActionOverride: &action}})
				s2r4Command(t, control, NativeControlCommand{ID: id + "-end", Outcome: &NativeOutcome{EpisodeID: id, TraceDigest: step.TraceDigest, Source: "r4-retention-control", OutcomeID: id, Provenance: "observed-sensor", Learn: false}})
			}
			controlPath := filepath.Join(dir, "retention-control.json")
			s2r3SaveCP(t, control, controlPath)
			// Full neural states/sequence are unchanged; omit only old adapter
			// dedup metadata in detached fresh-probe images, with exact parity
			// against the full oracle verified before this protocol was frozen.
			controlImage := s2r4OracleImage(t, controlPath, filepath.Join(dir, "oracle-control-evaluation-only.json"))
			candidateImage := s2r4OracleImage(t, path, filepath.Join(dir, "oracle-candidate-evaluation-only.json"))
			ae := s2r3AE1(t, controlImage, candidateImage, pop)
			name := fmt.Sprintf("s%d-%s", seed, arm.Name)
			results[name] = map[string]any{"restart_predictions_exact": true, "rollback_exact": true, "full_suffix_replay_exact": exact, "canonical_brain_exact": bytes.Equal(a, b), "serialized_state_exact": exact, "historical_WAL_prefix_exact": prefixExact, "source_sequence": 80735, "final_sequence": r.LastSequence(), "suffix_bytes": len(raw) - len(prefix), "full_suffix_replay_nanoseconds": replayNanos, "AE1": ae, "S1_S2": s2r4Retention(t, source, r, historical, pop), "native_policy_exact": reflect.DeepEqual(source.brain.NativeControlSnapshot(), r.brain.NativeControlSnapshot()), "witnesses_exact": reflect.DeepEqual(source.brain.BrainState.NResolution.Witnesses, r.brain.BrainState.NResolution.Witnesses)}
			s2r3SaveCP(t, r, filepath.Join(dir, "restarted.json"))
			s1Check(t, w.Close())
			s1Write(t, resultPath, results)
			t.Log("exact persistence", name)
		}
	}
}
