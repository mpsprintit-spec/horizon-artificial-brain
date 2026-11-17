//go:build horizon_s2r3 && horizon_s2r2

package runtime

// Opt-in acquisition/diagnosis only. Learning always uses the existing native
// journaled transaction. No fitted estimator or answer-generating engine here.
import (
	"bytes"
	"encoding/json"
	"fmt"
	"math"
	"math/rand"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"runtime/pprof"
	"sort"
	"strconv"
	"strings"
	"testing"
	"time"

	k "github.com/project-horizon/horizon-core/services/ai/knowledge"
)

var s2r3Seeds = []int{37, 83, 151}

const s2r3Checksum = "9016b845eed4e8437e87ada99b9641718a086c570e246dd0a62e57dffd1ce9d3"

func s2r3Paths(t *testing.T) (string, string, [][]k.NodeID) {
	t.Helper()
	root, work := os.Getenv("HORIZON_S2R3_SOURCE"), os.Getenv("HORIZON_S2R3_WORK")
	if root == "" || work == "" {
		t.Skip("requires authenticated R2 source and isolated R3 work")
	}
	if root == work || strings.HasPrefix(work, root+string(os.PathSeparator)) {
		t.Fatal("source/work overlap")
	}
	t.Setenv("HORIZON_S2_WORK", work)
	t.Setenv("HORIZON_S2_SOURCE", root)
	return root, work, s1Pop(t, filepath.Join(root, "source/source/source"))
}
func s2r3Source(root string) (string, string) {
	return filepath.Join(root, "s2r2/s2r2-continuation.json"), filepath.Join(root, "s2r2/training-combined/events.jsonl")
}
func s2r3JSONHash(t *testing.T, v any) string {
	t.Helper()
	b, e := json.Marshal(v)
	s1Check(t, e)
	return s1Digest(b)
}
func s2r3FileHash(t *testing.T, p string) string {
	t.Helper()
	b, e := os.ReadFile(p)
	s1Check(t, e)
	return s1Digest(b)
}
func s2r3Old(t *testing.T, cp string) []k.NodeID {
	r := orderedFixtureClone(t, cp)
	ids := []k.NodeID{}
	for _, f := range r.brain.BrainState.NResolution.Features {
		ids = append(ids, f.Node)
	}
	return ids
}
func s2r3Memory() map[string]any {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	rss, _ := os.ReadFile("/proc/self/status")
	selected := []string{}
	for _, line := range strings.Split(string(rss), "\n") {
		if strings.HasPrefix(line, "VmRSS:") || strings.HasPrefix(line, "VmHWM:") {
			selected = append(selected, line)
		}
	}
	return map[string]any{"heap_alloc": m.HeapAlloc, "heap_sys": m.HeapSys, "total_alloc": m.TotalAlloc, "mallocs": m.Mallocs, "process_rss": selected}
}
func s2r3Inventory(t *testing.T, r *BrainRuntime) map[string]any {
	b, e := r.brain.CanonicalJSON()
	s1Check(t, e)
	var v struct {
		Units    []any `json:"neural_units"`
		Synapses []any `json:"synapses"`
	}
	s1Check(t, json.Unmarshal(b, &v))
	return map[string]any{"features": len(r.brain.BrainState.NResolution.Features), "neurons": len(v.Units), "synapses": len(v.Synapses), "sequence": r.LastSequence(), "native_policy_updates": r.brain.NativeControlSnapshot().Updates, "capacity_metadata": r.brain.BrainState.NResolution.ExperimentalFeatureLimit}
}
func s2r3SaveCP(t *testing.T, r *BrainRuntime, p string) map[string]any {
	start := time.Now()
	s1Check(t, r.Checkpoint(p))
	elapsed := time.Since(start).Nanoseconds()
	s := s1Snapshot(t, p)
	sum, e := checkpointChecksumForSchema(s)
	s1Check(t, e)
	if sum != s.Checksum {
		t.Fatal("native checksum")
	}
	info, e := os.Stat(p)
	s1Check(t, e)
	return map[string]any{"path": p, "sequence": s.Sequence, "native_checksum": sum, "bytes": info.Size(), "nanoseconds": elapsed}
}
func s2r3Lesson(t *testing.T, r *BrainRuntime, id string, limit int, fit, check []s2Trial, pop [][]k.NodeID) (*CompositionLearningReceipt, map[string]any) {
	t.Helper()
	e := s2Lesson(1, fit, check, pop, s1Time(r))
	e.ID = id
	e.EpisodeID = id
	e.IndependenceGroup = id
	e.Source = "s2r3-native-observed-error"
	e.Resolution.Proposal = &k.NMetricProposalConfig{Protocol: k.NMetricBoundedV1, Strategy: "combined", Fractions: append([]float64(nil), s2r2Fractions...), PreserveShared: true, FeatureLimit: limit}
	before := s2r3Memory()
	start := time.Now()
	_, receipt, err := r.LearnCompositionWithDiagnosis(e, s1Time(r))
	elapsed := time.Since(start).Nanoseconds()
	s1Check(t, err)
	if receipt == nil {
		t.Fatal("missing native receipt")
	}
	return receipt, map[string]any{"wall_nanoseconds": elapsed, "before": before, "after": s2r3Memory(), "trials": len(receipt.Diagnosis.Trials)}
}
func s2r3Metric(t *testing.T, work string) s2Protocol {
	var p s2Protocol
	s1Read(t, filepath.Join(work, "METRIC_PROTOCOL.json"), &p)
	return p
}
func s2r3AE1(t *testing.T, before, after string, pop [][]k.NodeID) map[string]any {
	b, a := e1Retention(t, before, pop), e1Retention(t, after, pop)
	be, ae := e1Measure(t, before, pop), e1Measure(t, after, pop)
	d := s1AuditMax(s1AuditValue(t, b), s1AuditValue(t, a))
	ed := s1AuditMax(s1AuditValue(t, be), s1AuditValue(t, ae))
	c := a["C"].(*flexPolicyMetrics)
	return map[string]any{"control": b, "candidate": a, "control_E1": be, "candidate_E1": ae, "delta": d, "E1_delta": ed, "C_correct": int(c.Accuracy * 6), "E1_correct": int(math.Round(ae.Accuracy * 36)), "pass": d <= 1e-7 && ed <= 1e-7 && c.Accuracy == 1 && ae.Accuracy >= 35./36}
}

