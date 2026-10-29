package runtime

// All teacher meaning labels, answer examples and simulated rooms live here.
// The inference adapter contains neither these word classes nor these answers.
import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/project-horizon/horizon-core/services/ai/knowledge"
)

type dialogueMeaning [DialogueLexicalHeads]float64

var dialogueTeacherLexicon = map[string]dialogueMeaning{
	"lampu": {.25, 1, 0, 0, 0, 0, 0}, "kipas": {.5, 1, 0, 0, 0, 0, 0}, "pintu": {.75, 1, 0, 0, 0, 0, 0},
	"bagaimana": {0, 0, 0, -1, 0, 0, 0}, "gimana": {0, 0, 0, -1, 0, 0, 0}, "apakah": {0, 0, 0, 1, 0, 0, 0},
	"menyala": {0, 0, 1, 0, 0, 0, 1}, "nyala": {0, 0, 1, 0, 0, 0, 1}, "padam": {0, 0, -1, 0, 0, 0, 1}, "mati": {0, 0, -1, 0, 0, 0, 1},
	"terbuka": {0, 0, 1, 0, 0, 0, -1}, "tertutup": {0, 0, -1, 0, 0, 0, -1},
	"tidak": {0, 0, 0, 0, 1, 0, 0}, "tadi": {0, 0, 0, 0, 0, -1, 0}, "sekarang": {0, 0, 0, 0, 0, 1, 0},
	"itu": {}, "keadaannya": {}, "kondisi": {}, "keadaan": {}, "masih": {}, "sudah": {},
}
var dialogueTeacherEntities = []string{"lampu", "kipas", "pintu"}
var dialogueTeacherOutput = []string{"lampu", "kipas", "pintu", "itu", "sekarang", "tadi", "menyala", "padam", "terbuka", "tertutup", "ya,", "tidak,", "objek", "yang", "mana", "?", "."}

type dialogueTrainingRow struct {
	Fields map[int]float64
	Value  float64
}

