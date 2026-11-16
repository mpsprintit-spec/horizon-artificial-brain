package runtime

// Explicitly opt-in research adapter. All learned directions, quantizers and
// readouts are the existing canonical signed synapses; no estimator lives here.
import (
	"bytes"
	"encoding/json"
	"fmt"
	"math"
	"math/rand"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"testing"
	"time"

	k "github.com/project-horizon/horizon-core/services/ai/knowledge"
)

const s2Baseline = "11c6e0a78547d68a0be8f78b3c801e912722608d"
const s2SourceSHA = "d9bf6662a17c262922806672bcf53fe1484e5eb552ff0ec6699c21142333d64b"
const s2SourceWAL = "3c692c4dccaa33ebac748bcafe06be8b4a21a036317d3fe0e0ec813060b0b9dd"

// This type cannot carry the current operation's future state or error.
type s2Pre struct {
	Before                 map[k.NodeID]float64
	Previous               []float64
	Action, PreviousAction int
	At                     time.Time
	Forecast, History      []float64
	HistoryAction          int
}
type s2Trial struct {
	ID               string
	Pre              s2Pre
	Actual, Residual []float64
	Error, Estimate  float64
	AfterAt          time.Time
	InputSHA256      string
}
type s2Protocol struct {
	Baseline, SourceCheckpointSHA256, SourceJournalSHA256, DesignSHA256 string
	TrainingSeed, DevelopmentSeed                                       int64
	NewTrainingSeed                                                     int64
	HoldoutSeeds                                                        []int64
	FitN, DevelopmentN, HoldoutN, Lessons                               int
	Scale, Constant, DevelopmentSD, LowCut, HighCut, CalibrationLimit   float64
	Bins                                                                []float64
	Identifiable                                                        bool
	RelativeRMSE, Correlation, AblationRatio, RetentionTolerance        float64
}
type s2Metric struct {
	N                                        int
	MAE, RMSE, Correlation, CalibrationError float64
	LowN, HighN                              int
	LowMAE, HighMAE                          float64
	Bins                                     []map[string]any
}

