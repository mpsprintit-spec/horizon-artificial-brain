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

func checkpointRecoverySensor(id string, at time.Time, value float64) SensorObservation {
	sensor := testSensorObservation()
	sensor.SampleID = id
	sensor.Timestamp = at
	sensor.ChannelStates[0] = value
	return sensor
}

func TestCheckpointJournalRecoveryReplaysOnlyCommittedSuffix(t *testing.T) {
	path := filepath.Join(t.TempDir(), "journal.jsonl")
	checkpointPath := filepath.Join(t.TempDir(), "checkpoint.json")
	log, err := OpenEventLog(path)
	if err != nil { t.Fatal(err) }
	defer log.Close()

	at := time.Date(2026, 10, 12, 8, 0, 0, 0, time.UTC)
	r := NewBrainRuntime(knowledge.NewBootstrapBrain())
	r.SetClock(NewFixedClock(at))
	r.SetEventLog(log)
	orchestrator := NewCognitiveOrchestrator(r)
	first := checkpointRecoverySensor("sensor:a", at, 0.11)
	second := checkpointRecoverySensor("sensor:b", at.Add(time.Second), 0.22)
	third := checkpointRecoverySensor("sensor:c", at.Add(2*time.Second), 0.33)
	fourth := checkpointRecoverySensor("sensor:d", at.Add(3*time.Second), 0.44)

	for _, sensor := range []SensorObservation{first, second} {
		if _, err := orchestrator.ProcessSensorObservation(sensor); err != nil { t.Fatal(err) }
	}
	if err := r.Checkpoint(checkpointPath); err != nil { t.Fatalf("checkpoint: %v", err) }
	snapshot, err := readCheckpointForTest(checkpointPath)
	if err != nil { t.Fatal(err) }
	if snapshot.SchemaVersion != BrainSnapshotSchemaVersion || snapshot.JournalPrefixHash == "" ||
		snapshot.Sequence != 2 || !snapshot.LastObservationAt.Equal(second.Timestamp) ||
		len(snapshot.LastObservationPopulation) == 0 || snapshot.ProcessedEvents[second.SampleID] != 2 {
		t.Fatalf("checkpoint missing temporal or journal recovery metadata: %+v", snapshot)
	}
	for _, sensor := range []SensorObservation{third, fourth} {
		if _, err := orchestrator.ProcessSensorObservation(sensor); err != nil { t.Fatal(err) }
	}
	if r.LastSequence() != 4 { t.Fatalf("unexpected live sequence %d", r.LastSequence()) }

	recovered, err := RebuildRuntimeFromCheckpointAndJournal(log, checkpointPath)
	if err != nil { t.Fatalf("recover checkpoint+suffix: %v", err) }
	if recovered == r || recovered.brain == r.brain { t.Fatal("recovery reused live runtime or neural substrate") }
	if recovered.eventLog != nil { t.Fatal("recovered runtime unexpectedly published with event writer") }
	if recovered.LastSequence() != 4 { t.Fatalf("recovered wrong sequence %d", recovered.LastSequence()) }
	if !recovered.lastObservationAt.Equal(fourth.Timestamp) ||
		!reflect.DeepEqual(recovered.lastObservationPopulation, r.lastObservationPopulation) {
		t.Fatal("checkpoint suffix did not reproduce temporal state")
	}
	if !reflect.DeepEqual(recovered.processedEvents, r.processedEvents) {
		t.Fatal("checkpoint suffix did not restore processed event index")
	}
	expectedPath := filepath.Join(t.TempDir(), "expected.json")
	replayedPath := filepath.Join(t.TempDir(), "recovered.json")
	if err := r.Checkpoint(expectedPath); err != nil { t.Fatal(err) }
	if err := recovered.Checkpoint(replayedPath); err != nil { t.Fatal(err) }
	expected, err := readCheckpointForTest(expectedPath)
	if err != nil { t.Fatal(err) }
	actual, err := readCheckpointForTest(replayedPath)
	if err != nil { t.Fatal(err) }
	if !bytes.Equal(expected.Brain, actual.Brain) {
		t.Fatalf("recovered substrate differs from committed original: bytes %d vs %d", len(expected.Brain), len(actual.Brain))
	}
	if !reflect.DeepEqual(expected.Activation, actual.Activation) || !reflect.DeepEqual(expected.CognitiveState, actual.CognitiveState) {
		t.Fatal("recovered recurrent or cognitive state differs")
	}
	beforeNodes := len(recovered.brain.Registry.Nodes())
	if _, err := NewCognitiveOrchestrator(recovered).ProcessSensorObservation(fourth); err == nil {
		t.Fatal("checkpoint recovery allowed a duplicate sensor sample")
	}
	if len(recovered.brain.Registry.Nodes()) != beforeNodes {
		t.Fatal("duplicate sensor sample mutated recovered substrate")
	}
}

func TestCheckpointJournalRecoveryWithNoSuffix(t *testing.T) {
	path := filepath.Join(t.TempDir(), "journal.jsonl")
	checkpointPath := filepath.Join(t.TempDir(), "checkpoint.json")
	log, err := OpenEventLog(path)
	if err != nil { t.Fatal(err) }
	defer log.Close()
	at := time.Date(2026, 10, 12, 8, 0, 0, 0, time.UTC)
	r := NewBrainRuntime(knowledge.NewBootstrapBrain())
	r.SetEventLog(log)
	if _, err := NewCognitiveOrchestrator(r).ProcessSensorObservation(checkpointRecoverySensor("sensor:single", at, 0.1)); err != nil { t.Fatal(err) }
	if err := r.Checkpoint(checkpointPath); err != nil { t.Fatal(err) }
	recovered, err := RebuildRuntimeFromCheckpointAndJournal(log, checkpointPath)
	if err != nil { t.Fatal(err) }
	if recovered.LastSequence() != 1 || !recovered.lastObservationAt.Equal(at) {
		t.Fatalf("empty-suffix restore lost sensor history: seq=%d time=%s", recovered.LastSequence(), recovered.lastObservationAt)
	}
}

