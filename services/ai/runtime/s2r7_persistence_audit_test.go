//go:build horizon_s2r7_audit && horizon_s2r7 && horizon_s2r6 && horizon_s2r5 && horizon_s2r4 && horizon_s2r3 && horizon_s2r2

package runtime

import (
	"bytes"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"reflect"
	"testing"
	"time"
)

func TestS2R7PersistenceAudit(t *testing.T) {
	root, work, pop := s2r7Paths(t)
	phase, seed, _ := s2r7PhaseSeed(t)
	cp, wal := s2r3Source(root)
	prefix, e := os.ReadFile(wal)
	s1Check(t, e)
	source := orderedFixtureClone(t, cp)
	out := map[string]any{}
	base := filepath.Join(work, phase, fmt.Sprintf("seed-%d", seed))
	arms := []string{"A", "B", "C", "D", "E", "F", "G", "history-masked"}
	if phase == "confirmatory" {
		arms = append(arms, "iid", "iid-masked")
	}
	for _, arm := range arms {
		dir := filepath.Join(base, arm)
		path := filepath.Join(dir, "final.json")
		r, w := s1Open(t, path, filepath.Join(dir, "events.jsonl"))
		original := continuationBytes(t, r)
		started := time.Now()
		replay, err := RebuildRuntimeFromCheckpointAndJournal(w, cp)
		s1Check(t, err)
		elapsed := time.Since(started).Nanoseconds()
		if !bytes.Equal(original, continuationBytes(t, replay)) {
			t.Fatal("full suffix replay mismatch", seed, arm)
		}
		restarted, err := RebuildRuntimeFromCheckpointAndJournal(w, path)
		s1Check(t, err)
		if !bytes.Equal(original, continuationBytes(t, restarted)) {
			t.Fatal("checkpoint restart mismatch", arm)
		}
		raw, err := os.ReadFile(filepath.Join(dir, "events.jsonl"))
		s1Check(t, err)
		if !bytes.HasPrefix(raw, prefix) {
			t.Fatal("historical prefix changed")
		}
		a, err := r.brain.CanonicalJSON()
		s1Check(t, err)
		b, err := replay.brain.CanonicalJSON()
		s1Check(t, err)
		if !bytes.Equal(a, b) {
			t.Fatal("signed canonical reconstruction mismatch")
		}
		if len(r.brain.BrainState.NResolution.Features) != 32 || r.activation.Memory != r.brain || r.learning.Kb != r.brain || r.dnf.Brain() != r.brain {
			t.Fatal("architecture invariant")
		}
		if !reflect.DeepEqual(source.brain.NativeControlSnapshot(), r.brain.NativeControlSnapshot()) {
			t.Fatal("policy changed")
		}
		snap := s1Snapshot(t, path)
		sum, err := checkpointChecksumForSchema(snap)
		s1Check(t, err)
		if snap.Checksum != sum {
			t.Fatal("native checksum")
		}
		ae := s2r3AE1(t, cp, path, pop)
		if !ae["pass"].(bool) {
			t.Log("A-E1 retention gate FAIL", seed, arm)
		}
		for _, key := range []string{"delta", "E1_delta"} {
			if v, ok := ae[key].(float64); ok && (math.IsInf(v, 0) || math.IsNaN(v)) {
				ae[key+"_nonfinite"] = true
				ae[key] = fmt.Sprint(v)
			}
		}
		rollback := orderedFixtureClone(t, cp)
		if !bytes.Equal(continuationBytes(t, source), continuationBytes(t, rollback)) {
			t.Fatal("source rollback")
		}
		out[arm] = map[string]any{"restart_exact": true, "full_suffix_replay_exact": true, "signed_parameter_reconstruction_exact": true, "canonical_state_exact": true, "historical_WAL_prefix_exact": true, "source_rollback_exact": true, "source_checkpoint_sha256": s2r3FileHash(t, cp), "source_WAL_sha256": s2r3FileHash(t, wal), "native_checksum": sum, "final_sequence": r.LastSequence(), "suffix_bytes": len(raw) - len(prefix), "replay_nanoseconds": elapsed, "inventory": s2r3Inventory(t, r), "AE1": ae, "resources": s2r3Memory()}
		s1Check(t, w.Close())
		s2r7Write(t, filepath.Join(base, "PERSISTENCE_AUDIT.json"), out)
		t.Log("exact persistence; A-E1 pass:", seed, arm, ae["pass"])
	}
}
