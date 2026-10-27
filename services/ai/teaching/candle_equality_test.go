package teaching

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"math"
	"math/big"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"testing"
	"time"

	"github.com/project-horizon/horizon-core/services/ai/knowledge"
	brainruntime "github.com/project-horizon/horizon-core/services/ai/runtime"
)

func equalityQuestion(q Question) Question {
	if q.Relation == SumGreater {
		q.Relation = SumEqual
	} else {
		q.Relation = Equal
	}
	return q
}
func rawEquality(c chartCase, i int) bool {
	b := c.Present.Candles
	if i == 1 {
		val := func(v float64) *big.Rat {
			r, ok := new(big.Rat).SetString(strconv.FormatFloat(v, 'f', -1, 64))
			if !ok {
				panic("invalid decimal price")
			}
			return r
		}
		x := b[48]
		a := new(big.Rat).Add(val(x.H), val(x.L))
		d := new(big.Rat).Add(val(x.O), val(x.C))
		return a.Cmp(d) == 0
	}
	switch i {
	case 0:
		return b[48].C == b[48].O
	case 2:
		return b[48].C == b[45].C
	case 3:
		return b[45].C == b[42].C
	case 4:
		return b[40].C == b[40].O
	case 5:
		return b[40].C == b[36].C
	}
	panic("unknown equality scope")
}

