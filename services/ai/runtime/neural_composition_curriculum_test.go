package runtime

// Test-only simulator and teacher fixtures retained from the sealed 24 October
// composition protocol. Asset names/gold actions exist only in this _test file;
// production inference receives numerical present input without these labels.
import (
	"fmt"
	"github.com/project-horizon/horizon-core/services/ai/knowledge"
	"math"
	"math/rand"
	"slices"
	"sort"
)

var roomNames = []string{"lampu", "televisi", "kamera", "mikrofon", "speaker", "pintu", "kotak"}
var roomXY = [7][2]int{{1, 1}, {3, 1}, {5, 1}, {1, 3}, {3, 3}, {5, 3}, {1, 5}}

// The world and teacher know their physical assets. Only raster values,
// pointing coordinates and contact/setpoint vectors enter the Brain.
type roomWorld struct {
	State  [7]bool `json:"state"`
	Layout int     `json:"layout"`
}

func roomWorldAt(background, layout int) roomWorld {
	w := roomWorld{Layout: layout}
	for i := range w.State {
		w.State[i] = background == 1
	}
	if background == 2 {
		w.State = [7]bool{true, false, true, false, true, false, true}
	}
	if background == 3 {
		w.State = [7]bool{false, true, false, true, false, true, false}
	}
	return w
}
func (w roomWorld) xy(index int) [2]int {
	if w.Layout == 1 {
		if index == 0 {
			return roomXY[1]
		}
		if index == 1 {
			return roomXY[0]
		}
	}
	return roomXY[index]
}
func roomNorm(v int) float64 { return 2*float64(v)/7 - 1 }

type roomCase struct {
	ID      string    `json:"id"`
	Split   string    `json:"split"`
	Words   []string  `json:"words"`
	Target  int       `json:"teacher_target"`
	Set     bool      `json:"teacher_setpoint"`
	Noop    bool      `json:"teacher_noop"`
	Pointer int       `json:"pointer"`
	Before  roomWorld `json:"before"`
}

func roomCurriculum() ([]roomCase, []roomCase) {
	train := []roomCase{}
	held := []roomCase{}
	add := func(destination *[]roomCase, split string, words []string, target int, set, noop bool, pointer int) {
		for background := 0; background < 2; background++ {
			*destination = append(*destination, roomCase{ID: fmt.Sprintf("%s-%03d", split, len(*destination)), Split: split, Words: slices.Clone(words), Target: target, Set: set, Noop: noop, Pointer: pointer, Before: roomWorldAt(background, 0)})
		}
	}
	for target, name := range roomNames {
		positive, negative, otherPositive, otherNegative := "nyalakan", "matikan", "hidupkan", "nonaktifkan"
		if target >= 5 {
			positive, negative, otherPositive, otherNegative = "buka", "tutup", "bukalah", "tutuplah"
		}
		add(&train, "seen_named", []string{positive, name}, target, true, false, -1)
		add(&train, "seen_named", []string{negative, name}, target, false, false, -1)
		positiveTrain := target == 0 || target == 2 || target == 5
		negativeTrain := target == 0 || target == 2 || target == 6
		destination, split := &held, "heldout_verb_device"
		if positiveTrain {
			destination, split = &train, "seen_synonym"
		}
		add(destination, split, []string{otherPositive, name}, target, true, false, -1)
		destination, split = &held, "heldout_verb_device"
		if negativeTrain {
			destination, split = &train, "seen_synonym"
		}
		add(destination, split, []string{otherNegative, name}, target, false, false, -1)
		add(&train, "seen_deictic", []string{positive, "itu"}, target, true, false, target)
		add(&train, "seen_deictic", []string{negative, "itu"}, target, false, false, target)
		if target < 5 {
			destination, split = &held, "heldout_negation"
			if target == 0 || target == 2 {
				destination, split = &train, "seen_negation"
			}
			add(destination, split, []string{"jangan", positive, name}, target, true, true, -1)
			add(destination, split, []string{"jangan", negative, name}, target, false, true, -1)
		}
	}
	return train, held
}
func roomExtraCases(train []roomCase) []roomCase {
	out := []roomCase{}
	for _, c := range train {
		if c.Split != "seen_named" || c.Before.State[0] {
			continue
		}
		for background := 2; background <= 3; background++ {
			next := c
			next.Split = "heldout_mixed_state"
			next.Before = roomWorldAt(background, 0)
			next.ID = fmt.Sprintf("mixed-%d-%s", background, c.ID)
			out = append(out, next)
		}
		if c.Target < 2 {
			next := c
			next.Split = "heldout_moved_asset"
			next.Before.Layout = 1
			next.ID = "moved-" + c.ID
			out = append(out, next)
		}
	}
	// Untrained forms deliberately share neither the full learned command nor
	// a semantic normalization rule. They remain out-of-training evidence.
	for target := 0; target < 5; target++ {
		for background := 0; background < 2; background++ {
			out = append(out, roomCase{fmt.Sprintf("paraphrase-%d-%d", target, background), "heldout_new_paraphrase", []string{"buat", roomNames[target], "menyala"}, target, true, false, -1, roomWorldAt(background, 0)})
		}
	}
	return out
}

