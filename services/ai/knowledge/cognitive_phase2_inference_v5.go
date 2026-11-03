package knowledge

import (
	"errors"
	"time"
)

// Trace contains observed inputs only. No target, verified potential outcome,
// reference probability, or caller-provided neural weights can enter this API.
type Phase2ProbeV5 struct {
	Trace               []CognitiveInput `json:"trace"`
	ConditionObserved   bool             `json:"condition_observed"`
	CategoricalAblation bool             `json:"categorical_ablation"`
	RawTemperature      bool             `json:"raw_temperature"`
	Dense               bool             `json:"dense"`
}
type Phase2PredictionV5 struct {
	Readout           CognitiveReadout       `json:"readout"`
	RawJoint          []float64              `json:"raw_joint"`
	ContradictionMass float64                `json:"contradiction_mass"`
	Observed          []VerifiedMentalResult `json:"observed"`
	ParameterReadNS   int64                  `json:"parameter_read_ns"`
	GRUNS             int64                  `json:"gru_ns"`
	DecodeNS          int64                  `json:"decode_ns"`
	AssessmentBMPSNS  int64                  `json:"assessment_bmps_ns"`
	ActorNS           int64                  `json:"actor_ns"`
	UtilityNS         int64                  `json:"utility_ns"`
	SelectorNS        int64                  `json:"selector_ns"`
	SuccessNS         int64                  `json:"success_ns"`
}

func (k *KnowledgeBase) PreviewPhase2V5(req Phase2ProbeV5) (Phase2PredictionV5, error) {
	return k.previewPhase3V5(req, nil)
}

// The ordinary inference API above never accepts interventions. This separate
// entry point is for explicitly labelled offline causal experiments only.
func (k *KnowledgeBase) PreviewPhase3DiagnosticV5(req Phase2ProbeV5, do Phase3InterventionV5) (Phase2PredictionV5, error) {
	if !do.OfflineOnly {
		return Phase2PredictionV5{}, errors.New("causal interventions require explicit offline-only contract")
	}
	return k.previewPhase3V5(req, &do)
}