func TestS2R3Prepare(t *testing.T) {
	root, work, pop := s2r3Paths(t)
	if _, e := os.Stat(work); !os.IsNotExist(e) {
		t.Fatal("work must be new")
	}
	s2r2SourceManifest := map[string]struct {
		Bytes int    `json:"bytes"`
		SHA   string `json:"sha256"`
	}{}
	s1Read(t, filepath.Join(root, "MANIFEST.json"), &s2r2SourceManifest)
	for name, v := range s2r2SourceManifest {
		b, e := os.ReadFile(filepath.Join(root, name))
		s1Check(t, e)
		if len(b) != v.Bytes || s1Digest(b) != v.SHA {
			t.Fatal("BLOCKED: source payload", name)
		}
	}
	if len(s2r2SourceManifest) != 443 {
		t.Fatal("source payload count")
	}
	cp, wal := s2r3Source(root)
	s := s1Snapshot(t, cp)
	sum, e := checkpointChecksumForSchema(s)
	s1Check(t, e)
	if s.Sequence != 80735 || s.Checksum != s2r3Checksum || sum != s2r3Checksum {
		t.Fatal("BLOCKED: latest continuation")
	}
	s1Check(t, os.MkdirAll(work, 0700))
	s2r2Copy(t, cp, filepath.Join(work, "00-source.json"))
	s2r2Copy(t, wal, filepath.Join(work, "source-events.jsonl"))
	s2r2Copy(t, "../../../docs/research/s2_r3_capacity_20261117/PROTOCOL.md", filepath.Join(work, "PROTOCOL.md"))
	r, w := s2r2Fork(t, cp, wal, filepath.Join(work, "development-observer"))
	defer w.Close()
	if r.LastSequence() != 80735 {
		t.Fatal("source replay advanced unexpectedly")
	}
	inv := s2r3Inventory(t, r)
	if inv["features"] != 32 || inv["neurons"] != 274 || inv["synapses"] != 6408 {
		t.Fatal("source inventory", inv)
	}
	s1Write(t, filepath.Join(work, "SOURCE_VERIFICATION.json"), map[string]any{"payloads_verified": 443, "native_checksum": sum, "sequence": s.Sequence, "checkpoint_sha256": s2r3FileHash(t, cp), "journal_sha256": s2r3FileHash(t, wal), "native_owner_unified": r.activation.Memory == r.brain && r.learning.Kb == r.brain && r.dnf.Brain() == r.brain, "inventory": inv, "recovery_native": true})
	seeds := [][2]int64{{830011, 830017}, {830023, 830029}}
	values, fitValues := []float64{}, []float64{}
	for stream, pair := range seeds {
		fit := s2Collect(t, r, pop, pair[0], 64, stream == 1, nil, -1)
		check := s2Collect(t, r, pop, pair[1], 64, stream == 1, nil, -1)
		s1Write(t, filepath.Join(work, fmt.Sprintf("development-%d-fit.json", stream)), fit)
		s1Write(t, filepath.Join(work, fmt.Sprintf("development-%d-check.json", stream)), check)
		for _, row := range fit {
			fitValues = append(fitValues, row.Error)
		}
		for _, row := range check {
			values = append(values, row.Error)
		}
	}
	mean := 0.
	for _, v := range fitValues {
		mean += v / float64(len(fitValues))
	}
	devmean := 0.
	for _, v := range values {
		devmean += v / float64(len(values))
	}
	variance := 0.
	for _, v := range values {
		variance += (v - devmean) * (v - devmean) / float64(len(values))
	}
	sort.Float64s(values)
	p := s2Protocol{Baseline: s2Baseline, RelativeRMSE: .8, Correlation: .5, AblationRatio: 1.2, RetentionTolerance: 1e-7, Constant: mean, DevelopmentSD: math.Sqrt(variance), LowCut: values[len(values)/3], HighCut: values[2*len(values)/3], HoldoutN: 512, Lessons: 8, FitN: 512, DevelopmentN: 128, HoldoutSeeds: []int64{990011, 990031}}
	p.CalibrationLimit = math.Max(.003, .25*p.DevelopmentSD)
	for i := 1; i <= 5; i++ {
		p.Bins = append(p.Bins, values[len(values)-1]*float64(i)/5)
	}
	s1Write(t, filepath.Join(work, "METRIC_PROTOCOL.json"), p)
	s2r3SaveCP(t, r, filepath.Join(work, "development-observer/final.json"))
	s1Write(t, filepath.Join(work, "ACQUISITION_LOCK.json"), map[string]any{"training_seeds": s2r3Seeds, "protocol_sha256": s2r3FileHash(t, filepath.Join(work, "PROTOCOL.md")), "metric_protocol_sha256": s2r3FileHash(t, filepath.Join(work, "METRIC_PROTOCOL.json")), "before_training": true, "independent_holdout_acquired": false, "GOMAXPROCS": runtime.GOMAXPROCS(0)})
	t.Log("verified source 80735, 443 payloads; 256 fresh development observations; protocol registered")
}

