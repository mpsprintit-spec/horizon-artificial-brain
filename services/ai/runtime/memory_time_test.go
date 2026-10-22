package runtime

import (
	"bytes"
	"encoding/json"
	"errors"
	"math"
	"os"
	"path/filepath"
	"reflect"
	"testing"
	"time"

	"github.com/project-horizon/horizon-core/services/ai/knowledge"
	"github.com/project-horizon/horizon-core/services/ai/learning"
)

func memoryRuntime(t *testing.T) (*BrainRuntime, time.Time) {
	t.Helper()
	r := NewBrainRuntime(knowledge.NewBrain())
	at := time.Date(2026, 10, 20, 0, 0, 0, 123, time.UTC)
	r.SetClock(NewFixedClock(at))
	r.brain.BrainState.LearningPolicyState.CuriosityPressure = 0
	if _, err := r.LearnExperience(learning.Experience{Sequence: []string{"a", "z"}, Weight: .8, Confidence: .8}, at); err != nil {
		t.Fatal(err)
	}
	r.brain.BrainState.ExperienceTraces = []knowledge.ExperienceTrace{{CurrentAt: at, MemoryPriority: .6}}
	return r, at
}

func memorySynapse(r *BrainRuntime) *knowledge.Synapse {
	return r.brain.Registry.Get("a").Synapses[r.brain.Registry.Get("z").ID][0]
}

func TestPhysicalMemoryRuntimeIdleSchedulesAgree(t *testing.T) {
	for _, steps := range []int{1, 24, 96} {
		r, at := memoryRuntime(t)
		for i := 1; i <= steps; i++ {
			r.SetClock(NewFixedClock(at.Add(time.Duration(i) * 24 * time.Hour / time.Duration(steps))))
			if _, err := r.CognitiveThink(1); err != nil {
				t.Fatal(err)
			}
		}
		s := memorySynapse(r)
		if math.Abs(s.Weight-.8*math.Exp(-.4)) > 1e-12 || math.Abs(s.Confidence-.8*math.Exp(-.2)) > 1e-12 || math.Abs(r.brain.BrainState.ExperienceTraces[0].MemoryPriority-.6*math.Exp(-.4)) > 1e-12 {
			t.Fatal("idle schedule changed retention")
		}
		before, _ := r.brain.CanonicalJSON()
		for i := 0; i < 100; i++ {
			if _, err := r.CognitiveThink(1); err != nil {
				t.Fatal(err)
			}
		}
		after, _ := r.brain.CanonicalJSON()
		if !bytes.Equal(before, after) {
			t.Fatal("same-time idle thought compounded decay")
		}
	}
}

func TestPhysicalMemorySettlesBeforeReinforcement(t *testing.T) {
	r, at := memoryRuntime(t)
	if _, err := r.LearnExperience(learning.Experience{Sequence: []string{"a", "z"}, Weight: .4, Confidence: .4}, at.Add(time.Hour)); err != nil {
		t.Fatal(err)
	}
	want := (.8*math.Exp(-.4/24) + .4) / 2
	s := memorySynapse(r)
	if math.Abs(s.Weight-want) > 1e-12 || s.Dynamic.Frequency != 2 || !s.Dynamic.LastModification.Equal(at.Add(time.Hour)) {
		t.Fatalf("reinforcement used stale memory: got %.15f want %.15f", s.Weight, want)
	}
	if _, err := r.CognitiveThink(1); err != nil {
		t.Fatal(err)
	} // Fixed clock rolls back to t0.
	if s.Weight != want || !r.brain.BrainState.MemoryDecay.AppliedAt.Equal(at.Add(time.Hour)) {
		t.Fatal("rollback rewound/reapplied passive memory time")
	}
}

func TestPhysicalMemoryCheckpointAndJournalContinueExactly(t *testing.T) {
	at := time.Date(2026, 10, 20, 0, 0, 0, 987654321, time.UTC)
	live := NewBrainRuntime(knowledge.NewBrain())
	live.SetClock(NewFixedClock(at))
	log := openKnowledgeJournal(t)
	live.SetEventLog(log)
	if _, err := live.LearnExperience(learning.Experience{Sequence: []string{"x", "a", "b"}, Weight: .8, Confidence: .8}, at); err != nil {
		t.Fatal(err)
	}
	contextEpisode(t, live, "context", "x", "episode", false, at.Add(time.Hour))
	checkpoint := orderedSave(t, live)
	contextEpisode(t, live, "cue", "a", "episode", false, at.Add(2*time.Hour))
	sensor := testSensorObservation()
	sensor.SampleID, sensor.SessionID, sensor.Timestamp = "physical-sensor", "episode", at.Add(3*time.Hour)
	if _, err := live.ProcessSensorObservation(sensor); err != nil {
		t.Fatal(err)
	}
	genesis, err := RebuildRuntimeFromCommittedJournal(log, knowledge.NewBrain)
	if err != nil {
		t.Fatal(err)
	}
	suffix, err := RebuildRuntimeFromCheckpointAndJournal(log, checkpoint)
	if err != nil {
		t.Fatal(err)
	}
	standalone := NewBrainRuntime(knowledge.NewBrain())
	if err = standalone.RestoreCheckpoint(orderedSave(t, live)); err != nil {
		t.Fatal(err)
	}
	for _, other := range []*BrainRuntime{genesis, suffix, standalone} {
		assertCompleteContinuation(t, live, other)
	}
	live.SetEventLog(nil)
	var want CognitiveOutput
	for i, r := range []*BrainRuntime{live, genesis, suffix, standalone} {
		r.SetClock(NewFixedClock(at.Add(4 * time.Hour)))
		out, err := r.CognitiveThink(1)
		if err != nil {
			t.Fatal(err)
		}
		if i == 0 {
			want = out
		} else if !reflect.DeepEqual(want, out) {
			t.Fatal("next thought changed after clock recovery")
		}
		if !r.brain.BrainState.MemoryDecay.AppliedAt.Equal(at.Add(4 * time.Hour)) {
			t.Fatal("exact clock did not survive recovery")
		}
	}
	for _, r := range []*BrainRuntime{genesis, suffix, standalone} {
		assertCompleteContinuation(t, live, r)
	}
}

