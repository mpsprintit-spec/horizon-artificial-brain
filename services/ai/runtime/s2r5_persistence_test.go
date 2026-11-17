//go:build horizon_s2r5 && horizon_s2r4 && horizon_s2r3 && horizon_s2r2

package runtime

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"testing"
	"time"
)

func TestS2R5Persistence(t *testing.T) {
	root, work, pop := s2r5Paths(t)
	cp, wal := s2r3Source(root)
	prefix, e := os.ReadFile(wal)
	s1Check(t, e)
	filterSeed, filterArm := os.Getenv("HORIZON_S2R5_SEED"), os.Getenv("HORIZON_S2R5_PERSISTENCE_ARM")
	rollbackName, resultPath := "rollback-audit", filepath.Join(work, "persistence.json")
	if filterSeed != "" {
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
	for _, seed := range s2r5Seeds {
		for _, arm := range s2r5CompatibilityArms() {
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