type bindPatch struct {
	Values []float64  `json:"values"`
	XY     [2]float64 `json:"xy"`
}
type bindInput struct {
	Patches []bindPatch `json:"patches"`
	Point   [3]float64  `json:"point"`
	Words   [][]float64 `json:"word_vectors"`
}
type bindEpisode struct {
	ID       int         `json:"id"`
	Input    bindInput   `json:"input"`
	After    []bindPatch `json:"after"`
	Motor    []float64   `json:"motor"`
	Language bool        `json:"language"`
}
type bindWorld struct {
	State    [7]bool `json:"state"`
	Perm     [7]int  `json:"permutation"`
	Palette  [7]int  `json:"palette"`
	Polarity float64 `json:"actuator_polarity"`
}

func bindWorldNew(seed int64, background, layout int) bindWorld {
	w := bindWorld{Polarity: 1}
	if seed%2 == 1 && seed != 7 {
		w.Polarity = -1
	}
	palette := rand.New(rand.NewSource(seed + 117)).Perm(7)
	for i := 0; i < 7; i++ {
		w.Perm[i] = i
		w.Palette[i] = palette[i]
		w.State[i] = background == 1
	}
	if background == 2 {
		w.State = [7]bool{true, false, true, false, true, false, true}
	}
	if background == 3 {
		w.State = [7]bool{false, true, false, true, false, true, false}
	}
	if layout == 1 {
		w.Perm[0], w.Perm[1] = 1, 0
	}
	if layout >= 2 {
		rng := rand.New(rand.NewSource(seed*103 + int64(layout)*419))
		p := rng.Perm(7)
		for i := range w.Perm {
			w.Perm[i] = p[i]
		}
	}
	return w
}
func (w bindWorld) xy(target int) [2]float64 {
	p := roomXY[w.Perm[target]]
	return [2]float64{roomNorm(p[0]), roomNorm(p[1])}
}
func (w bindWorld) body(target int) []float64 {
	v := []float64{-.8, -.8, -.8}
	code := w.Palette[target] + 1
	for k := range v {
		if code&(1<<k) != 0 {
			v[k] = .8
		}
	}
	return v
}
func (w bindWorld) patches() []bindPatch {
	out := make([]bindPatch, 7)
	for i := 0; i < 7; i++ {
		v := append(w.body(i), -1.0, -1.0)
		if w.State[i] {
			v[3] = 1
			v[4] = 0
		}
		if i == 1 && w.State[1] && w.State[2] {
			v[4] = 1
		}
		if i == 4 && w.State[4] && w.State[3] {
			v[4] = 1
		}
		out[i] = bindPatch{v, w.xy(i)}
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].XY[1] == out[j].XY[1] {
			return out[i].XY[0] < out[j].XY[0]
		}
		return out[i].XY[1] < out[j].XY[1]
	})
	return out
}
func (w bindWorld) motor(target int, set, noop bool) []float64 {
	if noop {
		return []float64{0, 0, 0, 0}
	}
	xy := w.xy(target)
	v := -w.Polarity
	if set {
		v = w.Polarity
	}
	return []float64{xy[0], xy[1], v, 1}
}
func (w bindWorld) apply(motor []float64) (bindWorld, bool) {
	if len(motor) != 4 {
		return w, false
	}
	if motor[3] == 0 {
		return w, slices.Equal(motor, []float64{0, 0, 0, 0})
	}
	if motor[3] != 1 || math.Abs(motor[2]) != 1 {
		return w, false
	}
	target := -1
	for i := 0; i < 7; i++ {
		xy := w.xy(i)
		if math.Abs(xy[0]-motor[0]) < 1e-9 && math.Abs(xy[1]-motor[1]) < 1e-9 {
			target = i
		}
	}
	if target < 0 {
		return w, false
	}
	w.State[target] = motor[2]*w.Polarity > 0
	return w, true
}
func (w bindWorld) input(words []string, pointer int) bindInput {
	input := bindInput{Patches: w.patches(), Point: [3]float64{0, 0, -1}}
	if pointer >= 0 {
		xy := w.xy(pointer)
		input.Point = [3]float64{xy[0], xy[1], 1}
	}
	for _, word := range words {
		input.Words = append(input.Words, knowledge.EncodeObservation(word, "language").Values)
	}
	return input
}