func s2Paths(t *testing.T) (string, string) {
	t.Helper()
	a, b := os.Getenv("HORIZON_S2_SOURCE"), os.Getenv("HORIZON_S2_WORK")
	if a == "" || b == "" {
		t.Skip("requires explicit immutable S1 root and isolated S2 directory")
	}
	return a, b
}
func s2Source(t *testing.T, root string) {
	t.Helper()
	for name, want := range map[string]string{"s1/lineage/s1-continuation.json": s2SourceSHA, "s1/lineage/events.jsonl": s2SourceWAL} {
		b, e := os.ReadFile(filepath.Join(root, name))
		s1Check(t, e)
		if s1Digest(b) != want {
			t.Fatal("S1 source bytes differ", name)
		}
	}
	s := s1Snapshot(t, filepath.Join(root, "s1/lineage/s1-continuation.json"))
	cs, e := checkpointChecksumForSchema(s)
	s1Check(t, e)
	if s.BrainIdentity != BrainIdentity || s.Sequence != 75866 || cs != s.Checksum || cs != "15105e910a7d1fcc0b1cdf4b2883493c498abbabc5afae3626dd0f9bb18ba2bf" {
		t.Fatal("wrong S1 native lineage")
	}
	var b struct {
		Units    []any `json:"neural_units"`
		Synapses []any `json:"synapses"`
		Patterns []any `json:"temporal_patterns"`
	}
	s1Check(t, json.Unmarshal(s.Brain, &b))
	if len(b.Units) != 256 || len(b.Synapses) != 5820 || len(b.Patterns) != 24 {
		t.Fatal("wrong S1 inventory")
	}
}
func s2Input(p s2Pre, pop [][]k.NodeID) k.CompositionInput {
	s := s1Sample{Before: p.Before, Previous: p.Previous, Action: p.Action, PreviousAction: p.PreviousAction, BeforeAt: p.At}
	in := s1Input(3, s, pop)
	// Prepending keeps every original S1 coordinate at the same reverse lag.
	h := make([]float64, 3)
	copy(h, p.History)
	present, action := 0., 0.
	if len(p.History) == 3 {
		present = 1
		action = -1
		if p.HistoryAction == 1 {
			action = 1
		}
	}
	row1 := []float64{p.Forecast[0], p.Forecast[1], p.Forecast[2], h[0], h[1], h[2], 1}
	row2 := []float64{present, action, 0, 0, 0, 0, 1}
	in.Sequence = append([][]float64{row2, row1}, in.Sequence...)
	precision := .002
	in.RequiredPrecision = &precision
	return in
}
func s2Predict(t *testing.T, r *BrainRuntime, p s2Pre, pop [][]k.NodeID) float64 {
	t.Helper()
	in := s2Input(p, pop)
	o, e := r.ReadComposition(in, in.Scene.ObservedAt)
	s1Check(t, e)
	if o.NResolution == nil {
		return 0
	}
	return o.NResolution.Value
}
func s2Collect(t *testing.T, r *BrainRuntime, pop [][]k.NodeID, seed int64, n int, shift bool, history []float64, historyAction int) []s2Trial {
	t.Helper()
	_, work := s2Paths(t)
	ledger, err := os.OpenFile(filepath.Join(work, "preoperation.jsonl"), os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0600)
	s1Check(t, err)
	defer ledger.Close()
	rng := rand.New(rand.NewSource(seed))
	rows := make([]s2Trial, 0, n)
	for i := 0; i < n; i++ {
		id := fmt.Sprintf("s2-%d-%03d", seed, i)
		state := map[k.NodeID]float64{}
		lo, hi := .05, .65
		if shift {
			lo, hi = .07, .73
		}
		for _, p := range pop {
			amp := lo + (hi-lo)*rng.Float64()
			for _, v := range p {
				state[v] = amp
			}
		}
		prior := -1
		var prev []float64
		if i%2 == 1 {
			prior = 0
			prev = s1Previous(state, pop)
		}
		if shift && i%6 == 3 {
			prior = 1
			prev = s1Previous(state, pop)
			pid := id + "-prior-stop"
			s1Command(t, r, NativeControlCommand{ID: pid + "-begin", Begin: &NativeBegin{EpisodeID: pid, Observation: &NativeObservation{State: state, Source: "s2-neural-observer", SampleID: pid}}})
			a := 1
			step := s1Command(t, r, NativeControlCommand{ID: pid + "-step", Step: &NativeStep{EpisodeID: pid, ActionOverride: &a}})
			state = step.State
			s1Command(t, r, NativeControlCommand{ID: pid + "-close", Outcome: &NativeOutcome{EpisodeID: pid, TraceDigest: step.TraceDigest, Source: "s2-neural-observer", OutcomeID: pid, Provenance: "observed-sensor", Learn: false}})
		}
		s1Command(t, r, NativeControlCommand{ID: id + "-begin", Begin: &NativeBegin{EpisodeID: id, Observation: &NativeObservation{State: state, Source: "s2-controlled-neural-stimulus", SampleID: id}}})
		if prior == 0 {
			a := 0
			s1Command(t, r, NativeControlCommand{ID: id + "-prior", Step: &NativeStep{EpisodeID: id, ActionOverride: &a}})
		}
		p := s2Pre{Before: r.activation.CurrentState(), Previous: prev, Action: (i / 2) % 2, PreviousAction: prior, At: s1Time(r), History: append([]float64(nil), history...), HistoryAction: historyAction}
		s := s1Sample{Before: p.Before, Previous: p.Previous, Action: p.Action, PreviousAction: p.PreviousAction, BeforeAt: p.At}
		before := continuationBytes(t, r)
		p.Forecast = s1Predict(t, r, s, pop)
		estimate := s2Predict(t, r, p, pop)
		if !bytes.Equal(before, continuationBytes(t, r)) {
			t.Fatal("pre-operation forecast mutated state")
		}
		encoded, e := json.Marshal(s2Input(p, pop))
		s1Check(t, e)
		// Durable evidence is written before the command that produces the label.
		s1Check(t, json.NewEncoder(ledger).Encode(map[string]any{"id": id, "before_sequence": r.LastSequence(), "input_sha256": s1Digest(encoded), "forecast": p.Forecast, "estimate": estimate}))
		s1Check(t, ledger.Sync())
		a := p.Action
		step := s1Command(t, r, NativeControlCommand{ID: id + "-target", Step: &NativeStep{EpisodeID: id, ActionOverride: &a}})
		actual := s1Target(step.State, pop)
		residual := make([]float64, 3)
		history = make([]float64, 3)
		magnitude := 0.
		for q := range actual {
			residual[q] = actual[q] - p.Forecast[q]
			history[q] = math.Abs(residual[q])
			magnitude += history[q] / 3
		}
		historyAction = p.Action
		if !step.Stopped {
			stop := 1
			step = s1Command(t, r, NativeControlCommand{ID: id + "-stop", Step: &NativeStep{EpisodeID: id, ActionOverride: &stop}})
		}
		s1Command(t, r, NativeControlCommand{ID: id + "-close", Outcome: &NativeOutcome{EpisodeID: id, TraceDigest: step.TraceDigest, Source: "s2-error-observation", OutcomeID: id, Provenance: "observed-sensor", Learn: false}})
		rows = append(rows, s2Trial{ID: id, Pre: p, Actual: actual, Residual: residual, Error: magnitude, Estimate: estimate, AfterAt: s1Time(r), InputSHA256: s1Digest(encoded)})
	}
	return rows
}
func s2Predictions(t *testing.T, r *BrainRuntime, rows []s2Trial, pop [][]k.NodeID) []float64 {
	t.Helper()
	out := make([]float64, len(rows))
	for i, s := range rows {
		out[i] = s2Predict(t, r, s.Pre, pop)
	}
	return out
}
func s2Constant(rows []s2Trial, v float64) []float64 {
	out := make([]float64, len(rows))
	for i := range out {
		out[i] = v
	}
	return out
}
func s2MetricOf(rows []s2Trial, pred []float64, p s2Protocol) s2Metric {
	m := s2Metric{N: len(rows)}
	bins := make([][3]float64, len(p.Bins)+1)
	mx, my := 0., 0.
	for i, s := range rows {
		mx += pred[i] / float64(len(rows))
		my += s.Error / float64(len(rows))
	}
	cov, vx, vy := 0., 0., 0.
	for i, s := range rows {
		d := pred[i] - s.Error
		m.MAE += math.Abs(d)
		m.RMSE += d * d
		x, y := pred[i]-mx, s.Error-my
		cov += x * y
		vx += x * x
		vy += y * y
		if s.Error <= p.LowCut {
			m.LowN++
			m.LowMAE += math.Abs(d)
		}
		if s.Error >= p.HighCut {
			m.HighN++
			m.HighMAE += math.Abs(d)
		}
		b := sort.SearchFloat64s(p.Bins, pred[i])
		bins[b][0]++
		bins[b][1] += pred[i]
		bins[b][2] += s.Error
	}
	if len(rows) > 0 {
		m.MAE /= float64(len(rows))
		m.RMSE = math.Sqrt(m.RMSE / float64(len(rows)))
	}
	if vx*vy > 0 {
		m.Correlation = cov / math.Sqrt(vx*vy)
	}
	if m.LowN > 0 {
		m.LowMAE /= float64(m.LowN)
	}
	if m.HighN > 0 {
		m.HighMAE /= float64(m.HighN)
	}
	for i, b := range bins {
		if b[0] > 0 {
			m.CalibrationError += math.Abs(b[1]-b[2]) / float64(len(rows))
			m.Bins = append(m.Bins, map[string]any{"index": i, "n": int(b[0]), "estimated": b[1] / b[0], "actual": b[2] / b[0]})
		}
	}
	return m
}
func s2ProtocolRead(t *testing.T, work string) s2Protocol {
	t.Helper()
	var p s2Protocol
	b, e := os.ReadFile(filepath.Join(work, "PROTOCOL.json"))
	s1Check(t, e)
	h, e := os.ReadFile(filepath.Join(work, "PROTOCOL.sha256"))
	s1Check(t, e)
	if string(h) != s1Digest(b) {
		t.Fatal("frozen S2 protocol changed")
	}
	s1Check(t, json.Unmarshal(b, &p))
	return p
}

