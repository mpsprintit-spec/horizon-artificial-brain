package runtime

import (
	"errors"
	"time"

	"github.com/project-horizon/horizon-core/services/ai/learning"
)

type CognitiveOrchestrator struct { Runtime *BrainRuntime }
func NewCognitiveOrchestrator(runtime *BrainRuntime) *CognitiveOrchestrator { return &CognitiveOrchestrator{Runtime: runtime} }

type ObservationInput struct {
	Source string
	Modality string
	Tokens []string
	ContextTokens []string
	DataTokens []string
}

// ProcessObservation is the canonical integration boundary. Context and data
// are first grounded into the same Brain substrate, then the event is processed
// and interpreted. Grounding is provenance only: it never authorizes action and
// never promotes a candidate observation into trusted knowledge by itself.
func (o *CognitiveOrchestrator) ProcessObservation(event Event, observation ObservationInput, experience *learning.Experience) (CognitiveInterpretation, bool, error) {
	if o == nil || o.Runtime == nil { return CognitiveInterpretation{}, false, errors.New("cognitive orchestrator is not initialized") }

	observationContext := Observation{Source: observation.Source, Modality: observation.Modality, Tokens: append([]string(nil), observation.Tokens...), ContextTokens: append([]string(nil), observation.ContextTokens...), DataTokens: append([]string(nil), observation.DataTokens...)}
	grounded := make([]knowledge.GroundedRepresentation, 0, len(observation.ContextTokens)+len(observation.DataTokens))
	for _, token := range observation.ContextTokens {
		representation, err := o.Runtime.GroundObservation(token, observation.Source, observation.Modality)
		if err != nil { return CognitiveInterpretation{}, false, err }
		grounded = append(grounded, representation)
	}
	for _, token := range observation.DataTokens {
		representation, err := o.Runtime.GroundObservation(token, observation.Source, observation.Modality)
		if err != nil { return CognitiveInterpretation{}, false, err }
		grounded = append(grounded, representation)
	}

	// Keep event evidence synchronized with the typed observation boundary.
	event.Source = observation.Source
	event.Modality = observation.Modality
	event.ContextTokens = append([]string(nil), observation.ContextTokens...)
	event.DataTokens = append([]string(nil), observation.DataTokens...)

	output, err := o.Runtime.CognitiveProcess(event)
	if err != nil { return CognitiveInterpretation{}, false, err }

	learned := false
	if experience != nil && len(experience.Sequence) > 0 {
		if experience.ExperienceID == "" { experience.ExperienceID = event.ID }
		if experience.Source == "" { experience.Source = observation.Source }
		if experience.Modality == "" { experience.Modality = observation.Modality }
		if experience.Timestamp.IsZero() { experience.Timestamp = event.Timestamp }
		_, err = o.Runtime.LearnExperience(*experience, experience.Timestamp)
		if err != nil { return CognitiveInterpretation{}, false, err }
		learned = true
	}

	cognitive, err := o.Runtime.InterpretCognitive(output, observationContext)
	if err != nil { return CognitiveInterpretation{}, learned, err }
	cognitive.GroundedRepresentations = grounded
	return cognitive, learned, nil
}

func (o *CognitiveOrchestrator) LearnFromOutcome(outcome OutcomeEvent) (uint64, error) {
	if o == nil || o.Runtime == nil { return 0, errors.New("cognitive orchestrator is not initialized") }
	return o.Runtime.ObserveOutcome(outcome)
}

func NewExperienceFromObservation(event Event, observation ObservationInput, sequence []string) learning.Experience {
	timestamp := event.Timestamp
	if timestamp.IsZero() { timestamp = time.Now().UTC() }
	return learning.Experience{ExperienceID: event.ID, Sequence: append([]string(nil), sequence...), Weight: 0.50, Confidence: 0.20, Source: observation.Source, Modality: observation.Modality, Timestamp: timestamp, Reliability: 0.50, IndependenceGroup: event.ID}
}
