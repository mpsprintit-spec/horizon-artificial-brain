//go:build horizon_s2r7 && horizon_s2r6 && horizon_s2r5 && horizon_s2r4 && horizon_s2r3 && horizon_s2r2

package runtime

import (
	"fmt"
	k "github.com/project-horizon/horizon-core/services/ai/knowledge"
	"path/filepath"
	"testing"
)

func TestS2R7Conditioning(t *testing.T) {
	_, work, _ := s2r7Paths(t)
	for _, seed := range []int{37, 211} {
		base := filepath.Join(work, "development", fmt.Sprintf("seed-%d", seed))
		r := orderedFixtureClone(t, filepath.Join(base, "bootstrap/final.json"))
		var rows, eval []s2r7Row
		s1Read(t, filepath.Join(base, "utility-acquisition/D0-rows.json"), &rows)
		s1Read(t, filepath.Join(base, "transfer-unseen/acquisition/A-rows.json"), &eval)
		fit, check := s2r7Partition(rows, false, 271300000+int64(seed*1000)+111)
		inputs := []k.CompositionInput{}
		targets := []float64{}
		for _, row := range eval {
			inputs = append(inputs, row.Input)
			targets = append(targets, row.Utility)
		}
		records := []any{}
		for b := 0; b < 16; b++ {
			fi, ci := []k.CompositionInput{}, []k.CompositionInput{}
			fy, cy := []float64{}, []float64{}
			for _, row := range fit[64*b : 64*(b+1)] {
				fi = append(fi, row.Input)
				fy = append(fy, row.Utility)
			}
			for _, row := range check[64*b : 64*(b+1)] {
				ci = append(ci, row.Input)
				cy = append(cy, row.Utility)
			}
			probe, e := r.brain.S2R7ConditioningProbe(fi, fy, ci, cy, inputs)
			s1Check(t, e)
			records = append(records, map[string]any{"block": b, "probe": probe})
		}
		s2r7Write(t, filepath.Join(base, "CONDITIONING_INTERVENTION.json"), map[string]any{"records": records, "holdout_utility": targets, "source": "fresh R7 development, not R6 final", "not_installed": true})
	}
}