func TestS2R3Train(t *testing.T) {
	root, work, pop := s2r3Paths(t)
	seed, e := strconv.Atoi(os.Getenv("HORIZON_S2R3_SEED"))
	s1Check(t, e)
	if seed != 37 && seed != 83 && seed != 151 {
		t.Fatal("unregistered seed")
	}
	dir := filepath.Join(work, fmt.Sprintf("seed-%d", seed))
	s1Check(t, os.Mkdir(dir, 0700))
	t.Setenv("HORIZON_S2_WORK", dir)
	cp, wal := s2r3Source(root)
	observer, ow := s2r2Fork(t, cp, wal, filepath.Join(dir, "observer"))
	defer ow.Close()
	profile, e := os.Create(filepath.Join(dir, "learning-cpu.pprof"))
	s1Check(t, e)
	s1Check(t, pprof.StartCPUProfile(profile))
	defer func() { pprof.StopCPUProfile(); profile.Close() }()
	models := map[int]*BrainRuntime{}
	logs := map[int]*EventLog{}
	for _, cap := range []int{32, 64} {
		r, w := s2r2Fork(t, cp, wal, filepath.Join(dir, fmt.Sprintf("arm-%d", cap)))
		models[cap] = r
		logs[cap] = w
		defer w.Close()
	}
	p := s2r3Metric(t, work)
	source := orderedFixtureClone(t, cp)
	dev := []s2Trial{}
	for stream := 0; stream < 2; stream++ {
		var x []s2Trial
		s1Read(t, filepath.Join(work, fmt.Sprintf("development-%d-check.json", stream)), &x)
		dev = append(dev, x...)
	}
	s1expected := s1Predictions(t, source, s2S1Rows(dev), pop)
	progress := []map[string]any{}
	last := uint64(80735)
	for b := 0; b < 8; b++ {
		fit := s2Collect(t, observer, pop, int64(840000+1000*seed+20*b+1), 64, b%2 == 1, nil, -1)
		check := s2Collect(t, observer, pop, int64(840000+1000*seed+20*b+7), 64, b%2 == 1, nil, -1)
		s1Write(t, filepath.Join(dir, fmt.Sprintf("block-%d-fit.json", b)), fit)
		s1Write(t, filepath.Join(dir, fmt.Sprintf("block-%d-check.json", b)), check)
		events, err := ReadEventLog(filepath.Join(dir, "observer/events.jsonl"))
		s1Check(t, err)
		commands := []LoggedEvent{}
		for _, ev := range events {
			if ev.Sequence > last {
				if ev.Type != EventTypeNativeControl || ev.NativeControl == nil {
					t.Fatal("observer learned")
				}
				commands = append(commands, ev)
			}
		}
		last = observer.LastSequence()
		s1Write(t, filepath.Join(dir, fmt.Sprintf("block-%d-observed-commands.json", b)), commands)
		for _, cap := range []int{32, 64} {
			r := models[cap]
			for _, ev := range commands {
				_, err := r.ApplyNativeControl(*ev.NativeControl, ev.Timestamp)
				s1Check(t, err)
			}
			before := s2MetricOf(check, s2Predictions(t, r, check, pop), p)
			invBefore := s2r3Inventory(t, r)
			receipt, cost := s2r3Lesson(t, r, fmt.Sprintf("s2r3-seed-%d-block-%d", seed, b), cap, fit, check, pop)
			afterPred := s2Predictions(t, r, check, pop)
			after := s2MetricOf(check, afterPred, p)
			s1delta := s1AuditMax(s1AuditValue(t, s1expected), s1AuditValue(t, s1Predictions(t, r, s2S1Rows(dev), pop)))
			witnessExact := reflect.DeepEqual(source.brain.BrainState.NResolution.Witnesses, r.brain.BrainState.NResolution.Witnesses)
			policyExact := reflect.DeepEqual(source.brain.NativeControlSnapshot(), r.brain.NativeControlSnapshot())
			cpInfo := s2r3SaveCP(t, r, filepath.Join(dir, fmt.Sprintf("arm-%d/block-%d.json", cap, b)))
			s1Write(t, filepath.Join(dir, fmt.Sprintf("arm-%d/receipt-%d.json", cap, b)), receipt)
			row := map[string]any{"seed": seed, "block": b, "capacity": cap, "before": before, "after": after, "before_inventory": invBefore, "after_inventory": s2r3Inventory(t, r), "cost": cost, "S1_max_delta": s1delta, "witnesses_exact": witnessExact, "policy_exact": policyExact, "checkpoint": cpInfo, "committed": receipt.Diagnosis.Committed, "grown": receipt.Diagnosis.Grown, "cause": receipt.Diagnosis.Cause, "predictions": afterPred}
			progress = append(progress, row)
			s1Write(t, filepath.Join(dir, "progress.json"), progress)
			t.Logf("seed %d cap %d block %d commit %v growth %d features %d RMSE %.8f -> %.8f S1 %.3g", seed, cap, b, receipt.Diagnosis.Committed, receipt.Diagnosis.Grown, len(r.brain.BrainState.NResolution.Features), before.RMSE, after.RMSE, s1delta)
			if s1delta > 1e-12 || !witnessExact || !policyExact {
				t.Fatal("strict retention failure; evidence retained")
			}
		}
	}
	for _, cap := range []int{32, 64} {
		s2r3SaveCP(t, models[cap], filepath.Join(dir, fmt.Sprintf("arm-%d/final.json", cap)))
	}
	s2r3SaveCP(t, observer, filepath.Join(dir, "observer/final.json"))
	// A control receives the identical native schedule, with no numerical
	// lessons. Complete native stop trials align the remaining event count.
	s2Advance(t, observer, pop, models[32].LastSequence())
	s2r3SaveCP(t, observer, filepath.Join(dir, "observer/time-matched.json"))
	ret := map[string]any{}
	for _, cap := range []int{32, 64} {
		ret[strconv.Itoa(cap)] = s2r3AE1(t, filepath.Join(dir, "observer/time-matched.json"), filepath.Join(dir, fmt.Sprintf("arm-%d/final.json", cap)), pop)
	}
	s1Write(t, filepath.Join(dir, "AE1-retention.json"), ret)
}

