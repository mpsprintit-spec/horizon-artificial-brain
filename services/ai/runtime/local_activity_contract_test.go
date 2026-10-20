package runtime

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"testing"
	"time"

	"github.com/project-horizon/horizon-core/services/ai/knowledge"
	"github.com/project-horizon/horizon-core/services/ai/learning"
)

func TestLocalActivityPreservesActualPR33ReplayAndNextHistoricalThought(t *testing.T) {
	const fixture = "testdata/physical_memory_v1"
	data, err := os.ReadFile(filepath.Join(fixture, "reference.json"))
	if err != nil {
		t.Fatal(err)
	}
	var ref struct {
		Commit   string          `json:"commit"`
		Checksum string          `json:"checksum"`
		Next     CognitiveOutput `json:"next"`
		After    string          `json:"after_next_checksum"`
	}
	if err = json.Unmarshal(data, &ref); err != nil {
		t.Fatal(err)
	}
	if ref.Commit != "c6c7fc5ea36706d78e45064e31aee79ca23c557c" {
		t.Fatal("wrong fixture provenance")
	}
	data, err = os.ReadFile(filepath.Join(fixture, "commands.jsonl"))
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), "commands.jsonl")
	if err = os.WriteFile(path, data, 0600); err != nil {
		t.Fatal(err)
	}
	log, err := OpenEventLog(path)
	if err != nil {
		t.Fatal(err)
	}
	defer log.Close()
	genesis, err := RebuildRuntimeFromCommittedJournal(log, knowledge.NewBrain)
	if err != nil {
		t.Fatal(err)
	}
	suffix, err := RebuildRuntimeFromCheckpointAndJournal(log, filepath.Join(fixture, "prefix-checkpoint.json"))
	if err != nil {
		t.Fatal(err)
	}
	checksum := func(r *BrainRuntime) string {
		s, e := readCheckpointForTest(orderedSave(t, r))
		if e != nil {
			t.Fatal(e)
		}
		s.SchemaVersion = physicalMemoryBrainSnapshotSchemaVersion
		sum, e := checkpointChecksumV7(s)
		if e != nil {
			t.Fatal(e)
		}
		return sum
	}
	for _, r := range []*BrainRuntime{genesis, suffix} {
		if r.dynamicsEncoding != knowledge.CurrentDynamics || checksum(r) != ref.Checksum {
			t.Fatal("PR33 complete historical state or serving default changed")
		}
		r.SetClock(NewFixedClock(time.Date(2026, 10, 19, 11, 0, 0, 0, time.UTC)))
		r.dynamicsEncoding = knowledge.PhysicalMemoryDynamics
		next, err := r.CognitiveThink(1)
		r.dynamicsEncoding = knowledge.CurrentDynamics
		if err != nil {
			t.Fatal(err)
		}
		if !reflect.DeepEqual(next, ref.Next) || checksum(r) != ref.After {
			t.Fatal("PR33 historical next thought/state changed")
		}
	}
}

func TestLocalActivityMixedJournalRestartAndNextThought(t *testing.T) {
	at := time.Date(2026, 10, 20, 3, 0, 0, 987654321, time.UTC)
	live := NewBrainRuntime(knowledge.NewBrain())
	live.SetClock(NewFixedClock(at))
	log := openKnowledgeJournal(t)
	live.SetEventLog(log)
	live.dynamicsEncoding = knowledge.PhysicalMemoryDynamics
	if _, err := live.LearnExperience(learning.Experience{Sequence: []string{"cue", "old-target"}, Weight: .7, Confidence: .8}, at); err != nil {
		t.Fatal(err)
	}
	live.dynamicsEncoding = knowledge.CurrentDynamics
	for i := 0; i < 3; i++ {
		if _, _, err := NewCognitiveOrchestrator(live).ProcessObservation(Event{ID: "noise-" + string(rune('a'+i)), Timestamp: at.Add(time.Duration(i+1) * time.Millisecond), Cycles: 1}, ObservationInput{Source: "probe", Modality: "text", Tokens: []string{"unrelated"}}, nil); err != nil {
			t.Fatal(err)
		}
	}
	prefix := orderedSave(t, live)
	if _, err := live.LearnExperience(learning.Experience{Sequence: []string{"other-cue", "new-target"}, Weight: .7, Confidence: .8}, at.Add(time.Second)); err != nil {
		t.Fatal(err)
	}
	sensor := testSensorObservation()
	sensor.SampleID, sensor.Timestamp = "local-sensor", at.Add(2*time.Second)
	if _, err := live.ProcessSensorObservation(sensor); err != nil {
		t.Fatal(err)
	}
	live.SetClock(NewFixedClock(at.Add(3 * time.Second)))
	if _, err := live.CognitiveThink(1); err != nil {
		t.Fatal(err)
	}
	events, err := ReadEventLog(log.file.Name())
	if err != nil {
		t.Fatal(err)
	}
	if events[0].DynamicsEncoding != knowledge.PhysicalMemoryDynamics {
		t.Fatal("historical prefix lost its contract")
	}
	for _, event := range events[1:] {
		if event.DynamicsEncoding != knowledge.CurrentDynamics {
			t.Fatal("serving bypassed local activity version")
		}
	}
	genesis, err := RebuildRuntimeFromCommittedJournal(log, knowledge.NewBrain)
	if err != nil {
		t.Fatal(err)
	}
	suffix, err := RebuildRuntimeFromCheckpointAndJournal(log, prefix)
	if err != nil {
		t.Fatal(err)
	}
	standalone := NewBrainRuntime(knowledge.NewBrain())
	if err = standalone.RestoreCheckpoint(orderedSave(t, live)); err != nil {
		t.Fatal(err)
	}
	for _, r := range []*BrainRuntime{genesis, suffix, standalone} {
		assertCompleteContinuation(t, live, r)
	}
	live.SetEventLog(nil)
	var want CognitiveOutput
	for i, r := range []*BrainRuntime{live, genesis, suffix, standalone} {
		r.SetClock(NewFixedClock(at.Add(4 * time.Second)))
		out, err := r.CognitiveThink(1)
		if err != nil {
			t.Fatal(err)
		}
		if i == 0 {
			want = out
		} else if !reflect.DeepEqual(want, out) {
			t.Fatal("local plasticity next thought diverged after restart")
		}
	}
	for _, r := range []*BrainRuntime{genesis, suffix, standalone} {
		assertCompleteContinuation(t, live, r)
	}
}