func TestS2Prepare(t *testing.T) {
	root, work := s2Paths(t)
	s2Source(t, root)
	if _, e := os.Stat(work); !os.IsNotExist(e) {
		t.Fatal("S2 work must be new")
	}
	var ci struct {
		HeadSHA            string `json:"head_sha"`
		Status, Conclusion string
		Event              string
		ID                 uint64
	}
	ciBytes, e := os.ReadFile(os.Getenv("HORIZON_S2_CI"))
	s1Check(t, e)
	s1Check(t, json.Unmarshal(ciBytes, &ci))
	if ci.HeadSHA != s2Baseline || ci.Status != "completed" || ci.Conclusion != "success" || ci.Event != "push" {
		t.Fatal("post-merge main CI not SUCCESS; acquisition blocked")
	}
	s1Check(t, os.MkdirAll(filepath.Join(work, "lineage"), 0700))
	s1Check(t, os.WriteFile(filepath.Join(work, "main-ci.json"), ciBytes, 0600))
	for src, dst := range map[string]string{"s1/lineage/s1-continuation.json": "00-s1-source.json", "s1/lineage/events.jsonl": "lineage/events.jsonl", "source/populations.json": "populations.json"} {
		b, e := os.ReadFile(filepath.Join(root, src))
		s1Check(t, e)
		s1Check(t, os.WriteFile(filepath.Join(work, dst), b, 0600))
	}
	r, wal := s1Open(t, filepath.Join(work, "00-s1-source.json"), filepath.Join(work, "lineage/events.jsonl"))
	if r.brain.NativeControlSnapshot().Updates != 10752 {
		t.Fatal("wrong native policy counter")
	}
	pop := s1Pop(t, filepath.Join(root, "source"))
	dev := s2Collect(t, r, pop, 211, 64, false, nil, -1)
	fit := s2Collect(t, r, pop, 223, 64, false, nil, -1)
	s1Write(t, filepath.Join(work, "development.json"), dev)
	s1Write(t, filepath.Join(work, "training.json"), fit)
	values := make([]float64, len(dev))
	mean := 0.
	for i, s := range dev {
		values[i] = s.Error
		mean += s.Error / float64(len(dev))
	}
	sort.Float64s(values)
	variance := 0.
	for _, v := range values {
		variance += (v - mean) * (v - mean) / float64(len(values))
	}
	constant := 0.
	for _, s := range fit {
		constant += s.Error / float64(len(fit))
	}
	design, e := os.ReadFile(filepath.Join("../../../docs/research/s2_self_evaluation_20261116", "DESIGN.md"))
	s1Check(t, e)
	p := s2Protocol{Baseline: s2Baseline, SourceCheckpointSHA256: s2SourceSHA, SourceJournalSHA256: s2SourceWAL, DesignSHA256: s1Digest(design), TrainingSeed: 223, NewTrainingSeed: 227, DevelopmentSeed: 211, HoldoutSeeds: []int64{307, 401}, FitN: 128, DevelopmentN: 64, HoldoutN: 96, Lessons: 16, Scale: 1, Constant: constant, DevelopmentSD: math.Sqrt(variance), LowCut: values[len(values)/3], HighCut: values[2*len(values)/3], RelativeRMSE: .8, Correlation: .5, AblationRatio: 1.2, RetentionTolerance: 1e-7}
	p.CalibrationLimit = math.Max(.003, .25*p.DevelopmentSD)
	p.Identifiable = variance > 1e-8 && p.HighCut-p.LowCut > .003
	for i := 1; i <= 5; i++ {
		p.Bins = append(p.Bins, values[len(values)-1]*float64(i)/5)
	}
	s1Write(t, filepath.Join(work, "PROTOCOL.json"), p)
	raw, e := os.ReadFile(filepath.Join(work, "PROTOCOL.json"))
	s1Check(t, e)
	s1Check(t, os.WriteFile(filepath.Join(work, "PROTOCOL.sha256"), []byte(s1Digest(raw)), 0600))
	s1Check(t, r.Checkpoint(filepath.Join(work, "01-observed-untrained.json")))
	s1Check(t, wal.Close())
	t.Logf("S1 continuation verified; frozen protocol %s; SD %.6f low %.6f high %.6f identifiable %v", s1Digest(raw), p.DevelopmentSD, p.LowCut, p.HighCut, p.Identifiable)
}