func s2r3Inputs(rows []s2Trial, pop [][]k.NodeID) []k.CompositionInput {
	ins := make([]k.CompositionInput, len(rows))
	for i, s := range rows {
		ins[i] = s2Input(s.Pre, pop)
	}
	return ins
}
func s2r3Intervention(t *testing.T, r *BrainRuntime, rows []s2Trial, pop [][]k.NodeID, condition string, seed int64) []float64 {
	ins := s2r3Inputs(rows, pop)
	donors := make([]int, len(rows))
	for i := range donors {
		donors[i] = i
	}
	if strings.HasPrefix(condition, "permute-") {
		groups := map[string][]int{}
		keys := []string{}
		projection := make([]float64, len(rows))
		for i, row := range rows {
			projection[i] = flexMean(row.Pre.Before, pop[0])
		}
		sorted := append([]float64(nil), projection...)
		sort.Float64s(sorted)
		for i, row := range rows {
			stream := 0
			if i >= len(rows)/2 {
				stream = 1
			}
			key := fmt.Sprintf("%d/%d/%d/%d", stream, row.Pre.Action, row.Pre.PreviousAction, len(row.Pre.History))
			if condition == "permute-history" {
				key += fmt.Sprint("/", row.Pre.HistoryAction)
			} else {
				q := sort.SearchFloat64s(sorted, projection[i]) * 4 / len(sorted)
				key += fmt.Sprint("/", min(q, 3))
			}
			if _, ok := groups[key]; !ok {
				keys = append(keys, key)
			}
			groups[key] = append(groups[key], i)
		}
		rng := rand.New(rand.NewSource(seed))
		for _, key := range keys {
			g := groups[key]
			d := append([]int(nil), g...)
			rng.Shuffle(len(d), func(i, j int) { d[i], d[j] = d[j], d[i] })
			for j, i := range g {
				donors[i] = d[j]
			}
		}
	}
	out := make([]float64, len(rows))
	for i, base := range ins {
		in := base
		in.Sequence = make([][]float64, len(base.Sequence))
		for j, row := range base.Sequence {
			in.Sequence[j] = append([]float64(nil), row...)
		}
		switch condition {
		case "no-history":
			for j := 3; j < 6; j++ {
				in.Sequence[1][j] = 0
			}
		case "no-forecast":
			for j := 0; j < 3; j++ {
				in.Sequence[1][j] = 0
			}
		case "no-previous-operation":
			in.Sequence[len(in.Sequence)-1][4] = 0
		case "no-presence":
			in.Sequence[0][0] = 0
		case "no-history-operation":
			in.Sequence[0][1] = 0
		case "no-extras":
			for j := 0; j < 6; j++ {
				in.Sequence[1][j] = 0
			}
			in.Sequence[0][0] = 0
			in.Sequence[0][1] = 0
		case "permute-history":
			copy(in.Sequence[1][3:6], ins[donors[i]].Sequence[1][3:6])
		case "permute-forecast":
			copy(in.Sequence[1][:3], ins[donors[i]].Sequence[1][:3])
		default:
			t.Fatal("unknown intervention", condition)
		}
		o, e := r.ReadComposition(in, in.Scene.ObservedAt)
		s1Check(t, e)
		out[i] = o.NResolution.Value
	}
	return out
}

