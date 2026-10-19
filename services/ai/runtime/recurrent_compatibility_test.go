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
)

func TestRecurrentDynamicsPreserveActualPR31ReplayAndNextThought(t *testing.T) {
	const fixture = "testdata/ordered_dynamics_v1"
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
	if ref.Commit != "0f9a4651f1d31e6f1ceeb0ca4d66fddafeafa4dc" {
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
		s, err := readCheckpointForTest(orderedSave(t, r))
		if err != nil {
			t.Fatal(err)
		}
		s.SchemaVersion = orderedBrainSnapshotSchemaVersion
		sum, err := checkpointChecksumV7(s)
		if err != nil {
			t.Fatal(err)
		}
		return sum
	}
	for _, r := range []*BrainRuntime{genesis, suffix} {
		if r.dynamicsEncoding != knowledge.CurrentDynamics || r.activation.SnapshotState().ContextState != nil {
			t.Fatal("historical replay gained invented context")
		}
		if checksum(r) != ref.Checksum {
			t.Fatal("PR31 complete logical state changed")
		}
		r.SetClock(NewFixedClock(time.Date(2026, 10, 19, 10, 0, 0, 0, time.UTC)))
		next, err := r.CognitiveThink(1)
		if err != nil {
			t.Fatal(err)
		}
		if !reflect.DeepEqual(next, ref.Next) || checksum(r) != ref.After {
			t.Fatal("PR31 next thought or post-thought state changed")
		}
	}
}

func TestRecurrentCheckpointRejectsDowngradeAndMalformedEligibilityBeforeMutation(t *testing.T) {
	r := orderedFixtureClone(t, contextTraining(t, 7))
	contextObservation(t, r, "context", "xa", time.Unix(101, 0))
	snapshot, err := readCheckpointForTest(orderedSave(t, r))
	if err != nil {
		t.Fatal(err)
	}
	for _, mutate := range []func(*BrainSnapshot){
		func(s *BrainSnapshot) { s.SchemaVersion = 8 },
		func(s *BrainSnapshot) { s.Activation.ContextState.Frames[0].Population[0] = 999999 },
		func(s *BrainSnapshot) { s.Activation.ContextState.Frames[0].Ordinal = 0 },
		func(s *BrainSnapshot) { s.Activation.ContextState.Frames[0].At = time.Time{} },
		func(s *BrainSnapshot) { s.Activation.ContextState.CueStart = 0 },
		func(s *BrainSnapshot) {
			s.Activation.ContextState.Frames = append(s.Activation.ContextState.Frames, s.Activation.ContextState.Frames[0])
		},
	} {
		packet, _ := json.Marshal(snapshot)
		var changed BrainSnapshot
		_ = json.Unmarshal(packet, &changed)
		mutate(&changed)
		changed.Checksum, err = checkpointChecksumV7(changed)
		if err != nil {
			t.Fatal(err)
		}
		packet, _ = json.Marshal(changed)
		path := filepath.Join(t.TempDir(), "bad.json")
		if err = os.WriteFile(path, packet, 0600); err != nil {
			t.Fatal(err)
		}
		target := NewBrainRuntime(knowledge.NewBrain())
		before := continuationBytes(t, target)
		if err = target.RestoreCheckpoint(path); err == nil {
			t.Fatal("invalid recurrent context restored")
		}
		if !bytes.Equal(before, continuationBytes(t, target)) || target.RecoveryRequired() {
			t.Fatal("invalid checkpoint changed the target")
		}
	}
}

