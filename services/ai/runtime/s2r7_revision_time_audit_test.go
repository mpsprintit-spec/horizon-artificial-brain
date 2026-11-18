//go:build horizon_s2r7_audit && horizon_s2r7 && horizon_s2r6 && horizon_s2r5 && horizon_s2r4 && horizon_s2r3 && horizon_s2r2

package runtime

// The existing R6 oracle protocol pads a nonlearning observer control to the
// candidate revision because flexAt derives probe time from LastSequence.
// Keep the original unmatched and physical-time-only results separately.
import (
	"encoding/json"
	"fmt"
	"path/filepath"
	"testing"
	"time"
)

func TestS2R7RevisionTimeAudit(t *testing.T) {
	_, work, pop := s2r7Paths(t)
	_, seed, _ := s2r7PhaseSeed(t)
	base := filepath.Join(work, "confirmatory", fmt.Sprintf("seed-%d", seed))
	dir := filepath.Join(base, "revision-time-audit")
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
		A, E     any
		Sequence uint64
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
		family := "history-observer"
		if arm == "iid" || arm == "iid-masked" {
			family = "iid-observer"
		}
		key := fmt.Sprintf("%s-%d-%s", family, snap.Sequence, at.Format(time.RFC3339Nano))
		control, ok := cache[key]
		if !ok {
			source := filepath.Join(base, family, "observer-final.json")
			r := orderedFixtureClone(t, source)
			for r.LastSequence() < snap.Sequence {
				id := fmt.Sprintf("r7-oracle-control-%d", r.LastSequence())
				s2r4Command(t, r, NativeControlCommand{ID: id + "-begin", Begin: &NativeBegin{EpisodeID: id, Observation: &NativeObservation{State: flexCState(pop, 0, .4), Source: "r7-nonlearning-time-control", SampleID: id}}})
				action := 0
				s2r4Command(t, r, NativeControlCommand{ID: id + "-propagate", Step: &NativeStep{EpisodeID: id, ActionOverride: &action}})
				action = 1
				last := s2r4Command(t, r, NativeControlCommand{ID: id + "-stop", Step: &NativeStep{EpisodeID: id, ActionOverride: &action}})
				s2r4Command(t, r, NativeControlCommand{ID: id + "-outcome", Outcome: &NativeOutcome{EpisodeID: id, TraceDigest: last.TraceDigest, Source: "r7-nonlearning-time-control", OutcomeID: id, Provenance: "observed-sensor", Learn: false}})
			}
			r.mu.Lock()
			r.settleMemoryTimeLocked(at)
			r.mu.Unlock()
			p := filepath.Join(dir, arm, "control.json")
			s2r7Write(t, filepath.Join(dir, arm, "CONTROL_PROTOCOL.json"), map[string]any{"source": source, "candidate_sequence": snap.Sequence, "control_sequence": r.LastSequence(), "candidate_memory_time": at, "protocol": "unchanged R6 revision-matched observer control plus explicit physical-time match", "no_training": true})
			s2r3SaveCP(t, r, p)
			image := s2r4OracleImage(t, p, filepath.Join(dir, arm, "oracle-control-evaluation-only.json"))
			control.A = e1Retention(t, image, pop)
			control.E = e1Measure(t, image, pop)
			control.Sequence = r.LastSequence()
			cache[key] = control
		}
		candidate := prior[arm].AE1
		delta := s1AuditMax(s1AuditValue(t, control.A), s1AuditValue(t, candidate.Candidate))
		edelta := s1AuditMax(s1AuditValue(t, control.E), s1AuditValue(t, candidate.CandidateE1))
		pass := delta <= 1e-7 && edelta <= 1e-7 && candidate.CCorrect == 6 && candidate.E1Correct >= 35
		results[arm] = map[string]any{"candidate_oracles_reused": true, "candidate_unchanged": true, "original_unmatched_pass": candidate.Pass, "R6_revision_matched_criterion_pass": pass, "delta": delta, "E1_delta": edelta, "C_correct": candidate.CCorrect, "E1_correct": candidate.E1Correct, "candidate_sequence": snap.Sequence, "control_sequence": control.Sequence, "candidate_memory_applied_at": at, "control": control.A, "control_E1": control.E, "candidate": candidate.Candidate, "candidate_E1": candidate.CandidateE1, "scope": "source age/revision control from existing R6 protocol; not fitting or checkpoint selection; original unmatched failures retained"}
		s2r7Write(t, filepath.Join(base, "REVISION_TIME_AUDIT.json"), results)
		t.Log("unmatched/R6 revision-time-matched", seed, arm, candidate.Pass, pass)
	}
}
