package runtime

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"testing"
	"time"

	"github.com/project-horizon/horizon-core/services/ai/knowledge"
)

func TestLearnedContinuationFrozenV1(t *testing.T) {
	dir := filepath.Join("testdata", "learned_continuation_v1")
	var ref struct {
		Parent     string              `json:"parent_commit"`
		Encoding   string              `json:"encoding"`
		Population []knowledge.NodeID  `json:"population"`
		At         time.Time           `json:"at"`
		Readout    ContinuationReadout `json:"readout"`
		Canonical  string              `json:"canonical_sha256"`
		Journal    string              `json:"journal_sha256"`
		Records    int                 `json:"records"`
	}
	data, err := os.ReadFile(filepath.Join(dir, "reference.json"))
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(data, &ref); err != nil {
		t.Fatal(err)
	}
	digest := func(data []byte) string { sum := sha256.Sum256(data); return hex.EncodeToString(sum[:]) }
	journalPath := filepath.Join(dir, "events.jsonl")
	data, err = os.ReadFile(journalPath)
	if err != nil || digest(data) != ref.Journal {
		t.Fatalf("frozen journal changed: %v", err)
	}
	events, err := ReadEventLog(journalPath)
	if err != nil || len(events) != ref.Records || ref.Encoding != ContinuationEncodingV1 || ref.Parent != "99820e0c7912440f84fa4e0a223a3880b88bbf7b" {
		t.Fatalf("invalid frozen contract: %v", err)
	}
	r := NewBrainRuntime(knowledge.NewBrain())
	if err := ReplayEventLog(r, events); err != nil {
		t.Fatal(err)
	}
	before := continuationBytes(t, r)
	canonical, err := r.brain.CanonicalJSON()
	if err != nil || digest(canonical) != ref.Canonical {
		t.Fatalf("frozen neural state changed: %v", err)
	}
	out, err := r.ReadLearnedContinuation(&knowledge.ObservationTrace{Stimulus: [][]knowledge.NodeID{ref.Population}}, 8, ref.At)
	if err != nil || !reflect.DeepEqual(out, ref.Readout) || !bytes.Equal(before, continuationBytes(t, r)) {
		t.Fatalf("frozen v1 readout or read-only state changed: %+v, %v", out, err)
	}
}