func TestS2R3Freeze(t *testing.T) {
	_, work, _ := s2r3Paths(t)
	if _, e := os.Stat(filepath.Join(work, "CONFIRMATORY_LOCK.json")); !os.IsNotExist(e) {
		t.Fatal("already frozen")
	}
	lock := map[string]any{"before_holdout": true, "selection": "block 7 unconditionally", "holdout_seeds": []int{990011, 990031}, "per_stream": 512, "training_seeds": s2r3Seeds, "protocol": s2r3FileHash(t, filepath.Join(work, "PROTOCOL.md")), "metric_protocol": s2r3FileHash(t, filepath.Join(work, "METRIC_PROTOCOL.json"))}
	for _, seed := range s2r3Seeds {
		var ret map[string]struct {
			Pass bool `json:"pass"`
		}
		s1Read(t, filepath.Join(work, fmt.Sprintf("seed-%d/AE1-retention.json", seed)), &ret)
		for _, cap := range []int{32, 64} {
			if !ret[strconv.Itoa(cap)].Pass {
				t.Fatal("AE1 blocked final holdout", seed, cap)
			}
			p := fmt.Sprintf("seed-%d/arm-%d/final.json", seed, cap)
			lock[p] = s2r3FileHash(t, filepath.Join(work, p))
		}
	}
	for _, name := range []string{"../knowledge/neural_metric_learning.go", "../knowledge/neural_metric_proposal.go", "../knowledge/neural_metric.go", "../knowledge/neural_resolution.go", "../knowledge/neural_resolution_validation.go", "../knowledge/s2r3_probe.go", "s2r3_capacity_test.go", "s2r3_controls_test.go", "../../../scripts/analyze_s2r3_capacity.py"} {
		lock[name] = s2r3FileHash(t, name)
	}
	s1Write(t, filepath.Join(work, "CONFIRMATORY_LOCK.json"), lock)
}

