//go:build horizon_s2r7 && horizon_s2r6 && horizon_s2r5 && horizon_s2r4 && horizon_s2r3 && horizon_s2r2

package runtime

import (
	"fmt"
	k "github.com/project-horizon/horizon-core/services/ai/knowledge"
	"math/rand"
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"testing"
)

func s2r7ConditionalDonors(rows []s2r4Row, mean, sd []float64, seed int64) []int {
	x := make([][]float64, len(rows))
	for i, row := range rows {
		p := row.Trial.Pre
		v := []float64{}
		for j := 1; j <= 36; j++ {
			v = append(v, p.Before[k.NodeID(j)])
		}
		prev := p.Previous
		if len(prev) == 0 {
			prev = make([]float64, 9)
		}
		v = append(v, prev...)
		v = append(v, p.Forecast...)
		for j := range v {
			v[j] = (v[j] - mean[j]) / sd[j]
		}
		x[i] = v
	}
	rng := rand.New(rand.NewSource(seed))
	out := make([]int, len(rows))
	for i, row := range rows {
		type neighbor struct {
			index    int
			distance float64
		}
		near := []neighbor{}
		p := row.Trial.Pre
		for j, other := range rows {
			q := other.Trial.Pre
			if other.Stream == row.Stream || len(p.History) != 3 || p.Action != q.Action || p.PreviousAction != q.PreviousAction || p.HistoryAction != q.HistoryAction || len(q.History) != 3 {
				continue
			}
			d := 0.
			for c, v := range x[i] {
				d += (v - x[j][c]) * (v - x[j][c])
			}
			near = append(near, neighbor{j, d})
		}
		slices.SortStableFunc(near, func(a, b neighbor) int {
			if a.distance < b.distance {
				return -1
			}
			if a.distance > b.distance {
				return 1
			}
			return a.index - b.index
		})
		out[i] = i
		if len(near) > 0 {
			out[i] = near[rng.Intn(min(12, len(near)))].index
		}
	}
	return out
}

