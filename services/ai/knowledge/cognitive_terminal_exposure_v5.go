package knowledge

import (
	"errors"
	"math"
)

// An explicit experimental observation protocol, not a skill or a production
// action selector. Acquisitions are externally executed before terminal choice.
const Phase4TerminalProtocolV5 = "v5-phase4-controlled-terminal-exposure-v1"

type Phase4TerminalAcquisitionV5 struct {
	Observed        []VerifiedMentalResult `json:"observed,omitempty"`
	Cost            float64                `json:"cost"`
	FeedbackBlinded bool                   `json:"feedback_blinded,omitempty"`
	TraceDigest     string                 `json:"trace_digest"`
}

func phase4TerminalMaskV5() []bool { return []bool{false, false, false, true, true, true, true, true} }

// Uses only the current canonical N weights. Features must come from the
// authenticated frozen observation pipeline; no target/outcome verifier label
// is accepted. Batching amortizes canonical parameter reads, not learning.
func (k *KnowledgeBase) PreviewPhase4TerminalFeaturesV5(features [][]float64, draws []float64, greedy bool, drop Phase4DropV5) ([]Phase4PredictionV5, error) {
	if k.HasCoordinatedV5() {
		return nil, errors.New("coordinated state requires observed trace, not frozen historical feature cache")
	}
	k.RLock()
	defer k.RUnlock()
	s := k.BrainState.CognitiveModel
	if s == nil || s.Phase4V5 == nil || len(features) < 1 || len(features) > 256 || len(draws) != len(features) {
		return nil, errors.New("explicit Phase IV terminal features required")
	}
	w, e := k.cognitiveWeights(s)
	if e != nil {
		return nil, e
	}
	digest := CognitiveDigest(w)
	out := make([]Phase4PredictionV5, len(features))
	for i, input := range features {
		if len(input) != 58 || !finiteNumber(draws[i]) || draws[i] < 0 || draws[i] >= 1 {
			return nil, errors.New("invalid terminal feature/draw shape")
		}
		for _, v := range input {
			if !finiteNumber(v) {
				return nil, errors.New("nonfinite terminal feature")
			}
		}
		if input[48] < 0 || input[48] > 3 || input[48] != math.Trunc(input[48]) {
			return nil, errors.New("invalid acquired budget")
		}
		x := append([]float64(nil), input...)
		phase4DropFeatures(x, Phase4DropV5{s.Phase4V5.Spec.DropMUSE || drop.MUSE, s.Phase4V5.Spec.DropVOC || drop.VOC, s.Phase4V5.Spec.DropActor || drop.Actor})
		_, p, v := phase4Forward(s.Phase4V5, w, x, phase4TerminalMaskV5())
		a := samplePolicy(p, draws[i])
		if greedy {
			a = maxIndex(p)
		}
		out[i] = Phase4PredictionV5{Features: x, Policy: p, Value: v, Action: a, ParametersHash: digest}
	}
	return out, nil
}

func validatePhase4TerminalBatchV5(s *CognitiveModelState, w []float64, b Phase4BatchV5) error {
	for _, ep := range b.Episodes {
		if ep.CaseID == "" || len(ep.Frames) != 1 || ep.Acquisition == nil || len(ep.Acquisition.Observed) > 3 || ep.Acquisition.TraceDigest == "" {
			return errors.New("terminal protocol requires one observed terminal and explicit acquisition")
		}
		f := ep.Frames[0]
		acq := ep.Acquisition
		if len(f.Features) != 58 || len(f.Mask) != 8 || f.Action < 3 || f.Action > 7 || !finiteNumber(f.Reward) || !finiteNumber(f.Draw) || f.Draw < 0 || f.Draw >= 1 {
			return errors.New("invalid observed terminal frame")
		}
		for _, v := range f.Features {
			if !finiteNumber(v) {
				return errors.New("nonfinite terminal features")
			}
		}
		for a, valid := range f.Mask {
			if valid != (a >= 3) {
				return errors.New("terminal protocol cannot force an answer or acquire operations")
			}
		}
		if f.Features[48] != float64(3-len(acq.Observed)) {
			return errors.New("acquisition/remaining budget mismatch")
		}
		used := [3]bool{}
		cost := 0.
		for _, o := range acq.Observed {
			if o.Operation < 0 || o.Operation >= 3 || o.Outcome < 0 || o.Outcome >= 3 || used[o.Operation] {
				return errors.New("invalid acquired observation")
			}
			used[o.Operation] = true
			cost += f.Features[45+o.Operation]
		}
		if !finiteNumber(acq.Cost) || math.Abs(cost-acq.Cost) > 1e-12 {
			return errors.New("acquisition cost mismatch")
		}
		if f.Action == 7 && f.Reward != .35 || f.Action != 7 && f.Reward != 0 && f.Reward != 1 {
			return errors.New("terminal payoff changed")
		}
		x := append([]float64(nil), f.Features...)
		phase4DropFeatures(x, Phase4DropV5{s.Phase4V5.Spec.DropMUSE, s.Phase4V5.Spec.DropVOC, s.Phase4V5.Spec.DropActor})
		if CognitiveDigest(x) != CognitiveDigest(f.Features) {
			return errors.New("ablated terminal features present")
		}
		_, p, v := phase4Forward(s.Phase4V5, w, f.Features, f.Mask)
		if samplePolicy(p, f.Draw) != f.Action || math.Abs(p[f.Action]-f.BehaviorProbability) > 1e-12 || math.Abs(v-f.BehaviorValue) > 1e-12 {
			return errors.New("off-policy terminal choice")
		}
	}
	return nil
}