func s2Lesson(pass int, fit, dev []s2Trial, pop [][]k.NodeID, at time.Time) k.CompositionExposure {
	b := &k.NResolutionLesson{}
	for set, rows := range [][]s2Trial{fit, dev} {
		for _, s := range rows {
			o := k.NObservation{Input: s2Input(s.Pre, pop), Outcome: s.Error, ObservedAt: s.AfterAt, EpisodeID: s.ID}
			if set == 0 {
				b.Fit = append(b.Fit, o)
			} else {
				b.Check = append(b.Check, o)
			}
		}
	}
	id := fmt.Sprintf("s2-error-learning-%02d", pass)
	return k.CompositionExposure{ID: id, Source: "runtime-internal-error-inspection", EpisodeID: id, IndependenceGroup: "s2-streams-211-223", CausalGroup: "s2-observed-native-prediction-error", Reliability: 1, Input: b.Fit[0].Input, Motor: []float64{0, 0, 0, 0}, After: s1Scene(at), Resolution: b}
}
func TestS2Train(t *testing.T) {
	root, work := s2Paths(t)
	s2Source(t, root)
	p := s2ProtocolRead(t, work)
	if _, e := os.Stat(filepath.Join(work, "lineage/s2-final.json")); e == nil {
		t.Fatal("S2 acquisition already complete")
	}
	pop := s1Pop(t, filepath.Join(root, "source"))
	r, wal := s1Open(t, filepath.Join(work, "01-observed-untrained.json"), filepath.Join(work, "lineage/events.jsonl"))
	var fit, dev []s2Trial
	s1Read(t, filepath.Join(work, "training.json"), &fit)
	s1Read(t, filepath.Join(work, "development.json"), &dev)
	var diagnoses []k.NDiagnosis
	if _, e := os.Stat(filepath.Join(work, "learning.json")); e == nil {
		s1Read(t, filepath.Join(work, "learning.json"), &diagnoses)
	}
	progress := []map[string]any{}
	for pass := 1; pass <= p.Lessons; pass++ {
		if pass == 9 {
			newPath := filepath.Join(work, "training-new.json")
			if _, err := os.Stat(newPath); os.IsNotExist(err) {
				last := fit[len(fit)-1]
				history := []float64{}
				for _, v := range last.Residual {
					history = append(history, math.Abs(v))
				}
				fresh := s2Collect(t, r, pop, p.NewTrainingSeed, 64, false, history, last.Pre.Action)
				s1Write(t, newPath, fresh)
			}
			s1Read(t, newPath, &fit)
		}
		lesson := s2Lesson(pass, fit, dev, pop, s1Time(r))
		if _, ok := r.processedEvents[lesson.ID]; ok {
			continue
		}
		before := s2MetricOf(dev, s2Predictions(t, r, dev, pop), p)
		_, receipt, e := r.LearnCompositionWithDiagnosis(lesson, s1Time(r))
		s1Check(t, e)
		if receipt == nil {
			t.Fatal("missing native S2 learning receipt")
		}
		diagnoses = append(diagnoses, receipt.Diagnosis)
		after := s2MetricOf(dev, s2Predictions(t, r, dev, pop), p)
		progress = append(progress, map[string]any{"pass": pass, "before": before, "after": after, "committed": receipt.Diagnosis.Committed, "features": len(r.brain.BrainState.NResolution.Features)})
		s1Write(t, filepath.Join(work, "learning.json"), diagnoses)
		s1Write(t, filepath.Join(work, "progress.json"), progress)
		s1Check(t, r.Checkpoint(filepath.Join(work, "lineage/latest.json")))
		if pass == 1 || pass == 8 {
			s1Check(t, r.Checkpoint(filepath.Join(work, fmt.Sprintf("02-stage-%02d.json", pass))))
		}
		t.Logf("lesson %d committed=%v dev_rmse=%.8f features=%d cause=%s", pass, receipt.Diagnosis.Committed, after.RMSE, len(r.brain.BrainState.NResolution.Features), receipt.Diagnosis.Cause)
	}
	s1Check(t, r.Checkpoint(filepath.Join(work, "lineage/s2-final.json")))
	s1Check(t, wal.Close())
}

