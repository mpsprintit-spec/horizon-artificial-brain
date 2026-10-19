package runtime

import (
	"errors"
	"time"

	"github.com/project-horizon/horizon-core/services/ai/knowledge"
	"github.com/project-horizon/horizon-core/services/ai/learning"
)

type CognitiveOrchestrator struct { Runtime *BrainRuntime }
func NewCognitiveOrchestrator(runtime *BrainRuntime) *CognitiveOrchestrator { return &CognitiveOrchestrator{Runtime: runtime} }

type ObservationInput struct { Source string; Modality string; Tokens []string; ContextTokens []string; DataTokens []string }

// ProcessObservation is the canonical integration boundary. Context and data
// are first grounded into the same Brain substrate, then the event is processed
// and interpreted. Grounding is provenance only: it never authorizes action and
// never promotes a candidate observation into trusted knowledge by itself.
func (o *CognitiveOrchestrator) ProcessObservation(event Event, observation ObservationInput, experience *learning.Experience) (CognitiveInterpretation, bool, error) {
	return o.processObservation(event, observation, experience, true)
}

// processObservation performs the observation -> grounding -> cognition path.
// planInquiry controls whether this observation starts a new inquiry trajectory.
// Inquiry feedback uses false so the completed inquiry is closed before the
// next inquiry is planned.
func (o *CognitiveOrchestrator) processObservation(event Event, observation ObservationInput, experience *learning.Experience, planInquiry bool) (CognitiveInterpretation, bool, error) {
	if o == nil || o.Runtime == nil { return CognitiveInterpretation{}, false, errors.New("cognitive orchestrator is not initialized") }
	return o.Runtime.processObservationCommand(event, ObservationCommand{Input: observation, Experience: experience, PlanInquiry: planInquiry})
}

// RequestInquiry converts the selected internal policy decision into an adapter
// request. Creating the request is not execution and does not authorize an
// external action.
func (o *CognitiveOrchestrator) RequestInquiry(agenda InquiryAgenda) (InquiryRequest, error) {
	if o == nil || o.Runtime == nil {
		return InquiryRequest{}, errors.New("cognitive orchestrator is not initialized")
	}
	return o.Runtime.BuildInquiryRequest(agenda)
}

// ProcessInquiryObservation closes the selected inquiry only after an adapter
// returns an observation. The observed information gain is derived from the
// inquiry uncertainty and the uncertainty produced by the returned observation.
// A new inquiry is planned only after the previous trajectory is closed.
func (o *CognitiveOrchestrator) ProcessInquiryObservation(request InquiryRequest, event Event, observation ObservationInput, experience *learning.Experience) (CognitiveInterpretation, bool, error) {
	if o == nil || o.Runtime == nil {
		return CognitiveInterpretation{}, false, errors.New("cognitive orchestrator is not initialized")
	}
	if request.BrainIdentity != BrainIdentity {
		return CognitiveInterpretation{}, false, errors.New("inquiry request belongs to a different brain")
	}
	if request.RequestID == "" {
		return CognitiveInterpretation{}, false, errors.New("inquiry request ID is required")
	}
	if request.Action == "" {
		return CognitiveInterpretation{}, false, errors.New("inquiry request action is required")
	}
	return o.Runtime.processObservationCommand(event, ObservationCommand{Input: observation, Experience: experience, PlanInquiry: true, Inquiry: &request})
}

func (o *CognitiveOrchestrator) LearnFromOutcome(outcome OutcomeEvent) (uint64, error) { if o == nil || o.Runtime == nil { return 0, errors.New("cognitive orchestrator is not initialized") }; return o.Runtime.ObserveOutcome(outcome) }
func NewExperienceFromObservation(event Event, observation ObservationInput, sequence []string) learning.Experience { timestamp := event.Timestamp; if timestamp.IsZero() { timestamp = time.Now().UTC() }; return learning.Experience{ExperienceID: event.ID, Sequence: append([]string(nil), sequence...), Weight: 0.50, Confidence: 0.20, Source: observation.Source, Modality: observation.Modality, Timestamp: timestamp, Reliability: 0.50, IndependenceGroup: event.ID} }


func knowledgeClaimsFromGrounding(grounded []knowledge.GroundedRepresentation, at time.Time) []KnowledgeClaim {
	claims := make([]KnowledgeClaim, 0, len(grounded))
	for _, representation := range grounded {
		claims = append(claims, KnowledgeClaim{
			NodeID: representation.NodeID,
			Origin: KnowledgeOrigin{
				Source: representation.Source,
				Modality: representation.Modality,
				ObservedAt: at,
			},
			Verified: false,
			Status: representation.Status,
		})
	}
	return claims
}

func groundedNodeIDs(grounded []knowledge.GroundedRepresentation) []knowledge.NodeID {
	ids := make([]knowledge.NodeID, 0, len(grounded))
	for _, representation := range grounded {
		ids = append(ids, representation.NodeID)
	}
	return ids
}

func experienceID(experience *learning.Experience) string {
	if experience == nil { return "" }
	return experience.ExperienceID
}


func groundedPopulationNodeIDs(grounded []knowledge.GroundedRepresentation) []knowledge.NodeID {
	seen := make(map[knowledge.NodeID]struct{})
	ids := make([]knowledge.NodeID, 0)
	for _, representation := range grounded {
		for _, id := range representation.Population {
			if id == 0 {
				continue
			}
			if _, ok := seen[id]; ok {
				continue
			}
			seen[id] = struct{}{}
			ids = append(ids, id)
		}
	}
	return ids
}
