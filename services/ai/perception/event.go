package perception

import "time"

// PerceptionEvent is the normalized boundary between sensing/perception and
// the neural runtime. It carries observed input plus provenance without
// assigning semantic meaning to neural units.
type PerceptionEvent struct {
	ID         string
	Signals    []PerceptionSignal
	Context    []string
	Data       []string
	Source     string
	Modality   string
	ObservedAt time.Time
}

// NewEvent normalizes any perception layer into the common event boundary.
// The layer performs perception only; it does not assign neural semantics or
// action permissions.
func NewEvent(id, input, source string, observedAt time.Time, layer PerceptionLayer) (PerceptionEvent, error) {
	if layer == nil {
		layer = UserInputPerception{}
	}
	signals, err := layer.Perceive(input)
	if err != nil {
		return PerceptionEvent{}, err
	}
	if observedAt.IsZero() {
		if len(signals) > 0 && !signals[0].ObservedAt.IsZero() {
			observedAt = signals[0].ObservedAt
		} else {
			observedAt = time.Now().UTC()
		}
	}
	if source == "" {
		source = "user"
	}
	modality := "perception"
	if len(signals) > 0 && signals[0].Kind == PerceptionUserInput {
		modality = "user-input"
	}
	return PerceptionEvent{
		ID: id, Signals: signals, Source: source, Modality: modality, ObservedAt: observedAt.UTC(),
	}, nil
}

// NewUserInputEvent is the compatibility constructor for the default user
// input adapter.
func NewUserInputEvent(id, input, source string, observedAt time.Time) (PerceptionEvent, error) {
	return NewEvent(id, input, source, observedAt, UserInputPerception{})
}
