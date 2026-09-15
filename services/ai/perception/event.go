package perception

import "time"

// PerceptionEvent is the normalized boundary between sensing/perception and
// the neural runtime. It carries observed input plus provenance without
// assigning semantic meaning to neural units.
type PerceptionEvent struct {
	ID        string
	Signals   []PerceptionSignal
	Context   []string
	Data      []string
	Source    string
	Modality  string
	ObservedAt time.Time
}

// NewUserInputEvent creates the normalized event for the current user-input
// adapter. Other sensors can construct the same contract without changing the
// neural runtime boundary.
func NewUserInputEvent(id, input, source string, observedAt time.Time) (PerceptionEvent, error) {
	layer := UserInputPerception{}
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
	return PerceptionEvent{
		ID: id, Signals: signals, Source: source, Modality: "user-input", ObservedAt: observedAt.UTC(),
	}, nil
}