func TestPhysicalMemoryCheckpointRejectsInvalidClockBeforeMutation(t *testing.T) {
	r, _ := memoryRuntime(t)
	snapshot, err := readCheckpointForTest(orderedSave(t, r))
	if err != nil {
		t.Fatal(err)
	}
	// Mutate the semantic clock, independently of pretty/compact checkpoint
	// formatting. Assert the field exists so a future format change cannot
	// silently turn a negative test into an unchanged valid checkpoint.
	changeClock := func(field, value string) func(*BrainSnapshot) {
		return func(s *BrainSnapshot) {
			var document map[string]json.RawMessage
			if err := json.Unmarshal(s.Brain, &document); err != nil {
				t.Fatal(err)
			}
			var clock map[string]any
			if err := json.Unmarshal(document["memory_decay"], &clock); err != nil {
				t.Fatal(err)
			}
			if _, exists := clock[field]; !exists {
				t.Fatalf("clock field %s missing", field)
			}
			clock[field] = value
			changed, err := json.Marshal(clock)
			if err != nil {
				t.Fatal(err)
			}
			document["memory_decay"] = changed
			changed, err = json.Marshal(document)
			if err != nil {
				t.Fatal(err)
			}
			if bytes.Equal(changed, s.Brain) {
				t.Fatal("invalid clock mutation was a no-op")
			}
			s.Brain = changed
		}
	}
	for _, mutate := range []func(*BrainSnapshot){
		func(s *BrainSnapshot) { s.SchemaVersion = recurrentBrainSnapshotSchemaVersion },
		changeClock("applied_at", "0001-01-01T00:00:00Z"),
		changeClock("encoding", "future"),
	} {
		changed := snapshot
		mutate(&changed)
		changed.Checksum, err = checkpointChecksumV7(changed)
		if err != nil {
			t.Fatal(err)
		}
		packet, _ := json.Marshal(changed)
		path := filepath.Join(t.TempDir(), "bad.json")
		if err = os.WriteFile(path, packet, 0600); err != nil {
			t.Fatal(err)
		}
		target := NewBrainRuntime(knowledge.NewBrain())
		before := continuationBytes(t, target)
		if err = target.RestoreCheckpoint(path); err == nil || !bytes.Equal(before, continuationBytes(t, target)) || target.RecoveryRequired() {
			t.Fatal("invalid physical clock changed target")
		}
	}
}

func TestPhysicalMemoryUncertainAppendDoesNotAdvanceClock(t *testing.T) {
	r, at := memoryRuntime(t)
	log := openKnowledgeJournal(t)
	r.SetEventLog(log)
	before := continuationBytes(t, r)
	log.beforeAppend = func(LoggedEvent) error { return errors.New("injected") }
	if _, err := r.LearnExperience(learning.Experience{Sequence: []string{"a", "z"}}, at.Add(24*time.Hour)); !errors.Is(err, ErrRuntimeRecoveryRequired) {
		t.Fatal(err)
	}
	if !bytes.Equal(before, continuationBytes(t, r)) {
		t.Fatal("unacknowledged command decayed memory")
	}
}

func TestPhysicalMemoryReplaysActualPR32AndHistoricalNextThought(t *testing.T) {
	const fixture = "testdata/recurrent_dynamics_v1"
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
	if ref.Commit != "b2ea1f7a4395602fa9e1a067a5daf3ad2a458018" {
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
		s.SchemaVersion = recurrentBrainSnapshotSchemaVersion
		v, e := checkpointChecksumV7(s)
		if e != nil {
			t.Fatal(e)
		}
		return v
	}
	for _, r := range []*BrainRuntime{genesis, suffix} {
		if r.dynamicsEncoding != knowledge.CurrentDynamics || r.brain.BrainState.MemoryDecay != nil || checksum(r) != ref.Checksum {
			t.Fatal("PR32 historical state/default changed")
		}
		r.SetClock(NewFixedClock(time.Date(2026, 10, 19, 11, 0, 0, 0, time.UTC)))
		r.dynamicsEncoding = knowledge.RecurrentDynamics
		next, err := r.CognitiveThink(1)
		r.dynamicsEncoding = knowledge.CurrentDynamics
		if err != nil {
			t.Fatal(err)
		}
		if !reflect.DeepEqual(next, ref.Next) || checksum(r) != ref.After {
			t.Fatal("PR32 historical next thought changed")
		}
		// The first new-law command adopts the already-paid legacy time.
		r.SetClock(NewFixedClock(time.Date(2026, 10, 19, 12, 0, 0, 0, time.UTC)))
		if _, err = r.CognitiveThink(1); err != nil {
			t.Fatal(err)
		}
		if r.brain.BrainState.MemoryDecay == nil {
			t.Fatal("new serving command did not adopt physical time")
		}
	}
}
