package runtime

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	k "github.com/project-horizon/horizon-core/services/ai/knowledge"
)

// Acquisition is not repeated: all probes start at the authenticated E1 state.
// Readout/feedback probes mutate disposable evaluation copies only. The saved
// checkpoint and journal bytes must remain unchanged throughout this test.
func TestNativeIntegrationSavedContinuation(t *testing.T) {
	root := os.Getenv("HORIZON_NATIVE_CONTINUATION_DIR")
	if root == "" {
		t.Skip("prepare the saved PR69 lineage with prepare_native_continuation.py")
	}
	cp := filepath.Join(root, "07-E1-restarted.json")
	cpBytes, err := os.ReadFile(cp)
	if err != nil {
		t.Fatal(err)
	}
	journalBytes, err := os.ReadFile(filepath.Join(root, "events.jsonl"))
	if err != nil {
		t.Fatal(err)
	}
	var snapshot BrainSnapshot
	if err = json.Unmarshal(cpBytes, &snapshot); err != nil {
		t.Fatal(err)
	}
	var pop [][]k.NodeID
	data, err := os.ReadFile(filepath.Join(root, "populations.json"))
	if err != nil {
		t.Fatal(err)
	}
	if err = json.Unmarshal(data, &pop); err != nil || len(pop) != 9 {
		t.Fatal("invalid population lineage", err)
	}
	dir := t.TempDir()
	path := filepath.Join(dir, "events.jsonl")
	if err = os.WriteFile(path, journalBytes, 0600); err != nil {
		t.Fatal(err)
	}
	wal, err := OpenEventLog(path)
	if err != nil {
		t.Fatal(err)
	}
	defer wal.Close()
	r, err := RebuildRuntimeFromCheckpointAndJournal(wal, cp)
	if err != nil {
		t.Fatal(err)
	}
	owner := r.brain
	if r.LastSequence() != 75331 || owner.NativeControlSnapshot().Updates != 10752 || snapshot.Checksum != "68a1518ebf1eea887041a339851754880d7638866a135361693205bab4bf4b82" {
		t.Fatal("wrong saved lineage")
	}
	if r.activation.Memory != owner || r.learning.Kb != owner || r.dnf.Brain() != owner {
		t.Fatal("canonical owner split")
	}
	initial := continuationBytes(t, r)
	genesis, err := RebuildRuntimeFromCommittedJournal(wal, k.NewBrain)
	if err != nil || !bytes.Equal(initial, continuationBytes(t, genesis)) {
		t.Fatal("saved genesis replay differs", err)
	}
	r.SetEventLog(wal)
	r.SetClock(NewFixedClock(flexAt(r)))
	roundtrip := filepath.Join(dir, "roundtrip.json")
	if err = r.Checkpoint(roundtrip); err != nil {
		t.Fatal(err)
	}
	var written BrainSnapshot
	data, err = os.ReadFile(roundtrip)
	if err != nil {
		t.Fatal(err)
	}
	if err = json.Unmarshal(data, &written); err != nil || written.Checksum != snapshot.Checksum {
		t.Fatal("old checksum law changed", err)
	}
	restored, err := RebuildRuntimeFromCheckpointAndJournal(wal, roundtrip)
	if err != nil || !bytes.Equal(initial, continuationBytes(t, restored)) {
		t.Fatal("roundtrip differs", err)
	}
	retention := e1Retention(t, cp, pop)
	e1 := e1Measure(t, cp, pop)
	if e1.Accuracy != 35.0/36 || retention["C"].(*flexPolicyMetrics).Accuracy != 1 {
		t.Fatal("learned choice regression")
	}
	if _, err = r.CognitiveThink(1); err != nil {
		t.Fatal(err)
	}
	continued := continuationBytes(t, r)
	suffix, err := RebuildRuntimeFromCheckpointAndJournal(wal, cp)
	if err != nil || !bytes.Equal(continued, continuationBytes(t, suffix)) {
		t.Fatal("thought suffix differs", err)
	}
	if owner != r.brain || r.activation.Memory != owner || r.learning.Kb != owner || r.dnf.Brain() != owner {
		t.Fatal("owner changed on continuation")
	}
	actual := map[string]any{"initial": k.NativeDigest(json.RawMessage(initial)), "next_thought": k.NativeDigest(json.RawMessage(continued)), "retention": retention, "E1": e1, "sequence": uint64(75331), "native_updates": uint64(10752), "checksum": written.Checksum}
	encoded, err := json.MarshalIndent(actual, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	if capture := os.Getenv("HORIZON_NATIVE_REFERENCE_CAPTURE"); capture != "" {
		// One development oracle run on unchanged PR69; final CI never sets it.
		file, err := os.OpenFile(capture, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
		if err != nil {
			t.Fatal(err)
		}
		_, err = file.Write(encoded)
		file.Close()
		if err != nil {
			t.Fatal(err)
		}
	} else {
		expected, err := os.ReadFile("testdata/native_saved_e1_expected.json")
		if err != nil || !bytes.Equal(bytes.TrimSpace(expected), encoded) {
			t.Fatal("saved readouts/retention or full continuation differ from PR69 oracle", err)
		}
	}
	for name, before := range map[string][]byte{"07-E1-restarted.json": cpBytes, "events.jsonl": journalBytes} {
		after, err := os.ReadFile(filepath.Join(root, name))
		if err != nil || !bytes.Equal(before, after) {
			t.Fatal("source lineage was modified", name, err)
		}
	}
	t.Log("saved A–E1 seq75331/updates10752: exact genesis, suffix, checksum, retention and next-thought state; no acquisition")
}
