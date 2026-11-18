//go:build horizon_s2r8 && horizon_s2r4 && horizon_s2r3 && horizon_s2r2

package runtime

// Corrects only the frozen harness's unsupported hot rollback. All original
// failures remain recorded. Learner, corpora, thresholds and selected fourth
// block checkpoints are unchanged; this file never runs another main lesson.
import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"
)

func TestS2R8RecoveryCompanion(t *testing.T) {
	root, work, seed, pop := s2r8Paths(t)
	s2r8CheckLock(t, work)
	base := filepath.Join(work, fmt.Sprintf("seed-%d", seed))
	var corpus s2r8Corpus
	s1Read(t, filepath.Join(base, "TRAINING.json"), &corpus)
	sourceCP, sourceWAL := s2r3Source(root)
	for _, arm := range []string{"A", "B", "C", "D"} {
		if selected := os.Getenv("R8_RECOVERY_ARM"); selected != "" && selected != arm {
			continue
		}
		t.Run(arm, func(t *testing.T) {
			dir := filepath.Join(base, arm)
			var execution struct {
				Returncode int `json:"returncode"`
			}
			s1Read(t, filepath.Join(work, "execution", fmt.Sprintf("TestS2R8Train-%d-%s.json", seed, arm)), &execution)
			failure, err := os.ReadFile(filepath.Join(work, "execution", fmt.Sprintf("TestS2R8Train-%d-%s.log", seed, arm)))
			s1Check(t, err)
			resourceRecovered := false
			if bytes.Contains(failure, []byte("no space left on device")) {
				var proof struct {
					Exact bool `json:"four_scientific_checkpoints_match_exactly"`
				}
				s1Read(t, filepath.Join(dir, "RESOURCE_RECOVERY.json"), &proof)
				resourceRecovered = proof.Exact
			}
			if execution.Returncode != 1 || (!bytes.Contains(failure, []byte("native checkpoint restore requires a fresh detached runtime")) && !resourceRecovered) {
				t.Fatal("different frozen failure requires independent diagnosis")
			}
			scratch := filepath.Join(work, "companion-scratch", fmt.Sprintf("%d-%s", seed, arm))
			for revision := 2; ; revision++ {
				if _, e := os.Stat(scratch); os.IsNotExist(e) {
					break
				}
				scratch = filepath.Join(work, "companion-scratch", fmt.Sprintf("%d-%s-retry-%d", seed, arm, revision))
			}
			s1Check(t, os.MkdirAll(filepath.Dir(scratch), 0700))
			s1Check(t, os.Mkdir(scratch, 0700))
			final, continued, wal := filepath.Join(scratch, "final.json"), filepath.Join(scratch, "continued.json"), filepath.Join(scratch, "events.jsonl")
			for _, path := range []string{final, continued, wal} {
				original := filepath.Join(dir, filepath.Base(path))
				var data []byte
				if _, e := os.Stat(original + ".zst"); e == nil {
					data, err = exec.Command("zstd", "-q", "-d", "-c", original+".zst").Output()
				} else {
					data, err = os.ReadFile(original)
				}
				s1Check(t, err)
				s1Check(t, os.WriteFile(path, data, 0600))
			}
			fd := s1Snapshot(t, final)
			raw, err := os.ReadFile(wal)
			s1Check(t, err)
			sourcePrefix, err := os.ReadFile(sourceWAL)
			s1Check(t, err)
			prefix := []byte{}
			sequence := uint64(0)
			for _, line := range bytes.SplitAfter(raw, []byte{'\n'}) {
				if len(bytes.TrimSpace(line)) == 0 {
					continue
				}
				var row struct {
					Sequence uint64 `json:"sequence"`
				}
				s1Check(t, json.Unmarshal(line, &row))
				if row.Sequence > fd.Sequence {
					break
				}
				prefix = append(prefix, line...)
				sequence = row.Sequence
			}
			if sequence != fd.Sequence || !bytes.HasPrefix(prefix, sourcePrefix) {
				t.Fatal("final authenticated WAL boundary")
			}
			prefixPath := filepath.Join(scratch, "through-final.jsonl")
			s1Check(t, os.WriteFile(prefixPath, prefix, 0600))
			finalLog, err := OpenEventLog(prefixPath)
			s1Check(t, err)
			defer finalLog.Close()
			expected := orderedFixtureClone(t, final)
			want := continuationBytes(t, expected)
			restart, err := RebuildRuntimeFromCheckpointAndJournal(finalLog, final)
			s1Check(t, err)
			started := time.Now()
			replayed, err := RebuildRuntimeFromCheckpointAndJournal(finalLog, sourceCP)
			s1Check(t, err)
			replayNS := time.Since(started).Nanoseconds()
			result := map[string]any{"companion": true, "frozen_training_harness_returncode": execution.Returncode, "frozen_failure": "unsupported hot restore correctly rejected; rollback requires a fresh detached runtime", "restart_exact": bytes.Equal(want, continuationBytes(t, restart)), "full_suffix_replay_exact": bytes.Equal(want, continuationBytes(t, replayed)), "source_prefix_exact": bytes.HasPrefix(raw, sourcePrefix), "through_final_WAL_bytes": len(prefix), "through_final_WAL_sha256": s1Digest(prefix), "replay_nanoseconds": replayNS, "final_native_checksum": fd.Checksum, "inventory": s2r3Inventory(t, expected), "single_N_owner": restart.activation.Memory == restart.brain && restart.learning.Kb == restart.brain && restart.dnf.Brain() == restart.brain}
			if resourceRecovered {
				result["frozen_failure"] = "resource interruption; all four scientific checkpoints reconstructed exactly; original failure and interrupted files retained"
			}
			fullLog, err := OpenEventLog(wal)
			s1Check(t, err)
			defer fullLog.Close()
			actualContinued := orderedFixtureClone(t, continued)
			fullReplay, err := RebuildRuntimeFromCheckpointAndJournal(fullLog, sourceCP)
			s1Check(t, err)
			result["continued_full_replay_exact"] = bytes.Equal(continuationBytes(t, actualContinued), continuationBytes(t, fullReplay))
			// Execute exactly the already recorded extra lesson on a restarted final
			// copy, with its own journal, and compare to the recorded continued state.
			resumed, resumeLog := s2r2Fork(t, final, prefixPath, filepath.Join(scratch, "continued-copy"))
			defer resumeLog.Close()
			c2 := corpus
			c2.TrainingFinalTime = corpus.TrainingFinalTime.Add(10 * time.Nanosecond)
			receipt := s2r8Lesson(t, resumed, c2, arm, 0, s2r8Roots(t, root), pop)
			result["learning_after_restart_exact"] = bytes.Equal(continuationBytes(t, resumed), continuationBytes(t, actualContinued)) && s2r3JSONHash(t, receipt.Diagnosis) == s2r3JSONHash(t, actualContinued.brain.LastNDiagnosis())
			result["continued_learning_committed"] = receipt.Diagnosis.Committed
			rollback, rollbackLog := s2r2Fork(t, sourceCP, sourceWAL, filepath.Join(scratch, "source-rollback"))
			defer rollbackLog.Close()
			result["source_rollback_exact"] = bytes.Equal(continuationBytes(t, rollback), continuationBytes(t, orderedFixtureClone(t, sourceCP)))
			result["continued_native_checksum"] = s1Snapshot(t, continued).Checksum
			s2r8Write(t, filepath.Join(dir, "persistence.json"), result)
			s2r8Write(t, filepath.Join(dir, "RECOVERY_COMPANION.json"), result)
			for _, key := range []string{"restart_exact", "full_suffix_replay_exact", "source_prefix_exact", "continued_full_replay_exact", "learning_after_restart_exact", "source_rollback_exact", "single_N_owner"} {
				if result[key] != true {
					t.Error("recovery evidence failed", key)
				}
			}
			s1Check(t, finalLog.Close())
			s1Check(t, fullLog.Close())
			s1Check(t, resumeLog.Close())
			s1Check(t, rollbackLog.Close())
			s1Check(t, os.RemoveAll(scratch))
			t.Log(seed, arm, "corrected fresh-runtime rollback and all recovery checks", result["learning_after_restart_exact"])
		})
	}
}
