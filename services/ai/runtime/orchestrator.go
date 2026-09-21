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
	if o == nil || o.Runtime == nil { return CognitiveInterpretation{}, false, errors.New("cognitive orchestrator is not initialized") }
	observationContext := Observation{Source: observation.Source, Modality: observation.Modality, Tokens: append([]string(nil), observation.Tokens...), ContextTokens: append([]string(nil), observation.ContextTokens...), DataTokens: append([]string(nil), observation.DataTokens...)}
	grounded := make([]knowledge.GroundedRepresentation, 0, len(observation.Tokens)+len(observation.ContextTokens)+len(observation.DataTokens))
	seen := make(map[string]struct{}, len(observation.Tokens)+len(observation.ContextTokens)+len(observation.DataTokens))
	groundToken := func(token string) error {
		key := token
		if _, exists := seen[key]; exists { return nil }
		seen[key] = struct{}{}
		representation, err := o.Runtime.GroundObservationAt(token, observation.Source, observation.Modality, event.Timestamp)
		if err != nil { return err }
		grounded = append(grounded, representation)
		return nil
	}
	for _, token := range observation.Tokens { if err := groundToken(token); err != nil { return CognitiveInterpretation{}, false, err } }
	for _, token := range observation.ContextTokens { if err := groundToken(token); err != nil { return CognitiveInterpretation{}, false, err } }
	for _, token := range observation.DataTokens { if err := groundToken(token); err != nil { return CognitiveInterpretation{}, false, err } }
	event.Source = observation.Source; event.Modality = observation.Modality; event.ContextTokens = append([]string(nil), observation.ContextTokens...); event.DataTokens = append([]string(nil), observation.DataTokens...)
	output, err := o.Runtime.CognitiveProcess(event); if err != nil { return CognitiveInterpretation{}, false, err }
	learned := false
	if experience != nil && len(experience.Sequence) > 0 {
		if experience.ExperienceID == "" { experience.ExperienceID = event.ID }; if experience.Source == "" { experience.Source = observation.Source }; if experience.Modality == "" { experience.Modality = observation.Modality }; if experience.Timestamp.IsZero() { experience.Timestamp = event.Timestamp }
		_, err = o.Runtime.LearnExperience(*experience, experience.Timestamp); if err != nil { return CognitiveInterpretation{}, false, err }; learned = true
	}
	cognitive, err := o.Runtime.InterpretCognitive(output, observationContext)
	if err != nil { return CognitiveInterpretation{}, learned, err }
	cognitive.GroundedRepresentations = grounded
	inquiryAgenda, inquiryErr := BuildInquiryAgendaFromCognition(cognitive, output.Timestamp)
	if inquiryErr != nil {
		return CognitiveInterpretation{}, learned, inquiryErr
	}
	cognitive.InquiryAgenda = &inquiryAgenda
	cognitive.ChangeAwareness = InternalChangeAwareness{
		BrainIdentity: BrainIdentity,
		Sequence: output.Sequence,
		Timestamp: output.Timestamp,
		StateDelta: output.StateDelta,
		KnowledgeChanges: []KnowledgeInjectionEvent{{
			BrainIdentity: BrainIdentity,
			Sequence: output.Sequence,
			Timestamp: event.Timestamp,
			Claims: knowledgeClaimsFromGrounding(grounded, event.Timestamp),
			ExperienceID: experienceID(experience),
			ChangedNodeIDs: groundedNodeIDs(grounded),
		}},
	}
	return cognitive, learned, nil
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
