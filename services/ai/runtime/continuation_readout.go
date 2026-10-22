package runtime

import (
	"errors"
	"slices"
	"time"

	"github.com/project-horizon/horizon-core/services/ai/knowledge"
)

const (
	ContinuationEncodingV1     = "learned-continuation-v1"
	MaxContinuationSteps       = 32
	MaxContinuationInputFrames = 32
	MaxContinuationMembers     = 64 // Per simultaneous population, including predictions.
	MaxContinuationScalars     = 512
	ContinuationConfidenceGate = .5 // A conservative I/O gate, not calibrated accuracy.
)

type ContinuationStatus string

const (
	ContinuationComplete    ContinuationStatus = "complete"
	ContinuationUnknown     ContinuationStatus = "unknown"
	ContinuationUncertain   ContinuationStatus = "uncertain"
	ContinuationUndecodable ContinuationStatus = "undecodable"
	ContinuationStepLimit   ContinuationStatus = "step-limit"
	ContinuationScalarLimit ContinuationStatus = "scalar-limit"
	ContinuationMemberLimit ContinuationStatus = "member-limit"
)

type ContinuationStep struct {
	Forecast ContinuationForecast       `json:"forecast"`
	Spelling *knowledge.SpellingReadout `json:"spelling,omitempty"`
}

// ContinuationForecast summarizes the selected native population. Alternative
// populations are counted, not copied into an unbounded diagnostic response.
type ContinuationForecast struct {
	NodeIDs            []knowledge.NodeID `json:"node_ids,omitempty"`
	MemberCount        int                `json:"member_count"`
	Alternatives       int                `json:"alternatives"`
	Confidence         float64            `json:"confidence"`
	UnknownProbability float64            `json:"unknown_probability"`
	ContextSupport     float64            `json:"context_support"`
	Ambiguity          float64            `json:"ambiguity"`
	EvidenceMass       float64            `json:"evidence_mass"`
}

func summarizeContinuationForecast(a *knowledge.PredictionAssessment) ContinuationForecast {
	out := ContinuationForecast{MemberCount: len(a.NodeIDs), Alternatives: len(a.Alternatives), Confidence: a.Confidence, UnknownProbability: a.UnknownProbability, ContextSupport: a.ContextSupport, Ambiguity: a.Ambiguity, EvidenceMass: a.EvidenceMass}
	if len(a.NodeIDs) <= MaxContinuationMembers {
		out.NodeIDs = slices.Clone(a.NodeIDs)
	}
	return out
}

// ContinuationReadout is a detached, bounded hypothesis. Steps may describe an
// unfinished attempt; Text/Scalars are published only after a learned empty
// spelling terminates it. MinConfidence is the weakest individual forecast,
// not a joint probability, factual confidence, or evidence of language fluency.
type ContinuationReadout struct {
	BrainIdentity string             `json:"brain_identity"`
	Sequence      uint64             `json:"sequence"`
	Hypothetical  bool               `json:"hypothetical"`
	At            time.Time          `json:"at"`
	Encoding      string             `json:"encoding"`
	Status        ContinuationStatus `json:"status"`
	Steps         []ContinuationStep `json:"steps,omitempty"`
	Text          string             `json:"text,omitempty"`
	Scalars       []uint32           `json:"scalars,omitempty"`
	MinConfidence float64            `json:"min_confidence"`
}