func s2Ablate(t *testing.T, r *BrainRuntime, source *BrainRuntime) int {
	t.Helper()
	old := map[k.NodeID]bool{}
	for _, h := range source.brain.BrainState.NResolution.Heads {
		for _, q := range h.Query {
			old[q] = true
		}
	}
	query := map[k.NodeID]bool{}
	for _, h := range r.brain.BrainState.NResolution.Heads {
		for _, q := range h.Query {
			if !old[q] {
				query[q] = true
			}
		}
	}
	n := 0
	r.brain.Registry.VisitNodes(func(node *k.ConceptNode) {
		for target, list := range node.Synapses {
			if query[target] {
				for _, edge := range list {
					if edge.Dynamic.Weight != 0 {
						n++
					}
					edge.Dynamic.Weight = 0
					edge.Weight = 0
				}
			}
		}
	})
	return n
}
func s2S1Rows(rows []s2Trial) []s1Sample {
	out := make([]s1Sample, len(rows))
	for i, s := range rows {
		out[i] = s1Sample{ID: s.ID, Before: s.Pre.Before, Previous: s.Pre.Previous, Action: s.Pre.Action, PreviousAction: s.Pre.PreviousAction, BeforeAt: s.Pre.At, AfterAt: s.AfterAt, Prediction: s.Pre.Forecast, Actual: s.Actual}
	}
	return out
}
func s2ContextGaps(rows []s2Trial, pred []float64, p s2Protocol) map[string]float64 {
	out := map[string]float64{}
	for a, name := range []string{"propagate", "stop"} {
		lo, hi := 0., 0.
		nl, nh := 0, 0
		for i, s := range rows {
			if s.Pre.Action == a {
				if s.Error <= p.LowCut {
					lo += pred[i]
					nl++
				}
				if s.Error >= p.HighCut {
					hi += pred[i]
					nh++
				}
			}
		}
		if nl > 0 && nh > 0 {
			out[name] = hi/float64(nh) - lo/float64(nl)
		}
	}
	return out
}
func TestS2Evaluate(t *testing.T) {
	root, work := s2Paths(t)
	p := s2ProtocolRead(t, work)
	s2Source(t, root)
	if _, e := os.Stat(filepath.Join(work, "results.json")); !os.IsNotExist(e) {
		t.Fatal("final holdout already opened; preserve results")
	}
	pop := s1Pop(t, filepath.Join(root, "source"))
	final := filepath.Join(work, "lineage/s2-final.json")
	base := orderedFixtureClone(t, filepath.Join(work, "00-s1-source.json"))
	candidate := orderedFixtureClone(t, final)
	control := orderedFixtureClone(t, filepath.Join(work, "01-observed-untrained.json"))
	ablate := orderedFixtureClone(t, final)
	removed := s2Ablate(t, ablate, base)
	var firstFit, newFit []s2Trial
	s1Read(t, filepath.Join(work, "training.json"), &firstFit)
	s1Read(t, filepath.Join(work, "training-new.json"), &newFit)
	fullConstant := 0.
	for _, s := range append(append([]s2Trial(nil), firstFit...), newFit...) {
		fullConstant += s.Error / float64(len(firstFit)+len(newFit))
	}
	rows := []s2Trial{}
	pred := map[string][]float64{}
	bySeed := map[string]map[string]s2Metric{}
	gates := map[string]bool{"identifiability": p.Identifiable, "holdout": true, "no_future_leakage": true, "read_only": true}
	maxS1Drift := 0.
	for si, seed := range p.HoldoutSeeds {
		actual := orderedFixtureClone(t, final)
		actual.SetClock(NewFixedClock(s1Base))
		part := s2Collect(t, actual, pop, seed, p.HoldoutN, si == 1, nil, -1)
		streams := map[string][]float64{"A_S1": s2Predictions(t, base, part, pop), "B_constant": s2Constant(part, p.Constant), "B_full_training_constant": s2Constant(part, fullConstant), "C_untrained": s2Predictions(t, control, part, pop), "D_S2": s2Predictions(t, candidate, part, pop), "E_ablated": s2Predictions(t, ablate, part, pop)}
		bySeed[fmt.Sprint(seed)] = map[string]s2Metric{}
		for name, x := range streams {
			bySeed[fmt.Sprint(seed)][name] = s2MetricOf(part, x, p)
			pred[name] = append(pred[name], x...)
		}
		d := bySeed[fmt.Sprint(seed)]
		gates["holdout"] = gates["holdout"] && d["D_S2"].RMSE <= p.RelativeRMSE*d["B_constant"].RMSE && d["D_S2"].RMSE <= p.RelativeRMSE*d["B_full_training_constant"].RMSE && d["D_S2"].RMSE <= p.RelativeRMSE*d["C_untrained"].RMSE
		for i, s := range part {
			if streams["D_S2"][i] != s.Estimate {
				t.Fatal("offline estimate differs from pre-operation estimate")
			}
			raw, e := json.Marshal(s2Input(s.Pre, pop))
			s1Check(t, e)
			if s.InputSHA256 != s1Digest(raw) {
				t.Fatal("pre-operation input changed after outcome")
			}
		}
		s1rows := s2S1Rows(part)
		a, b := s1Predictions(t, base, s1rows, pop), s1Predictions(t, candidate, s1rows, pop)
		for i := range a {
			for q := range a[i] {
				maxS1Drift = math.Max(maxS1Drift, math.Abs(a[i][q]-b[i][q]))
			}
		}
		rows = append(rows, part...)
	}
	metrics := map[string]s2Metric{}
	for name, x := range pred {
		metrics[name] = s2MetricOf(rows, x, p)
	}
	trained := metrics["D_S2"]
	constant := metrics["B_constant"]
	gaps := s2ContextGaps(rows, pred["D_S2"], p)
	gates["calibration"] = trained.CalibrationError <= p.CalibrationLimit
	gates["discrimination"] = trained.Correlation >= p.Correlation && gaps["propagate"] > 0 && gaps["stop"] > 0 && len(gaps) == 2
	gates["low_high"] = trained.LowN > 0 && trained.HighN > 0 && trained.LowMAE <= constant.LowMAE && trained.HighMAE <= constant.HighMAE
	gates["synaptic_ablation"] = removed > 0 && metrics["E_ablated"].RMSE >= p.AblationRatio*trained.RMSE
	witnessFree := orderedFixtureClone(t, final)
	witnessFree.brain.BrainState.NResolution.Witnesses = nil
	gates["witness_free"] = reflect.DeepEqual(pred["D_S2"], s2Predictions(t, witnessFree, rows, pop))
	historyRows := append([]s2Trial(nil), rows...)
	for i := range historyRows {
		historyRows[i].Pre.History = nil
		historyRows[i].Pre.HistoryAction = -1
	}
	historyMetric := s2MetricOf(rows, s2Predictions(t, candidate, historyRows, pop), p)
	var learning []k.NDiagnosis
	s1Read(t, filepath.Join(work, "learning.json"), &learning)
	commits := 0
	for _, d := range learning {
		if d.Committed {
			commits++
		}
	}
	gates["learning"] = commits > 0
	gates["S1_retention"] = maxS1Drift <= p.RetentionTolerance
	gates["native_policy"] = reflect.DeepEqual(base.brain.NativeControlSnapshot(), candidate.brain.NativeControlSnapshot())
	s1metric := s1MetricOf(s2S1Rows(rows), s1Predictions(t, candidate, s2S1Rows(rows), pop))
	result := map[string]any{"protocol_sha256": func() string {
		b, e := os.ReadFile(filepath.Join(work, "PROTOCOL.json"))
		s1Check(t, e)
		return s1Digest(b)
	}(), "status": "PARTIAL", "gates": gates, "metrics": metrics, "by_seed": bySeed, "within_operation_estimated_high_low_gap": gaps, "history_removed": historyMetric, "history_incremental_rmse": historyMetric.RMSE - trained.RMSE, "ablated_synapses": removed, "native_numeric_commits": commits, "native_numeric_commands": len(learning), "S1_prediction_drift": maxS1Drift, "S1_on_same_S2_distribution": s1metric, "S1_historical_status": "PARTIAL", "S1_historical_retention_gate": false, "sequence": candidate.LastSequence(), "native_policy_updates": candidate.brain.NativeControlSnapshot().Updates, "production_promoted": false}
	s1Write(t, filepath.Join(work, "holdout.json"), rows)
	s1Write(t, filepath.Join(work, "predictions.json"), pred)
	s1Write(t, filepath.Join(work, "results.json"), result)
	t.Logf("fixed final stage: S2 RMSE %.8f constant %.8f untrained %.8f corr %.4f calibration %.8f S1 drift %.3g", trained.RMSE, constant.RMSE, metrics["C_untrained"].RMSE, trained.Correlation, trained.CalibrationError, maxS1Drift)
}

