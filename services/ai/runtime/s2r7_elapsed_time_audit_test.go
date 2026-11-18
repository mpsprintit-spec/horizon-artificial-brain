//go:build horizon_s2r7_audit && horizon_s2r7 && horizon_s2r6 && horizon_s2r5 && horizon_s2r4 && horizon_s2r3 && horizon_s2r2

package runtime

// Supplementary causal diagnosis only. Original unmatched-clock failures are
// preserved and never replaced, fitted to, or used to promote a checkpoint.
import (
	"encoding/json"
	"fmt"
	"path/filepath"
	"testing"
	"time"
)

func TestS2R7ElapsedTimeAudit(t *testing.T) {
	root, work, pop := s2r7Paths(t)
	_, seed, _ := s2r7PhaseSeed(t)
	cp, _ := s2r3Source(root)
	base := filepath.Join(work, "confirmatory", fmt.Sprintf("seed-%d", seed))
	dir := filepath.Join(base, "elapsed-time-audit")
	var prior map[string]struct {
		AE1 struct {
			Candidate   any  `json:"candidate"`
			CandidateE1 any  `json:"candidate_E1"`
			CCorrect    int  `json:"C_correct"`
			E1Correct   int  `json:"E1_correct"`
			Pass        bool `json:"pass"`
		}
	}
	s1Read(t, filepath.Join(base, "PERSISTENCE_AUDIT.json"), &prior)
	cache := map[string]struct {
		A any
		E any
	}{}
	results := map[string]any{}
	for _, arm := range []string{"A", "B", "C", "D", "E", "F", "G", "history-masked", "iid", "iid-masked"} {
		snap := s1Snapshot(t, filepath.Join(base, arm, "final.json"))
		var state struct {
			MemoryDecay struct {
				AppliedAt time.Time `json:"applied_at"`
			} `json:"memory_decay"`
		}
		s1Check(t, json.Unmarshal(snap.Brain, &state))
		at := state.MemoryDecay.AppliedAt
		if at.IsZero() {
			t.Fatal("missing physical time")
		}
		key := at.Format(time.RFC3339Nano)
		control, ok := cache[key]
		if !ok {
			r := orderedFixtureClone(t, cp)
			r.mu.Lock()
			r.settleMemoryTimeLocked(at)
			r.mu.Unlock()
			p := filepath.Join(dir, arm, "control-natural-time.json")
			s2r7Write(t, filepath.Join(dir, arm, "CONTROL_TIME.json"), map[string]any{"source": cp, "candidate_applied_at": key, "natural_memory_time_law": true})
			s2r3SaveCP(t, r, p)
			image := s2r4OracleImage(t, p, filepath.Join(dir, arm, "oracle-control-evaluation-only.json"))
			control.A = e1Retention(t, image, pop)
			control.E = e1Measure(t, image, pop)
			cache[key] = control
		}
		candidate := prior[arm].AE1
		delta := s1AuditMax(s1AuditValue(t, control.A), s1AuditValue(t, candidate.Candidate))
		edelta := s1AuditMax(s1AuditValue(t, control.E), s1AuditValue(t, candidate.CandidateE1))
		pass := delta <= 1e-7 && edelta <= 1e-7 && candidate.CCorrect == 6 && candidate.E1Correct >= 35
		results[arm] = map[string]any{"candidate_oracles_reused_from_frozen_audit": true, "candidate_checkpoint_unchanged": true, "unadjusted_original_pass": candidate.Pass, "original_criterion_unchanged": true, "time_matched_pass": pass, "delta": delta, "E1_delta": edelta, "control": control.A, "control_E1": control.E, "candidate": candidate.Candidate, "candidate_E1": candidate.CandidateE1, "candidate_applied_at": key, "scope": "supplementary detached natural-time control; not training, selection or promotion"}
		s2r7Write(t, filepath.Join(base, "ELAPSED_TIME_AUDIT.json"), results)
		t.Log("unadjusted/time-matched", seed, arm, candidate.Pass, pass)
	}
}