func validateContinuationSeed(seed *knowledge.ObservationTrace, maxSteps int) error {
	if maxSteps < 1 || maxSteps > MaxContinuationSteps || seed == nil || len(seed.Stimulus) == 0 {
		return errors.New("continuation requires a nonempty neural seed and a bounded positive step count")
	}
	if len(seed.Stimulus) > MaxContinuationInputFrames || len(seed.Context) > MaxContinuationInputFrames || len(seed.Data) > MaxContinuationInputFrames || len(seed.Stimulus)+len(seed.Context)+len(seed.Data) > MaxContinuationInputFrames {
		return errors.New("continuation seed exceeds frame budget")
	}
	for _, stream := range [][][]knowledge.NodeID{seed.Stimulus, seed.Context, seed.Data} {
		for _, frame := range stream {
			if len(frame) == 0 || len(frame) > MaxContinuationMembers {
				return errors.New("continuation seed exceeds population budget or has an empty frame")
			}
			seen := map[knowledge.NodeID]bool{}
			for _, id := range frame {
				if id <= 0 || seen[id] {
					return errors.New("continuation seed contains an invalid or duplicate neural member")
				}
				seen[id] = true
			}
		}
	}
	return nil
}

// ReadLearnedContinuation feeds forecast populations into a local hypothetical
// prefix and realizes each via the learned spelling projection on the same
// Brain. It never feeds its outputs into observation/learning, or changes the
// native Answer. Teacher lessons use the existing numeric LearnExperience API.
// Only callers holding the supervisor lease may use a managed runtime.
func (r *BrainRuntime) ReadLearnedContinuation(seed *knowledge.ObservationTrace, maxSteps int, at time.Time) (ContinuationReadout, error) {
	out := ContinuationReadout{BrainIdentity: BrainIdentity, Encoding: ContinuationEncodingV1, Hypothetical: true, Status: ContinuationUnknown}
	if r == nil || r.brain == nil {
		return out, errors.New("brain runtime is not initialized")
	}
	// Validate budgets before copying caller-owned frames.
	if err := validateContinuationSeed(seed, maxSteps); err != nil {
		return out, err
	}
	seed = seed.Clone()
	r.mu.Lock()
	defer r.mu.Unlock()
	if err := r.ensureWritableLocked(); err != nil {
		return out, err
	}
	out.Sequence = r.seq
	if r.dynamicsEncoding != knowledge.CurrentDynamics {
		return out, errors.New("learned continuation requires the current explicit neural dynamics")
	}
	if err := seed.Validate(func(id knowledge.NodeID) bool { return r.brain.Registry.GetByID(id) != nil }); err != nil {
		return out, err
	}
	if at.IsZero() {
		at = r.nowLocked()
	}
	at = at.UTC()
	out.At = at
	var scalars []uint32
	for i := 0; i < maxSteps; i++ {
		assessment := r.activation.PreviewLearnedPopulation(seed, at)
		out.Steps = append(out.Steps, ContinuationStep{Forecast: summarizeContinuationForecast(assessment)})
		if len(assessment.NodeIDs) == 0 {
			return out, nil
		}
		if i == 0 || assessment.Confidence < out.MinConfidence {
			out.MinConfidence = assessment.Confidence
		}
		if assessment.Confidence <= ContinuationConfidenceGate {
			out.Status = ContinuationUncertain
			return out, nil
		}
		if len(assessment.NodeIDs) > MaxContinuationMembers {
			out.Status = ContinuationMemberLimit
			return out, nil
		}
		spelling, err := r.brain.ReadLearnedSpelling(assessment.NodeIDs)
		if err != nil {
			return out, err
		}
		out.Steps[i].Spelling = &spelling
		if spelling.Status != knowledge.SpellingDecoded {
			out.Status = ContinuationUndecodable
			return out, nil
		}
		if len(spelling.Scalars) == 0 {
			out.Status = ContinuationComplete
			out.Scalars = scalars
			runes := make([]rune, len(scalars))
			for j, scalar := range scalars {
				runes[j] = rune(scalar)
			}
			out.Text = string(runes)
			return out, nil
		}
		if len(scalars)+len(spelling.Scalars) > MaxContinuationScalars {
			out.Status = ContinuationScalarLimit
			return out, nil
		}
		scalars = append(scalars, spelling.Scalars...)
		seed.Stimulus = append(seed.Stimulus, slices.Clone(assessment.NodeIDs))
	}
	out.Status = ContinuationStepLimit
	return out, nil
}
