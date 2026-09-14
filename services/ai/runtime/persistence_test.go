package runtime

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/project-horizon/horizon-core/services/ai/knowledge"
	"github.com/project-horizon/horizon-core/services/ai/learning"
)

func TestBrainRuntimeCheckpointAndRestore(t *testing.T) {
	brain := knowledge.NewBrain()
	saya := brain.Store("saya")
	ingin := brain.Store("ingin")
	brain.Connect(saya, ingin, 0.8, 0.7, false)
	r := NewBrainRuntime(brain)

	if _, err := r.LearnExperience(testExperience(), time.Now().UTC()); err != nil {
		t.Fatalf("learn experience: %v", err)
	}
	before := r.LastSequence()

	path := filepath.Join(t.TempDir(), "brain.json")
	if err := r.Checkpoint(path); err != nil {
		t.Fatalf("checkpoint: %v", err)
	}

	restoredBrain := knowledge.NewBrain()
	restored := NewBrainRuntime(restoredBrain)
	if err := restored.RestoreCheckpoint(path); err != nil {
		t.Fatalf("restore: %v", err)
	}
	if restored.LastSequence() != before {
		t.Fatalf("sequence mismatch: got %d want %d", restored.LastSequence(), before)
	}
	if restoredBrain.Fetch("saya") == nil || restoredBrain.Fetch("ingin") == nil {
		t.Fatal("restored brain is missing nodes")
	}
	if len(restoredBrain.Fetch("saya").OutboundAll()) == 0 {
		t.Fatal("restored brain is missing dynamic connection")
	}
}

func TestBrainRuntimeRejectsCorruptCheckpoint(t *testing.T) {
	brain := knowledge.NewBrain()
	brain.Store("air")
	r := NewBrainRuntime(brain)
	path := filepath.Join(t.TempDir(), "brain.json")
	if err := r.Checkpoint(path); err != nil {
		t.Fatalf("checkpoint: %v", err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read checkpoint: %v", err)
	}
	data[len(data)-2] ^= 1
	if err := os.WriteFile(path, data, 0600); err != nil {
		t.Fatalf("corrupt checkpoint: %v", err)
	}
	if err := r.RestoreCheckpoint(path); err == nil {
		t.Fatal("expected checksum failure")
	}
}

func TestBrainRuntimeRejectsActivationStateTampering(t *testing.T) {
	brain := knowledge.NewBrain()
	brain.Store("saya")
	r := NewBrainRuntime(brain)
	if _, _, err := r.Process(Event{ID: "activation-check", Stimulus: []string{"saya"}, Cycles: 1, Timestamp: time.Now().UTC()}); err != nil {
		t.Fatalf("process: %v", err)
	}

	path := filepath.Join(t.TempDir(), "brain.json")
	if err := r.Checkpoint(path); err != nil {
		t.Fatalf("checkpoint: %v", err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read checkpoint: %v", err)
	}
	var snapshot BrainSnapshot
	if err := json.Unmarshal(data, &snapshot); err != nil {
		t.Fatalf("decode checkpoint: %v", err)
	}
	if len(snapshot.Activation.InternalState) == 0 {
		t.Fatal("checkpoint did not contain recurrent activation state")
	}
	for nodeID, value := range snapshot.Activation.InternalState {
		snapshot.Activation.InternalState[nodeID] = value + 0.001
		break
	}
	tampered, err := json.MarshalIndent(snapshot, "", "  ")
	if err != nil {
		t.Fatalf("encode tampered checkpoint: %v", err)
	}
	if err := os.WriteFile(path, tampered, 0600); err != nil {
		t.Fatalf("write tampered checkpoint: %v", err)
	}

	if err := r.RestoreCheckpoint(path); err == nil {
		t.Fatal("expected checksum failure after activation state tampering")
	}
}

func testExperience() learning.Experience {
	return learning.Experience{Sequence: []string{"saya", "ingin", "belajar"}, Weight: 0.8, Confidence: 0.6, ExperienceID: "checkpoint-test", Source: "test", Modality: "language", Reliability: 0.8, IndependenceGroup: "checkpoint"}
}