func s2Advance(t *testing.T, r *BrainRuntime, pop [][]k.NodeID, target uint64) {
	t.Helper()
	for r.LastSequence() < target {
		gap := target - r.LastSequence()
		if gap < 3 {
			t.Fatal("cannot align time by complete native trials", gap)
		}
		id := fmt.Sprintf("s2-time-control-%d", r.LastSequence())
		s1Command(t, r, NativeControlCommand{ID: id + "-begin", Begin: &NativeBegin{EpisodeID: id, Observation: &NativeObservation{State: flexCState(pop, 0, .4), Source: "s2-time-control", SampleID: id}}})
		a := 1
		if gap%3 != 0 {
			a = 0
		}
		step := s1Command(t, r, NativeControlCommand{ID: id + "-step", Step: &NativeStep{EpisodeID: id, ActionOverride: &a}})
		if !step.Stopped {
			stop := 1
			step = s1Command(t, r, NativeControlCommand{ID: id + "-stop", Step: &NativeStep{EpisodeID: id, ActionOverride: &stop}})
		}
		s1Command(t, r, NativeControlCommand{ID: id + "-close", Outcome: &NativeOutcome{EpisodeID: id, TraceDigest: step.TraceDigest, Source: "s2-time-control", OutcomeID: id, Provenance: "observed-sensor", Learn: false}})
	}
}
func TestS2Retention(t *testing.T) {
	root, work := s2Paths(t)
	p := s2ProtocolRead(t, work)
	pop := s1Pop(t, filepath.Join(root, "source"))
	eval := filepath.Join(work, "retention")
	s1Check(t, os.MkdirAll(eval, 0700))
	controlWAL := filepath.Join(eval, "control-events.jsonl")
	if _, e := os.Stat(controlWAL); e == nil {
		t.Fatal("retention evaluation exists")
	}
	b, e := os.ReadFile(filepath.Join(root, "s1/lineage/events.jsonl"))
	s1Check(t, e)
	s1Check(t, os.WriteFile(controlWAL, b, 0600))
	control, cw := s1Open(t, filepath.Join(work, "00-s1-source.json"), controlWAL)
	events, e := ReadEventLog(filepath.Join(work, "lineage/events.jsonl"))
	s1Check(t, e)
	observations, lessons := 0, 0
	for _, ev := range events {
		if ev.Sequence <= 75866 {
			continue
		}
		if ev.Type == EventTypeCompositionExposure {
			lessons++
			continue
		}
		if ev.Type != EventTypeNativeControl || ev.NativeControl == nil || !ev.Timestamp.Equal(s1Base) {
			t.Fatal("unexpected S2 schedule")
		}
		c := *ev.NativeControl
		if c.Begin != nil && c.Begin.Training || c.Outcome != nil && c.Outcome.Learn {
			t.Fatal("control would learn policy")
		}
		_, e = control.ApplyNativeControl(c, ev.Timestamp)
		s1Check(t, e)
		observations++
	}
	final := filepath.Join(work, "lineage/s2-final.json")
	candidate := orderedFixtureClone(t, final)
	s2Advance(t, control, pop, candidate.LastSequence())
	cp := filepath.Join(eval, "control-time-matched.json")
	s1Check(t, control.Checkpoint(cp))
	s1Check(t, cw.Close())
	before, after := e1Retention(t, cp, pop), e1Retention(t, final, pop)
	be, ae := e1Measure(t, cp, pop), e1Measure(t, final, pop)
	delta := s1AuditMax(s1AuditValue(t, before), s1AuditValue(t, after))
	c := after["C"].(*flexPolicyMetrics)
	passed := delta <= p.RetentionTolerance && s1AuditMax(s1AuditValue(t, be), s1AuditValue(t, ae)) <= p.RetentionTolerance && c.Accuracy == 1 && ae.Accuracy >= 35./36
	s1Write(t, filepath.Join(work, "retention.json"), map[string]any{"control": before, "candidate": after, "control_E1": be, "candidate_E1": ae, "max_delta": delta, "C_correct": int(c.Accuracy * 6), "E1_correct": int(math.Round(ae.Accuracy * 36)), "common_sequence": candidate.LastSequence(), "native_observation_commands": observations, "numerical_commands_replaced": lessons, "pass": passed, "historical_S1_retention_failure_retained": true})
	var result map[string]any
	s1Read(t, filepath.Join(work, "results.json"), &result)
	result["gates"].(map[string]any)["AE1_retention"] = passed
	s1Write(t, filepath.Join(work, "results.json"), result)
	t.Logf("A-E1 paired oracle delta %.12g C=%d/6 E1=%d/36 pass=%v", delta, int(c.Accuracy*6), int(math.Round(ae.Accuracy*36)), passed)
}

func s2SynapseChanges(t *testing.T, a, b string) map[string]int {
	t.Helper()
	read := func(path string) map[string]map[string]any {
		s := s1Snapshot(t, path)
		var v struct {
			Synapses []map[string]any `json:"synapses"`
		}
		s1Check(t, json.Unmarshal(s.Brain, &v))
		out := map[string]map[string]any{}
		for _, x := range v.Synapses {
			key := fmt.Sprint(x["source_id"], "/", x["target_id"], "/", x["inhibitory"])
			delete(x, "activation")
			out[key] = x
		}
		return out
	}
	old, new := read(a), read(b)
	out := map[string]int{"added": 0, "changed": 0, "removed": 0}
	for key, x := range old {
		y, ok := new[key]
		if !ok {
			out["removed"]++
		} else if !reflect.DeepEqual(x, y) {
			out["changed"]++
		}
	}
	for key := range new {
		if _, ok := old[key]; !ok {
			out["added"]++
		}
	}
	return out
}

