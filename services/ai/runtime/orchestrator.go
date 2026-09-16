package runtime

import (
	"errors"
	"time"

	"github.com/project-horizon/horizon-core/services/ai/learning"
)

// CognitiveOrchestrator is the canonical integration boundary for the neural
// cognitive cycle. It coordinates processing, optional experience recording,
// and typed interpretation without turning neural activation into an action.
// Action authorization/execution remains outside this layer.
type CognitiveOrchestrator struct {
	Runtime *BrainRuntime
}

func NewCognitiveOrchestrator(runtime *BrainRuntime) *CognitiveOrchestrator {
	return &CognitiveOrchestrator{Runtime: runtime}
}

// ObservationInput contains the substrate-neutral evidence associated with an
// event. It is deliberately separate from action or authorization state.
type ObservationInput struct {
	Source        string
	Modality      string
	Tokens        []string
	ContextTokens []string
	DataTokens    []string
}

// ProcessObservation executes the canonical perception-to-interpretation
// boundary and, when an experience is supplied, records it through the same
// runtime. The returned cognitive interpretation is safe to pass to a
// presentation layer; Recommendation remains nil until an explicit action
// proposal stage creates one.
func (o *CognitiveOrchestrator) ProcessObservation(event Event, observation ObservationInput, experience *learning.Experience) (CognitiveInterpretation, bool, error) {
	if o == nil || o.Runtime == nil {
		return CognitiveInterpretation{}, false, errors.New("cognitive orchestrator is not initialized")
	}

	output, err := o.Runtime.CognitiveProcess(event)
	if err != nil {
		return CognitiveInterpretation{}, false, err
	}

	learned := false
	if experience != nil && len(experience.Sequence) > 0 {
		if experience.ExperienceID == "" {
			experience.ExperienceID = event.ID
		}
		if experience.Source == "" {
			experience.Source = observation.Source
		}
		if experience.Modality == "" {
			experience.Modality = observation.Modality
		}
		if experience.Timestamp.IsZero() {
			experience.Timestamp = event.Timestamp
		}
		_, err = o.Runtime.LearnExperience(*experience, experience.Timestamp)
		learned = err == nil
		if err != nil {
			return CognitiveInterpretation{}, false, err
		}
	}

	cognitive, err := o.Runtime.InterpretCognitive(output, Observation{
		Source:        observation.Source,
		Modality:      observation.Modality,
		Tokens:        append([]string(nil), observation.Tokens...),
		ContextTokens: append([]string(nil), observation.ContextTokens...),
		DataTokens:    append([]string(nil), observation.DataTokens...),
	})
	if err != nil {
		return CognitiveInterpretation{}, learned, err
	}
	return cognitive, learned, nil
}

// LearnFromOutcome records an observed action consequence through the runtime's
// event log and learning substrate. The outcome must retain the RequestID that
// identifies the originating action recommendation.
func (o *CognitiveOrchestrator) LearnFromOutcome(outcome OutcomeEvent) (uint64, error) {
	if o == nil || o.Runtime == nil {
		return 0, errors.New("cognitive orchestrator is not initialized")
	}
	return o.Runtime.ObserveOutcome(outcome)
}

// NewExperienceFromObservation constructs an experience without deciding that
// the observation is permanently trusted. Promotion policy remains a separate
// concern and can reject or downgrade this candidate later.
func NewExperienceFromObservation(event Event, observation ObservationInput, sequence []string) learning.Experience {
	timestamp := event.Timestamp
	if timestamp.IsZero() {
		timestamp = time.Now().UTC()
	}
	return learning.Experience{
		ExperienceID:      event.ID,
		Sequence:          append([]string(nil), sequence...),
		Weight:            0.50,
		Confidence:        0.20,
		Source:            observation.Source,
		Modality:          observation.Modality,
		Timestamp:         timestamp,
		Reliability:       0.50,
		IndependenceGroup: event.ID,
	}
}
