package knowledge

// Exact finite-support realization of Callaway et al. (2018), eqs. 2--5.
// No Horizon uncertainty score is substituted for VOI, VPI, or VPI_sub.
import (
	"errors"
	"math"
	"math/rand"
	"sort"
	"time"
)

type MetaComputation struct {
	Cost       float64     `json:"cost"`
	Likelihood [][]float64 `json:"likelihood"` // P(result | hypothesis, computation)
	Groups     []int       `json:"groups"`     // equal labels = same relevant parameter value
}
type MetaBelief struct {
	Probabilities []float64         `json:"probabilities"`
	Utilities     [][]float64       `json:"utilities"` // hypothesis x terminal option
	Computations  []MetaComputation `json:"computations"`
	Remaining     int               `json:"remaining"`
	Horizon       int               `json:"horizon,omitempty"`
}

func (b MetaBelief) Validate() error {
	n := len(b.Probabilities)
	if n == 0 || len(b.Utilities) != n || len(b.Utilities[0]) == 0 || b.Remaining < 0 || b.Horizon < 0 || (b.Horizon > 0 && b.Remaining > b.Horizon) {
		return errors.New("invalid finite metalevel belief")
	}
	sum := 0.0
	for i, p := range b.Probabilities {
		if !finiteNumber(p) || p < 0 {
			return errors.New("invalid hypothesis probability")
		}
		sum += p
		if len(b.Utilities[i]) != len(b.Utilities[0]) {
			return errors.New("utility shape mismatch")
		}
		for _, u := range b.Utilities[i] {
			if !finiteNumber(u) {
				return errors.New("nonfinite utility")
			}
		}
	}
	if math.Abs(sum-1) > 1e-9 {
		return errors.New("belief probabilities must sum to one")
	}
	for _, c := range b.Computations {
		if !finiteNumber(c.Cost) || c.Cost <= 0 || len(c.Likelihood) != n || len(c.Groups) != n || len(c.Likelihood[0]) == 0 {
			return errors.New("invalid computation cost or transition model")
		}
		for _, row := range c.Likelihood {
			if len(row) != len(c.Likelihood[0]) {
				return errors.New("transition shape mismatch")
			}
			s := 0.0
			for _, p := range row {
				if !finiteNumber(p) || p < 0 {
					return errors.New("invalid transition probability")
				}
				s += p
			}
			if math.Abs(s-1) > 1e-9 {
				return errors.New("transition probabilities must sum to one")
			}
		}
	}
	return nil
}
func (b MetaBelief) terminal() (float64, int) {
	v := make([]float64, len(b.Utilities[0]))
	for i, p := range b.Probabilities {
		for j, u := range b.Utilities[i] {
			v[j] += p * u
		}
	}
	j := maxIndex(v)
	return v[j], j
}
func (b MetaBelief) After(computation, result int) (MetaBelief, error) {
	if err := b.Validate(); err != nil {
		return b, err
	}
	if computation < 0 || computation >= len(b.Computations) || result < 0 || result >= len(b.Computations[computation].Likelihood[0]) || b.Remaining == 0 {
		return b, errors.New("invalid metalevel transition")
	}
	p := make([]float64, len(b.Probabilities))
	s := 0.0
	for i, q := range b.Probabilities {
		p[i] = q * b.Computations[computation].Likelihood[i][result]
		s += p[i]
	}
	if s <= 0 {
		return b, errors.New("impossible metalevel observation")
	}
	for i := range p {
		p[i] /= s
	}
	b.Probabilities = p
	if b.Horizon == 0 {
		b.Horizon = b.Remaining
	}
	b.Remaining--
	return b, nil
}

type BMPSFeatures struct {
	Myopic  float64 `json:"myopic"`
	Perfect float64 `json:"perfect"`
	Subset  float64 `json:"subset"`
	Cost    float64 `json:"cost"`
}

func BMPSInformation(b MetaBelief) ([]BMPSFeatures, error) {
	if err := b.Validate(); err != nil {
		return nil, err
	}
	base, _ := b.terminal()
	vpi := 0.0
	for i, p := range b.Probabilities {
		vpi += p * b.Utilities[i][maxIndex(b.Utilities[i])]
	}
	vpi = math.Max(0, vpi-base)
	out := make([]BMPSFeatures, len(b.Computations))
	for a, c := range b.Computations {
		voi := 0.0
		for o := range c.Likelihood[0] {
			p := 0.0
			post := make([]float64, len(b.Probabilities))
			for i, q := range b.Probabilities {
				post[i] = q * c.Likelihood[i][o]
				p += post[i]
			}
			if p == 0 {
				continue
			}
			for i := range post {
				post[i] /= p
			}
			bb := b
			bb.Probabilities = post
			v, _ := bb.terminal()
			voi += p * v
		}
		groups := map[int][]float64{}
		for i, g := range c.Groups {
			if groups[g] == nil {
				groups[g] = make([]float64, len(b.Utilities[0]))
			}
			for j, u := range b.Utilities[i] {
				groups[g][j] += b.Probabilities[i] * u
			}
		}
		sub := 0.0
		keys := make([]int, 0, len(groups))
		for group := range groups {
			keys = append(keys, group)
		}
		sort.Ints(keys)
		for _, group := range keys {
			v := groups[group]
			sub += v[maxIndex(v)]
		}
		out[a] = BMPSFeatures{Myopic: math.Max(0, voi-base), Perfect: vpi, Subset: math.Max(0, sub-base), Cost: c.Cost}
	}
	return out, nil
}