func dialogueSave(t *testing.T, name string, v any) {
	t.Helper()
	root := os.Getenv("HORIZON_DIALOGUE_RESULTS")
	if root == "" {
		return
	}
	compositionCheck(t, os.MkdirAll(root, 0700))
	b, e := json.MarshalIndent(v, "", "  ")
	compositionCheck(t, e)
	compositionCheck(t, os.WriteFile(filepath.Join(root, name), append(b, '\n'), 0600))
}
func dialogueTeach(t *testing.T, r *BrainRuntime, head, phase int, fit, check []dialogueTrainingRow, tol float64) knowledge.NDiagnosis {
	t.Helper()
	at := nBase.Add(time.Duration(100+phase) * time.Minute)
	convert := func(rows []dialogueTrainingRow, label string) []knowledge.NObservation {
		out := []knowledge.NObservation{}
		for i, row := range rows {
			out = append(out, knowledge.NObservation{Input: DialogueNumericInput(head, row.Fields, at.Add(-time.Second), tol), Outcome: row.Value, ObservedAt: at, EpisodeID: fmt.Sprintf("dialogue-%d-%d-%s-%d", head, phase, label, i)})
		}
		return out
	}
	e := knowledge.CompositionExposure{ID: fmt.Sprintf("dialogue-%d-%d", head, phase), Source: "paired-room-teacher", EpisodeID: fmt.Sprintf("lesson-%d-%d", head, phase), IndependenceGroup: fmt.Sprintf("head-%d-phase-%d", head, phase), CausalGroup: "observed-room-language", Reliability: 1,
		Input: DialogueNumericInput(head, fit[0].Fields, at.Add(-time.Second), tol), After: nScene(at), Motor: []float64{0, 0, 0, 0}, Resolution: &knowledge.NResolutionLesson{Fit: convert(fit, "fit"), Check: convert(check, "check")}}
	e.After = e.Input.Scene
	e.After.ObservedAt = at
	r.SetClock(NewFixedClock(at))
	_, receipt, err := r.LearnCompositionWithDiagnosis(e, at)
	compositionCheck(t, err)
	if receipt == nil {
		t.Fatal("missing diagnosis")
	}
	return receipt.Diagnosis
}
func dialogueRowsPass(t *testing.T, r *BrainRuntime, head int, rows []dialogueTrainingRow, tol float64) (int, float64) {
	t.Helper()
	count, maxerr := 0, 0.
	for _, row := range rows {
		o, e := r.ReadComposition(DialogueNumericInput(head, row.Fields, nBase.Add(48*time.Hour), tol), nBase.Add(48*time.Hour))
		compositionCheck(t, e)
		if o.NResolution != nil {
			err := math.Abs(o.NResolution.Value - row.Value)
			maxerr = math.Max(maxerr, err)
			if o.Accepted && err <= tol {
				count++
			}
		}
	}
	return count, maxerr
}
func dialogueFitStage(t *testing.T, r *BrainRuntime, head int, fit, check []dialogueTrainingRow, tol float64, phase *int, limit int) []knowledge.NDiagnosis {
	t.Helper()
	ds := []knowledge.NDiagnosis{}
	for attempt := 0; attempt < limit; attempt++ {
		*phase++
		d := dialogueTeach(t, r, head, *phase, fit, check, tol)
		ds = append(ds, d)
		n, err := dialogueRowsPass(t, r, head, check, tol)
		t.Logf("dialogue head=%d attempt=%d %s committed=%v check=%d/%d err=%g N=%d", head, attempt, d.Cause, d.Committed, n, len(check), err, len(r.brain.BrainState.NResolution.Features))
		if n == len(check) {
			break
		}
	}
	return ds
}
func dialogueVocabulary(t *testing.T, r *BrainRuntime) DialogueCodec {
	t.Helper()
	forms := []string{}
	for word := range dialogueTeacherLexicon {
		forms = append(forms, word)
	}
	sort.Strings(forms)
	c := DialogueCodec{Version: "grounded-dialogue-io-v1", OutputStep: 1.0 / 32}
	for i, form := range forms {
		c.Input = append(c.Input, DialogueInputToken{Form: form, Slot: i})
	}
	for i, word := range dialogueTeacherOutput {
		alphabet := make([]float64, 32)
		alphabet[i] = 1
		g, e := r.GroundVectorAt(knowledge.NewNeuralVector(alphabet), nBase)
		compositionCheck(t, e)
		exposure, e := knowledge.NewSpellingExposure(g.Population, word, .8, .8)
		compositionCheck(t, e)
		for j := 0; j < 4; j++ {
			_, e = r.LearnSpelling(exposure, nBase)
			compositionCheck(t, e)
		}
		c.Output = append(c.Output, DialogueOutputToken{Code: float64(i+1) * c.OutputStep, Population: g.Population})
	}
	compositionCheck(t, c.Validate())
	return c
}
func dialogueOutputCode(word string) float64 {
	if word == "" {
		return 0
	}
	for i, w := range dialogueTeacherOutput {
		if w == word {
			return float64(i+1) / 32
		}
	}
	panic("unknown output form")
}

// Teacher realization is used exclusively to provide observed training labels
// and a sealed reference for evaluation. Production inference never calls it.
func dialogueTeacherWords(ref, state, temporal, kind, agreement, valid float64) []string {
	if valid == 0 {
		return []string{"objek", "yang", "mana", "?", "", ""}
	}
	i := int(math.Round(ref*4)) - 1
	word := "padam"
	if state > 0 {
		word = "menyala"
	}
	if i == 2 {
		word = "tertutup"
		if state > 0 {
			word = "terbuka"
		}
	}
	prefix := ""
	if kind > 0 {
		prefix = "tidak,"
		if agreement > 0 {
			prefix = "ya,"
		}
	}
	when := "sekarang"
	if temporal < 0 {
		when = "tadi"
	}
	return []string{prefix, dialogueTeacherEntities[i], "itu", when, word, "."}
}