func TestHNLEqualityLessonFromCandleBrain(t *testing.T) {
	dataset, previous := os.Getenv("HORIZON_HNL_CURRICULUM"), os.Getenv("HORIZON_HNL_EQUALITY_PREVIOUS")
	if dataset == "" || previous == "" {
		t.Skip("explicit chart data and stage-7 trained checkpoint required")
	}
	raw, e := os.ReadFile(dataset)
	if e != nil {
		t.Fatal(e)
	}
	var data struct {
		Groups []struct {
			Split string      `json:"split"`
			Cases []chartCase `json:"cases"`
		} `json:"groups"`
	}
	if e = json.Unmarshal(raw, &data); e != nil {
		t.Fatal(e)
	}
	sets := map[string][]chartCase{}
	for _, g := range data.Groups {
		sets[g.Split] = append(sets[g.Split], g.Cases...)
	}
	for k := range sets {
		sort.Slice(sets[k], func(i, j int) bool {
			a, b := sets[k][i], sets[k][j]
			if a.Present.At == b.Present.At {
				return a.Present.Asset < b.Present.Asset
			}
			return a.Present.At < b.Present.At
		})
	}
	dir := t.TempDir()
	for _, n := range []string{"trained.json", "events.jsonl"} {
		v, e := os.ReadFile(filepath.Join(previous, n))
		if e != nil {
			t.Fatal(e)
		}
		if e = os.WriteFile(filepath.Join(dir, n), v, 0600); e != nil {
			t.Fatal(e)
		}
	}
	log, e := brainruntime.OpenEventLog(filepath.Join(dir, "events.jsonl"))
	if e != nil {
		t.Fatal(e)
	}
	defer log.Close()
	r, e := brainruntime.RebuildRuntimeFromCheckpointAndJournal(log, filepath.Join(dir, "trained.json"))
	if e != nil {
		t.Fatal(e)
	}
	r.SetEventLog(log)
	r.SetClock(brainruntime.NewFixedClock(base.Add(48 * time.Hour)))
	b := Bridge{Runtime: r, Clock: brainruntime.NewFixedClock(base.Add(48 * time.Hour))}
	start := filepath.Join(dir, "equality-start.json")
	if e = r.Checkpoint(start); e != nil {
		t.Fatal(e)
	}
	oldResults := struct {
		Evaluation map[string]struct {
			Records []struct {
				ID       string    `json:"id"`
				Values   []float64 `json:"values"`
				Statuses string    `json:"statuses"`
			} `json:"records"`
		} `json:"evaluation"`
	}{}
	v, e := os.ReadFile(filepath.Join(previous, "candle_curriculum_results.json"))
	if e != nil {
		t.Fatal(e)
	}
	if e = json.Unmarshal(v, &oldResults); e != nil {
		t.Fatal(e)
	}
	pools := [2][]Example{}
	for i, c := range sets["train"] {
		v := chartExample(c, uint64(i+1))
		at := time.UnixMilli(c.Present.At).UTC()
		if at.Before(time.Date(2024, 6, 21, 0, 0, 0, 0, time.UTC)) {
			pools[0] = append(pools[0], v)
		}
		if !at.Before(time.Date(2024, 7, 10, 0, 0, 0, 0, time.UTC)) {
			pools[1] = append(pools[1], v)
		}
	}
	tasks := chartTasks()
	receipts := []map[string]any{}
	for qi, task := range tasks[:6] {
		p := Packet{Version: ExtendedVersion, Kind: Teach, ID: uint64(201 + qi), Source: 16, Stream: 3, At: base.Add(3*time.Hour + time.Duration(qi)*time.Second).UnixNano(), Question: equalityQuestion(task.Q)}
		for set, pool := range pools {
			n := 0
			for _, source := range pool {
				others := []Question{qX, qY, qRelation}
				for j, other := range tasks[:6] {
					if j != qi && !(qi == 2 && j == 3) && !(qi == 3 && j == 2) {
						others = append(others, other.Q)
					}
				}
				a, z, ok := contrastPair(task.Q, source, .001, others)
				if !ok {
					continue
				}
				middle := cloneExample(a)
				for i := range middle.Samples {
					for j := range middle.Samples[i].Values {
						middle.Samples[i].Values[j] = (a.Samples[i].Values[j] + z.Samples[i].Values[j]) / 2
					}
				}
				variants := []Example{a, middle, z}
				// Remove an unrelated flat interval as a shortcut for equality of
				// the target interval. The alignment circuit is then constant.
				if qi == 2 || qi == 3 {
					for i := range variants {
						if qi == 2 {
							setRef(&variants[i], Ref{6, 3}, atRef(variants[i], Ref{3, 3}))
						} else {
							setRef(&variants[i], Ref{0, 3}, atRef(variants[i], Ref{3, 3}))
						}
					}
				}
				if qi != 1 {
					for row := range variants[0].Samples {
						high, low := -1., 1.
						for _, v := range variants {
							high = math.Max(high, math.Max(v.Samples[row].Values[0], v.Samples[row].Values[3]))
							low = math.Min(low, math.Min(v.Samples[row].Values[0], v.Samples[row].Values[3]))
						}
						for i := range variants {
							variants[i].Samples[row].Values[1] = high + .03
							variants[i].Samples[row].Values[2] = low - .01
						}
					}
				}
				valid := true
				for _, v := range variants {
					if !validChart(v) {
						valid = false
					}
				}
				if !valid {
					continue
				}
				// Other supplied distinctions must have constant truth across
				// these three examples, so they cannot stand in for the target.
				for _, q := range others {
					x, y, z := teacherMargin(q, variants[0]), teacherMargin(q, variants[1]), teacherMargin(q, variants[2])
					if sign(x) != sign(y) || sign(z) != sign(y) {
						valid = false
					}
				}
				if !valid {
					continue
				}
				for j, v := range variants {
					v.Role = Role(set + 1)
					v.Episode = p.ID*100000 + uint64(set*1000+n*3+j+1)
					v.ObservedAt = p.At - 1000000
					value := -1.
					if j == 1 {
						value = 1
					}
					v.Outcome = &Evidence{Value: value, ObservedAt: p.At - 1}
					p.Examples = append(p.Examples, v)
				}
				n++
				if n == 8 {
					break
				}
			}
			if n != 8 {
				t.Fatalf("not enough equality prototypes for %s set%d: %d", task.Name, set, n)
			}
		}
		out := exchange(t, b, p).Reply
		saveCandleLesson(t, p, fmt.Sprintf("equality-%02d-%s.hnl", qi+1, task.Name))
		receipts = append(receipts, map[string]any{"task": task.Name, "reply": out})
		if !out.Committed {
			t.Fatalf("equality lesson did not improve %s: %+v", task.Name, out)
		}
		t.Logf("equality %s committed cause%d learnedN%d", task.Name, out.Cause, out.LearnedN)
	}
	final := filepath.Join(dir, "equality-trained.json")
	if e = r.Checkpoint(final); e != nil {
		t.Fatal(e)
	}
	metrics := map[string]any{}
	explained, totalUncertain, oldChanged := 0, 0, 0
	for _, split := range []string{"test2025", "test2026"} {
		correct, available := make([]int, 6), make([]int, 6)
		records := []map[string]any{}
		for j, c := range sets[split] {
			v := chartExample(c, uint64(600000+j))
			outcomes := []*Response{}
			for q, task := range tasks[:6] {
				out := exchange(t, b, askChart(equalityQuestion(task.Q), v)).Reply
				outcomes = append(outcomes, out)
				if out.Status == 3 {
					available[q]++
					if (out.Value > 0) == rawEquality(c, q) {
						correct[q]++
					}
				}
			}
			old := oldResults.Evaluation[split].Records[j]
			if old.ID != c.ID {
				t.Fatal("case ordering changed")
			}
			statuses, e := base64.StdEncoding.DecodeString(old.Statuses)
			if e != nil {
				t.Fatal(e)
			}
			for q, task := range tasks {
				now := exchange(t, b, askChart(task.Q, v)).Reply
				if now.Value != old.Values[q] || now.Status != statuses[q] {
					oldChanged++
				}
				if statuses[q] == 2 {
					totalUncertain++
					eq := q
					if q == 6 {
						if outcomes[2].Status == 3 && outcomes[2].Value > 0 || outcomes[3].Status == 3 && outcomes[3].Value > 0 {
							explained++
						}
					} else if outcomes[eq].Status == 3 && outcomes[eq].Value > 0 {
						explained++
					}
				}
			}
			vals := []float64{}
			flags := []int{}
			for _, o := range outcomes {
				vals = append(vals, o.Value)
				flags = append(flags, int(o.Status))
			}
			records = append(records, map[string]any{"id": c.ID, "values": vals, "statuses": flags})
		}
		metrics[split] = map[string]any{"cases": len(sets[split]), "correct_accepted": correct, "accepted": available, "records": records}
		t.Logf("equality %s accepted%v correct%v", split, available, correct)
	}
	if oldChanged != 0 {
		t.Fatalf("equality learning changed %d prior readouts", oldChanged)
	}
	retained := 0
	for _, v := range grid([]float64{-.43, -.21, .17, .61}, []float64{-.49, .37}) {
		for i, q := range []Question{qX, qY, qRelation} {
			out := exchange(t, b, askFixture(q, v.x, v.y)).Reply
			truth := []float64{sign(v.x), sign(v.y), sign(v.x * v.y)}[i]
			if out.Status == 3 && sign(out.Value) == truth {
				retained++
			}
		}
	}
	if retained != 24 {
		t.Fatal("old foundation lost")
	}
	after := filepath.Join(dir, "equality-after-read.json")
	if e = r.Checkpoint(after); e != nil {
		t.Fatal(e)
	}
	first, _ := os.ReadFile(final)
	second, _ := os.ReadFile(after)
	if !bytes.Equal(first, second) {
		t.Fatal("equality inference mutated Brain")
	}
	restored := brainruntime.NewBrainRuntime(knowledge.NewBrain())
	if e = restored.RestoreCheckpoint(final); e != nil {
		t.Fatal(e)
	}
	replayed, e := brainruntime.RebuildRuntimeFromCheckpointAndJournal(log, start)
	if e != nil {
		t.Fatal(e)
	}
	probe := askChart(equalityQuestion(tasks[1].Q), chartExample(sets["test2026"][0], 950000))
	wanted := exchange(t, b, probe)
	for _, rt := range []*brainruntime.BrainRuntime{restored, replayed} {
		got := exchange(t, Bridge{Runtime: rt, Clock: b.Clock}, probe)
		x, _ := Marshal(wanted)
		y, _ := Marshal(got)
		if !bytes.Equal(x, y) {
			t.Fatal("equality changed after restore/replay")
		}
	}
	count := len(exchange(t, b, askFixture(qX, .43, .37)).Reply.Units)
	evidence(t, "equality_results.json", map[string]any{"teacher_driven_additional_stage_after_diagnosis": true, "lessons": 6, "receipts": receipts, "evaluation": metrics, "previous_knowledge_retained": retained, "prior_readouts_changed": oldChanged, "explained_uncertain": explained, "total_previously_uncertain": totalUncertain, "learned_n_units": count, "read_only_inference": true, "restore_reply_equal": true, "journal_suffix_replay_reply_equal": true})
	if e = log.Close(); e != nil {
		t.Fatal(e)
	}
	if out := os.Getenv("HORIZON_HNL_RESULTS"); out != "" {
		for _, n := range []string{"equality-start.json", "equality-trained.json", "events.jsonl"} {
			v, e := os.ReadFile(filepath.Join(dir, n))
			if e != nil {
				t.Fatal(e)
			}
			name := n
			if n == "events.jsonl" {
				name = "equality-events.jsonl"
			}
			if e = os.WriteFile(filepath.Join(out, name), v, 0600); e != nil {
				t.Fatal(e)
			}
		}
	}
}
