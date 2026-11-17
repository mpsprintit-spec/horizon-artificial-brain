//go:build horizon_s2r3 && horizon_s2r2

package runtime

import (
	"fmt"
	"path/filepath"
	"testing"
)

// Supplementary DEVELOPMENT diagnosis, never another final-holdout condition.
// The preregistered no-extras condition concerns the S2-added channels. This
// separate stored-development audit also removes inherited S1 temporal inputs.
func TestS2R3DevelopmentInputAudit(t *testing.T) {
	_, work, pop := s2r3Paths(t)
	rows := []s2Trial{}
	for stream := 0; stream < 2; stream++ {
		var x []s2Trial
		s1Read(t, filepath.Join(work, fmt.Sprintf("development-%d-check.json", stream)), &x)
		rows = append(rows, x...)
	}
	p := s2r3Metric(t, work)
	pred := map[string][]float64{}
	metrics := map[string]any{}
	for _, seed := range s2r3Seeds {
		for _, cap := range []int{32, 64} {
			name := fmt.Sprintf("s%d-c%d", seed, cap)
			r := orderedFixtureClone(t, filepath.Join(work, fmt.Sprintf("seed-%d/arm-%d/final.json", seed, cap)))
			pred[name+"/full"] = s2Predictions(t, r, rows, pop)
			for _, condition := range []string{"no-history", "no-forecast", "no-previous-operation", "no-presence", "no-history-operation", "no-extras"} {
				pred[name+"/"+condition] = s2r3Intervention(t, r, rows, pop, condition, 0)
			}
			for _, condition := range []string{"permute-history", "permute-forecast"} {
				pred[name+"/"+condition] = s2r3Intervention(t, r, rows, pop, condition, 970001)
			}
			out := []float64{}
			for _, in := range s2r3Inputs(rows, pop) {
				for j := 0; j < 6; j++ {
					in.Sequence[1][j] = 0
				}
				in.Sequence[0][0] = 0
				in.Sequence[0][1] = 0
				// Last two original S1 rows hold nine previous-state projections,
				// then current operation, previous operation and temporal presence.
				for j := 0; j < 6; j++ {
					in.Sequence[len(in.Sequence)-2][j] = 0
				}
				for j := 0; j < 3; j++ {
					in.Sequence[len(in.Sequence)-1][j] = 0
				}
				in.Sequence[len(in.Sequence)-1][4] = 0
				in.Sequence[len(in.Sequence)-1][5] = 0
				o, e := r.ReadComposition(in, in.Scene.ObservedAt)
				s1Check(t, e)
				out = append(out, o.NResolution.Value)
			}
			pred[name+"/current-state-and-operation-only"] = out
		}
	}
	for name, v := range pred {
		metrics[name] = s2MetricOf(rows, v, p)
	}
	s1Write(t, filepath.Join(work, "development-input-audit.json"), map[string]any{"data": "128 stored development Check cases; zero final-holdout cases", "post_confirmatory_supplement": true, "new_experiences": 0, "new_learning": 0, "model_selection": false, "all_additional_removed_includes_S1_temporal_context": true, "predictions": pred, "metrics": metrics})
}