type BMPSDecision struct {
	Computation     int            `json:"computation"`
	TerminalAction  int            `json:"terminal_action"`
	TerminalUtility float64        `json:"terminal_utility"`
	Features        []BMPSFeatures `json:"features"`
	VOC             []float64      `json:"voc"`
	Weights         [4]float64     `json:"weights"`
}

func ValidateBMPSWeights(w [4]float64, horizon int) error {
	if horizon < 1 {
		return errors.New("invalid BMPS horizon")
	}
	s := 0.0
	for i, x := range w {
		if !finiteNumber(x) {
			return errors.New("nonfinite BMPS weight")
		}
		if i < 3 {
			if x < 0 || x > 1 {
				return errors.New("BMPS mixture outside simplex")
			}
			s += x
		}
	}
	if math.Abs(s-1) > 1e-9 || w[3] < 1-1e-9 || w[3] > float64(horizon)+1e-9 {
		return errors.New("BMPS constraints violated")
	}
	return nil
}
func SelectBMPS(b MetaBelief, w [4]float64) (BMPSDecision, error) {
	if err := ValidateBMPSWeights(w, max(b.Horizon, b.Remaining, 1)); err != nil {
		return BMPSDecision{}, err
	}
	f, err := BMPSInformation(b)
	if err != nil {
		return BMPSDecision{}, err
	}
	u, j := b.terminal()
	d := BMPSDecision{Computation: -1, TerminalAction: j, TerminalUtility: u, Features: f, Weights: w, VOC: make([]float64, len(f))}
	best := 0.0
	for i, x := range f {
		v := w[0]*x.Myopic + w[1]*x.Perfect + w[2]*x.Subset - w[3]*x.Cost
		d.VOC[i] = v
		if b.Remaining > 0 && v > best {
			best = v
			d.Computation = i
		}
	}
	return d, nil
}

type BMPSEvaluation struct {
	Point    [3]float64 `json:"point"`
	Return   float64    `json:"return"`
	Episodes int        `json:"episodes"`
}

func BMPSWeights(point [3]float64, h int) [4]float64 {
	return [4]float64{point[0], (1 - point[0]) * point[1], (1 - point[0]) * (1 - point[1]), 1 + (float64(h)-1)*point[2]}
}
func (e BMPSEvaluation) Validate() error {
	for _, x := range e.Point {
		if !finiteNumber(x) || x < 0 || x > 1 {
			return errors.New("invalid BO coordinate")
		}
	}
	if !finiteNumber(e.Return) || e.Episodes < 1 {
		return errors.New("BO requires measured episode return")
	}
	return nil
}
func rbf(a, b [3]float64) float64 {
	d := 0.0
	for i := range a {
		d += math.Pow((a[i]-b[i])/.35, 2)
	}
	return math.Exp(-.5 * d)
}
func cholesky(a [][]float64) ([][]float64, error) {
	n := len(a)
	l := make([][]float64, n)
	for i := range l {
		l[i] = make([]float64, n)
		for j := 0; j <= i; j++ {
			s := a[i][j]
			for k := 0; k < j; k++ {
				s -= l[i][k] * l[j][k]
			}
			if i == j {
				if s <= 0 {
					return nil, errors.New("BO covariance not positive definite")
				}
				l[i][j] = math.Sqrt(s)
			} else {
				l[i][j] = s / l[j][j]
			}
		}
	}
	return l, nil
}
func cholSolve(l [][]float64, b []float64) []float64 {
	n := len(l)
	y := make([]float64, n)
	for i := range y {
		y[i] = b[i]
		for j := 0; j < i; j++ {
			y[i] -= l[i][j] * y[j]
		}
		y[i] /= l[i][i]
	}
	x := make([]float64, n)
	for i := n - 1; i >= 0; i-- {
		x[i] = y[i]
		for j := i + 1; j < n; j++ {
			x[i] -= l[j][i] * x[j]
		}
		x[i] /= l[i][i]
	}
	return x
}

