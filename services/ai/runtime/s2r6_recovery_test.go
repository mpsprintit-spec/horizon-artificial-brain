//go:build horizon_s2r6 && horizon_s2r5 && horizon_s2r4 && horizon_s2r3 && horizon_s2r2

package runtime

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

func TestS2R6Preflight(t *testing.T) {
	_, work, _ := s2r6Paths(t)
	seed := s2r6Seed(t)
	out := map[string]any{}
	for _, arm := range s2r6AllArms() {
		path := filepath.Join(work, fmt.Sprintf("training/seed-%d/%s/final.json", seed, arm))
		r := orderedFixtureClone(t, path)
		snap := s1Snapshot(t, path)
		sum, err := checkpointChecksumForSchema(snap)
		s1Check(t, err)
		if sum != snap.Checksum || len(r.brain.BrainState.NResolution.Features) != 32 {
			t.Fatal("preflight", arm)
		}
		out[arm] = map[string]any{"native_checksum": sum, "inventory": s2r3Inventory(t, r)}
	}
	s1Write(t, filepath.Join(work, fmt.Sprintf("PREFLIGHT-%d.json", seed)), out)
}

func TestS2R6DevelopmentPersistence(t *testing.T) {
	root, work, _ := s2r6Paths(t)
	cp, wal := s2r3Source(root)
	prefix, err := os.ReadFile(wal)
	s1Check(t, err)
	out := map[string]any{}
	for _, seed := range []int{37, 83} {
		for _, arm := range []string{"C", "F"} {
			dir := filepath.Join(work, fmt.Sprintf("development/seed-%d/%s", seed, arm))
			r, w := s1Open(t, filepath.Join(dir, "final.json"), filepath.Join(dir, "events.jsonl"))
			replay, err := RebuildRuntimeFromCheckpointAndJournal(w, cp)
			s1Check(t, err)
			if !bytes.Equal(continuationBytes(t, r), continuationBytes(t, replay)) {
				t.Fatal("development replay mismatch", seed, arm)
			}
			raw, err := os.ReadFile(filepath.Join(dir, "events.jsonl"))
			s1Check(t, err)
			if !bytes.HasPrefix(raw, prefix) {
				t.Fatal("development prefix")
			}
			out[fmt.Sprintf("%d-%s", seed, arm)] = map[string]any{"full_suffix_replay_exact": true, "restart_exact": true, "historical_prefix_exact": true, "native_utility_protocol": r.brain.BrainState.NResolution.Last.ProposalProtocol}
			s1Check(t, w.Close())
		}
	}
	s1Write(t, filepath.Join(work, "DEVELOPMENT_PERSISTENCE.json"), out)
}