type dialogueEvalCase struct {
	ID       string                  `json:"id"`
	Split    string                  `json:"split"`
	Request  GroundedDialogueRequest `json:"request"`
	Expected string                  `json:"expected"`
	Valid    bool                    `json:"valid"`
	Focus    float64                 `json:"focus"`
	Temporal float64                 `json:"temporal"`
	State    float64                 `json:"state"`
}

func dialogueSealedCases() []dialogueEvalCase {
	out := []dialogueEvalCase{}
	at := nBase.Add(48 * time.Hour)
	for i, name := range dialogueTeacherEntities {
		for mask := 0; mask < 8; mask++ {
			for historyMask := 0; historyMask < 2; historyMask++ {
				room := DialogueRoom{Current: []float64{-1, -1, -1}, Previous: []float64{1, 1, 1}, Kinds: []float64{1, 1, -1}, ObservedAt: at.Add(-time.Second), PreviousAt: at.Add(-time.Hour)}
				for j := range room.Current {
					if mask&(1<<j) != 0 {
						room.Current[j] = 1
					}
					if historyMask == 1 {
						room.Previous[j] = -room.Previous[j]
					}
				}
				for variant := 0; variant < 8; variant++ {
					text := fmt.Sprintf("Bagaimana keadaan %s sekarang?", name)
					history := []string{}
					temporal, kind, claim, neg := 1., -1., 0., 0.
					split := "named-new-context"
					positive := "menyala"
					if i == 2 {
						positive = "terbuka"
					}
					switch variant {
					case 1:
						text = fmt.Sprintf("Gimana kondisi %s?", name)
						split = "lexical-paraphrase"
					case 2:
						text = fmt.Sprintf("Apakah %s masih %s?", name, positive)
						kind, claim = 1, 1
						split = "polar"
					case 3:
						text = fmt.Sprintf("Apakah %s tidak %s?", name, positive)
						kind, claim, neg = 1, 1, 1
						split = "negation"
					case 4:
						text = "Bagaimana keadaannya tadi?"
						history = []string{fmt.Sprintf("Bagaimana keadaan %s?", name)}
						temporal = -1
						split = "history-and-time"
					case 5:
						text = "Bagaimana kondisi itu sekarang?"
						history = []string{fmt.Sprintf("Bagaimana keadaan %s?", name)}
						split = "history-reference"
					case 6:
						text = "Apakah itu tidak " + positive + " tadi?"
						history = []string{fmt.Sprintf("Bagaimana kondisi %s?", name)}
						kind, claim, neg, temporal = 1, 1, 1, -1
						split = "history-negation-time"
					case 7:
						text = "Bagaimana keadaannya?"
						history = []string{fmt.Sprintf("Bagaimana keadaan %s?", dialogueTeacherEntities[(i+1)%3]), fmt.Sprintf("Bagaimana kondisi %s?", name)}
						split = "latest-context-overrides"
					}
					state := room.Current[i]
					if temporal < 0 {
						state = room.Previous[i]
					}
					agreement := state * claim * (1 - 2*neg)
					out = append(out, dialogueEvalCase{ID: fmt.Sprintf("room-%d-%d-%d-%d", i, mask, historyMask, variant), Split: split, Request: GroundedDialogueRequest{Text: text, History: history, Room: room, At: at}, Expected: dialogueText(dialogueTeacherWords(float64(i+1)/4, state, temporal, kind, agreement, 1)), Valid: true, Focus: float64(i+1) / 4, Temporal: temporal, State: state})
				}
			}
		}
	}
	room := DialogueRoom{Current: []float64{1, -1, 1}, Previous: []float64{-1, 1, -1}, Kinds: []float64{1, 1, -1}, ObservedAt: at.Add(-time.Second), PreviousAt: at.Add(-time.Hour)}
	for i, text := range []string{"Bagaimana keadaannya?", "Bagaimana keadaan lampu pintu?", "Apakah lampu terbuka?", "Apakah pintu menyala?", "Apakah lampu?", "Bagaimana keadaan mangkuk?"} {
		out = append(out, dialogueEvalCase{ID: fmt.Sprintf("clarify-%d", i), Split: "must-clarify", Request: GroundedDialogueRequest{Text: text, Room: room, At: at}, Expected: "Objek yang mana?", Valid: false})
	}
	return out
}