// GP posterior + expected-improvement acquisition. The candidate pool is a
// numerical acquisition optimizer; it is NOT replacement grid-search of policy
// returns. Only the chosen candidate is evaluated in the environment.
func ProposeBMPS(history []BMPSEvaluation, seed int64) ([3]float64, error) {
	for _, e := range history {
		if err := e.Validate(); err != nil {
			return [3]float64{}, err
		}
	}
	rng := rand.New(rand.NewSource(seed + int64(len(history))*104729))
	point := func() [3]float64 { return [3]float64{rng.Float64(), rng.Float64(), rng.Float64()} }
	if len(history) < 4 {
		return point(), nil
	}
	n := len(history)
	a := make([][]float64, n)
	ys := make([]float64, n)
	mean, best := 0.0, math.Inf(-1)
	for _, e := range history {
		mean += e.Return
		best = math.Max(best, e.Return)
	}
	mean /= float64(n)
	scale := 0.0
	for _, e := range history {
		scale += math.Pow(e.Return-mean, 2)
	}
	scale = math.Max(.05, math.Sqrt(scale/float64(n)))
	for i, e := range history {
		a[i] = make([]float64, n)
		ys[i] = (e.Return - mean) / scale
		for j, f := range history {
			a[i][j] = rbf(e.Point, f.Point)
		}
		a[i][i] += 1e-4
	}
	l, err := cholesky(a)
	if err != nil {
		return [3]float64{}, err
	}
	alpha := cholSolve(l, ys)
	chosen := [3]float64{}
	maxEI := -1.0
	for c := 0; c < 256; c++ {
		x := point()
		if c < 8 {
			x = [3]float64{float64(c & 1), float64((c >> 1) & 1), float64((c >> 2) & 1)}
		}
		kv := make([]float64, n)
		mu := 0.0
		for i, e := range history {
			kv[i] = rbf(x, e.Point)
			mu += kv[i] * alpha[i]
		}
		v := cholSolve(l, kv)
		variance := 1.0
		for i := range v {
			variance -= kv[i] * v[i]
		}
		sd := scale * math.Sqrt(math.Max(1e-12, variance))
		mu = mean + scale*mu
		imp := mu - best
		z := imp / sd
		ei := imp*.5*(1+math.Erf(z/math.Sqrt2)) + sd*math.Exp(-z*z/2)/math.Sqrt(2*math.Pi)
		if ei > maxEI {
			maxEI = ei
			chosen = x
		}
	}
	return chosen, nil
}
func (k *KnowledgeBase) cognitiveBMPSWeights(s *CognitiveModelState) [4]float64 {
	var w [4]float64
	for i, id := range s.Layout.BMPS {
		w[i], _ = k.compositionParameter([]NodeID{s.Layout.BiasInput}, []NodeID{id})
	}
	w[3] *= float64(s.Config.Horizon)
	return w
}
func (k *KnowledgeBase) CognitiveBMPSWeights() ([4]float64, error) {
	k.RLock()
	defer k.RUnlock()
	if k.BrainState.CognitiveModel == nil {
		return [4]float64{}, errors.New("cognitive model not initialized")
	}
	return k.cognitiveBMPSWeights(k.BrainState.CognitiveModel), nil
}
func (k *KnowledgeBase) FitCognitiveBMPS(history []BMPSEvaluation, evidence CognitiveEvidence, at time.Time) error {
	if err := evidence.Validate(); err != nil {
		return err
	}
	if evidence.Digest != CognitiveDigest(history) {
		return errors.New("BO evidence digest mismatch")
	}
	if len(history) == 0 {
		return errors.New("empty BO history")
	}
	k.Lock()
	defer k.Unlock()
	s := k.BrainState.CognitiveModel
	if s == nil {
		return errors.New("cognitive model not initialized")
	}
	best := 0
	for i, e := range history {
		if err := e.Validate(); err != nil {
			return err
		}
		p, err := ProposeBMPS(history[:i], s.Config.Seed)
		if err != nil {
			return err
		}
		if p != e.Point {
			return errors.New("BO history was not generated by frozen GP/EI search")
		}
		if e.Return > history[best].Return {
			best = i
		}
	}
	w := BMPSWeights(history[best].Point, s.Config.Horizon)
	if err := ValidateBMPSWeights(w, s.Config.Horizon); err != nil {
		return err
	}
	for i, id := range s.Layout.BMPS {
		v := w[i]
		if i == 3 {
			v /= float64(s.Config.Horizon)
		}
		k.adjustCompositionParameter([]NodeID{s.Layout.BiasInput}, []NodeID{id}, v, 1, at, true)
	}
	s.Search = append([]BMPSEvaluation{}, history...)
	e := evidence
	s.LastEvidence = &e
	return nil
}
