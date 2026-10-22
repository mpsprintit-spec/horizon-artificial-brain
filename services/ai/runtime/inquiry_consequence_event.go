package runtime

import (
	"errors"
	"github.com/project-horizon/horizon-core/services/ai/knowledge"
	"math"
	"time"
)

type InquiryConsequenceEvent struct {
	Action          InquiryAction      `json:"action"`
	Valence         float64            `json:"valence"`
	InformationGain float64            `json:"information_gain"`
	PredictionError float64            `json:"prediction_error"`
	Reliability     float64            `json:"reliability"`
	TargetNodeIDs   []knowledge.NodeID `json:"target_node_ids,omitempty"`
	OutcomeNodeIDs  []knowledge.NodeID `json:"outcome_node_ids,omitempty"`
	CausalLink      string             `json:"causal_link,omitempty"`
}

// RecordInquiryConsequenceEvent applies a learned inquiry consequence and
// records the exact preference update in the durable event log.
func (r *BrainRuntime) RecordInquiryConsequenceEvent(event InquiryConsequenceEvent, at time.Time) (uint64, error) {
	if r == nil || event.Action == "" {
		return 0, nil
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if err := r.ensureWritableLocked(); err != nil {
		return r.seq, err
	}
	if at.IsZero() {
		at = r.nowLocked()
	}
	at = at.UTC()
	for _, value := range []float64{event.Valence, event.InformationGain, event.PredictionError, event.Reliability} {
		if math.IsNaN(value) || math.IsInf(value, 0) {
			return r.seq, errors.New("consequence values must be finite")
		}
	}
	event.Valence = clampSigned(event.Valence)
	event.InformationGain = clamp01(event.InformationGain)
	event.PredictionError = clamp01(event.PredictionError)
	event.Reliability = clamp01(event.Reliability)
	event.TargetNodeIDs = append([]knowledge.NodeID(nil), event.TargetNodeIDs...)
	event.OutcomeNodeIDs = append([]knowledge.NodeID(nil), event.OutcomeNodeIDs...)
	err := r.runJournalCommandLocked(LoggedEvent{Type: EventTypeConsequence, Timestamp: at, Consequence: &event}, func() error {
		r.applyInquiryConsequenceLocked(event, at)
		r.seq++
		return nil
	})
	return r.seq, err
}

// applyInquiryConsequenceLocked is shared with the complete inquiry outcome
// transaction. The caller owns journaling and the single logical revision.
func (r *BrainRuntime) applyInquiryConsequenceLocked(event InquiryConsequenceEvent, at time.Time) {
	if r.brain != nil {
		r.brain.RecordInquiryInformationExperience(string(event.Action), event.InformationGain, event.Reliability, at)
	}
	if r.inquiryValence == nil {
		r.inquiryValence = make(map[InquiryAction]float64)
	}
	previous := r.inquiryValence[event.Action]
	if r.learning != nil && len(event.TargetNodeIDs) > 0 && len(event.OutcomeNodeIDs) > 0 {
		r.learning.LearnOutcomeTrace(event.TargetNodeIDs, event.OutcomeNodeIDs, clamp01(0.5+0.5*absFloat(event.Valence)), clamp01(event.Reliability), knowledge.ExperienceEvidence{
			ExperienceID: event.CausalLink, Source: "inquiry-consequence", Modality: "outcome",
			Timestamp: at, Reliability: event.Reliability, IndependenceGroup: event.CausalLink,
			CausalLink: event.CausalLink,
		}, at)
	}
	r.inquiryValence[event.Action] = clampSigned(previous + inquiryPreferenceLearningRate*(event.Valence-previous))
}
