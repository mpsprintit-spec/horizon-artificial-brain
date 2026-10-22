package runtime

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"testing"
	"time"

	"github.com/project-horizon/horizon-core/services/ai/knowledge"
	"github.com/project-horizon/horizon-core/services/ai/learning"
)

func TestObservedFeedbackPreservesActualPR34ReplayAndNextHistoricalThought(t *testing.T) {
	const fixture = "testdata/local_activity_v1"
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
	if ref.Commit != "dadbd38e0e4aa06879c4b9c3d799199253efdcd9" {
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
			t.Fatal("PR34 complete historical state or serving default changed")
		}
		r.SetClock(NewFixedClock(time.Date(2026, 10, 19, 11, 0, 0, 0, time.UTC)))
		r.dynamicsEncoding = knowledge.LocalActivityDynamics
		next, err := r.CognitiveThink(1)
		r.dynamicsEncoding = knowledge.CurrentDynamics
		if err != nil {
			t.Fatal(err)
		}
		if !reflect.DeepEqual(next, ref.Next) || checksum(r) != ref.After {
			t.Fatal("PR34 historical next thought/state changed")
		}
	}
}

func TestObservedFeedbackPendingCueRestartAndNextOutcome(t *testing.T) {
	r := NewBrainRuntime(knowledge.NewBrain())
	at := time.Date(2026, 10, 20, 4, 0, 0, 0, time.UTC)
	r.SetClock(NewFixedClock(at))
	log := openKnowledgeJournal(t)
	r.SetEventLog(log)
	for _, word := range []string{"a", "b", "c"} {
		if _, err := r.LearnExperience(learning.Experience{Sequence: []string{word}}, at); err != nil {
			t.Fatal(err)
		}
	}
	contextEpisode(t, r, "cue", "a", "one", false, at)
	prefix := orderedSave(t, r) // The outcome has not arrived yet.
	r.SetClock(NewFixedClock(at))
	for i := 0; i < 4; i++ {
		if _, err := r.CognitiveThink(1); err != nil {
			t.Fatal(err)
		}
	}
	contextEpisode(t, r, "outcome", "c", "one", false, at.Add(time.Second))
	genesis, err := RebuildRuntimeFromCommittedJournal(log, knowledge.NewBrain)
	if err != nil {
		t.Fatal(err)
	}
	suffix, err := RebuildRuntimeFromCheckpointAndJournal(log, prefix)
	if err != nil {
		t.Fatal(err)
	}
	standalone := NewBrainRuntime(knowledge.NewBrain())
	if err = standalone.RestoreCheckpoint(orderedSave(t, r)); err != nil {
		t.Fatal(err)
	}
	for _, other := range []*BrainRuntime{genesis, suffix, standalone} {
		assertCompleteContinuation(t, r, other)
	}
	r.SetEventLog(nil)
	var want CognitiveOutput
	for i, candidate := range []*BrainRuntime{r, genesis, suffix, standalone} {
		candidate.SetClock(NewFixedClock(at.Add(time.Second)))
		out, err := candidate.CognitiveThink(1)
		if err != nil {
			t.Fatal(err)
		}
		if i == 0 {
			want = out
		} else if !reflect.DeepEqual(want, out) {
			t.Fatal("thought changed after feedback restart")
		}
		contextEpisode(t, candidate, "next", "b", "one", false, at.Add(2*time.Second))
	}
	for _, other := range []*BrainRuntime{genesis, suffix, standalone} {
		assertCompleteContinuation(t, r, other)
	}
}

func TestObservedFeedbackCheckpointValidationPrecedesMutation(t *testing.T) {
	r, at := reversalRuntime(t, 7)
	contextEpisode(t, r, "cue", "a", "one", false, at)
	snapshot, err := readCheckpointForTest(orderedSave(t, r))
	if err != nil {
		t.Fatal(err)
	}
	for _, mutate := range []func(*BrainSnapshot){
		func(s *BrainSnapshot) { s.SchemaVersion = physicalMemoryBrainSnapshotSchemaVersion },
		func(s *BrainSnapshot) { s.Activation.ObservationPrediction.At = time.Time{} },
		func(s *BrainSnapshot) {
			s.Activation.ObservationPrediction.At = s.Activation.ObservationPrediction.At.Add(time.Hour)
		},
		func(s *BrainSnapshot) { s.Activation.ObservationPrediction.EpisodeID = "other" },
		func(s *BrainSnapshot) {
			s.Activation.ObservationPrediction.Source = map[knowledge.NodeID]float64{999999: 1}
		},
		func(s *BrainSnapshot) {
			for id := range s.Activation.ObservationPrediction.Source {
				s.Activation.ObservationPrediction.Source[id] = .5
			}
		},
		func(s *BrainSnapshot) { s.Activation.ObservationPrediction.Prediction[999999] = 1 },
		func(s *BrainSnapshot) {
			s.Activation.ObservationPrediction.Patterns = map[knowledge.PatternID]float64{999999: 1}
		},
		func(s *BrainSnapshot) { s.Activation.LastTrace = nil },
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
		if err = target.RestoreCheckpoint(path); err == nil || !bytes.Equal(before, continuationBytes(t, target)) || target.RecoveryRequired() {
			t.Fatal("invalid observed provenance changed restore target")
		}
	}
}
