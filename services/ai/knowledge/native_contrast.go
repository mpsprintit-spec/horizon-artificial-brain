package knowledge

import (
	"errors"
	"math"
	"time"
)

// A bounded pending empirical trial, not another model or a replay database.
// No weight changes until its matched observed trial arrives. Both trials must
// keep parameters unchanged. Feature contrasts hold actions fixed; action
// contrasts hold the initial state and continuation fixed. Randomization/matching
// of external conditions is the experiment adapter's responsibility.
type NativeContrast struct {
	Mode      string         `json:"mode,omitempty"`
	ID        string         `json:"id"`
	Source    string         `json:"source"`
	OutcomeID string         `json:"outcome_id"`
	Reward    float64        `json:"reward"`
	Episode   *NativeEpisode `json:"episode"`
}

func validateNativeContrast(s *NativeControlState, id, source, outcome string, learn bool, mode string) error {
	if mode != "" && (mode != "action" || id == "") {
		return errors.New("invalid contrast mode")
	}
	if s == nil {
		return errors.New("missing native controller")
	}
	if id == "" {
		if s.PendingContrast != nil && learn {
			return errors.New("pending contrast freezes learning")
		}
		return nil
	}
	if !s.Config.SupportsExperiments() || s.Config.Lambda != 1 || !learn || s.Episode == nil || !s.Episode.Training || !s.Episode.Stopped || source == "" || outcome == "" {
		return errors.New("contrast requires version2 or later, lambda1, completed observed training trials")
	}
	if mode == "action" {
		return validateNativeActionContrast(s, id, source, outcome)
	}
	if p := s.PendingContrast; p != nil {
		ep := s.Episode
		if p.Mode != mode || p.ID != id || p.Source != source || p.OutcomeID == outcome || p.Episode.ID == ep.ID || len(p.Episode.Decisions) != len(ep.Decisions) {
			return errors.New("unmatched contrast")
		}
		for i, d := range ep.Decisions {
			if d.Action != p.Episode.Decisions[i].Action || d.Cost != p.Episode.Decisions[i].Cost {
				return errors.New("contrast action sequences differ")
			}
		}
	}
	return nil
}

func (k *KnowledgeBase) ValidateNativeContrast(id, source, outcome string, learn bool, modes ...string) error {
	mode := ""
	if len(modes) > 0 {
		mode = modes[0]
	}
	k.RLock()
	defer k.RUnlock()
	return validateNativeContrast(k.BrainState.NativeControl, id, source, outcome, learn, mode)
}

// Paired return regression minimizes ((G_A-G_B)-(Q_A-Q_B))^2. Shared
// activity cancels algebraically, including bias. This estimates conditional
// intervention effects only under matched exogenous conditions; a pair tag
// alone cannot certify those conditions. Ordinary unpaired Sarsa is preserved.
// ContrastMean additionally fits the factual pair mean through the canonical
// bias synapses only. This calibrates the otherwise unidentified intercept,
// without assigning common feature weights a causal effect. It is an explicit
// experimental option, not a general solution to nonlinear/contextual credit.
func (k *KnowledgeBase) LearnNativeObservedOutcome(reward float64, learn bool, id, source, outcome string, at time.Time, modes ...string) (NativeLearning, error) {
	mode := ""
	if len(modes) > 0 {
		mode = modes[0]
	}
	if err := k.ValidateNativeContrast(id, source, outcome, learn, mode); err != nil {
		return NativeLearning{}, err
	}
	if id == "" {
		return k.LearnNativeOutcome(reward, learn, at)
	}
	k.Lock()
	defer k.Unlock()
	s := k.BrainState.NativeControl
	if err := validateNativeContrast(s, id, source, outcome, learn, mode); err != nil {
		return NativeLearning{}, err
	}
	if !nativeFinite(reward) || reward < -1 || reward > 1 {
		return NativeLearning{}, errors.New("invalid contrast reward")
	}
	ep := s.Episode
	out := NativeLearning{Return: reward, Contrast: true}
	for _, d := range ep.Decisions {
		out.Return -= d.Cost
	}
	if s.PendingContrast == nil {
		s.PendingContrast = &NativeContrast{Mode: mode, ID: id, Source: source, OutcomeID: outcome, Reward: reward, Episode: ep}
		s.Episode = nil
		out.ContrastPending = true
		return out, nil
	}
	if mode == "action" {
		return k.learnNativeActionContrast(reward, at)
	}
	p := s.PendingContrast
	type edge struct{ from, to NodeID }
	grad := map[edge]float64{}
	diffReturn := p.Reward - reward
	meanReturn := (p.Reward + reward) / 2
	for t := len(ep.Decisions) - 1; t >= 0; t-- {
		a, b := p.Episode.Decisions[t], ep.Decisions[t]
		if t < len(ep.Decisions)-1 {
			diffReturn *= s.Config.Gamma
		}
		if t == len(ep.Decisions)-1 {
			meanReturn -= a.Cost
		} else {
			meanReturn = -a.Cost + s.Config.Gamma*meanReturn
		}
		if s.Config.ContrastMean {
			meanResidual := meanReturn - (a.Values[a.Action]+b.Values[b.Action])/2
			for _, to := range s.Outputs[a.Action] {
				grad[edge{s.Bias, to}] += meanResidual
			}
		}
		residual := diffReturn - (a.Values[a.Action] - b.Values[b.Action])
		out.TDError += math.Abs(residual)
		xa, err := k.nativeFeatures(s, a.State)
		if err != nil {
			return NativeLearning{}, err
		}
		xb, err := k.nativeFeatures(s, b.State)
		if err != nil {
			return NativeLearning{}, err
		}
		for key := range xb {
			if _, ok := xa[key]; !ok {
				xa[key] = 0
			}
		}
		norm := 0.0
		for _, key := range nativeIDs(xa) {
			xa[key] -= xb[key]
			if math.Abs(xa[key]) < 1e-14 {
				xa[key] = 0
			}
			norm += xa[key] * xa[key]
		}
		norm = math.Max(1, norm)
		for _, key := range nativeIDs(xa) {
			if xa[key] == 0 {
				continue
			}
			for _, to := range s.Outputs[a.Action] {
				grad[edge{key, to}] += residual * xa[key] / norm
			}
		}
	}
	// Canonical source/effector order also fixes floating update ordering.
	for _, from := range s.ParameterSources() {
		for _, group := range s.Outputs {
			for _, to := range group {
				delta := grad[edge{from, to}] * s.Config.LearningRate
				if delta == 0 {
					continue
				}
				syn := k.Registry.GetByID(from).Synapses[to][0]
				old := nativeWeight(syn)
				next := math.Max(-7.5, math.Min(7.5, old+delta))
				if next == old {
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

// Discarding is explicit and journaled by the runtime; it never changes weights
// or erases the original empirical event. It releases a failed pairing boundary.
func (k *KnowledgeBase) DiscardNativeContrast(id string) error {
	k.Lock()
	defer k.Unlock()
	s := k.BrainState.NativeControl
	if s == nil || s.Episode != nil || s.PendingContrast == nil || s.PendingContrast.ID != id {
		return errors.New("no matching idle pending contrast")
	}
	s.PendingContrast = nil
	return nil
}
