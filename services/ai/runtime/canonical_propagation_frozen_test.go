package runtime

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/project-horizon/horizon-core/services/ai/knowledge"
)

func TestCanonicalPropagationFrozenV1(t *testing.T) {
	dir := filepath.Join("testdata", "canonical_propagation_v1")
	var ref struct {
		Parent   string                     `json:"parent_commit"`
		Dynamics knowledge.DynamicsEncoding `json:"dynamics_encoding"`
		Journal  string                     `json:"journal_sha256"`
		Records  int                        `json:"records"`
		Initial  string                     `json:"initial_checkpoint_sha256"`
		Clock    time.Time                  `json:"initial_clock"`
		Future   []struct {
			At         time.Time `json:"at"`
			Output     string    `json:"output_sha256"`
			Checkpoint string    `json:"checkpoint_sha256"`
		} `json:"future"`
	}
	data, err := os.ReadFile(filepath.Join(dir, "reference.json"))
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(data, &ref); err != nil {
		t.Fatal(err)
	}
	if ref.Parent != "3f3716cb283372a18cf12a0ae17fdf2ff515118f" || ref.Dynamics != knowledge.CanonicalPropagationDynamics || len(ref.Future) != 8 {
		t.Fatal("wrong frozen propagation provenance")
	}
	digest := func(data []byte) string { sum := sha256.Sum256(data); return hex.EncodeToString(sum[:]) }
	path := filepath.Join(dir, "events.jsonl")
	data, err = os.ReadFile(path)
	if err != nil || digest(data) != ref.Journal {
		t.Fatal("frozen journal changed", err)
	}
	events, err := ReadEventLog(path)
	if err != nil || len(events) != ref.Records {
		t.Fatal("invalid frozen journal", err)
	}
	r := NewCanonicalPropagationRuntime()
	r.SetClock(NewFixedClock(ref.Clock))
	if err := ReplayEventLog(r, events); err != nil {
		t.Fatal(err)
	}
	if digest(canonicalPropagationCheckpointBytes(t, r)) != ref.Initial {
		t.Fatal("frozen complete initial checkpoint changed")
	}
	for i, future := range ref.Future {
		r.SetClock(NewFixedClock(future.At))
		out, err := r.CognitiveThink(1)
		if err != nil {
			t.Fatal(err)
		}
		data, err := json.Marshal(out)
		if err != nil {
			t.Fatal(err)
		}
		if digest(data) != future.Output || digest(canonicalPropagationCheckpointBytes(t, r)) != future.Checkpoint {
			t.Fatalf("frozen native output or complete checkpoint changed at step %d", i)
		}
	}
}
