//go:build horizon_s2r4 && horizon_s2r3 && horizon_s2r2

package runtime

// Detached behavioral evaluation only. Complete research checkpoints and full
// WAL are always retained and used by all recovery/replay tests.
import (
	"bytes"
	"fmt"
	"path/filepath"
	"testing"
)

func s2r4OracleImage(t *testing.T, source, destination string) string {
	t.Helper()
	s := s1Snapshot(t, source)
	brain := append([]byte(nil), s.Brain...)
	// Old command-dedup entries are not inputs to fresh behavioral probes. Keep
	// the entire Brain, activation, clock, policy, sequence and learned state.
	s.ProcessedEvents = nil
	s.JournalPrefixHash = ""
	sum, e := checkpointChecksumForSchema(s)
	s1Check(t, e)
	s.Checksum = sum
	s1Write(t, destination, s)
	r := orderedFixtureClone(t, destination)
	b, e := r.brain.CanonicalJSON()
	s1Check(t, e)
	original := orderedFixtureClone(t, source)
	a, e := original.brain.CanonicalJSON()
	s1Check(t, e)
	if !bytes.Equal(a, b) || len(brain) == 0 || r.LastSequence() != s.Sequence {
		t.Fatal("oracle image changed neural state/sequence")
	}
	return destination
}

func TestS2R4OracleImageParity(t *testing.T) {
	_, work, pop := s2r4Paths(t)
	dir := filepath.Join(work, "training/seed-37/A32")
	candidate := s2r4OracleImage(t, filepath.Join(dir, "final.json"), filepath.Join(dir, "oracle-parity-candidate.json"))
	control := s2r4OracleImage(t, filepath.Join(dir, "retention-control.json"), filepath.Join(dir, "oracle-parity-control.json"))
	got := s2r3AE1(t, control, candidate, pop)
	var prior map[string]struct{ AE1 any }
	s1Read(t, filepath.Join(work, "persistence.json"), &prior)
	delta := s1AuditMax(s1AuditValue(t, got), s1AuditValue(t, prior["s37-A32"].AE1))
	if delta != 0 {
		t.Fatal("behavioral oracle changed after removing adapter dedup metadata", delta)
	}
	s1Write(t, filepath.Join(work, "ORACLE_IMAGE_PARITY.json"), map[string]any{"full_neural_state_preserved": true, "sequence_preserved": true, "original_checkpoint_and_WAL_untouched": true, "behavioral_oracle_max_delta": delta, "scope": "fresh unseen behavioral probes only; never recovery/continuation", "source": fmt.Sprint("seed-37/A32 original preregistered oracle")})
}