func (k *KnowledgeBase) previewPhase3V5(req Phase2ProbeV5, do *Phase3InterventionV5) (Phase2PredictionV5, error) {
	k.RLock()
	defer k.RUnlock()
	out := Phase2PredictionV5{}
	s := k.BrainState.CognitiveModel
	if s == nil || s.NeuralMeta == nil || len(req.Trace) == 0 || len(req.Trace) > s.Config.Horizon {
		return out, errors.New("phase2 trace requires one canonical native model and bounded observed history")
	}
	c, n := s.Config, *s.Config.NeuralMeta
	start := time.Now()
	w, err := k.cognitiveWeights(s)
	if err != nil {
		return out, err
	}
	if do != nil {
		w = append([]float64(nil), w...)
		if err = do.applyParameters(w); err != nil {
			return out, err
		}
	}
	out.ParameterReadNS = time.Since(start).Nanoseconds()
	h := make([]float64, c.Hidden)
	observed := map[int]int{}
	var x CognitiveInput
	var features []float64
	start = time.Now()
	for _, input := range req.Trace {
		if err := input.Validate(c); err != nil {
			return out, err
		}
		if input.NeuralMeta.Mode == "adapter-bmps" || input.Belief != nil {
			return out, errors.New("reference probabilities cannot enter experimental learned-world inference")
		}
		x = input
		if x.Reset || x.NoRecurrence {
			clear(h)
		}
		features = append([]float64(nil), x.Features...)
		if x.NeuralMeta.NoFeedback {
			clear(features[n.ResultOffset : n.ResultOffset+n.Outcomes])
			clear(features[n.OperationOffset : n.OperationOffset+n.Computations])
		} else {
			op, o := -1, -1
			for j := 0; j < n.Computations; j++ {
				if features[n.OperationOffset+j] == 1 {
					if op >= 0 {
						return out, errors.New("ambiguous observed operation")
					}
					op = j
				}
			}
			for j := 0; j < n.Outcomes; j++ {
				if features[n.ResultOffset+j] == 1 {
					if o >= 0 {
						return out, errors.New("ambiguous observed result")
					}
					o = j
				}
			}
			if op >= 0 && o >= 0 {
				if old, ok := observed[op]; ok && old != o {
					return out, errors.New("conflicting actual observations")
				}
				observed[op] = o
			}
		}
		h = cognitiveGRU(c, s.Layout, w, features, h).h
	}
	out.GRUNS = time.Since(start).Nanoseconds()
	worldHidden := h
	if do != nil && do.WorldHidden != nil {
		if err = phase3VectorV5(do.WorldHidden, c.Hidden, false); err != nil {
			return out, err
		}
		worldHidden = append([]float64(nil), do.WorldHidden...)
	}
	start = time.Now()
	var z []float64
	if req.CategoricalAblation {
		z = mat(w, s.NeuralMeta.Joint, worldHidden)
		addBias(z, w, s.NeuralMeta.JointBias)
	} else {
		z, err = phase2LogitsV5(s, w, worldHidden, req.Dense)
		if err != nil {
			return out, err
		}
	}
	temps := k.metaTemperatures(s)
	temp := temps[0]
	if req.RawTemperature {
		temp = 1
	}
	joint := metaSoftmax(z, temp)
	if do != nil && do.Joint != nil {
		if err = phase3VectorV5(do.Joint, n.States(), true); err != nil {
			return out, err
		}
		joint = append([]float64(nil), do.Joint...)
	}
	out.RawJoint = append([]float64(nil), joint...)
	allowed := make([]bool, len(joint))
	mass := 0.
	for state, v := range joint {
		allowed[state] = true
		for op, o := range observed {
			if metaOutcome(n, state, op) != o {
				allowed[state] = false
				break
			}
		}
		if allowed[state] {
			mass += v
		}
	}
	out.ContradictionMass = 1 - mass
	if req.ConditionObserved {
		if mass <= 0 {
			return out, errors.New("impossible observed conditioning; no oracle fallback")
		}
		for state := range joint {
			if allowed[state] {
				joint[state] /= mass
			} else {
				joint[state] = 0
			}
		}
	}
	for op := 0; op < n.Computations; op++ {
		if o, ok := observed[op]; ok {
			out.Observed = append(out.Observed, VerifiedMentalResult{Operation: op, Outcome: o})
		}
	}
	out.DecodeNS = time.Since(start).Nanoseconds()
	if do != nil && do.ReadoutHidden != nil {
		if err = phase3VectorV5(do.ReadoutHidden, c.Hidden, false); err != nil {
			return out, err
		}
		h = append([]float64(nil), do.ReadoutHidden...)
	}
	start = time.Now()
	p := metaClassBelief(n, joint)
	actor, _, value := cognitiveHeads(s.Layout, w, h, x.Mask)
	out.ActorNS = time.Since(start).Nanoseconds()
	utilityStart := time.Now()
	r := CognitiveReadout{Hidden: h, Policy: actor, Value: value, ParametersHash: k.cognitiveParameterDigest(w), ClassBelief: p, JointBelief: joint, Success: make([]float64, 5), AssessmentMeaning: "phase2 fixed-trace inference with frozen native MUSE readouts"}
	muse := x.NeuralMeta.Mode == "muse-bmps"
	var u, q [][]float64
	if x.NeuralMeta.Mode != "recurrent" || x.NeuralMeta.Diagnostics {
		u, q = metaUtilities(s, w, h, features, p, temps, muse)
	}
	if do != nil {
		if do.ActorPolicy != nil {
			if err = phase3VectorV5(do.ActorPolicy, c.Actions, true); err != nil {
				return out, err
			}
			for a, v := range do.ActorPolicy {
				if !x.Mask[a] && v != 0 {
					return out, errors.New("intervened actor violates action mask")
				}
			}
			actor = append([]float64(nil), do.ActorPolicy...)
			r.Policy = actor
		}
		if do.Competence != nil {
			if err = phase3MatrixV5(do.Competence, n.Classes, n.Classes+1); err != nil {
				return out, err
			}
			q = do.Competence
			if muse {
				for y := range u {
					for a := 0; a < n.Classes; a++ {
						u[y][a] = q[y][a]
					}
				}
			}
		}
		if do.Reward != nil {
			if err = phase3MatrixV5(do.Reward, n.Classes, n.Classes+1); err != nil {
				return out, err
			}
			for y := range u {
				for a := range u[y] {
					if !muse || a == n.Classes {
						u[y][a] = do.Reward[y][a]
					}
				}
			}
		}
	}
	if x.NeuralMeta.Diagnostics {
		r.TerminalCompetence = q
		r.TerminalReward = make([][]float64, n.Classes)
		for y := range r.TerminalReward {
			r.TerminalReward[y] = make([]float64, n.Classes+1)
			for a := 0; a <= n.Classes; a++ {
				_, r.TerminalReward[y][a] = metaPlan(s, w, h, features, y, n.Computations+a, temps)
			}
		}
		if do != nil && do.Reward != nil {
			r.TerminalReward = do.Reward
		}
	}
	out.UtilityNS = time.Since(utilityStart).Nanoseconds()
	selectorStart := time.Now()
	r.Action = samplePolicy(actor, x.Draw)
	if x.NeuralMeta.Greedy {
		r.Action = maxIndex(actor)
	}
	switch x.NeuralMeta.Mode {
	case "recurrent":
	case "bmps", "muse-bmps":
		b, mapping, e := phase2MetaBeliefV5(s, joint, u, x)
		if e != nil {
			return out, e
		}
		weights := k.cognitiveBMPSWeights(s)
		if do != nil && do.BMPSWeights != nil {
			weights = *do.BMPSWeights
		}
		d, e := SelectBMPS(b, weights)
		if e != nil {
			return out, e
		}
		r.BMPS = &d
		if d.Computation >= 0 {
			r.Action = mapping[d.Computation]
		} else {
			r.Action = n.Computations + d.TerminalAction
		}
		r.NativeLikelihood = b.Computations
	case "muse":
		return out, errors.New("phase2 tests assessment and BMPS, not a rewritten MUSE regulation rollout")
	}
	if !x.Mask[r.Action] {
		return out, errors.New("experimental native selector produced invalid action")
	}
	out.SelectorNS = time.Since(selectorStart).Nanoseconds()
	successStart := time.Now()
	answer := r.Action - n.Computations
	if answer < 0 || answer >= n.Classes {
		answer = maxIndex(p)
		if q != nil {
			scores := make([]float64, n.Classes)
			for y, py := range p {
				for a := range scores {
					scores[a] += py * q[y][a]
				}
			}
			answer = maxIndex(scores)
		}
	}
	if answer < n.Classes {
		for y, py := range p {
			sa, _ := metaPlan(s, w, h, features, y, n.Computations+answer, temps)
			for i := range r.Success {
				r.Success[i] += py * sa[i]
			}
		}
	}
	out.Readout = r
	out.SuccessNS = time.Since(successStart).Nanoseconds()
	out.AssessmentBMPSNS = time.Since(start).Nanoseconds()
	return out, nil
}