type bindCase struct {
	ID         string   `json:"id"`
	Split      string   `json:"split"`
	Words      []string `json:"words"`
	Target     int      `json:"teacher_target"`
	Set        bool     `json:"teacher_set"`
	Noop       bool     `json:"teacher_noop"`
	Pointer    int      `json:"pointer"`
	Background int      `json:"background"`
	Layout     int      `json:"layout"`
}

func bindCases() ([]bindCase, []bindCase) {
	oldTrain, oldHeld := roomCurriculum()
	train, held := []bindCase{}, []bindCase{}
	convert := func(c roomCase) bindCase {
		bg := 0
		if c.Before.State[0] {
			bg = 1
		}
		return bindCase{c.ID, c.Split, c.Words, c.Target, c.Set, c.Noop, c.Pointer, bg, 0}
	}
	for _, c := range oldTrain {
		next := convert(c)
		if slices.Equal(c.Words, []string{"nyalakan", "televisi"}) {
			next.Split = "heldout_canonical_tv"
			held = append(held, next)
		} else {
			train = append(train, next)
		}
	}
	for _, c := range oldHeld {
		held = append(held, convert(c))
	}
	for _, c := range roomExtraCases(oldTrain) {
		next := convert(c)
		if c.Split == "heldout_mixed_state" {
			next.Background = 2
			if c.Before.State[0] == false {
				next.Background = 3
			}
		}
		next.Layout = c.Before.Layout
		held = append(held, next)
	}
	// Two trained clause referents, shared vocabulary and an explicit ordering
	// contrast. No parser is provided; numerical position gates must be learned.
	for _, pair := range [][2]int{{0, 2}, {2, 0}} {
		for _, verb := range []string{"nyalakan", "matikan"} {
			for bg := 0; bg < 2; bg++ {
				train = append(train, bindCase{fmt.Sprintf("scope-train-%d", len(train)), "seen_scope", []string{verb, roomNames[pair[0]], "jangan", roomNames[pair[1]]}, pair[0], verb == "nyalakan", false, -1, bg, 0})
			}
		}
	}
	for a := 0; a < 7; a++ {
		for b := 0; b < 7; b++ {
			if a == b || a < 5 && b >= 5 || a >= 5 && b < 5 || a == 0 && b == 2 || a == 2 && b == 0 {
				continue
			}
			verbs := []string{"nyalakan", "matikan"}
			if a >= 5 {
				verbs = []string{"buka", "tutup"}
			}
			for j, verb := range verbs {
				for bg := 0; bg < 2; bg++ {
					held = append(held, bindCase{fmt.Sprintf("scope-held-%d", len(held)), "heldout_scope", []string{verb, roomNames[a], "jangan", roomNames[b]}, a, j == 0, false, -1, bg, 0})
				}
			}
		}
	}
	// Fresh layouts/backgrounds, sealed before any training or main evaluation.
	for layout := 2; layout <= 4; layout++ {
		for target := 0; target < 7; target++ {
			verbs := []string{"nyalakan", "matikan"}
			if target >= 5 {
				verbs = []string{"buka", "tutup"}
			}
			for j, verb := range verbs {
				for bg := 2; bg <= 3; bg++ {
					held = append(held, bindCase{fmt.Sprintf("fresh-%d-%d-%d-%d", layout, target, j, bg), "fresh_layout_and_state", []string{verb, roomNames[target]}, target, j == 0, false, -1, bg, layout})
				}
			}
		}
	}
	return train, held
}
func bindEpisodeFor(c bindCase, seed int64, id int, words []string) bindEpisode {
	w := bindWorldNew(seed, c.Background, c.Layout)
	input := w.input(words, c.Pointer)
	motor := w.motor(c.Target, c.Set, c.Noop)
	after, valid := w.apply(motor)
	if !valid {
		panic("bad teacher motor")
	}
	return bindEpisode{ID: id, Input: input, After: after.patches(), Motor: motor, Language: true}
}
