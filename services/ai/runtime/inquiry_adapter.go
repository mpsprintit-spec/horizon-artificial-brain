package runtime

import (
	"errors"
	"fmt"
	"time"

	"github.com/project-horizon/horizon-core/services/ai/knowledge"
)

// InquiryRequest is the adapter-facing representation of an internal
// information-seeking decision. It describes what the brain selected; it does
// not authorize execution or directly invoke a device.
type InquiryRequest struct {
	RequestID string
	BrainIdentity string
	Sequence uint64
	Action InquiryAction
	Uncertainty float64
	ExpectedInformationGain float64
	Score float64
	RequiresAuthorization bool
	TargetNodeIDs []knowledge.NodeID
	CreatedAt time.Time
}

// InquiryAdapter is an external observation boundary. Horizon can hand a
// selected InquiryRequest to an adapter, but the runtime never invokes an
// adapter automatically. The adapter returns an observation that must pass
// through the normal grounding/cognition path.
type InquiryAdapter interface {
	Observe(InquiryRequest) (ObservationInput, error)
}

func (r *BrainRuntime) BuildInquiryRequest(agenda InquiryAgenda) (InquiryRequest, error) {
	if r == nil || r.brain == nil {
		return InquiryRequest{}, errors.New("brain runtime is not initialized")
	}
	if agenda.BrainIdentity != BrainIdentity {
		return InquiryRequest{}, errors.New("inquiry agenda belongs to a different brain")
	}
	if agenda.Selected == nil {
		return InquiryRequest{}, errors.New("inquiry agenda has no selected action")
	}
	if agenda.Sequence == 0 {
		return InquiryRequest{}, errors.New("inquiry agenda sequence is required")
	}
	if agenda.CreatedAt.IsZero() {
		return InquiryRequest{}, errors.New("inquiry agenda timestamp is required")
	}

	r.mu.Lock()
	targets := append([]knowledge.NodeID(nil), r.lastCognitiveState.ActiveNodeIDs...)
	r.mu.Unlock()

	requestID := fmt.Sprintf("inquiry-%d-%s", agenda.Sequence, agenda.Selected.Action)
	return InquiryRequest{
		RequestID: requestID,
		BrainIdentity: BrainIdentity,
		Sequence: agenda.Sequence,
		Action: agenda.Selected.Action,
		Uncertainty: clamp01(agenda.Uncertainty),
		ExpectedInformationGain: clamp01(agenda.Selected.InformationValue),
		Score: clamp01(agenda.Selected.Score / 1.85),
		RequiresAuthorization: agenda.Selected.RequiresAuthorization,
		TargetNodeIDs: targets,
		CreatedAt: agenda.CreatedAt.UTC(),
	}, nil
}
