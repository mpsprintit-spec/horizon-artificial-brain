package runtime

// Opt-in, bounded research adapter. Token IDs, coordinate addresses and Unicode
// detokenization are I/O scaffolding. Lexical roles, reference, time, agreement,
// compatibility and each emitted token are fitted by the active V6 Brain.
// This is not the ordinary Pulse language path or a general language model.
import (
	"errors"
	"math"
	"slices"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	"github.com/project-horizon/horizon-core/services/ai/knowledge"
)

const DialogueLexicalHeads = 7
const DialogueSurfaceSlots = 6

// One quarter of the smallest referent-code gap (0.25); numeric I/O
// precision, not a confidence threshold. Surface precision uses its own alphabet.
const dialoguePrecision = .25 / 4

type DialogueInputToken struct {
	Form string `json:"form"`
	Slot int    `json:"slot"`
}
type DialogueOutputToken struct {
	Code       float64            `json:"code"`
	Population []knowledge.NodeID `json:"population"`
}
type DialogueCodec struct {
	Version    string                `json:"version"`
	Input      []DialogueInputToken  `json:"input"`
	Output     []DialogueOutputToken `json:"output"`
	OutputStep float64               `json:"output_step"`
}
type DialogueRoom struct {
	Current    []float64 `json:"current"`
	Previous   []float64 `json:"previous"`
	Kinds      []float64 `json:"kinds"`
	ObservedAt time.Time `json:"observed_at"`
	PreviousAt time.Time `json:"previous_at"`
}
type GroundedDialogueRequest struct {
	Text    string       `json:"text"`
	History []string     `json:"history,omitempty"` // Observed utterances, never gold answers.
	Room    DialogueRoom `json:"room"`
	At      time.Time    `json:"at"`
}
type DialogueNumericTrace struct {
	Head      int                          `json:"head"`
	Fields    map[int]float64              `json:"fields"`
	Precision float64                      `json:"precision"`
	Readout   knowledge.CompositionReadout `json:"readout"`
}
type GroundedDialogueReadout struct {
	Status        string                      `json:"status"`
	Reason        string                      `json:"reason,omitempty"`
	Text          string                      `json:"text,omitempty"`
	Words         []knowledge.SpellingReadout `json:"words,omitempty"`
	Focus         float64                     `json:"focus"`
	Temporal      float64                     `json:"temporal"`
	ObservedState float64                     `json:"observed_state"`
	Agreement     float64                     `json:"agreement"`
	MeaningValid  bool                        `json:"meaning_valid"`
	Trace         []DialogueNumericTrace      `json:"trace"`
}

// Query coordinates distinguish learned tasks; they contain no word meanings.
func DialogueQuestion(head int) []float64 {
	q := make([]float64, 16)
	q[0], q[1], q[2] = .625, float64(head)/64, -.375
	return q
}

// Separate numeric fields preserve the boundaries between a written alphabet,
// contextual interpretation and a selected meaning sent to the speech motor.
func DialogueNumericInput(head int, fields map[int]float64, at time.Time, precision float64) knowledge.CompositionInput {
	rows := make([][]float64, 64)
	for i := range rows {
		rows[i] = []float64{0, 0, 0, 0, 0, 0, 1}
	}
	for index, value := range fields {
		if index >= 0 && index < 384 {
			rows[63-index/6][index%6] = value
		}
	}
	return knowledge.CompositionInput{Words: [][]float64{DialogueQuestion(head)}, Sequence: rows,
		RequiredPrecision: &precision, Scene: knowledge.CompositionScene{ObservedAt: at, Point: []float64{0, 0, -1}, Patches: []knowledge.CompositionPatch{{Values: []float64{0, 0, 0, -1, 0}, XY: []float64{0, 0}}}}}
}

