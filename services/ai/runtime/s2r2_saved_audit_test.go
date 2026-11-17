//go:build horizon_s2r2

package runtime

// Read-only audit of already recorded cases. No new holdout acquisition, fitting
// or checkpoint selection. Detached ablations are never saved as brain state.
import (
	"path/filepath"
	"reflect"
	"testing"
)

func TestS2R2SavedReadoutAndS1Attribution(t *testing.T) {
	root, work := s2r2Paths(t)
	pop := s1Pop(t, filepath.Join(root, "source/source"))
	r, w := s1Open(t, filepath.Join(work, "s2r2-final.json"), filepath.Join(work, "s2r2-final-events.jsonl"))
	defer w.Close()
	var rows []s2Trial
	s1Read(t, filepath.Join(work, "holdout.json"), &rows)
	var saved map[string][]float64
	s1Read(t, filepath.Join(work, "predictions.json"), &saved)
	conditions := []string{"full", "no-forecast", "no-history-magnitude", "no-history-action", "no-history-presence", "no-history-magnitude-and-action", "permute-history-conditional", "permute-action-conditional", "permute-forecast", "permute-extras", "new-feature-clamp", "new-readout-clamp", "all-readout-clamp"}
	for _, condition := range conditions {
		if !reflect.DeepEqual(saved[condition], s2r2Intervention(t, r, rows, pop, condition)) {
			t.Fatal("saved intervention differs after native restart", condition)
		}
	}
	source := orderedFixtureClone(t, filepath.Join(work, "00-source.json"))
	continued := orderedFixtureClone(t, filepath.Join(work, "s2r2-continuation.json"))
	s1rows := s2S1Rows(rows)
	a := s1Predictions(t, source, s1rows, pop)
	b := s1Predictions(t, r, s1rows, pop)
	c := s1Predictions(t, continued, s1rows, pop)
	d1 := s1AuditMax(s1AuditValue(t, a), s1AuditValue(t, b))
	d2 := s1AuditMax(s1AuditValue(t, a), s1AuditValue(t, c))
	if d1 != 0 || d2 != 0 {
		t.Fatal("S1 restart/continuation drift", d1, d2)
	}
	ablated := orderedFixtureClone(t, filepath.Join(work, "s2r2-continuation.json"))
	edges := s1AblateReadout(ablated)
	probe := s1Predictions(t, ablated, s1rows, pop)
	change := s1AuditMax(s1AuditValue(t, c), s1AuditValue(t, probe))
	if edges == 0 || change == 0 {
		t.Fatal("S1 readout attribution missing")
	}
	s1Write(t, filepath.Join(work, "saved-readout-audit.json"), map[string]any{"stored_holdout_cases": len(rows), "new_holdout_cases": 0, "new_learning_commands": 0, "restarted_intervention_conditions_exact": len(conditions), "S1_coordinates": 3 * len(rows), "S1_restart_delta": d1, "S1_continuation_delta": d2, "S1_readout_edges_ablated_on_disposable_copy": edges, "S1_readout_ablation_max_delta": change, "pass": true})
}
