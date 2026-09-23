package runtime

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/project-horizon/horizon-core/services/ai/knowledge"
)

func TestBrainRuntimeCheckpointRestoresInquiryValence(t *testing.T) {
	r := NewBrainRuntime(knowledge.NewBrain())
	r.RecordInquiryConsequence(InquiryFocus, 1)
	before := r.InquiryPriorExperience(InquiryFocus, 0.50)
	if before <= 0.50 {
		t.Fatalf("expected learned positive preference, got %v", before)
	}

	path := filepath.Join(t.TempDir(), "brain.json")
	if err := r.Checkpoint(path); err != nil {
		t.Fatalf("checkpoint: %v", err)
	}

	restored := NewBrainRuntime(knowledge.NewBrain())
	if err := restored.RestoreCheckpoint(path); err != nil {
		t.Fatalf("restore: %v", err)
	}
	after := restored.InquiryPriorExperience(InquiryFocus, 0.50)
	if after != before {
		t.Fatalf("restored inquiry preference mismatch: got %v want %v", after, before)
	}
}

func TestBrainRuntimeCheckpointChecksumCoversInquiryValence(t *testing.T) {
	r := NewBrainRuntime(knowledge.NewBrain())
	r.RecordInquiryConsequence(InquiryFocus, 1)
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
	snapshot.InquiryValence[InquiryFocus] = -1
	tampered, err := json.MarshalIndent(snapshot, "", "  ")
	if err != nil {
		t.Fatalf("encode tampered checkpoint: %v", err)
	}
	if err := os.WriteFile(path, tampered, 0600); err != nil {
		t.Fatalf("write tampered checkpoint: %v", err)
	}
	if err := r.RestoreCheckpoint(path); err == nil {
		t.Fatal("expected checksum failure after inquiry valence tampering")
	}
}

var _ = time.Time{}