func (c DialogueCodec) Validate() error {
	if c.Version != "grounded-dialogue-io-v1" || len(c.Input) == 0 || len(c.Input) > 40 || len(c.Output) == 0 || len(c.Output) > 40 || !dialogueFinite(c.OutputStep) || c.OutputStep <= 0 || c.OutputStep > 1 {
		return errors.New("invalid dialogue codec")
	}
	forms, slots := map[string]bool{}, map[int]bool{}
	for _, x := range c.Input {
		if x.Form == "" || len(x.Form) > 64 || x.Form != strings.ToLower(x.Form) || x.Slot < 0 || x.Slot >= 40 || forms[x.Form] || slots[x.Slot] {
			return errors.New("invalid or repeated input alphabet coordinate")
		}
		forms[x.Form], slots[x.Slot] = true, true
	}
	for i, x := range c.Output {
		if !dialogueFinite(x.Code) || x.Code <= 0 || x.Code > 1 || len(x.Population) == 0 || len(x.Population) > knowledge.MaxSpellingPopulation {
			return errors.New("invalid output alphabet coordinate")
		}
		seen := map[knowledge.NodeID]bool{}
		for _, id := range x.Population {
			if id == 0 || seen[id] {
				return errors.New("invalid output population")
			}
			seen[id] = true
		}
		for j := 0; j < i; j++ {
			if math.Abs(c.Output[j].Code-x.Code) < c.OutputStep*.999999 {
				return errors.New("overlapping output alphabet coordinates")
			}
		}
	}
	return nil
}
func dialogueFinite(x float64) bool { return !math.IsNaN(x) && !math.IsInf(x, 0) }
func dialogueTokens(s string) []string {
	return strings.FieldsFunc(strings.ToLower(s), func(r rune) bool { return !unicode.IsLetter(r) && !unicode.IsNumber(r) && r != '-' })
}
func dialogueSnap(x float64, alphabet []float64) (float64, bool) {
	best, distance := 0., math.Inf(1)
	for _, a := range alphabet {
		if d := math.Abs(a - x); d < distance {
			best, distance = a, d
		}
	}
	gap := math.Inf(1)
	for i, a := range alphabet {
		for _, b := range alphabet[:i] {
			if d := math.Abs(a - b); d > 0 {
				gap = math.Min(gap, d)
			}
		}
	}
	return best, distance <= gap/4
}
func dialogueText(words []string) string {
	var b strings.Builder
	for _, w := range words {
		if w == "" {
			continue
		}
		punct := w == "." || w == "," || w == "?" || w == "!" || w == ":" || w == ";"
		if b.Len() > 0 && !punct {
			b.WriteByte(' ')
		}
		if b.Len() == 0 {
			r, n := utf8.DecodeRuneInString(w)
			w = string(unicode.ToUpper(r)) + w[n:]
		}
		b.WriteString(w)
	}
	return b.String()
}