// Exact conditioning can give zero-probability tuples. Compact their support
// instead of adding a floor or weakening the production underflow validator.
func phase2MetaBeliefV5(s *CognitiveModelState, joint []float64, u [][]float64, input CognitiveInput) (MetaBelief, []int, error) {
	n := *s.Config.NeuralMeta
	b := MetaBelief{Horizon: s.Config.Horizon, Remaining: input.NeuralMeta.Remaining}
	tuples := []int{}
	for theta := 0; theta < len(joint)/n.Classes; theta++ {
		mass := 0.
		v := make([]float64, n.Classes+1)
		for y := 0; y < n.Classes; y++ {
			q := joint[theta*n.Classes+y]
			mass += q
			for a := range v {
				v[a] += q * u[y][a]
			}
		}
		if mass == 0 {
			continue
		}
		for a := range v {
			v[a] /= mass
		}
		tuples = append(tuples, theta)
		b.Probabilities = append(b.Probabilities, mass)
		b.Utilities = append(b.Utilities, v)
	}
	mapping := []int{}
	for op := 0; op < n.Computations; op++ {
		if !input.Mask[op] {
			continue
		}
		c := MetaComputation{Cost: input.NeuralMeta.Costs[op]}
		for _, theta := range tuples {
			o := metaOutcome(n, theta*n.Classes, op)
			row := make([]float64, n.Outcomes)
			row[o] = 1
			c.Likelihood = append(c.Likelihood, row)
			c.Groups = append(c.Groups, o)
		}
		mapping = append(mapping, op)
		b.Computations = append(b.Computations, c)
	}
	if e := b.Validate(); e != nil {
		return b, nil, e
	}
	return b, mapping, nil
}