func TestS2Persistence(t *testing.T) {
	root, work := s2Paths(t)
	p := s2ProtocolRead(t, work)
	s2Source(t, root)
	pop := s1Pop(t, filepath.Join(root, "source"))
	walPath := filepath.Join(work, "lineage/events.jsonl")
	final := filepath.Join(work, "lineage/s2-final.json")
	r, wal := s1Open(t, final, walPath)
	expected := continuationBytes(t, r)
	var rows []s2Trial
	s1Read(t, filepath.Join(work, "holdout.json"), &rows)
	var saved map[string][]float64
	s1Read(t, filepath.Join(work, "predictions.json"), &saved)
	restartPred := s2Predictions(t, r, rows, pop)
	if !reflect.DeepEqual(saved["D_S2"], restartPred) {
		t.Fatal("S2 restart changed predictions")
	}
	restarted := filepath.Join(work, "lineage/s2-restarted.json")
	s1Check(t, r.Checkpoint(restarted))
	if s1Snapshot(t, final).Checksum != s1Snapshot(t, restarted).Checksum {
		t.Fatal("restart native checksum mismatch")
	}
	routes := []map[string]any{}
	for _, cp := range []string{filepath.Join(work, "00-s1-source.json"), filepath.Join(work, "01-observed-untrained.json"), final} {
		s := s1Snapshot(t, cp)
		sum, e := checkpointChecksumForSchema(s)
		s1Check(t, e)
		if sum != s.Checksum {
			t.Fatal("checkpoint native checksum differs")
		}
		recovered, e := RebuildRuntimeFromCheckpointAndJournal(wal, cp)
		s1Check(t, e)
		if !bytes.Equal(expected, continuationBytes(t, recovered)) || !reflect.DeepEqual(restartPred, s2Predictions(t, recovered, rows, pop)) {
			t.Fatal("S2 suffix replay differs", cp)
		}
		routes = append(routes, map[string]any{"from": filepath.Base(cp), "from_sequence": s.Sequence, "recovered_sequence": recovered.LastSequence(), "checksum": sum, "exact_complete_state": true})
	}
	rollback := filepath.Join(work, "rollback-observed.json")
	b, e := os.ReadFile(filepath.Join(work, "01-observed-untrained.json"))
	s1Check(t, e)
	s1Check(t, os.WriteFile(rollback, b, 0600))
	s1AuditPrefix(t, walPath, rollback+".events.jsonl", s1Snapshot(t, rollback).Sequence)
	rw, e := OpenEventLog(rollback + ".events.jsonl")
	s1Check(t, e)
	rr, e := RebuildRuntimeFromCheckpointAndJournal(rw, rollback)
	s1Check(t, e)
	if !bytes.Equal(continuationBytes(t, rr), continuationBytes(t, orderedFixtureClone(t, rollback))) {
		t.Fatal("rollback pair differs")
	}
	s1Check(t, rw.Close())
	old, e := os.ReadFile(filepath.Join(root, "s1/lineage/events.jsonl"))
	s1Check(t, e)
	all, e := os.ReadFile(walPath)
	s1Check(t, e)
	if !bytes.HasPrefix(all, old) {
		t.Fatal("S1 WAL prefix modified")
	}
	diff := s2SynapseChanges(t, filepath.Join(work, "00-s1-source.json"), final)
	var result map[string]any
	s1Read(t, filepath.Join(work, "results.json"), &result)
	gates := result["gates"].(map[string]any)
	gates["synaptic_change"] = diff["added"]+diff["changed"] > 0 && diff["removed"] == 0
	gates["persistence"] = true
	var fit []s2Trial
	s1Read(t, filepath.Join(work, "training.json"), &fit)
	last := fit[len(fit)-1]
	history := []float64{}
	for _, v := range last.Residual {
		history = append(history, math.Abs(v))
	}
	continuation := s2Collect(t, r, pop, 607, 4, false, history, last.Pre.Action)
	s1Write(t, filepath.Join(work, "continuation-trials.json"), continuation)
	continued := filepath.Join(work, "lineage/s2-continuation.json")
	s1Check(t, r.Checkpoint(continued))
	after := continuationBytes(t, r)
	replayed, e := RebuildRuntimeFromCheckpointAndJournal(wal, final)
	s1Check(t, e)
	if !bytes.Equal(after, continuationBytes(t, replayed)) || !reflect.DeepEqual(saved["D_S2"], s2Predictions(t, replayed, rows, pop)) {
		t.Fatal("continuation changed S2 acquisition")
	}
	if s1Original(t, r, filepath.Join(root, "source")) != 0 {
		t.Fatal("A-E1 parameters no longer retained")
	}
	checks := []map[string]any{}
	for _, cp := range []string{final, restarted, continued} {
		raw, e := os.ReadFile(cp)
		s1Check(t, e)
		s := s1Snapshot(t, cp)
		sum, e := checkpointChecksumForSchema(s)
		s1Check(t, e)
		if sum != s.Checksum {
			t.Fatal("final checkpoint checksum mismatch")
		}
		checks = append(checks, map[string]any{"file": filepath.Base(cp), "bytes": len(raw), "sha256": s1Digest(raw), "sequence": s.Sequence, "native_checksum": sum})
	}
	pass := true
	for _, v := range gates {
		if v != true {
			pass = false
		}
	}
	if pass {
		result["status"] = "PASS"
	} else if result["native_numeric_commits"].(float64) == 0 {
		result["status"] = "BLOCKED"
	} else {
		result["status"] = "PARTIAL"
	}
	result["continuation_sequence"] = r.LastSequence()
	result["synaptic_changes"] = diff
	s1Write(t, filepath.Join(work, "results.json"), result)
	s1Write(t, filepath.Join(work, "persistence.json"), map[string]any{"routes": routes, "rollback_exact": true, "restart_exact": true, "continuation_exact": true, "S1_journal_prefix_exact": true, "S1_journal_bytes": len(old), "checkpoints": checks, "native_policy_updates": r.brain.NativeControlSnapshot().Updates, "continued_without_acquisition": true, "new_continuation_trials": 4, "protocol_sha256": func() string {
		v, e := os.ReadFile(filepath.Join(work, "PROTOCOL.json"))
		s1Check(t, e)
		return s1Digest(v)
	}(), "retention_tolerance": p.RetentionTolerance})
	s1Check(t, wal.Close())
	s2Source(t, root)
	t.Log("S2", result["status"], "continued sequence", r.LastSequence(), "synaptic changes", diff)
}

func TestS2Verify(t *testing.T) {
	root, work := s2Paths(t)
	s2Source(t, root)
	s2ProtocolRead(t, work)
	pop := s1Pop(t, filepath.Join(root, "source"))
	r, wal := s1Open(t, filepath.Join(work, "lineage/s2-continuation.json"), filepath.Join(work, "lineage/events.jsonl"))
	var rows []s2Trial
	s1Read(t, filepath.Join(work, "holdout.json"), &rows)
	var pred map[string][]float64
	s1Read(t, filepath.Join(work, "predictions.json"), &pred)
	if !reflect.DeepEqual(pred["D_S2"], s2Predictions(t, r, rows, pop)) {
		t.Fatal("shipped checkpoint lost S2 readout")
	}
	fromFinal, e := RebuildRuntimeFromCheckpointAndJournal(wal, filepath.Join(work, "lineage/s2-final.json"))
	s1Check(t, e)
	if !bytes.Equal(continuationBytes(t, r), continuationBytes(t, fromFinal)) {
		t.Fatal("shipped S2 suffix differs")
	}
	s1Check(t, wal.Close())
	t.Log("verified S2 continuation", r.LastSequence(), "policy updates", r.brain.NativeControlSnapshot().Updates)
}

