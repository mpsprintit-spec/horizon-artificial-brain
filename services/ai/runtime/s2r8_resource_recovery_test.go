//go:build horizon_s2r8 && horizon_s2r4 && horizon_s2r3 && horizon_s2r2

package runtime

// Recover only interrupted output/continuation checks. The fourth-block
// scientific checkpoint must authenticate and match deterministic replay.
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

func r8ResourceRead(t *testing.T, p string) []byte {
	t.Helper()
	b, e := os.ReadFile(p)
	if os.IsNotExist(e) {
		b, e = exec.Command("zstd", "-q", "-d", "-c", p+".zst").Output()
	}
	s1Check(t, e)
	return b
}
func r8ResourcePreserve(t *testing.T, p string) {
	t.Helper()
	if _, e := os.Stat(p); e == nil {
		s1Check(t, os.Rename(p, p+".resource-original"))
	}
	if _, e := os.Stat(p + ".zst"); e == nil {
		s1Check(t, os.Rename(p+".zst", p+".resource-original.zst"))
	}
}

func TestS2R8ResourceRecovery(t *testing.T) {
	root, work, seed, pop := s2r8Paths(t)
	s2r8CheckLock(t, work)
	base := filepath.Join(work, fmt.Sprintf("seed-%d", seed))
	arm := os.Getenv("R8_RECOVERY_ARM")
	dir := filepath.Join(base, arm)
	failure, e := os.ReadFile(filepath.Join(work, "execution", fmt.Sprintf("TestS2R8Train-%d-%s.log", seed, arm)))
	s1Check(t, e)
	if !bytes.Contains(failure, []byte("no space left on device")) {
		t.Fatal("not a resource-interrupted model")
	}
	var corpus s2r8Corpus
	s1Read(t, filepath.Join(base, "TRAINING.json"), &corpus)
	scratch := filepath.Join(work, "resource-replay", fmt.Sprintf("%d-%s", seed, arm))
	s1Check(t, os.MkdirAll(scratch, 0700))
	observerCP := filepath.Join(scratch, "observer.json")
	s1Check(t, os.WriteFile(observerCP, r8ResourceRead(t, filepath.Join(base, "training-observer/final.json")), 0600))
	startSeq := s1Snapshot(t, observerCP).Sequence
	originalWAL := r8ResourceRead(t, filepath.Join(dir, "events.jsonl"))
	startWAL := []byte{}
	originalEvents := []LoggedEvent{}
	for _, line := range bytes.SplitAfter(originalWAL, []byte{'\n'}) {
		if len(bytes.TrimSpace(line)) == 0 {
			continue
		}
		var event LoggedEvent
		s1Check(t, json.Unmarshal(line, &event))
		if event.Sequence <= startSeq {
			startWAL = append(startWAL, line...)
		} else {
			originalEvents = append(originalEvents, event)
		}
	}
	if len(originalEvents) < 4 {
		t.Fatal("four committed learning records missing")
	}
	walPath := filepath.Join(scratch, "events.jsonl")
	s1Check(t, os.WriteFile(walPath, startWAL, 0600))
	r, w := s1Open(t, observerCP, walPath)
	defer w.Close()
	source := orderedFixtureClone(t, observerCP)
	var old []s2Trial
	s1Read(t, filepath.Join(root, "s2r2/holdout.json"), &old)
	curves := []any{map[string]any{"block": 0, "exposures": 0, "inventory": s2r3Inventory(t, r), "monitor": s2r8Score(corpus.Monitor, s2Predictions(t, r, corpus.Monitor, pop))}}
	receipts := []*CompositionLearningReceipt{}
	retention := []any{}
	var previous []any
	if data, err := os.ReadFile(filepath.Join(dir, "retention.json")); err == nil {
		_ = json.Unmarshal(data, &previous)
	}
	for b := 0; b < 4; b++ {
		event := originalEvents[b]
		if event.CompositionExposure == nil {
			t.Fatal("unexpected committed suffix record")
		}
		before := s2r3Memory()
		cpu := s2r8CPU()
		at := time.Now()
		r.SetClock(NewFixedClock(event.Timestamp))
		_, receipt, err := r.LearnCompositionWithDiagnosis(*event.CompositionExposure, event.Timestamp)
		s1Check(t, err)
		ns := time.Since(at).Nanoseconds()
		cpu = s2r8CPU() - cpu
		after := s2r3Memory()
		receipts = append(receipts, receipt)
		expectedPath := filepath.Join(scratch, fmt.Sprintf("original-block-%d.json", b+1))
		s1Check(t, os.WriteFile(expectedPath, r8ResourceRead(t, filepath.Join(dir, fmt.Sprintf("block-%d.json", b+1))), 0600))
		expected := orderedFixtureClone(t, expectedPath)
		if !bytes.Equal(continuationBytes(t, r), continuationBytes(t, expected)) {
			t.Fatal("resource replay changed scientific state", b+1)
		}
		if len(previous) > b {
			retention = append(retention, previous[b])
		} else {
			oracle := filepath.Join(scratch, fmt.Sprintf("oracle-%d", b))
			ae := s2r8AE1(t, observerCP, expectedPath, oracle, pop)
			retention = append(retention, map[string]any{"block": b + 1, "AE1": ae, "historical_S1_S2": s2r8NumericRetention(t, source, r, old, pop), "recovered_after_resource_interruption": true})
			s1Check(t, os.RemoveAll(oracle))
		}
		curves = append(curves, map[string]any{"block": b + 1, "exposures": (b + 1) * 64, "inventory": s2r3Inventory(t, r), "monitor": s2r8Score(corpus.Monitor, s2Predictions(t, r, corpus.Monitor, pop)), "learning_nanoseconds": ns, "learning_cpu_nanoseconds": cpu, "memory_before": before, "memory_after": after, "committed": receipt.Diagnosis.Committed, "cause": receipt.Diagnosis.Cause, "trials": len(receipt.Diagnosis.Trials), "measurement_origin": "deterministic resource-recovery replay; original partial timing files preserved"})
	}
	final := filepath.Join(dir, "final.json")
	r8ResourcePreserve(t, final)
	s1Check(t, r.Checkpoint(final))
	finalChecksum := s1Snapshot(t, final).Checksum
	c2 := corpus
	c2.TrainingFinalTime = corpus.TrainingFinalTime.Add(10 * time.Nanosecond)
	s2r8Lesson(t, r, c2, arm, 0, s2r8Roots(t, root), pop)
	continued := filepath.Join(dir, "continued.json")
	r8ResourcePreserve(t, continued)
	s1Check(t, r.Checkpoint(continued))
	s1Check(t, w.Close())
	r8ResourcePreserve(t, filepath.Join(dir, "events.jsonl"))
	s1Check(t, os.WriteFile(filepath.Join(dir, "events.jsonl"), r8ResourceRead(t, walPath), 0600))
	for name, value := range map[string]any{"curves.json": curves, "receipts.json": receipts, "retention.json": retention} {
		p := filepath.Join(dir, name)
		r8ResourcePreserve(t, p)
		s2r8Write(t, p, value)
	}
	s2r8Write(t, filepath.Join(dir, "RESOURCE_RECOVERY.json"), map[string]any{"original_failure_retained": true, "four_scientific_checkpoints_match_exactly": true, "selected_block": 4, "source_sequence": 80735, "final_checksum": finalChecksum, "learner_changed": false, "final_holdout_used": false, "timing_remeasured_on_same_committed_commands": true})
	s1Check(t, os.RemoveAll(scratch))
	for _, name := range []string{"final.json", "continued.json", "events.jsonl"} {
		s2r8Zip(t, filepath.Join(dir, name))
	}
}
