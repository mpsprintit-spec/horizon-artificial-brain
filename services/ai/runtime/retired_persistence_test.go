package runtime

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestRetiredCognitiveJournalCannotAppendOrReplay(t *testing.T) {
	at := time.Date(2026, 11, 15, 0, 0, 0, 0, time.UTC)
	cases := map[string]LoggedEvent{
		"old-command":      {SchemaVersion: 3, Type: "cognitive_model_command"},
		"smuggled-payload": {SchemaVersion: 3, Type: EventTypeCognitiveThink, Event: &Event{Cycles: 1, Timestamp: at}, RetiredCognition: json.RawMessage(`{"weights":[1,2]}`)},
	}
	for version := 5; version <= 13; version++ {
		cases[fmt.Sprintf("schema%d", version)] = LoggedEvent{SchemaVersion: version, Type: "cognitive_model_command"}
	}
	for name, event := range cases {
		t.Run(name, func(t *testing.T) {
			event.BrainIdentity, event.Sequence, event.Timestamp = BrainIdentity, 1, at
			path := filepath.Join(t.TempDir(), "events.jsonl")
			wal, err := OpenEventLog(path)
			if err != nil {
				t.Fatal(err)
			}
			if err := wal.Append(event); err == nil || !strings.Contains(err.Error(), "retired cognitive") {
				t.Fatalf("append accepted retired controller: %v", err)
			}
			if err := wal.Close(); err != nil {
				t.Fatal(err)
			}
			data, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			if len(data) != 0 {
				t.Fatal("refused append changed the WAL")
			}
			encoded, err := json.Marshal(event)
			if err != nil {
				t.Fatal(err)
			}
			encoded = append(encoded, '\n')
			if err := os.WriteFile(path, encoded, 0600); err != nil {
				t.Fatal(err)
			}
			if _, err := ReadEventLog(path); err == nil {
				t.Fatal("reader accepted a retired record")
			}
			rt := NewBrainRuntime(nil)
			before := continuationBytes(t, rt)
			if err := ReplayEventLog(rt, []LoggedEvent{event}); err == nil {
				t.Fatal("replay accepted a retired record")
			}
			if !bytes.Equal(before, continuationBytes(t, rt)) {
				t.Fatal("refused replay changed native state")
			}
			// Validate the entire input before applying an otherwise valid prefix.
			event.Sequence = 2
			prefix := LoggedEvent{SchemaVersion: 3, BrainIdentity: BrainIdentity, Sequence: 1, Timestamp: at, Type: EventTypeCognitiveThink, Event: &Event{Cycles: 1, Timestamp: at}}
			if err := ReplayEventLog(rt, []LoggedEvent{prefix, event}); err == nil {
				t.Fatal("mixed replay accepted a retired suffix")
			}
			if !bytes.Equal(before, continuationBytes(t, rt)) {
				t.Fatal("mixed replay applied a prefix before refusal")
			}
			retained, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			if !bytes.Equal(retained, encoded) {
				t.Fatal("refused recovery changed historical journal bytes")
			}
		})
	}
}

func TestRetiredCognitiveCheckpointRefusedWithoutMutation(t *testing.T) {
	source, ids := nativeFixture(t)
	cfg := nativeControlConfig(ids)
	nativeApply(t, source, NativeControlCommand{ID: "native-init", Initialize: &cfg})
	path := filepath.Join(t.TempDir(), "native.json")
	if err := source.Checkpoint(path); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var original BrainSnapshot
	if err := json.Unmarshal(data, &original); err != nil {
		t.Fatal(err)
	}
	cases := map[string]BrainSnapshot{}
	for version := 15; version <= 19; version++ {
		snapshot := original
		snapshot.SchemaVersion = version
		cases[fmt.Sprintf("checkpoint%d", version)] = snapshot
	}
	for _, nested := range []bool{false, true} {
		var brain map[string]json.RawMessage
		if err := json.Unmarshal(original.Brain, &brain); err != nil {
			t.Fatal(err)
		}
		if nested {
			brain["brain_state"] = json.RawMessage(`{"cognitive_model":{"weights":[1,2]}}`)
		} else {
			brain["cognitive_model"] = json.RawMessage(`{"weights":[1,2]}`)
		}
		snapshot := original
		snapshot.Brain, err = json.Marshal(brain)
		if err != nil {
			t.Fatal(err)
		}
		snapshot.Checksum, err = checkpointChecksumForSchema(snapshot)
		if err != nil {
			t.Fatal(err)
		}
		cases[fmt.Sprintf("smuggled/nested=%v", nested)] = snapshot
	}
	for name, snapshot := range cases {
		t.Run(name, func(t *testing.T) {
			encoded, err := json.Marshal(snapshot)
			if err != nil {
				t.Fatal(err)
			}
			path := filepath.Join(t.TempDir(), "historical.json")
			if err := os.WriteFile(path, encoded, 0600); err != nil {
				t.Fatal(err)
			}
			destination := NewBrainRuntime(nil)
			before := continuationBytes(t, destination)
			if err := destination.RestoreCheckpoint(path); err == nil {
				t.Fatal("restore accepted retired checkpoint state")
			}
			if !bytes.Equal(before, continuationBytes(t, destination)) {
				t.Fatal("refused checkpoint changed native state")
			}
			retained, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			if !bytes.Equal(retained, encoded) {
				t.Fatal("refused restore changed historical checkpoint bytes")
			}
		})
	}
}
