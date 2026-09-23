package runtime

import "time"

type InquiryConsequenceEvent struct {
	Action InquiryAction `json:"action"`
	Valence float64 `json:"valence"`
}

// RecordInquiryConsequenceEvent applies a learned inquiry consequence and
// records the exact preference update in the durable event log.
func (r *BrainRuntime) RecordInquiryConsequenceEvent(event InquiryConsequenceEvent, at time.Time) (uint64, error) {
	if r == nil || event.Action == "" { return 0, nil }
	event.Valence = clampSigned(event.Valence)
	r.mu.Lock(); defer r.mu.Unlock()
	if r.inquiryValence == nil { r.inquiryValence = make(map[InquiryAction]float64) }
	previous := r.inquiryValence[event.Action]
	r.inquiryValence[event.Action] = clampSigned(previous + inquiryPreferenceLearningRate*(event.Valence-previous))
	nextSeq := r.seq + 1
	if r.eventLog != nil {
		if at.IsZero() { at = r.nowLocked() }
		if err := r.eventLog.Append(LoggedEvent{SchemaVersion: EventLogSchemaVersion, BrainIdentity: BrainIdentity, Sequence: nextSeq, Type: EventTypeConsequence, Timestamp: at.UTC(), Consequence: &event}); err != nil { return r.seq, err }
	}
	r.seq = nextSeq
	return r.seq, nil
}