// Post-acquisition verification only: saved experiences and native suffixes,
// without opening a new holdout, selecting a stage, or acquiring a lesson.
func TestS2RestartRetention(t *testing.T) {
	root, work := s2Paths(t)
	p := s2ProtocolRead(t, work)
	pop := s1Pop(t, filepath.Join(root, "source"))
	continued := filepath.Join(work, "lineage/s2-continuation.json")
	candidate, cw := s1Open(t, continued, filepath.Join(work, "lineage/events.jsonl"))
	base := orderedFixtureClone(t, filepath.Join(work, "00-s1-source.json"))
	var rows []s2Trial
	s1Read(t, filepath.Join(work, "holdout.json"), &rows)
	s1rows := s2S1Rows(rows)
	a, b := s1Predictions(t, base, s1rows, pop), s1Predictions(t, candidate, s1rows, pop)
	drift := s1AuditMax(s1AuditValue(t, a), s1AuditValue(t, b))
	if drift > p.RetentionTolerance {
		t.Fatal("S1 restart drift", drift)
	}
	controlCP := filepath.Join(work, "retention/control-time-matched.json")
	controlWAL := filepath.Join(work, "retention/control-continued-events.jsonl")
	if _, e := os.Stat(controlWAL); os.IsNotExist(e) {
		raw, err := os.ReadFile(filepath.Join(work, "retention/control-events.jsonl"))
		s1Check(t, err)
		s1Check(t, os.WriteFile(controlWAL, raw, 0600))
	}
	// An interrupted audit can restore its existing admitted native prefix.
	// No source/candidate state, sequence or receipt is reconstructed manually.
	control, wal := s1Open(t, controlCP, controlWAL)
	events, e := ReadEventLog(filepath.Join(work, "lineage/events.jsonl"))
	s1Check(t, e)
	finalSequence := s1Snapshot(t, filepath.Join(work, "lineage/s2-final.json")).Sequence
	commands := 0
	for _, ev := range events {
		if ev.Sequence <= finalSequence {
			continue
		}
		if ev.NativeControl == nil || ev.Type != EventTypeNativeControl {
			t.Fatal("continuation reacquired S2")
		}
		c := *ev.NativeControl
		if c.Outcome != nil && c.Outcome.Learn {
			t.Fatal("continuation learned policy")
		}
		if _, exists := control.processedEvents[c.ID]; !exists {
			_, e = control.ApplyNativeControl(c, ev.Timestamp)
			s1Check(t, e)
		}
		commands++
	}
	if control.LastSequence() != candidate.LastSequence() {
		t.Fatal("continued control clock not matched")
	}
	matched := filepath.Join(work, "retention/control-continued.json")
	s1Check(t, control.Checkpoint(matched))
	s1Check(t, wal.Close())
	br, ar := e1Retention(t, matched, pop), e1Retention(t, continued, pop)
	be, ae := e1Measure(t, matched, pop), e1Measure(t, continued, pop)
	delta := s1AuditMax(s1AuditValue(t, br), s1AuditValue(t, ar))
	edelta := s1AuditMax(s1AuditValue(t, be), s1AuditValue(t, ae))
	c := ar["C"].(*flexPolicyMetrics)
	if delta > p.RetentionTolerance || edelta > p.RetentionTolerance || c.Accuracy != 1 || ae.Accuracy < 35./36 {
		t.Fatal("continued retention failed", delta, edelta)
	}
	s1Write(t, filepath.Join(work, "restart-retention.json"), map[string]any{"S1_prediction_drift": drift, "S1_metric": s1MetricOf(s1rows, b), "AE1_delta": delta, "E1_delta": edelta, "C_correct": int(c.Accuracy * 6), "E1_correct": int(math.Round(ae.Accuracy * 36)), "control": br, "candidate": ar, "control_E1": be, "candidate_E1": ae, "common_sequence": candidate.LastSequence(), "continuation_native_commands": commands, "new_acquisition": false})
	s1Check(t, cw.Close())
	t.Log("restart retention S1 drift", drift, "A-E1 delta", delta, "C 6/6 E1 35/36")
}

func TestS2SavedS1Attribution(t *testing.T) {
	root, work := s2Paths(t)
	pop := s1Pop(t, filepath.Join(root, "source"))
	var rows []s2Trial
	s1Read(t, filepath.Join(work, "holdout.json"), &rows)
	samples := s2S1Rows(rows)
	continued := filepath.Join(work, "lineage/s2-continuation.json")
	base := orderedFixtureClone(t, filepath.Join(work, "00-s1-source.json"))
	candidate := orderedFixtureClone(t, continued)
	ablated := orderedFixtureClone(t, continued)
	removed := s1AblateReadout(ablated)
	a, b, c := s1Predictions(t, base, samples, pop), s1Predictions(t, candidate, samples, pop), s1Predictions(t, ablated, samples, pop)
	if !reflect.DeepEqual(a, b) {
		t.Fatal("restored S1 readout differs")
	}
	truth := make([][]float64, len(samples))
	for i, s := range samples {
		truth[i] = s.Actual
	}
	cr, corr := s1Context(samples, truth, b)
	metric, ablation := s1MetricOf(samples, b), s1MetricOf(samples, c)
	if removed == 0 || ablation.RMSE <= 1.2*metric.RMSE {
		t.Fatal("S1 predictive synapses no longer causal")
	}
	projection := []map[string]any{}
	for q := 0; q < 3; q++ {
		rmse, correct := 0., 0
		for i, s := range samples {
			d := b[i][q] - s.Actual[q]
			rmse += d * d
			if math.Abs(d) <= .05 {
				correct++
			}
		}
		projection = append(projection, map[string]any{"projection": q, "RMSE": math.Sqrt(rmse / float64(len(samples))), "within_0_05": correct, "coordinates": len(samples)})
	}
	s1Write(t, filepath.Join(work, "S1-attribution.json"), map[string]any{"source_candidate_predictions_exact": true, "candidate": metric, "readout_ablated": ablation, "ablated_synapses": removed, "context_contrast_RMSE": cr, "context_contrast_correlation": corr, "projections": projection, "observations": "saved S2 holdout only; no new S1 acquisition/evaluation distribution"})
	t.Log("S1 saved observations RMSE", metric.RMSE, "readout ablated", ablation.RMSE, "context correlation", corr)
}
