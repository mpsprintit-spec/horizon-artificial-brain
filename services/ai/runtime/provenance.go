package runtime

import (
	"errors"
	"time"

	"github.com/project-horizon/horizon-core/services/ai/knowledge"
	"github.com/project-horizon/horizon-core/services/ai/learning"
)

// KnowledgeOrigin identifies the source and temporal lineage of a neural claim.
type KnowledgeOrigin struct {
	Source            string
	Modality          string
	ObservedAt        time.Time
	ExperienceID      string
	IndependenceGroup string
}

// KnowledgeClaim is a provenance record, not a second memory store.
// Verified can only be set through the explicit verification boundary.
type KnowledgeClaim struct {
	NodeID     knowledge.NodeID
	Origin     KnowledgeOrigin
	Verified   bool
	Status     knowledge.GroundingStatus
}

// KnowledgeInjectionEvent describes how observed information entered the
// canonical Brain substrate.
type KnowledgeInjectionEvent struct {
	BrainIdentity  string
	Sequence       uint64
	Timestamp      time.Time
	Claims         []KnowledgeClaim
	ExperienceID   string
	ChangedNodeIDs []knowledge.NodeID
}

// InternalChangeAwareness exposes state and knowledge changes to the cognitive
// boundary without creating a parallel memory subsystem.
type InternalChangeAwareness struct {
	BrainIdentity   string
	Sequence        uint64
	Timestamp       time.Time
	StateDelta      CognitiveStateDelta
	KnowledgeChanges []KnowledgeInjectionEvent
}

// VerifyKnowledgeClaim is the explicit provenance-to-trust boundary.
// Grounding alone never marks a claim as verified.
func (r *BrainRuntime) VerifyKnowledgeClaim(claim KnowledgeClaim, evidence learning.Evidence, at time.Time) (KnowledgeClaim, learning.PromotionDecision, error) {
	if r == nil || r.promotion == nil {
		return claim, learning.PromotionCandidate, errors.New("promotion engine is not initialized")
	}
	if claim.NodeID == 0 {
		return claim, learning.PromotionCandidate, errors.New("knowledge claim node ID is required")
	}
	if at.IsZero() {
		at = r.now()
	}
	decision, err := r.promotion.Apply(claim.NodeID, evidence, at)
	if err != nil {
		return claim, decision, err
	}
	if decision == learning.PromotionAccepted {
		claim.Verified = true
	}
	return claim, decision, nil
}