func TestGroundedDialogueStagedLanguageCurriculum(t *testing.T) {
	if os.Getenv("HORIZON_DIALOGUE_TRAIN") == "" {
		t.Skip("explicit paired language curriculum required")
	}
	sealed := dialogueSealedCases()
	raw, e := json.MarshalIndent(sealed, "", "  ")
	compositionCheck(t, e)
	hash := sha256.Sum256(raw)
	dialogueSave(t, "sealed_cases.json", sealed)
	r, log, dir, start := mNew(t, knowledge.NeuralCompositionMetricV6)
	codec := dialogueVocabulary(t, r)
	dialogueSave(t, "codec.json", codec)
	baseline, e := r.ReadGroundedDialogue(codec, sealed[0].Request)
	compositionCheck(t, e)
	if baseline.Text != "" {
		t.Fatal("untrained Brain emitted an answer")
	}
	phase := 0
	receipts := map[string][]knowledge.NDiagnosis{}
	// Basic physical/relational coordinates, followed by lexical meaning. A
	// coordinate lesson is a teacher-provided primitive, not a discovered cause.
	for h, field := range []int{49, 50, 60, 61, 62, 66, 67, 77, 72, 76} {
		rows := []dialogueTrainingRow{}
		for rep := 0; rep < 4; rep++ {
			for _, v := range []float64{-.75, -.5, -.25, 0, .25, .5, .75, 1} {
				rows = append(rows, dialogueTrainingRow{Fields: map[int]float64{field: v}, Value: v})
			}
		}
		receipts[fmt.Sprintf("primitive-%d", h)] = dialogueFitStage(t, r, 17+h, rows, rows, dialoguePrecision, &phase, 3)
	}
	lexrows := func(head int) []dialogueTrainingRow {
		rows := []dialogueTrainingRow{}
		for _, entry := range codec.Input {
			value := 1.
			if head < 7 {
				value = dialogueTeacherLexicon[entry.Form][head]
			}
			rows = append(rows, dialogueTrainingRow{Fields: map[int]float64{entry.Slot: 1}, Value: value})
		}
		for len(rows) < 32 {
			rows = append(rows, dialogueTrainingRow{Fields: map[int]float64{}, Value: 0})
		}
		return rows
	}
	rows := lexrows(27)
	receipts["alphabet-presence"] = dialogueFitStage(t, r, 27, rows, rows, dialoguePrecision, &phase, 5)
	for head := 0; head < 7; head++ {
		rows := lexrows(head)
		receipts[fmt.Sprintf("lexical-%d", head)] = dialogueFitStage(t, r, head, rows, rows, dialoguePrecision, &phase, 16)
	}
	// Reference binding includes named, deictic, missing and ambiguous contexts.
	refAxisRows := []dialogueTrainingRow{}
	for rep := 0; rep < 4; rep++ {
		for _, v := range []float64{-.75, -.5, -.25, 0, .25, .5, .75, 1} {
			refAxisRows = append(refAxisRows, dialogueTrainingRow{Fields: map[int]float64{48: v}, Value: v})
		}
	}
	receipts["reference-coordinate"] = dialogueFitStage(t, r, 28, refAxisRows, refAxisRows, dialoguePrecision, &phase, 4)
	// First bind an explicit referent, then learn contextual substitution, then
	// reject multiple referents. Every stage preserves previously correct cases.
	for stage := 0; stage < 4; stage++ {
		focusRows := []dialogueTrainingRow{}
		for _, previous := range []float64{0, .25, .5, .75} {
			for _, count := range []int{0, 1, 2} {
				for _, ref := range []float64{.25, .5, .75} {
					if stage == 0 && count != 1 || stage == 1 && count == 2 || stage == 2 && count == 2 && previous != 0 {
						continue
					}
					v := 0.
					sum := ref
					if count == 0 {
						sum = 0
						v = previous
					}
					if count == 1 {
						v = ref
					}
					if count == 2 {
						sum = .5 + ref
					}
					focusRows = append(focusRows, dialogueTrainingRow{Fields: map[int]float64{48: sum / 2, 49: float64(count) / 2, 50: previous}, Value: v})
				}
			}
		}
		receipts[fmt.Sprintf("reference-%d", stage)] = dialogueFitStage(t, r, 7, focusRows, focusRows, dialoguePrecision, &phase, 20)
	}
	timeRows := []dialogueTrainingRow{}
	for rep := 0; rep < 8; rep++ {
		for _, v := range []float64{-1, 0, 1} {
			y := 1.
			if v < 0 {
				y = -1
			}
			timeRows = append(timeRows, dialogueTrainingRow{Fields: map[int]float64{54: v}, Value: y})
		}
	}
	receipts["temporal"] = dialogueFitStage(t, r, 8, timeRows, timeRows, dialoguePrecision, &phase, 4)
	// Learn state-versus-claim before composing it with negation. The fully
	// balanced three-factor problem has no linear or pairwise error signal.
	for stage := 0; stage < 2; stage++ {
		agreementRows := []dialogueTrainingRow{}
		for _, state := range []float64{-1, 1} {
			for _, claim := range []float64{-1, 0, 1} {
				for _, neg := range []float64{0, 1} {
					for _, kind := range []float64{-1, 1} {
						if stage == 0 && neg != 0 {
							continue
						}
						agreementRows = append(agreementRows, dialogueTrainingRow{Fields: map[int]float64{60: state, 61: claim, 62: neg, 63: kind}, Value: state * claim * (1 - 2*neg)})
					}
				}
			}
		}
		receipts[fmt.Sprintf("agreement-%d", stage)] = dialogueFitStage(t, r, 9, agreementRows, agreementRows, dialoguePrecision, &phase, 12)
	}
	validRows := []dialogueTrainingRow{}
	for _, objectType := range []float64{-1, 1} {
		for _, kind := range []float64{-1, 1} {
			for _, claimType := range []float64{-1, 0, 1} {
				claimed := math.Abs(claimType)
				v := 0.
				if kind < 0 || claimType == objectType {
					v = 1
				}
				validRows = append(validRows, dialogueTrainingRow{Fields: map[int]float64{64: 1, 63: kind, 66: claimType, 67: objectType, 68: claimed}, Value: v})
			}
		}
	}
	receipts["compatibility"] = dialogueFitStage(t, r, 10, validRows, validRows, dialoguePrecision, &phase, 12)
	// Teach surface motor slots from selected meanings, never from the prompt
	// or a whole-answer lookup. Some complete meaning combinations are excluded.
	surfaceFit := make([][]dialogueTrainingRow, 6)
	surfaceCheck := make([][]dialogueTrainingRow, 6)
	for i := 0; i < 3; i++ {
		for _, state := range []float64{-1, 1} {
			for _, temporal := range []float64{-1, 1} {
				for _, style := range []int{0, 1, 2} {
					if i == 1 && state < 0 && temporal < 0 {
						continue
					}
					kind, agree := -1., 0.
					if style > 0 {
						kind = 1
						agree = float64(2*style - 3)
					}
					sensorKind := 1.
					if i == 2 {
						sensorKind = -1
					}
					fields := map[int]float64{72: float64(i+1) / 4, 60: state, 74: temporal, 63: kind, 76: agree, 77: 1, 67: sensorKind}
					words := dialogueTeacherWords(float64(i+1)/4, state, temporal, kind, agree, 1)
					for slot, word := range words {
						surfaceFit[slot] = append(surfaceFit[slot], dialogueTrainingRow{Fields: fields, Value: dialogueOutputCode(word)})
						surfaceCheck[slot] = append(surfaceCheck[slot], dialogueTrainingRow{Fields: fields, Value: dialogueOutputCode(word)})
					}
				}
			}
		}
	}
	for slot, word := range dialogueTeacherWords(0, 0, 0, 0, 0, 0) {
		for rep := 0; rep < 4; rep++ {
			row := dialogueTrainingRow{Fields: map[int]float64{77: 0}, Value: dialogueOutputCode(word)}
			surfaceFit[slot] = append(surfaceFit[slot], row)
			surfaceCheck[slot] = append(surfaceCheck[slot], row)
		}
	}
	for slot := 0; slot < 6; slot++ {
		receipts[fmt.Sprintf("surface-%d", slot)] = dialogueFitStage(t, r, 11+slot, surfaceFit[slot], surfaceCheck[slot], codec.OutputStep/4, &phase, 12)
	}
	dialogueSave(t, "training_receipts.json", receipts)
	checkpoint := filepath.Join(dir, "language-brain.json")
	compositionCheck(t, r.Checkpoint(checkpoint))
	before, e := os.ReadFile(checkpoint)
	compositionCheck(t, e)
	sequence := r.LastSequence()
	counts := map[string]map[string]int{}
	results := []any{}
	latencies := []float64{}
	samples := []any{}
	for _, c := range sealed {
		begin := time.Now()
		o, e := r.ReadGroundedDialogue(codec, c.Request)
		compositionCheck(t, e)
		latencies = append(latencies, float64(time.Since(begin).Nanoseconds())/1e6)
		if counts[c.Split] == nil {
			counts[c.Split] = map[string]int{}
		}
		counts[c.Split]["total"]++
		if o.Text == c.Expected {
			counts[c.Split]["exact"]++
		}
		if o.MeaningValid == c.Valid && (!c.Valid || o.Focus == c.Focus && o.Temporal == c.Temporal && o.ObservedState == c.State) {
			counts[c.Split]["meaning"]++
		}
		if o.Status == "untrained" {
			counts[c.Split]["abstained"]++
		}
		if len(samples) < 8 || strings.Contains(c.ID, "-0-0-4") || c.Split == "must-clarify" {
			samples = append(samples, map[string]any{"case": c, "output": o})
		}
		results = append(results, map[string]any{"id": c.ID, "split": c.Split, "expected": c.Expected, "actual": o.Text, "status": o.Status, "reason": o.Reason, "meaning_valid": o.MeaningValid, "focus": o.Focus, "temporal": o.Temporal, "state": o.ObservedState, "agreement": o.Agreement})
	}
	afterPath := filepath.Join(dir, "language-after.json")
	compositionCheck(t, r.Checkpoint(afterPath))
	after, e := os.ReadFile(afterPath)
	compositionCheck(t, e)
	if !bytes.Equal(before, after) || sequence != r.LastSequence() {
		t.Fatal("language inference mutated canonical Brain")
	}
	compositionCheck(t, log.Close())
	root := os.Getenv("HORIZON_DIALOGUE_RESULTS")
	if root != "" {
		for _, f := range []string{"language-brain.json", "start.json", "events.jsonl"} {
			b, e := os.ReadFile(filepath.Join(dir, f))
			compositionCheck(t, e)
			compositionCheck(t, os.WriteFile(filepath.Join(root, f), b, 0600))
		}
	}
	_ = start
	dialogueSave(t, "evaluation.json", results)
	dialogueSave(t, "samples.json", samples)
	dialogueSave(t, "results.json", map[string]any{"sealed_sha256": hex.EncodeToString(hash[:]), "baseline": baseline, "counts": counts, "learned_n": len(r.brain.BrainState.NResolution.Features), "heads": len(r.brain.BrainState.NResolution.Heads), "total_units": len(r.brain.Registry.Nodes()), "native_sequence": sequence, "readonly": true, "p50_ms": mPct(latencies, .5), "p95_ms": mPct(latencies, .95), "phases": phase})
	t.Logf("dialogue final counts=%v N=%d heads=%d", counts, len(r.brain.BrainState.NResolution.Features), len(r.brain.BrainState.NResolution.Heads))
	// Report measured failures rather than turn an unfinished language curriculum
	// into a false green assertion. Safety/determinism assertions above still fail.
}