func TestS2R7Evaluate(t *testing.T) {
	root, work, pop := s2r7Paths(t)
	_, seed, _ := s2r7PhaseSeed(t)
	if _, err := os.Stat(filepath.Join(work, "CONFIRMATORY_LOCK.json")); err != nil {
		t.Fatal("final not sealed")
	}
	cp, wal := s2r3Source(root)
	base := filepath.Join(work, "confirmatory", fmt.Sprintf("seed-%d", seed))
	dir := filepath.Join(work, "final", fmt.Sprintf("seed-%d", seed))
	source := orderedFixtureClone(t, cp)
	readers := map[string]*BrainRuntime{}
	descs := map[string]string{}
	for _, arm := range []string{"A", "B", "C", "D", "E", "F", "G"} {
		readers[arm] = orderedFixtureClone(t, filepath.Join(base, arm, "final.json"))
		descs[arm] = "D3"
		if arm == "A" {
			descs[arm] = "D0"
		}
		if arm == "B" {
			descs[arm] = "D1"
		}
	}
	masked := orderedFixtureClone(t, filepath.Join(base, "history-masked/final.json"))
	iid := orderedFixtureClone(t, filepath.Join(base, "iid/final.json"))
	iidMasked := orderedFixtureClone(t, filepath.Join(base, "iid-masked/final.json"))
	var scale struct{ Mean, SD []float64 }
	s1Read(t, filepath.Join(work, "PERMUTATION_LOCK.json"), &scale)
	for ci, condition := range []string{"seen", "unseen", "shifted", "independent"} {
		od := filepath.Join(dir, condition, "native-observer")
		observer, ow := s2r2Fork(t, cp, wal, od)
		rng := int64(291000000 + seed*1000 + ci*1000000)
		var native []s2r4Row
		if condition != "independent" {
			_, native = s2r7Acquire(t, observer, readers, pop, filepath.Join(dir, condition, "utility"), rng, 16, 16, condition == "shifted", condition != "seen", descs, false)
		} else {
			native = s2r4Collect(t, observer, pop, filepath.Join(dir, condition, "iid-acquisition"), "independent", rng, 16, 16, false)
		}
		s1Check(t, ow.Close())
		history := map[string]any{}
		historyReader, historyMasked := readers["G"], masked
		if condition == "independent" {
			historyReader, historyMasked = iid, iidMasked
		}
		full, cost := s2r4Predict(t, historyReader, native, pop, s2r4Arm{}, "full", nil)
		zero, _ := s2r4Predict(t, historyMasked, native, pop, s2r4Arm{NoHistory: true}, "full", nil)
		magnitude, _ := s2r4Predict(t, historyReader, native, pop, s2r4Arm{}, "no-history", nil)
		history["full"], history["matched_no_history"], history["magnitude_ablation"], history["latency_ns"], history["source"] = full, zero, magnitude, cost, s2Predictions(t, source, s2r4Trials(native), pop)
		for j := 0; j < 10; j++ {
			donors := s2r7ConditionalDonors(native, scale.Mean, scale.SD, 292000037+int64(j))
			pred, _ := s2r4Predict(t, historyReader, native, pop, s2r4Arm{}, "permuted", donors)
			history[fmt.Sprintf("permute-%d", j)] = pred
			history[fmt.Sprintf("donors-%d", j)] = donors
		}
		s2r7Write(t, filepath.Join(dir, condition, "history.json"), history)
		s2r7Write(t, filepath.Join(dir, condition, "native-rows.json"), native)
		if condition != "independent" {
			for arm, r := range readers {
				var rows []s2r7Row
				s1Read(t, filepath.Join(dir, condition, "utility", arm+"-rows.json"), &rows)
				mask, association, permuted, retro, contribution := []float64{}, []float64{}, []float64{}, []float64{}, []float64{}
				exact := 0
				for _, row := range rows {
					ctx := row.Context
					ctx.Pathways = slices.Clone(ctx.Pathways)
					slices.Reverse(ctx.Pathways)
					in := s2r7Input(ctx, row.Original, row.State, row.Descriptor, nil)
					if reflect.DeepEqual(in, row.Input) {
						exact++
					}
					permuted = append(permuted, s2r6Read(t, r, in))
					ctx = row.Context
					ctx.Pathways = slices.Clone(ctx.Pathways)
					for i := range ctx.Pathways {
						ctx.Pathways[i].Delta = 0
					}
					if row.Descriptor == "D0" {
						ctx.Original.Measurements = slices.Clone(ctx.Original.Measurements)
						ctx.Original.Measurements[3] = 0
					}
					mask = append(mask, s2r6Read(t, r, s2r7Input(ctx, row.Original, row.State, row.Descriptor, nil)))
					ctx = row.Context
					ctx.Pathways = slices.Clone(ctx.Pathways)
					if len(ctx.Pathways) == 2 {
						ctx.Pathways[0].Delta, ctx.Pathways[1].Delta = ctx.Pathways[1].Delta, ctx.Pathways[0].Delta
					}
					association = append(association, s2r6Read(t, r, s2r7Input(ctx, row.Original, row.State, row.Descriptor, nil)))
					retro = append(retro, s2r6Read(t, r, s2r7Input(row.Context, row.Original, row.State, row.Descriptor, &row.Actual)))
					ci := row.Input
					ci.Words = [][]float64{slices.Clone(ci.Words[0])}
					ci.Words[0][4] = 0
					ci.Words[0][6] = 1
					contribution = append(contribution, s2r6Read(t, r, ci))
				}
				s2r7Write(t, filepath.Join(dir, condition, "ablation-"+arm+".json"), map[string]any{"delta_mask": mask, "association_swap": association, "permutation": permuted, "permutation_exact_inputs": exact, "retrospective": retro, "contribution": contribution, "N": len(rows)})
			}
		}
	}
	var historical []s2Trial
	s1Read(t, filepath.Join(root, "s2r2/holdout.json"), &historical)
	retained := map[string]any{}
	for _, arm := range []string{"A", "B", "C", "D", "E", "F", "G"} {
		r := readers[arm]
		hist := s2r4Retention(t, source, r, historical, pop)
		candidate := hist["S2_candidate"].(s2Metric)
		hist["historical_contract_pass"] = candidate.RMSE <= .01766457
		hist["strict_S1_pass"] = hist["S1_max_delta"].(float64) == 0
		retained[arm] = map[string]any{"historical": hist, "new": map[string]any{}}
	}
	for ni, name := range []string{"normal", "shifted"} {
		od := filepath.Join(dir, "retention-"+name)
		observer, w := s2r2Fork(t, cp, wal, od)
		t.Setenv("HORIZON_S2_WORK", od)
		rows := s2r5OldCollect(t, observer, pop, 295000000+int64(seed*1000+ni*1000000), 256, ni == 1, nil, -1)
		s2r7Write(t, filepath.Join(od, "rows.json"), rows)
		s1Check(t, w.Close())
		for arm, r := range readers {
			value := s2r4Retention(t, source, r, rows, pop)
			value["paired_same_inputs"] = true
			value["source_predictions"] = s2Predictions(t, source, rows, pop)
			value["candidate_predictions"] = s2Predictions(t, r, rows, pop)
			retained[arm].(map[string]any)["new"].(map[string]any)[name] = value
		}
	}
	s2r7Write(t, filepath.Join(dir, "retention.json"), retained)
}