func TestS2R3Final(t *testing.T) {
	root, work, pop := s2r3Paths(t)
	var lock map[string]any
	s1Read(t, filepath.Join(work, "CONFIRMATORY_LOCK.json"), &lock)
	for _, name := range []string{"../knowledge/neural_metric_learning.go", "../knowledge/neural_metric_proposal.go", "../knowledge/neural_metric.go", "../knowledge/neural_resolution.go", "../knowledge/neural_resolution_validation.go", "../knowledge/s2r3_probe.go", "s2r3_capacity_test.go", "s2r3_controls_test.go", "../../../scripts/analyze_s2r3_capacity.py"} {
		if lock[name] != s2r3FileHash(t, name) {
			t.Fatal("frozen code changed", name)
		}
	}
	flag, e := os.OpenFile(filepath.Join(work, "FINAL_STARTED"), os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
	s1Check(t, e)
	flag.Close()
	cp, wal := s2r3Source(root)
	observer, ow := s2r2Fork(t, cp, wal, filepath.Join(work, "final-observer"))
	t.Setenv("HORIZON_S2_WORK", filepath.Join(work, "final-observer"))
	rows := s2Collect(t, observer, pop, 990011, 512, false, nil, -1)
	rows = append(rows, s2Collect(t, observer, pop, 990031, 512, true, nil, -1)...)
	s1Write(t, filepath.Join(work, "holdout.json"), rows)
	s2r3SaveCP(t, observer, filepath.Join(work, "final-observer/final.json"))
	s1Check(t, ow.Close())
	old := s2r3Old(t, cp)
	p := s2r3Metric(t, work)
	predictions := map[string][]float64{"constant": s2Constant(rows, p.Constant)}
	source := orderedFixtureClone(t, cp)
	predictions["source"] = s2Predictions(t, source, rows, pop)
	retention := map[string]any{}
	for _, seed := range s2r3Seeds {
		for _, cap := range []int{32, 64} {
			name := fmt.Sprintf("s%d-c%d", seed, cap)
			rel := fmt.Sprintf("seed-%d/arm-%d/final.json", seed, cap)
			path := filepath.Join(work, rel)
			if lock[rel] != s2r3FileHash(t, path) {
				t.Fatal("frozen checkpoint changed")
			}
			r := orderedFixtureClone(t, path)
			before := continuationBytes(t, r)
			audit, e := r.brain.S2R3Inspect(s2r3Inputs(rows, pop), old)
			s1Check(t, e)
			for condition, v := range audit.Predictions {
				predictions[name+"/"+condition] = v
			}
			for _, condition := range []string{"no-history", "no-forecast", "no-previous-operation", "no-presence", "no-history-operation", "no-extras"} {
				predictions[name+"/"+condition] = s2r3Intervention(t, r, rows, pop, condition, 0)
			}
			for permutation := 0; permutation < 10; permutation++ {
				for _, condition := range []string{"permute-history", "permute-forecast"} {
					predictions[fmt.Sprintf("%s/%s-%02d", name, condition, permutation)] = s2r3Intervention(t, r, rows, pop, condition, int64(970001+permutation))
				}
			}
			s1Write(t, filepath.Join(work, name+"-feature-audit.json"), audit)
			delta := s1AuditMax(s1AuditValue(t, s1Predictions(t, source, s2S1Rows(rows), pop)), s1AuditValue(t, s1Predictions(t, r, s2S1Rows(rows), pop)))
			retention[name] = map[string]any{"S1_max_delta": delta, "coordinates": 3 * len(rows), "pass": delta <= 1e-12}
			if !bytes.Equal(before, continuationBytes(t, r)) {
				t.Fatal("evaluation mutated brain")
			}
			t.Log("evaluated", name, "features", len(r.brain.BrainState.NResolution.Features), "pooled RMSE", s2MetricOf(rows, predictions[name+"/full"], p).RMSE)
		}
	}
	s1Write(t, filepath.Join(work, "predictions.json"), predictions)
	s1Write(t, filepath.Join(work, "holdout-S1-retention.json"), retention)
	metrics := map[string]any{}
	for name, v := range predictions {
		metrics[name] = map[string]any{"pooled": s2MetricOf(rows, v, p), "normal": s2MetricOf(rows[:512], v[:512], p), "shifted": s2MetricOf(rows[512:], v[512:], p)}
	}
	s1Write(t, filepath.Join(work, "metrics.json"), metrics)
}

func TestS2R3Persistence(t *testing.T) {
	root, work, pop := s2r3Paths(t)
	cp, wal := s2r3Source(root)
	source, sw := s2r2Fork(t, cp, wal, filepath.Join(work, "rollback-audit"))
	defer sw.Close()
	if !bytes.Equal(continuationBytes(t, source), continuationBytes(t, orderedFixtureClone(t, cp))) {
		t.Fatal("rollback mismatch")
	}
	var rows []s2Trial
	s1Read(t, filepath.Join(work, "holdout.json"), &rows)
	var predictions map[string][]float64
	s1Read(t, filepath.Join(work, "predictions.json"), &predictions)
	prefix, e := os.ReadFile(wal)
	s1Check(t, e)
	results := map[string]any{}
	for _, seed := range s2r3Seeds {
		for _, cap := range []int{32, 64} {
			name := fmt.Sprintf("s%d-c%d", seed, cap)
			dir := filepath.Join(work, fmt.Sprintf("seed-%d/arm-%d", seed, cap))
			path := filepath.Join(dir, "final.json")
			journal := filepath.Join(dir, "events.jsonl")
			start := time.Now()
			r, w := s1Open(t, path, journal)
			restore := time.Since(start).Nanoseconds()
			expected := continuationBytes(t, r)
			start = time.Now()
			replayed, e := RebuildRuntimeFromCheckpointAndJournal(w, cp)
			s1Check(t, e)
			replayTime := time.Since(start).Nanoseconds()
			exact := bytes.Equal(expected, continuationBytes(t, replayed))
			if !exact {
				t.Fatal("full suffix replay", name)
			}
			if !reflect.DeepEqual(s2Predictions(t, r, rows, pop), predictions[name+"/full"]) {
				t.Fatal("restart predictions", name)
			}
			raw, e := os.ReadFile(journal)
			s1Check(t, e)
			if !bytes.HasPrefix(raw, prefix) {
				t.Fatal("historical WAL modified")
			}
			cpInfo := s2r3SaveCP(t, r, filepath.Join(dir, "restarted.json"))
			if s1Snapshot(t, path).Checksum != s1Snapshot(t, filepath.Join(dir, "restarted.json")).Checksum {
				t.Fatal("restart checksum")
			}
			ret := s2r3AE1(t, filepath.Join(work, fmt.Sprintf("seed-%d/observer/time-matched.json", seed)), filepath.Join(dir, "restarted.json"), pop)
			results[name] = map[string]any{"restart_exact": true, "full_suffix_replay_exact": exact, "rollback_exact": true, "historical_journal_prefix_exact": true, "policy_exact": reflect.DeepEqual(r.brain.NativeControlSnapshot(), source.brain.NativeControlSnapshot()), "feature_metadata_exact": reflect.DeepEqual(r.brain.BrainState.NResolution, replayed.brain.BrainState.NResolution), "witnesses_exact": reflect.DeepEqual(r.brain.BrainState.NResolution.Witnesses, source.brain.BrainState.NResolution.Witnesses), "recovery_nanoseconds": restore, "full_suffix_replay_nanoseconds": replayTime, "checkpoint": cpInfo, "journal_bytes": len(raw), "suffix_bytes": len(raw) - len(prefix), "AE1": ret}
			s1Check(t, w.Close())
			s1Write(t, filepath.Join(work, "persistence.json"), results)
			t.Log("native persistence exact", name)
		}
	}
}
