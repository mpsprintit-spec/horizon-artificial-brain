//go:build horizon_s2r5 && horizon_s2r4 && horizon_s2r3 && horizon_s2r2

package runtime

import (
	"fmt"
	k "github.com/project-horizon/horizon-core/services/ai/knowledge"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"sort"
	"strconv"
	"testing"
	"time"
)

func TestS2R5EvaluateFinal(t *testing.T) {
	root, work, pop := s2r5Paths(t)
	s2r4CheckLock(t, work)
	cp, _ := s2r3Source(root)
	source := orderedFixtureClone(t, cp)
	seed, e := strconv.Atoi(os.Getenv("HORIZON_S2R5_SEED"))
	s1Check(t, e)
	oldFeatures := []k.NodeID{}
	for _, f := range source.brain.BrainState.NResolution.Features {
		oldFeatures = append(oldFeatures, f.Node)
	}
	var oldRows []s2Trial
	s1Read(t, filepath.Join(root, "s2r2/holdout.json"), &oldRows)
	for _, arm := range s2r5Arms() {
		dir := filepath.Join(work, fmt.Sprintf("training/seed-%d/%s", seed, arm.Name))
		start := time.Now()
		r := orderedFixtureClone(t, filepath.Join(dir, "final.json"))
		recovery := time.Since(start).Nanoseconds()
		resources := map[string]any{}
		ret := s2r4Retention(t, source, r, oldRows, pop)
		cand := ret["S2_candidate"].(s2Metric)
		_ = cand
		ret["fixed_external_limit"] = .01766457
		ret["strict_S1_pass"] = ret["S1_max_delta"].(float64) == 0
		ret["fixed_external_pass"] = cand.RMSE <= .01766457
		s1Write(t, filepath.Join(dir, "final-retention.json"), ret)
		extra := []any{}
		for i := 0; i < 2; i++ {
			var rows []s2Trial
			s1Read(t, filepath.Join(work, fmt.Sprintf("new-retention-%d/rows.json", i)), &rows)
			v := s2r4Retention(t, source, r, rows, pop)
			v["strict_S1_pass"] = v["S1_max_delta"].(float64) == 0
			extra = append(extra, v)
		}
		s1Write(t, filepath.Join(dir, "new-retention.json"), extra)
		strictS1 := map[string]any{}
		for _, condition := range []string{"persistent", "independent", "shifted"} {
			hd := filepath.Join(work, "holdout", condition)
			var rows []s2r4Row
			s1Read(t, filepath.Join(hd, "rows.json"), &rows)
			var interventions s2r4Interventions
			s1Read(t, filepath.Join(hd, "input-interventions.json"), &interventions)
			modes := []string{"full", "no-history", "no-forecast", "no-history-operation"}
			keys := []string{}
			for key := range interventions.Donors {
				keys = append(keys, key)
			}
			sort.Strings(keys)
			modes = append(modes, keys...)
			predictions := map[string][]float64{}
			costs := []int64{}
			var before, after runtime.MemStats
			runtime.GC()
			runtime.ReadMemStats(&before)
			for _, mode := range modes {
				p, d := s2r4Predict(t, r, rows, pop, s2r4Arm{NoHistory: arm.Mask}, mode, interventions.Donors[mode])
				predictions[mode] = p
				if mode == "full" {
					costs = d
					runtime.ReadMemStats(&after)
				}
			}
			s1Write(t, filepath.Join(dir, "predictions-"+condition+".json"), predictions)
			sort.Slice(costs, func(i, j int) bool { return costs[i] < costs[j] })
			resources[condition] = map[string]any{"N": len(rows), "inference_p50_nanoseconds": costs[len(costs)/2], "inference_p95_nanoseconds": costs[95*len(costs)/100], "allocated_bytes_per_inference": float64(after.TotalAlloc-before.TotalAlloc) / float64(len(rows)), "allocations_per_inference": float64(after.Mallocs-before.Mallocs) / float64(len(rows)), "recovery_nanoseconds": recovery, "inventory": s2r3Inventory(t, r), "memory": s2r3Memory()}
			trials := s2r4Trials(rows)
			a := s1Predictions(t, source, s2S1Rows(trials), pop)
			b := s1Predictions(t, r, s2S1Rows(trials), pop)
			delta := s1AuditMax(s1AuditValue(t, a), s1AuditValue(t, b))
			strictS1[condition] = map[string]any{"max_delta": delta, "exact": reflect.DeepEqual(a, b), "samples": len(rows)}
			// All circuits, contributions and single-feature interventions on 2 streams.
			inputs := s2r3Inputs(s2r4Trials(s2r4Subset(rows, 0, 2)), pop)
			if arm.Mask {
				for i := range inputs {
					for j := 3; j < 6; j++ {
						inputs[i].Sequence[1][j] = 0
					}
				}
			}
			trace, e := r.brain.S2R3Inspect(inputs, oldFeatures)
			s1Check(t, e)
			s1Write(t, filepath.Join(dir, "feature-trace-"+condition+".json"), trace)
		}
		s1Write(t, filepath.Join(dir, "resources.json"), resources)
		s1Write(t, filepath.Join(dir, "strict-s1.json"), strictS1)
		t.Log("evaluated", seed, arm.Name)
	}
}

func TestS2R5CalibrationControls(t *testing.T) {
	root, work, pop := s2r5Paths(t)
	if _, e := os.Stat(filepath.Join(work, "FINAL_STARTED")); e == nil {
		t.Fatal("controls must precede final")
	}
	cp, _ := s2r3Source(root)
	source := orderedFixtureClone(t, cp)
	var old []s2Trial
	s1Read(t, filepath.Join(root, "s2r2/holdout.json"), &old)
	outdir := filepath.Join(work, "calibration-controls")
	s1Check(t, os.Mkdir(outdir, 0700))
	for _, seed := range s2r5Seeds {
		var rows []s2r4Row
		s1Read(t, filepath.Join(work, fmt.Sprintf("training/seed-%d/observer-persistent/rows.json", seed)), &rows)
		ids := []string{}
		for _, row := range rows {
			ids = append(ids, row.Trial.ID)
		}
		a := orderedFixtureClone(t, filepath.Join(work, fmt.Sprintf("training/seed-%d/A/final.json", seed)))
		out := map[string]any{"rows": rows, "ids": ids, "source": s2Predictions(t, source, s2r4Trials(rows), pop), "A": s2Predictions(t, a, s2r4Trials(rows), pop), "historical_rows": old, "historical_source": s2Predictions(t, source, old, pop), "historical_A": s2Predictions(t, a, old, pop)}
		s1Write(t, filepath.Join(outdir, fmt.Sprintf("seed-%d.json", seed)), out)
	}
}
