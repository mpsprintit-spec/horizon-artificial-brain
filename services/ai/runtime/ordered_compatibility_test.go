package runtime

import (
	"bytes"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"testing"
	"time"

	"github.com/project-horizon/horizon-core/services/ai/knowledge"
	"github.com/project-horizon/horizon-core/services/ai/learning"
)

func TestPreOrderedSchema2ReplayMatchesOriginalStateAndNextThought(t *testing.T) {
	const fixture = "testdata/legacy_dynamics_v2"
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
	if ref.Commit != "2f1537a957b15e400566dc5ec7dba9cd67312a44" {
		t.Fatal("unexpected fixture origin")
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
	for _, r := range []*BrainRuntime{genesis, suffix} {
		if r.dynamicsEncoding != knowledge.CurrentDynamics || r.surfaceEncoding != knowledge.WrittenTextEncoding {
			t.Fatal("replay leaked historical dynamics into serving")
		}
		if surfaceCheckpointChecksum(t, r) != ref.Checksum {
			t.Fatal("pre-ordered complete state differs from PR30")
		}
		r.SetClock(NewFixedClock(time.Date(2026, 10, 19, 10, 0, 0, 0, time.UTC)))
		// Pin the historical continuation to its recorded law. New serving
		// commands intentionally use the corrected passive-time dynamics.
		r.dynamicsEncoding = knowledge.LegacyDynamics
		next, err := r.CognitiveThink(1)
		r.dynamicsEncoding = knowledge.CurrentDynamics
		if err != nil {
			t.Fatal(err)
		}
		if !reflect.DeepEqual(next, ref.Next) || surfaceCheckpointChecksum(t, r) != ref.After {
			t.Fatal("pre-ordered next recurrent transition changed")
		}
	}
}

func TestOrderedJournalAndCheckpointContractsRejectDowngradesBeforeMutation(t *testing.T) {
	r := NewBrainRuntime(knowledge.NewBrain())
	log := openKnowledgeJournal(t)
	r.SetEventLog(log)
	at := time.Unix(100, 0).UTC()
	if _, err := r.LearnExperience(learning.Experience{ContextSequence: []string{"na", "lu"}, Sequence: []string{"cue", "target"}}, at); err != nil {
		t.Fatal(err)
	}
	events, err := ReadEventLog(log.file.Name())
	if err != nil {
		t.Fatal(err)
	}
	for _, mutation := range []func(*LoggedEvent){
		func(e *LoggedEvent) { e.DynamicsEncoding = "future" },
		func(e *LoggedEvent) { e.SchemaVersion = 2 },
		func(e *LoggedEvent) { e.DynamicsEncoding = knowledge.LegacyDynamics },
	} {
		packet, _ := json.Marshal(events)
		var changed []LoggedEvent
		_ = json.Unmarshal(packet, &changed)
		mutation(&changed[0])
		target := NewBrainRuntime(knowledge.NewBrain())
		before := continuationBytes(t, target)
		if err := ReplayEventLog(target, changed); err == nil {
			t.Fatal("invalid neural contract replayed")
		}
		if !bytes.Equal(before, continuationBytes(t, target)) {
			t.Fatal("invalid contract changed replay target")
		}
	}
	if _, err := r.ProcessObservationEnvelope(transactionObservation("trace", at.Add(time.Second), "na")); err != nil {
		t.Fatal(err)
	}
	path := orderedSave(t, r)
	snapshot, err := readCheckpointForTest(path)
	if err != nil {
		t.Fatal(err)
	}
	for _, mutation := range []func(*BrainSnapshot){
		func(s *BrainSnapshot) { s.SchemaVersion = 7 },
		func(s *BrainSnapshot) { s.Activation.LastTrace.Stimulus[0][0] = 999999 },
		func(s *BrainSnapshot) {
			s.Brain = bytes.Replace(s.Brain, []byte("ordered-observation-v1"), []byte("future-observation-v1"), 1)
		},
	} {
		packet, _ := json.Marshal(snapshot)
		var changed BrainSnapshot
		_ = json.Unmarshal(packet, &changed)
		mutation(&changed)
		changed.Checksum, err = checkpointChecksumForSchema(changed)
		if err != nil {
			t.Fatal(err)
		}
		packet, _ = json.Marshal(changed)
		bad := filepath.Join(t.TempDir(), "bad.json")
		if err = os.WriteFile(bad, packet, 0600); err != nil {
			t.Fatal(err)
		}
		target := NewBrainRuntime(knowledge.NewBrain())
		before := continuationBytes(t, target)
		if err = target.RestoreCheckpoint(bad); err == nil {
			t.Fatal("invalid ordered checkpoint restored")
		}
		if !bytes.Equal(before, continuationBytes(t, target)) {
			t.Fatal("invalid checkpoint changed target")
		}
	}
}

func TestOrderedInvalidInputsAndAppendFailureCannotMutate(t *testing.T) {
	at := time.Unix(100, 0).UTC()
	for _, input := range []struct {
		trace      *knowledge.ObservationTrace
		experience *learning.Experience
	}{
		{trace: &knowledge.ObservationTrace{Stimulus: [][]knowledge.NodeID{{99999}}}},
		{trace: &knowledge.ObservationTrace{Stimulus: [][]knowledge.NodeID{{}}}},
		{experience: &learning.Experience{NeuralSequence: [][]knowledge.NodeID{{99999}}}},
		{experience: &learning.Experience{ContextSequence: []string{""}, Sequence: []string{"na"}}},
	} {
		r := NewBrainRuntime(knowledge.NewBrain())
		log := openKnowledgeJournal(t)
		r.SetEventLog(log)
		before := continuationBytes(t, r)
		var err error
		if input.experience != nil {
			_, err = r.LearnExperience(*input.experience, at)
		} else {
			_, err = r.CognitiveProcess(Event{ID: "bad", Timestamp: at, Cycles: 1, NeuralTrace: input.trace})
		}
		if err == nil || r.RecoveryRequired() || !bytes.Equal(before, continuationBytes(t, r)) {
			t.Fatalf("invalid ordered input mutated state: %v", err)
		}
		data, _ := os.ReadFile(log.file.Name())
		if len(data) != 0 {
			t.Fatal("invalid command appended")
		}
	}
	r := NewBrainRuntime(knowledge.NewBrain())
	log := openKnowledgeJournal(t)
	r.SetEventLog(log)
	before := continuationBytes(t, r)
	log.beforeAppend = func(LoggedEvent) error { return errors.New("injected") }
	if _, err := r.LearnExperience(learning.Experience{ContextSequence: []string{"unseen-context"}, Sequence: []string{"unseen-cue", "unseen-target"}}, at); !errors.Is(err, ErrRuntimeRecoveryRequired) {
		t.Fatalf("append error=%v", err)
	}
	if !bytes.Equal(before, continuationBytes(t, r)) {
		t.Fatal("failed append grounded ordered learning input")
	}
}