// ReadGroundedDialogue holds one canonical owner revision for interpretation and
// synaptic spelling. It neither learns nor mutates conversational memory. Raw
// history is replayed through the learned interpreter for every call.
func (r *BrainRuntime) ReadGroundedDialogue(codec DialogueCodec, request GroundedDialogueRequest) (GroundedDialogueReadout, error) {
	out := GroundedDialogueReadout{Status: "untrained"}
	if r == nil || r.brain == nil {
		return out, errors.New("runtime unavailable")
	}
	if err := codec.Validate(); err != nil {
		return out, err
	}
	codec, err := freezeComposition(codec)
	if err != nil {
		return out, err
	}
	request, err = freezeComposition(request)
	if err != nil {
		return out, err
	}
	room := request.Room
	if request.At.IsZero() || room.ObservedAt.IsZero() || room.PreviousAt.IsZero() || room.ObservedAt.After(request.At) || room.PreviousAt.After(room.ObservedAt) || len(room.Current) != 3 || len(room.Previous) != 3 || len(room.Kinds) != 3 || len(request.History) > 16 || len(request.Text) > 1024 {
		return out, errors.New("invalid or future dialogue observations")
	}
	for _, v := range [][]float64{room.Current, room.Previous, room.Kinds} {
		for _, x := range v {
			if !dialogueFinite(x) || math.Abs(x) != 1 {
				return out, errors.New("dialogue sensor alphabet is binary")
			}
		}
	}
	for _, s := range request.History {
		if len(s) > 1024 {
			return out, errors.New("dialogue history bound exceeded")
		}
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if err := r.ensureWritableLocked(); err != nil {
		return out, err
	}
	if r.composition == nil || r.composition.Config.Version != knowledge.NeuralCompositionMetricV6 {
		return out, errors.New("dialogue requires explicit V6 research Brain")
	}
	read := func(head int, fields map[int]float64, precision float64) (float64, bool, error) {
		in := DialogueNumericInput(head, fields, request.At, precision)
		o, e := r.brain.ReadComposition(r.composition.Config, in, request.At)
		o.Sequence, o.BrainIdentity = r.seq, BrainIdentity
		copied := make(map[int]float64, len(fields))
		for k, v := range fields {
			copied[k] = v
		}
		out.Trace = append(out.Trace, DialogueNumericTrace{Head: head, Fields: copied, Precision: precision, Readout: o})
		if e != nil {
			return 0, false, e
		}
		if !o.Accepted || o.NResolution == nil {
			return 0, false, nil
		}
		return o.NResolution.Value, true, nil
	}
	lexical := func(text string) ([]float64, string, error) {
		tokens := dialogueTokens(text)
		if len(tokens) == 0 || len(tokens) > 8 {
			return nil, "utterance-bound", nil
		}
		pool := make([]float64, DialogueLexicalHeads)
		for _, word := range tokens {
			slot := -1
			for _, entry := range codec.Input {
				if word == entry.Form {
					slot = entry.Slot
					break
				}
			}
			if slot < 0 {
				return nil, "untrained-token", nil
			}
			for h := 0; h < DialogueLexicalHeads; h++ {
				v, ok, e := read(h, map[int]float64{slot: 1}, dialoguePrecision)
				if e != nil {
					return nil, "", e
				}
				if !ok {
					return nil, "lexical-precision", nil
				}
				alphabet := []float64{-1, 0, 1}
				if h == 0 {
					alphabet = []float64{0, .25, .5, .75}
				}
				v, ok = dialogueSnap(v, alphabet)
				if !ok {
					return nil, "lexical-code", nil
				}
				pool[h] += v
			}
		}
		return pool, "", nil
	}
	focus := 0.
	interpretFocus := func(pool []float64) (float64, bool, error) {

		if pool[0] > 3 || pool[1] > 3 {
			return 0, false, nil
		}
		v, ok, e := read(7, map[int]float64{48: pool[0] / 2, 49: pool[1] / 2, 50: focus, 51: 0}, dialoguePrecision)
		if e != nil || !ok {
			return 0, false, e
		}
		v, ok = dialogueSnap(v, []float64{0, .25, .5, .75})
		return v, ok, nil
	}
	for _, turn := range request.History {
		pool, reason, e := lexical(turn)
		if e != nil {
			return out, e
		}
		if reason != "" {
			focus = 0
			continue
		}
		v, ok, e := interpretFocus(pool)
		if e != nil {
			return out, e
		}
		if ok {
			focus = v
		} else {
			focus = 0
		}
	}
	pool, reason, err := lexical(request.Text)
	if err != nil {
		return out, err
	}
	valid, kind, state, temporal, agreement, sensorKind := 0., 0., 0., 1., 0., 0.
	ref := 0.
	if reason == "" {
		ok := false
		ref, ok, err = interpretFocus(pool)
		if err != nil {
			return out, err
		}
		if !ok {
			out.Reason = "reference-precision"
			return out, nil
		}
		if math.Abs(pool[2]) > 1 || math.Abs(pool[3]) != 1 || pool[4] > 1 || math.Abs(pool[5]) > 1 || math.Abs(pool[6]) > 1 {
			reason = "unsupported-composition"
		} else {
			temporal, ok, err = read(8, map[int]float64{54: pool[5]}, dialoguePrecision)
			if err != nil {
				return out, err
			}
			if !ok {
				out.Reason = "temporal-precision"
				return out, nil
			}
			temporal, ok = dialogueSnap(temporal, []float64{-1, 1})
			if !ok {
				out.Reason = "temporal-code"
				return out, nil
			}
			// Addressing a measured coordinate is sensor I/O. The address and
			// temporal choice above came from learned heads, not word rules.
			index := int(math.Round(ref*4)) - 1
			found := 0.
			if index >= 0 && index < len(room.Current) {
				found = 1
				state = room.Current[index]
				if temporal < 0 {
					state = room.Previous[index]
				}
				sensorKind = room.Kinds[index]
			}
			kind = pool[3]
			if found == 1 {
				agreement, ok, err = read(9, map[int]float64{60: state, 61: pool[2], 62: pool[4], 63: kind}, dialoguePrecision)
				if err != nil {
					return out, err
				}
				if !ok {
					out.Reason = "agreement-precision"
					return out, nil
				}
				agreement, ok = dialogueSnap(agreement, []float64{-1, 0, 1})
				if !ok {
					out.Reason = "agreement-code"
					return out, nil
				}
				valid, ok, err = read(10, map[int]float64{64: found, 63: kind, 66: pool[6], 67: sensorKind, 68: math.Abs(pool[2])}, dialoguePrecision)
				if err != nil {
					return out, err
				}
				if !ok {
					out.Reason = "compatibility-precision"
					return out, nil
				}
				valid, ok = dialogueSnap(valid, []float64{0, 1})
				if !ok {
					out.Reason = "compatibility-code"
					return out, nil
				}
			}
		}
	}
	if reason != "" {
		out.Reason = reason
		valid = 0
	}
	out.Focus, out.Temporal, out.ObservedState, out.Agreement, out.MeaningValid = ref, temporal, state, agreement, valid == 1
	fields := map[int]float64{72: ref, 60: state, 74: temporal, 63: kind, 76: agreement, 77: valid, 67: sensorKind}
	if valid == 0 {
		fields = map[int]float64{77: 0}
		if out.Reason == "" {
			out.Reason = "unresolved-meaning"
		}
	}
	words := []string{}
	for position := 0; position < DialogueSurfaceSlots; position++ {
		value, ok, e := read(11+position, fields, codec.OutputStep/4)
		if e != nil {
			return out, e
		}
		if !ok {
			out.Reason = "surface-precision"
			return out, nil
		}
		if math.Abs(value) <= codec.OutputStep/4 {
			continue
		}
		var ids []knowledge.NodeID
		for _, entry := range codec.Output {
			if math.Abs(value-entry.Code) <= codec.OutputStep/4 {
				ids = slices.Clone(entry.Population)
				break
			}
		}
		if len(ids) == 0 {
			out.Reason = "surface-code"
			return out, nil
		}
		spelling, e := r.brain.ReadLearnedSpelling(ids)
		if e != nil {
			return out, e
		}
		out.Words = append(out.Words, spelling)
		if spelling.Status != knowledge.SpellingDecoded {
			out.Reason = "untrained-spelling"
			return out, nil
		}
		words = append(words, spelling.Text)
	}
	out.Text = dialogueText(words)
	out.Status = "answered"
	if valid == 0 {
		out.Status = "clarification-needed"
	}
	return out, nil
}
