package knowledge

import (
	"errors"
	"math"
	"slices"
	"time"
	"unicode/utf8"
)

const (
	SpellingCodecV1       = "unicode-synaptic-v1"
	MaxSpellingScalars    = 64
	MaxSpellingPopulation = 64
	spellingScalarBits    = 21
	spellingEndBit        = spellingScalarBits
)

var spellingScope = ProjectionScope{Modality: "written-output", Field: SpellingCodecV1}

// SpellingExposure is paired numerical I/O supervision. Population identifies
// an existing neural state, not a word or concept ID. Scalars are transport
// numbers only; learned spellings live in ordinary shared-Brain synapses.
type SpellingExposure struct {
	Codec      string   `json:"codec"`
	Population []NodeID `json:"population"`
	Scalars    []uint32 `json:"scalars"`
	Weight     float64  `json:"weight"`
	Confidence float64  `json:"confidence"`
}

func (e SpellingExposure) Validate() error {
	if e.Codec != SpellingCodecV1 || len(e.Population) == 0 || len(e.Population) > MaxSpellingPopulation || len(e.Scalars) > MaxSpellingScalars {
		return errors.New("unsupported spelling codec or exposure size")
	}
	for _, value := range []float64{e.Weight, e.Confidence} {
		if math.IsNaN(value) || math.IsInf(value, 0) || value <= 0 || value > 1 {
			return errors.New("spelling supervision requires finite weight/confidence in (0,1]")
		}
	}
	if err := validateSpellingPopulation(e.Population); err != nil {
		return err
	}
	for _, scalar := range e.Scalars {
		if !validSpellingScalar(scalar) {
			return errors.New("spelling supervision contains an invalid Unicode scalar")
		}
	}
	return nil
}

// NewSpellingExposure only converts external text into Unicode scalars. It
// performs no lexical lookup, neural recruitment, learning or interpretation.
func NewSpellingExposure(population []NodeID, text string, weight, confidence float64) (SpellingExposure, error) {
	if err := validateSpellingPopulation(population); err != nil {
		return SpellingExposure{}, err
	}
	if !utf8.ValidString(text) {
		return SpellingExposure{}, errors.New("spelling text is not valid UTF-8")
	}
	e := SpellingExposure{Codec: SpellingCodecV1, Population: slices.Clone(population), Weight: weight, Confidence: confidence}
	for _, scalar := range text {
		if len(e.Scalars) == MaxSpellingScalars {
			return SpellingExposure{}, errors.New("spelling text exceeds scalar budget")
		}
		e.Scalars = append(e.Scalars, uint32(scalar))
	}
	return e, e.Validate()
}

type SpellingStatus string

const (
	SpellingDecoded    SpellingStatus = "decoded"
	SpellingUntrained  SpellingStatus = "untrained"
	SpellingIncomplete SpellingStatus = "incomplete-support"
	SpellingAmbiguous  SpellingStatus = "ambiguous"
	SpellingInvalid    SpellingStatus = "invalid-scalar"
	SpellingLimit      SpellingStatus = "limit-reached"
)

// SpellingReadout is an I/O realization, not a semantic answer. Margin and
// conductance describe opposing learned connections; neither is a probability
// of understanding or correctness. Text/scalars are returned only on completion.
type SpellingReadout struct {
	Status      SpellingStatus `json:"status"`
	Text        string         `json:"text,omitempty"`
	Scalars     []uint32       `json:"scalars,omitempty"`
	MinMargin   float64        `json:"min_margin"`
	Conductance float64        `json:"conductance"`
	Position    int            `json:"position"`
}

type spellingCoordinate struct{ position, bit int }

func validSpellingScalar(s uint32) bool { return s <= utf8.MaxRune && (s < 0xd800 || s > 0xdfff) }

func validateSpellingPopulation(ids []NodeID) error {
	if len(ids) > MaxSpellingPopulation {
		return errors.New("spelling population exceeds readout budget")
	}
	seen := map[NodeID]bool{}
	for _, id := range ids {
		if id <= 0 || seen[id] {
			return errors.New("spelling population contains zero, negative or duplicate IDs")
		}
		seen[id] = true
	}
	return nil
}

