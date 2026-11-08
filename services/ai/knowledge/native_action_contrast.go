package knowledge

import (
	"errors"
	"math"
	"time"
)

// Action contrasts compare two factual executions, never a predicted alternative.
// MatchKey binds the adapter's exogenous trial block; it is not policy input and
// cannot itself certify hidden-world matching. Runtime verifies observable state
// equality and identical continuation after the assigned first operation.
func validateNativeActionContrast(s *NativeControlState, id, source, outcome string) error {
	ep := s.Episode
	if s.Config.Version != NativeControlCausalVersion || s.Config.Gamma != 1 || s.Config.ContrastMean || !ep.Intervention || ep.MatchKey == "" || ep.Truncated || len(ep.Decisions) == 0 {
		return errors.New("action contrasts need v3, gamma1, factual matched intervention trials")
	}
	p := s.PendingContrast
	if p == nil {
		return nil
	}
	a, b := p.Episode, ep
	if p.Mode != "action" || p.ID != id || p.Source != source || p.OutcomeID == outcome || a.ID == b.ID || a.MatchKey != b.MatchKey || a.Decisions[0].Before != b.Decisions[0].Before || a.Decisions[0].Action == b.Decisions[0].Action {
		return errors.New("unmatched action contrast")
	}
	// Terminal stop is absorbing. Pad a stopped single-decision arm with stop so
	// propagate->stop versus stop is a valid controlled comparison. Reject a
	// change in a later nonterminal operation; it would change two treatments.
	for i := 1; i < max(len(a.Decisions), len(b.Decisions)); i++ {
		x, y := "stop", "stop"
		ax, bx := -1, -1
		if i < len(a.Decisions) {
			ax = a.Decisions[i].Action
			x = s.Config.Operations[ax].Kind
		}
		if i < len(b.Decisions) {
			bx = b.Decisions[i].Action
			y = s.Config.Operations[bx].Kind
		}
		if x != y || x != "stop" && ax != bx {
			return errors.New("action contrast changed continuation")
		}
	}
	return nil
}

func nativeUndiscountedReturn(ep *NativeEpisode, reward float64) float64 {
	for _, d := range ep.Decisions {
		reward -= d.Cost
	}
	return reward
}

// Fits the observed net-return advantage Q(s,a)-Q(s,b). The common absolute
// value is intentionally unidentified: it does not affect greedy choice. A
// common exogenous reward offset cancels exactly when gamma=1 and matching holds.
// Credit is for the initial controlled operation under this continuation, not
// for all operations in an arbitrary, differently scheduled trajectory.
// Caller holds the Brain write lock and has validated the pair.
func (k *KnowledgeBase) learnNativeActionContrast(reward float64, at time.Time) (NativeLearning, error) {
	s := k.BrainState.NativeControl
	p, ep := s.PendingContrast, s.Episode
	a, b := p.Episode.Decisions[0], ep.Decisions[0]
	ga, gb := nativeUndiscountedReturn(p.Episode, p.Reward), nativeUndiscountedReturn(ep, reward)
	delta := (ga - gb) - (a.Values[a.Action] - b.Values[b.Action])
	out := NativeLearning{Contrast: true, Return: gb, TDError: math.Abs(delta)}
	x, err := k.nativeFeatures(s, a.State)
	if err != nil {
		return out, err
	}
	norm := 0.0
	for _, id := range nativeIDs(x) {
		norm += x[id] * x[id]
	}
	norm = 2 * math.Max(1, norm)
	for _, id := range nativeIDs(x) {
		for j, action := range []int{a.Action, b.Action} {
			sign := 1.0
			if j == 1 {
				sign = -1
			}
			for _, to := range s.Outputs[action] {
				syn := k.Registry.GetByID(id).Synapses[to][0]
				old := nativeWeight(syn)
				next := math.Max(-7.5, math.Min(7.5, old+s.Config.LearningRate*sign*delta*x[id]/norm))
				if old == next {
					continue
				}
				syn.Weight = math.Abs(next) / nativeReadoutScale
				syn.Inhibitory = next < 0
				syn.Dynamic.Weight = syn.Weight
				syn.Dynamic.LastModification = at
				syn.Dynamic.Frequency++
				out.UpdatedSynapses++
				out.MaxDelta = math.Max(out.MaxDelta, math.Abs(next-old))
			}
		}
	}
	s.Updates++
	s.Episode = nil
	s.PendingContrast = nil
	return out, nil
}