func TestCheckpointJournalRecoveryRejectsUnboundAndTamperedSnapshots(t *testing.T) {
	path := filepath.Join(t.TempDir(), "journal.jsonl")
	checkpointPath := filepath.Join(t.TempDir(), "checkpoint.json")
	log, err := OpenEventLog(path)
	if err != nil { t.Fatal(err) }
	defer log.Close()
	r := NewBrainRuntime(knowledge.NewBootstrapBrain())
	r.SetEventLog(log)
	sensor := checkpointRecoverySensor("sensor:anchored", time.Date(2026, 10, 12, 8, 0, 0, 0, time.UTC), 0.1)
	if _, err := NewCognitiveOrchestrator(r).ProcessSensorObservation(sensor); err != nil { t.Fatal(err) }
	if err := r.Checkpoint(checkpointPath); err != nil { t.Fatal(err) }
	data, err := os.ReadFile(checkpointPath)
	if err != nil { t.Fatal(err) }
	var original BrainSnapshot
	if err := json.Unmarshal(data, &original); err != nil { t.Fatal(err) }

	cases := []struct {
		name string
		change func(*BrainSnapshot) error
	}{
		{"temporal-tamper", func(s *BrainSnapshot) error { s.LastObservationAt = s.LastObservationAt.Add(time.Second); return nil }},
		{"processed-index-tamper", func(s *BrainSnapshot) error { s.ProcessedEvents = cloneProcessedEventIndex(s.ProcessedEvents); s.ProcessedEvents[sensor.SampleID] = 1234; return nil }},
		{"sequence-tamper", func(s *BrainSnapshot) error { s.Sequence = 0; return nil }},
		{"forged-anchor", func(s *BrainSnapshot) error {
			s.JournalPrefixHash = "not-the-real-journal-digest"
			var err error
			s.Checksum, err = checkpointChecksumV5(*s)
			return err
		}},
		{"unanchored", func(s *BrainSnapshot) error {
			s.JournalPrefixHash = ""
			var err error
			s.Checksum, err = checkpointChecksumV5(*s)
			return err
		}},
		{"future-sequence", func(s *BrainSnapshot) error {
			s.Sequence = 2
			var err error
			s.Checksum, err = checkpointChecksumV5(*s)
			return err
		}},
		{"legacy-schema", func(s *BrainSnapshot) error {
			s.SchemaVersion = legacyBrainSnapshotSchemaVersion
			s.JournalPrefixHash = ""
			s.LastObservationPopulation = nil
			s.LastObservationAt = time.Time{}
			s.ProcessedEvents = nil
			var err error
			s.Checksum, err = checkpointChecksum(s.Brain, s.Activation, s.CognitiveState, s.InquiryValence)
			return err
		}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			s := original
			if err := tc.change(&s); err != nil { t.Fatal(err) }
			changed, err := json.Marshal(s)
			if err != nil { t.Fatal(err) }
			if err := os.WriteFile(checkpointPath, changed, 0600); err != nil { t.Fatal(err) }
			recovered, err := RebuildRuntimeFromCheckpointAndJournal(log, checkpointPath)
			if err == nil || recovered != nil {
				t.Fatalf("unsafe checkpoint accepted: recovered=%v err=%v", recovered, err)
			}
		})
	}
}

func TestLegacyV4CheckpointCanRestoreButCannotOffsetReplay(t *testing.T) {
	path := filepath.Join(t.TempDir(), "legacy.json")
	r := NewBrainRuntime(knowledge.NewBootstrapBrain())
	if err := r.Checkpoint(path); err != nil { t.Fatal(err) }
	snapshot, err := readCheckpointForTest(path)
	if err != nil { t.Fatal(err) }
	snapshot.SchemaVersion = legacyBrainSnapshotSchemaVersion
	snapshot.ActionBindings = nil
	snapshot.EvidenceLedger = nil
	snapshot.JournalPrefixHash = ""
	snapshot.LastObservationPopulation = nil
	snapshot.LastObservationAt = time.Time{}
	snapshot.ProcessedEvents = nil
	snapshot.Checksum, err = checkpointChecksum(snapshot.Brain, snapshot.Activation, snapshot.CognitiveState, snapshot.InquiryValence)
	if err != nil { t.Fatal(err) }
	data, err := json.Marshal(snapshot)
	if err != nil { t.Fatal(err) }
	if err := os.WriteFile(path, data, 0600); err != nil { t.Fatal(err) }
	restored := NewBrainRuntime(knowledge.NewBootstrapBrain())
	if err := restored.RestoreCheckpoint(path); err != nil {
		t.Fatalf("legacy v4 checkpoint should remain restorable: %v", err)
	}
}

func TestPoisonedRuntimeCannotRestoreCheckpointInPlace(t *testing.T) {
	r := NewBrainRuntime(knowledge.NewBootstrapBrain())
	path := filepath.Join(t.TempDir(), "snapshot.json")
	if err := r.Checkpoint(path); err != nil { t.Fatal(err) }
	r.mu.Lock()
	r.recoveryRequired = true
	r.mu.Unlock()
	if err := r.RestoreCheckpoint(path); !errors.Is(err, ErrRuntimeRecoveryRequired) {
		t.Fatalf("poisoned runtime accepted in-place recovery: %v", err)
	}
	if !r.RecoveryRequired() { t.Fatal("poisoned state was cleared by failed recovery") }
}