// Caller holds k.mu. The only receptive-field metadata here is an I/O bit
// coordinate: position/64 and bit/32 are exact dyadic values. No character,
// codepoint, spelling, source population or semantic label is stored in it.
func (k *KnowledgeBase) spellingBankLocked() (map[spellingCoordinate]NodeID, map[NodeID]bool, error) {
	bank := map[spellingCoordinate]NodeID{}
	members := map[NodeID]bool{}
	k.projectionMu.RLock()
	defer k.projectionMu.RUnlock()
	for _, p := range k.ProjectionPopulations {
		if p.Scope == nil || *p.Scope != spellingScope {
			continue
		}
		if len(p.Prototype) != 2 || len(p.Units) != 1 {
			return nil, nil, errors.New("invalid spelling bit bank")
		}
		position, bit := p.Prototype[0]*64, p.Prototype[1]*32
		if math.IsNaN(position) || math.IsInf(position, 0) || position < 0 || position > MaxSpellingScalars || math.Trunc(position) != position || math.IsNaN(bit) || math.IsInf(bit, 0) || bit < 0 || bit > spellingEndBit || math.Trunc(bit) != bit {
			return nil, nil, errors.New("invalid spelling bit coordinates")
		}
		coordinate := spellingCoordinate{int(position), int(bit)}
		id := p.Units[0].NodeID
		if bank[coordinate] != 0 || members[id] || k.Registry.GetByID(id) == nil {
			return nil, nil, errors.New("duplicate or missing spelling bit unit")
		}
		bank[coordinate], members[id] = id, true
	}
	return bank, members, nil
}

func (k *KnowledgeBase) validateSpellingSourcesLocked(ids []NodeID, bankMembers map[NodeID]bool) error {
	if err := validateSpellingPopulation(ids); err != nil {
		return err
	}
	for _, id := range ids {
		if k.Registry.GetByID(id) == nil || bankMembers[id] {
			return errors.New("spelling source must be an existing non-output neural unit")
		}
	}
	return nil
}

// ValidateSpellingExposure performs read-only preflight before journal append.
func (k *KnowledgeBase) ValidateSpellingExposure(e SpellingExposure) error {
	if k == nil {
		return ErrNilBrain
	}
	if err := e.Validate(); err != nil {
		return err
	}
	k.mu.RLock()
	defer k.mu.RUnlock()
	_, members, err := k.spellingBankLocked()
	if err != nil {
		return err
	}
	return k.validateSpellingSourcesLocked(e.Population, members)
}

// LearnSpellingAt recruits shared output bit cells and teaches their ordinary
// excitatory/inhibitory synapses. It does not store an answer string or create
// another Brain. Opposite evidence uses the existing reconsolidation gain and
// retains the previous connection/frequency instead of deleting its history.
func (k *KnowledgeBase) LearnSpellingAt(e SpellingExposure, at time.Time) error {
	if k == nil {
		return ErrNilBrain
	}
	if err := e.Validate(); err != nil {
		return err
	}
	if at.IsZero() {
		return errors.New("spelling learning requires event time")
	}
	at = at.UTC()
	k.mu.Lock()
	defer k.mu.Unlock()
	bank, members, err := k.spellingBankLocked()
	if err != nil {
		return err
	}
	if err := k.validateSpellingSourcesLocked(e.Population, members); err != nil {
		return err
	}
	sources := slices.Clone(e.Population)
	slices.Sort(sources)
	for position := 0; position <= len(e.Scalars); position++ {
		for bit := 0; bit <= spellingEndBit; bit++ {
			end := position == len(e.Scalars)
			if end && bit != spellingEndBit {
				continue
			}
			coordinate := spellingCoordinate{position, bit}
			id := bank[coordinate]
			if id == 0 {
				vector := NewNeuralVector([]float64{float64(position) / 64, float64(bit) / 32})
				node := k.createRepresentationPrototypeAt(vector, at)
				id = node.ID
				bank[coordinate] = id
				k.registerProjectionPopulation(ProjectionPopulation{Scope: cloneProjectionScope(&spellingScope), Prototype: vector.Values, Units: []PopulationUnit{{NodeID: id, Activation: 1}}, Provenance: BootstrapProvenance{Origin: "supervised_numeric_io", DirectExperience: false, Status: "learned_output_coordinates"}})
			}
			on := end
			if bit != spellingEndBit {
				on = e.Scalars[position]&(1<<uint(bit)) != 0
			}
			target := k.Registry.GetByID(id)
			for _, sourceID := range sources {
				source := k.Registry.GetByID(sourceID)
				k.Registry.mu.Lock()
				if opposing := source.Synapses[id].FindDynamic(on); opposing != nil {
					delta := ReconsolidationGain * e.Confidence * clamp01(opposing.Dynamic.Weight) * clamp01(opposing.Dynamic.Confidence)
					opposing.Dynamic.Weight = clamp01(opposing.Dynamic.Weight - delta)
					opposing.Dynamic.Confidence = clamp01(opposing.Dynamic.Confidence - delta*.5)
					opposing.Dynamic.LastModification = at
					syncSynapseLegacyState(opposing)
				}
				k.Registry.mu.Unlock()
				k.ConnectAt(source, target, e.Weight, e.Confidence, !on, at)
			}
		}
	}
	return nil
}

