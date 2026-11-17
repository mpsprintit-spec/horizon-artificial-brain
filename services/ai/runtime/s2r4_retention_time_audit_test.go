//go:build horizon_s2r4 && horizon_s2r3 && horizon_s2r2

package runtime

// Supplementary post-evaluation causal audit. Never used to fit/select a Brain
// or replace the preregistered unadjusted retention results.
import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestS2R4ElapsedMemoryControlAudit(t *testing.T) {
	_, work, pop := s2r4Paths(t)
	results := map[string]any{}
	cache := filepath.Join(work, "elapsed-memory-audit.json")
	if _, e := os.Stat(cache); e == nil {
		s1Read(t, cache, &results)
	}
	var recorded map[string]struct {
		AE1 struct {
			Candidate   any `json:"candidate"`
			CandidateE1 any `json:"candidate_E1"`
			CCorrect    int `json:"C_correct"`
			E1Correct   int `json:"E1_correct"`
		} `json:"AE1"`
	}
	s1Read(t, filepath.Join(work, "persistence.json"), &recorded)
	for _, seed := range s2r3Seeds {
		for _, arm := range s2r4Arms {
			if only := os.Getenv("HORIZON_S2R4_TIME_ARM"); only != "" && only != arm.Name {
				continue
			}
			name := fmt.Sprintf("s%d-%s", seed, arm.Name)
			if _, ok := results[name]; ok {
				continue
			}
			dir := filepath.Join(work, fmt.Sprintf("training/seed-%d/%s", seed, arm.Name))
			controlPath := filepath.Join(dir, "retention-control.json")
			if _, e := os.Stat(controlPath); e != nil {
				continue
			}
			candidatePath := filepath.Join(dir, "final.json")
			snap := s1Snapshot(t, candidatePath)
			var b struct {
				MemoryDecay struct {
					AppliedAt time.Time `json:"applied_at"`
				} `json:"memory_decay"`
			}
			s1Check(t, json.Unmarshal(snap.Brain, &b))
			if b.MemoryDecay.AppliedAt.IsZero() {
				t.Fatal("missing physical memory time")
			}
			control := orderedFixtureClone(t, controlPath)
			// The same native physical-memory time law that numerical commands invoke;
			// no parameter assignment, training target, gate change or model promotion.
			control.mu.Lock()
			control.settleMemoryTimeLocked(b.MemoryDecay.AppliedAt)
			control.mu.Unlock()
			auditDir := filepath.Join(work, "elapsed-memory-audit", fmt.Sprintf("s%d-%s", seed, arm.Name))
			s1Check(t, os.MkdirAll(auditDir, 0700))
			normalized := filepath.Join(auditDir, "control-natural-time.json")
			s2r3SaveCP(t, control, normalized)
			// Candidate oracles already measured on these exact frozen bytes by
			// the primary retention audit. Reuse them; only the changed control
			// needs another measurement. This cannot select or improve a model.
			prior, ok := recorded[name]
			if !ok {
				t.Fatal("primary retention result missing", name)
			}
			image := s2r4OracleImage(t, normalized, filepath.Join(auditDir, "control-evaluation-only.json"))
			controlAE := e1Retention(t, image, pop)
			controlE := e1Measure(t, image, pop)
			d := s1AuditMax(s1AuditValue(t, controlAE), s1AuditValue(t, prior.AE1.Candidate))
			ed := s1AuditMax(s1AuditValue(t, controlE), s1AuditValue(t, prior.AE1.CandidateE1))
			ae := map[string]any{"control": controlAE, "control_E1": controlE, "candidate": prior.AE1.Candidate, "candidate_E1": prior.AE1.CandidateE1, "candidate_oracles_reused_from_primary_frozen_checkpoint": true, "delta": d, "E1_delta": ed, "C_correct": prior.AE1.CCorrect, "E1_correct": prior.AE1.E1Correct, "pass": d <= 1e-7 && ed <= 1e-7 && prior.AE1.CCorrect == 6 && prior.AE1.E1Correct >= 35}
			results[name] = map[string]any{"supplementary_post_evaluation": true, "used_for_training_or_selection": false, "unadjusted_results_preserved": true, "control_is_detached_evaluation_only": true, "candidate_applied_at": b.MemoryDecay.AppliedAt, "AE1_elapsed_time_matched": ae, "interpretation": "Native control commands do not settle physical memory time; numerical learning commands do. Original control matched revision but not applied_at. This audit isolates natural memory decay; it does not redefine or erase the original strict retention failure."}
			s1Write(t, cache, results)
		}
	}
	s1Write(t, filepath.Join(work, "elapsed-memory-audit.json"), results)
	t.Log("supplementary elapsed-time controls", len(results))
}