func TestRecurrentEpisodeValidationAndAppendFailurePrecedeContextReset(t *testing.T) {
	at := time.Unix(101, 0).UTC()
	r := orderedFixtureClone(t, contextTraining(t, 17))
	contextEpisode(t, r, "context", "xa", "one", false, at)
	log := openKnowledgeJournal(t)
	r.SetEventLog(log)
	before := continuationBytes(t, r)
	for _, event := range []Event{
		{ID: "blank-episode", EpisodeID: " ", Timestamp: at.Add(time.Second), ResetContext: true},
		{ID: "mismatch", EpisodeID: "two", Timestamp: at.Add(time.Second), Observation: &ObservationEnvelope{EventID: "mismatch", SessionID: "one", Timestamp: at.Add(time.Second), Source: "probe", Modality: "text"}},
	} {
		_, _, err := NewCognitiveOrchestrator(r).ProcessObservation(event, ObservationInput{Source: "probe", Modality: "text", Tokens: []string{"unseen"}}, nil)
		if err == nil || r.RecoveryRequired() || !bytes.Equal(before, continuationBytes(t, r)) {
			t.Fatal("invalid episode mutated the neural/context state")
		}
	}
	log.beforeAppend = func(LoggedEvent) error { return errors.New("injected") }
	_, _, err := NewCognitiveOrchestrator(r).ProcessObservation(Event{ID: "reset", EpisodeID: "two", ResetContext: true, Timestamp: at.Add(time.Second)}, ObservationInput{Source: "probe", Modality: "text", Tokens: []string{"unseen"}}, nil)
	if !errors.Is(err, ErrRuntimeRecoveryRequired) || !bytes.Equal(before, continuationBytes(t, r)) {
		t.Fatal("uncertain append reset context or grounded an unseen surface")
	}
}

func TestRecurrentEpisodeControlsRequireTheirRecordedDynamics(t *testing.T) {
	e := LoggedEvent{SchemaVersion: 3, BrainIdentity: BrainIdentity, Sequence: 1, Type: EventTypeProcess, Timestamp: time.Unix(100, 0), Event: &Event{ID: "episode", EpisodeID: "one", ResetContext: true, Timestamp: time.Unix(100, 0)}}
	for _, dynamics := range []knowledge.DynamicsEncoding{knowledge.LegacyDynamics, knowledge.OrderedDynamics} {
		e.DynamicsEncoding = dynamics
		if err := validateJournalPayload(e); err == nil {
			t.Fatal("episode controls were silently accepted under older dynamics")
		}
	}
	e.DynamicsEncoding = knowledge.RecurrentDynamics
	if err := validateJournalPayload(e); err != nil {
		t.Fatal(err)
	}
	e.SchemaVersion = 2
	if err := validateJournalPayload(e); err == nil {
		t.Fatal("new dynamics were downgraded to schema 2")
	}
}

func TestRecurrentSensorIntentCannotCarryAnIgnoredResetEvenWithValidCommitHash(t *testing.T) {
	r := NewBrainRuntime(knowledge.NewBrain())
	log := openKnowledgeJournal(t)
	r.SetEventLog(log)
	if _, err := r.ProcessSensorObservation(testSensorObservation()); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(log.file.Name())
	if err != nil {
		t.Fatal(err)
	}
	records := bytes.Split(bytes.TrimSpace(data), []byte("\n"))
	if len(records) != 2 {
		t.Fatal("expected sensor intent and commit")
	}
	var intent, commit LoggedEvent
	if err = json.Unmarshal(records[0], &intent); err != nil {
		t.Fatal(err)
	}
	if err = json.Unmarshal(records[1], &commit); err != nil {
		t.Fatal(err)
	}
	intent.Event.ResetContext = true
	commit.IntentHash, err = sensorIntentHash(intent)
	if err != nil {
		t.Fatal(err)
	}
	a, _ := json.Marshal(intent)
	b, _ := json.Marshal(commit)
	path := filepath.Join(t.TempDir(), "ignored-reset.jsonl")
	if err = os.WriteFile(path, append(append(append(a, '\n'), b...), '\n'), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err = ReadEventLog(path); err == nil {
		t.Fatal("sensor reset was accepted but would be ignored on replay")
	}
	target := NewBrainRuntime(knowledge.NewBrain())
	before := continuationBytes(t, target)
	intent.Type = EventTypeProcess
	if err = ReplayEventLog(target, []LoggedEvent{intent}); err == nil || !bytes.Equal(before, continuationBytes(t, target)) {
		t.Fatal("ignored sensor reset was not rejected before replay mutation")
	}
}