// ReadLearnedSpelling forwards a detached, equally weighted population through
// learned signed conductance. Bit coordinates merely drive the Unicode output
// codec. Labels, source fingerprints, node representations, pattern result names
// and the input/teacher text are never read. No neural state/time is modified.
func (k *KnowledgeBase) ReadLearnedSpelling(population []NodeID) (SpellingReadout, error) {
	out := SpellingReadout{Status: SpellingUntrained}
	if k == nil {
		return out, ErrNilBrain
	}
	if len(population) == 0 {
		return out, nil
	}
	k.mu.RLock()
	defer k.mu.RUnlock()
	bank, members, err := k.spellingBankLocked()
	if err != nil {
		return out, err
	}
	if err := k.validateSpellingSourcesLocked(population, members); err != nil {
		return out, err
	}
	sources := slices.Clone(population)
	slices.Sort(sources)
	// ConnectAt also serves low-level numerical adapters and serializes its
	// dynamic fields with the registry lock. Hold that read lock while pooling
	// conductance, without recursively calling GetByID under a pending writer.
	k.Registry.mu.RLock()
	defer k.Registry.mu.RUnlock()
	out.MinMargin, out.Conductance = 1, math.Inf(1)
	readBit := func(position, bit int) (bool, SpellingStatus) {
		id := bank[spellingCoordinate{position, bit}]
		if id == 0 {
			return false, SpellingUntrained
		}
		positive, negative, covered := 0.0, 0.0, 0
		for _, sourceID := range sources {
			node := k.Registry.byID[sourceID]
			known := false
			for _, inhibitory := range []bool{false, true} {
				s := node.Synapses[id].FindDynamic(inhibitory)
				if s == nil || s.Dynamic.Frequency <= 0 {
					continue
				}
				weight, confidence := s.Dynamic.Weight, s.Dynamic.Confidence
				if math.IsNaN(weight) || math.IsInf(weight, 0) || weight < 0 || weight > 1 || math.IsNaN(confidence) || math.IsInf(confidence, 0) || confidence < 0 || confidence > 1 {
					continue
				}
				conductance := weight * confidence
				known = known || conductance > 0
				if inhibitory {
					negative += conductance
				} else {
					positive += conductance
				}
			}
			if known {
				covered++
			}
		}
		if covered == 0 {
			return false, SpellingUntrained
		}
		if covered != len(sources) {
			return false, SpellingIncomplete
		}
		margin := math.Abs(positive-negative) / (positive + negative)
		out.MinMargin = min(out.MinMargin, margin)
		out.Conductance = min(out.Conductance, (positive+negative)/float64(len(sources)))
		if margin <= 1e-12 {
			return false, SpellingAmbiguous
		}
		return positive > negative, SpellingDecoded
	}
	var scalars []uint32
	for position := 0; position <= MaxSpellingScalars; position++ {
		out.Position = position
		end, status := readBit(position, spellingEndBit)
		if status != SpellingDecoded {
			out.Status = status
			break
		}
		if end {
			out.Status, out.Scalars = SpellingDecoded, scalars
			runes := make([]rune, len(scalars))
			for i, scalar := range scalars {
				runes[i] = rune(scalar)
			}
			out.Text = string(runes)
			return out, nil
		}
		if position == MaxSpellingScalars {
			out.Status = SpellingLimit
			break
		}
		var scalar uint32
		for bit := 0; bit < spellingScalarBits; bit++ {
			on, status := readBit(position, bit)
			if status != SpellingDecoded {
				out.Status = status
				goto incomplete
			}
			if on {
				scalar |= 1 << uint(bit)
			}
		}
		if !validSpellingScalar(scalar) {
			out.Status = SpellingInvalid
			break
		}
		scalars = append(scalars, scalar)
	}
incomplete:
	if math.IsInf(out.Conductance, 0) {
		out.Conductance, out.MinMargin = 0, 0
	}
	return out, nil
}
