package runtime

import (
	"testing"
	"time"

	"github.com/project-horizon/horizon-core/services/ai/knowledge"
	"github.com/project-horizon/horizon-core/services/ai/learning"
)

func TestVerifyKnowledgeClaimRequiresAcceptedEvidence(t *testing.T) {
	runtime := NewBrainRuntime(nil)
	node := runtime.DNF()
	_ = node

	at := time.Date(2026, 9, 20, 4, 30, 0, 0, time.UTC)
	grounded, err := runtime.GroundObservation("cahaya", "sensor-a", "text")
	if err != nil {
		t.Fatalf("GroundObservation() error = %v", err)
	}
	claim := KnowledgeClaim{
		NodeID: grounded.NodeID,
		Origin: KnowledgeOrigin{
			Source: "sensor-a",
			Modality: "text",
			ObservedAt: at,
			ExperienceID: "experience-1",
			IndependenceGroup: "source-a",
		},
		Status: grounded.Status,
	}

	rejected, decision, err := runtime.VerifyKnowledgeClaim(
		claim,
		learning.Evidence{
			Weight: .30, Confidence: .30, Reliability: .30,
			IndependentSources: 1,
		},
		at,
	)
	if err != nil {
		t.Fatalf("VerifyKnowledgeClaim() rejected evidence error = %v", err)
	}
	if decision != learning.PromotionCandidate {
		t.Fatalf("decision = %v, want candidate", decision)
	}
	if rejected.Verified {
		t.Fatal("claim became verified from insufficient evidence")
	}

	accepted, decision, err := runtime.VerifyKnowledgeClaim(
		claim,
		learning.Evidence{
			Weight: .80, Confidence: .80, Reliability: .80,
			IndependentSources: 2,
		},
		at,
	)
	if err != nil {
		t.Fatalf("VerifyKnowledgeClaim() accepted evidence error = %v", err)
	}
	if decision != learning.PromotionAccepted {
		t.Fatalf("decision = %v, want accepted", decision)
	}
	if !accepted.Verified {
		t.Fatal("accepted claim remained unverified")
	}
	if accepted.NodeID != grounded.NodeID {
		t.Fatalf("claim NodeID = %d, want %d", accepted.NodeID, grounded.NodeID)
	}
}

func TestKnowledgeClaimDoesNotBecomeVerifiedByGrounding(t *testing.T) {
	claim := KnowledgeClaim{
		NodeID: 1,
		Status: knowledge.GroundingCandidate,
		Origin: KnowledgeOrigin{Source: "sensor"},
	}
	if claim.Verified {
		t.Fatal("grounded candidate must begin unverified")
	}
}
