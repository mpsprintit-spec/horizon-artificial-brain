package runtime

// Post-acquisition audit only. Historical oracles, protocols, checkpoints and
// learning streams are immutable inputs. Every sequence advance below is an
// admitted native command on a disposable evaluation lineage, never an edit.
import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"reflect"
	"testing"
	"time"

	k "github.com/project-horizon/horizon-core/services/ai/knowledge"
)

func s1AuditDir(t *testing.T) string {
	t.Helper()
	d := os.Getenv("HORIZON_S1_AUDIT")
	if d == "" {
		t.Skip("requires explicit independent audit output directory")
	}
	source, work := s1Paths(t)
	for _, p := range []string{source, work} {
		rel, e := filepath.Rel(filepath.Clean(p), filepath.Clean(d))
		s1Check(t, e)
		if rel == "." || (rel != ".." && !bytes.HasPrefix([]byte(rel), []byte(".."+string(os.PathSeparator)))) {
			t.Fatal("audit output overlaps immutable inputs")
		}
	}
	s1Check(t, os.MkdirAll(d, 0700))
	return d
}

func s1AuditAdvance(t *testing.T, r *BrainRuntime, pop [][]k.NodeID, target uint64) {
	t.Helper()
	for r.LastSequence() < target {
		gap := target - r.LastSequence()
		if gap < 3 {
			t.Fatal("target cannot be reached by a complete native trial", gap)
		}
		id := fmt.Sprintf("s1-final-audit-%d", r.LastSequence())
		s1Command(t, r, NativeControlCommand{ID: id + "-begin", Begin: &NativeBegin{EpisodeID: id, Observation: &NativeObservation{State: flexCState(pop, 0, .4), Source: "independent-retention-control", SampleID: id}}})
		a := 1
		if gap%3 != 0 {
			a = 0
		}
		step := s1Command(t, r, NativeControlCommand{ID: id + "-step", Step: &NativeStep{EpisodeID: id, ActionOverride: &a}})
		if !step.Stopped {
			stop := 1
			step = s1Command(t, r, NativeControlCommand{ID: id + "-stop", Step: &NativeStep{EpisodeID: id, ActionOverride: &stop}})
		}
		s1Command(t, r, NativeControlCommand{ID: id + "-outcome", Outcome: &NativeOutcome{EpisodeID: id, TraceDigest: step.TraceDigest, Source: "independent-retention-control", OutcomeID: id, Provenance: "observed-sensor", Learn: false}})
	}
	if r.brain.NativeControlSnapshot().Updates != 10752 || r.brain.BrainState.NResolution != nil {
		t.Fatal("retention control acquired learning")
	}
}

// Copy an authenticated existing prefix verbatim into a NEW evaluation WAL.
// No source file or record is rewritten. Native recovery validates the prefix.
func s1AuditPrefix(t *testing.T, from, to string, sequence uint64) {
	t.Helper()
	f, e := os.Open(from)
	s1Check(t, e)
	defer f.Close()
	out, e := os.OpenFile(to, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
	s1Check(t, e)
	scan := bufio.NewScanner(f)
	scan.Buffer(make([]byte, 64<<10), 16<<20)
	for scan.Scan() {
		var row struct {
			Sequence uint64 `json:"sequence"`
		}
		s1Check(t, json.Unmarshal(scan.Bytes(), &row))
		if row.Sequence > sequence {
			break
		}
		_, e = out.Write(append(append([]byte(nil), scan.Bytes()...), '\n'))
		s1Check(t, e)
	}
	s1Check(t, scan.Err())
	s1Check(t, out.Close())
}

func s1AuditMax(a, b any) float64 {
	maximum := 0.
	switch x := a.(type) {
	case float64:
		y, ok := b.(float64)
		if !ok {
			return math.Inf(1)
		}
		return math.Abs(x - y)
	case []any:
		y, ok := b.([]any)
		if !ok || len(x) != len(y) {
			return math.Inf(1)
		}
		for i, v := range x {
			maximum = math.Max(maximum, s1AuditMax(v, y[i]))
		}
	case map[string]any:
		y, ok := b.(map[string]any)
		if !ok || len(x) != len(y) {
			return math.Inf(1)
		}
		for key, v := range x {
			u, ok := y[key]
			if !ok {
				return math.Inf(1)
			}
			maximum = math.Max(maximum, s1AuditMax(v, u))
		}
	default:
		if !reflect.DeepEqual(a, b) {
			return math.Inf(1)
		}
	}
	return maximum
}
func s1AuditValue(t *testing.T, v any) any {
	t.Helper()
	raw, e := json.Marshal(v)
	s1Check(t, e)
	var out any
	s1Check(t, json.Unmarshal(raw, &out))
	return out
}

func TestS1FinalRetentionAudit(t *testing.T) {
	source, work := s1Paths(t)
	d := s1AuditDir(t)
	s1Source(t, source)
	pop := s1Pop(t, source)
	final := filepath.Join(work, "lineage/s1-final.json")
	var saved s1Result
	s1Read(t, filepath.Join(work, "results.json"), &saved)
	if saved.Status != "PARTIAL" || saved.Gates["retention"] || saved.Sequence != 75849 {
		t.Fatal("historical result changed")
	}

	// A: original A–E1, with no adapter registration or prediction lessons.
	aWal := filepath.Join(d, "control-a-events.jsonl")
	s1AuditPrefix(t, filepath.Join(source, "events.jsonl"), aWal, 75331)
	a, aw := s1Open(t, filepath.Join(source, "07-E1-restarted.json"), aWal)
	// Keep the original memory watermark too. Native trials are metadata-only
	// for that watermark: unlike cognitive A/B/D probes, C/E1 factual native
	// trials do not settle the source's remaining 1 ms. Record that residual
	// separately, rather than altering A or requiring false exact equality.
	s1AuditAdvance(t, a, pop, saved.Sequence)
	ap := filepath.Join(d, "control-a.json")
	s1Check(t, a.Checkpoint(ap))
	s1Check(t, aw.Close())

	// B: identical numeric registration and all 480 original native observation
	// commands at their original timestamps. Replace the 36 prediction-learning
	// revisions by 12 complete stop-only nonlearning trials at that same time.
	bWal := filepath.Join(d, "control-b-events.jsonl")
	s1AuditPrefix(t, filepath.Join(work, "lineage/events.jsonl"), bWal, 75333)
	b, bw := s1Open(t, filepath.Join(work, "01-initialized-control.json"), bWal)
	events, e := ReadEventLog(filepath.Join(work, "lineage/events.jsonl"))
	s1Check(t, e)
	observations := 0
	for _, ev := range events {
		if ev.Sequence <= 75333 || ev.Sequence > saved.Sequence {
			continue
		}
		if ev.Type == EventTypeCompositionExposure {
			continue
		}
		if ev.Type != EventTypeNativeControl || ev.NativeControl == nil || !ev.Timestamp.Equal(s1Base) {
			t.Fatal("unrecognized acquisition schedule")
		}
		c := *ev.NativeControl
		if (c.Begin != nil && c.Begin.Training) || (c.Outcome != nil && c.Outcome.Learn) {
			t.Fatal("control would learn")
		}
		_, e = b.ApplyNativeControl(c, ev.Timestamp)
		s1Check(t, e)
		observations++
	}
	if observations != 480 || b.LastSequence() != 75813 {
		t.Fatal("observation schedule differs", observations, b.LastSequence())
	}
	s1AuditAdvance(t, b, pop, saved.Sequence)
	bp := filepath.Join(d, "control-b.json")
	s1Check(t, b.Checkpoint(bp))
	s1Check(t, bw.Close())

	// Reuse the historical oracle without modifying a line of its code.
	aRet, bRet, cRet := e1Retention(t, ap, pop), e1Retention(t, bp, pop), e1Retention(t, final, pop)
	aE, bE, cE := e1Measure(t, ap, pop), e1Measure(t, bp, pop), e1Measure(t, final, pop)
	av, bv, cv := s1AuditValue(t, aRet), s1AuditValue(t, bRet), s1AuditValue(t, cRet)
	aDelta, bDelta := s1AuditMax(av, cv), s1AuditMax(bv, cv)
	aABDDeltas := map[string]float64{}
	for _, domain := range []string{"A", "B", "D"} {
		aABDDeltas[domain] = s1AuditMax(av.(map[string]any)[domain], cv.(map[string]any)[domain])
		if aABDDeltas[domain] > 1e-12 {
			t.Fatal("time-matched original margin differs", domain, aABDDeltas[domain])
		}
	}
	// Exact attribution is B versus C: registration, observation schedule and
	// memory watermark match, while B has NO S1 acquisition. A's raw C/E1
	// residual is not silently rounded away or presented as exact equality.
	if bDelta != 0 || !reflect.DeepEqual(bE, cE) {
		t.Fatal("functional retention regression", aDelta, bDelta)
	}
	if !reflect.DeepEqual(cv, s1AuditValue(t, saved.RetentionAfter)) {
		t.Fatal("historical S1 oracle no longer reproduced")
	}

	// Separate timestamp intervention: one unchanged A probe, same original
	// checkpoint/sequence/input, two prospective evaluation times. Only the
	// observation timestamp differs; no state, decay law or oracle is replaced.
	var cfg flexCurriculum
	s1Read(t, "testdata/continual_flexibility_v1.json", &cfg)
	probe := cfg.A[0]
	clockProbe := func(at time.Time) map[string]any {
		r := orderedFixtureClone(t, filepath.Join(source, "07-E1-restarted.json"))
		before, _ := r.brain.CanonicalJSON()
		_, e := r.CognitiveProcess(Event{ID: "independent-time-only-probe", Timestamp: at, Cycles: 1, EpisodeID: "flex-evaluation", ResetContext: true, NeuralTrace: &k.ObservationTrace{Stimulus: flexFrames(pop, probe.Sequence[:len(probe.Sequence)-1], false), Context: flexFrames(pop, probe.Context, false)}})
		s1Check(t, e)
		pred := r.activation.PredictionSnapshot().State
		margin := flexMean(pred, pop[probe.Sequence[len(probe.Sequence)-1]]) - flexMean(pred, pop[cfg.A[1].Sequence[len(cfg.A[1].Sequence)-1]])
		after, _ := r.brain.CanonicalJSON()
		return map[string]any{"at": at, "margin": margin, "before": json.RawMessage(before), "after": json.RawMessage(after)}
	}
	t0 := time.Date(2026, 11, 14, 0, 0, 0, 0, time.UTC).Add(75332 * time.Millisecond)
	p0, p1 := clockProbe(t0), clockProbe(t0.Add(518*time.Millisecond))
	if math.Abs(p0["margin"].(float64)-saved.RetentionBefore["A"].([]any)[0].(float64)) > 1e-14 {
		t.Fatal("time-only baseline differs")
	}
	if math.Abs(p1["margin"].(float64)-cRet["A"].([]float64)[0]) > 1e-12 {
		t.Fatal("timestamp intervention did not explain A margin")
	}
	// Raw neural snapshots allow an independent numerical audit of the native
	// exponential law, learning fields and transient activation in Python.
	s1Write(t, filepath.Join(d, "time-intervention.json"), []any{p0, p1})
	for name, path := range map[string]string{"A": ap, "B": bp, "C": final} {
		r := orderedFixtureClone(t, path)
		if s1Original(t, r, source) != 0 {
			t.Fatal("old parameters changed", name)
		}
	}
	s1Write(t, filepath.Join(d, "retention.json"), map[string]any{
		"historical_status": saved.Status, "historical_retention_gate": false, "historical_tolerance": 1e-7,
		"source_sequence": 75331, "common_sequence": saved.Sequence, "source_first_probe_at": t0, "common_first_probe_at": t0.Add(518 * time.Millisecond),
		"A_historical": saved.RetentionBefore, "A_time_matched": aRet, "B_schedule_matched": bRet, "C_S1": cRet,
		"A_time_matched_E1": aE, "B_schedule_matched_E1": bE, "C_S1_E1": cE,
		"B_native_observation_commands": observations, "B_replacement_stop_trials": 12, "B_prediction_learning_commands": 0,
		"A_C_max_absolute_delta": aDelta, "B_C_max_absolute_delta": bDelta, "B_C_oracles_exact": reflect.DeepEqual(bv, cv), "B_C_E1_exact": reflect.DeepEqual(bE, cE),
		"A_C_ABD_max_absolute_delta": aABDDeltas, "A_C_E1_max_absolute_delta": s1AuditMax(s1AuditValue(t, aE), s1AuditValue(t, cE)),
		"A_memory_watermark": a.brain.BrainState.MemoryDecay, "B_memory_watermark": b.brain.BrainState.MemoryDecay,
		"control_A_checkpoint": filepath.Base(ap), "control_B_checkpoint": filepath.Base(bp), "native_policy_updates": 10752,
	})
	t.Logf("retention: historical PARTIAL retained; A/C max %.17g, B/C %.17g; C 6/6 E1 35/36", aDelta, bDelta)
}

func TestS1FinalPersistenceAudit(t *testing.T) {
	source, work := s1Paths(t)
	d := s1AuditDir(t)
	s1Source(t, source)
	walPath := filepath.Join(work, "lineage/events.jsonl")
	wal, e := OpenEventLog(walPath)
	s1Check(t, e)
	defer wal.Close()
	paths := []string{filepath.Join(source, "07-E1-restarted.json"), filepath.Join(work, "02-observed-pretraining.json"), filepath.Join(work, "lineage/s1-final.json"), filepath.Join(work, "lineage/s1-continuation.json")}
	checks := []map[string]any{}
	var canonical []byte
	for _, p := range paths {
		s := s1Snapshot(t, p)
		sum, e := checkpointChecksumForSchema(s)
		s1Check(t, e)
		if sum != s.Checksum {
			t.Fatal("native checksum differs", p)
		}
		r, e := RebuildRuntimeFromCheckpointAndJournal(wal, p)
		s1Check(t, e)
		if canonical == nil {
			canonical = continuationBytes(t, r)
		}
		if !bytes.Equal(canonical, continuationBytes(t, r)) || r.LastSequence() != 75866 {
			t.Fatal("saved suffix replay differs", p)
		}
		checks = append(checks, map[string]any{"checkpoint": filepath.Base(p), "checkpoint_sequence": s.Sequence, "native_checksum": sum, "recovered_sequence": r.LastSequence(), "complete_state_equal": true})
	}
	rollbacks := []map[string]any{}
	for _, n := range []string{"rollback-before-numeric-registration.json", "rollback-before-numeric-banks.json"} {
		p := filepath.Join(work, n)
		s := s1Snapshot(t, p)
		sum, e := checkpointChecksumForSchema(s)
		s1Check(t, e)
		if sum != s.Checksum || s.SchemaVersion != 20 {
			t.Fatal("rollback lost native state", n)
		}
		w, e := OpenEventLog(p + ".events.jsonl")
		s1Check(t, e)
		r, e := RebuildRuntimeFromCheckpointAndJournal(w, p)
		s1Check(t, e)
		plain := orderedFixtureClone(t, p)
		if !bytes.Equal(continuationBytes(t, r), continuationBytes(t, plain)) {
			t.Fatal("rollback pair differs", n)
		}
		s1Check(t, w.Close())
		rollbacks = append(rollbacks, map[string]any{"checkpoint": n, "sequence": s.Sequence, "schema": s.SchemaVersion, "complete_state_equal": true, "native_checksum": sum})
	}
	a, e := os.ReadFile(filepath.Join(source, "events.jsonl"))
	s1Check(t, e)
	b, e := os.ReadFile(walPath)
	s1Check(t, e)
	if !bytes.HasPrefix(b, a) {
		t.Fatal("original journal prefix lost")
	}
	s1Write(t, filepath.Join(d, "persistence.json"), map[string]any{"recovery": checks, "rollback": rollbacks, "original_journal_prefix": true, "original_journal_bytes": len(a), "continuation_journal_bytes": len(b), "new_acquisition_commands": 0})
	t.Log("native checksums, both rollback pairs and four suffix recovery routes verified without new acquisition")
}

// Isolate the passive phase from the historic observation oracle. That oracle
// also performs its existing activity plasticity on its disposable probe copy;
// treating the whole probe as pure decay would be an incorrect causal model.
func TestS1FinalPassiveTimeAudit(t *testing.T) {
	source, _ := s1Paths(t)
	d := s1AuditDir(t)
	s1Source(t, source)
	rows := []map[string]any{}
	for _, at := range []time.Time{s1Base, s1Base.Add(518 * time.Millisecond)} {
		r := orderedFixtureClone(t, filepath.Join(source, "07-E1-restarted.json"))
		before, e := r.brain.CanonicalJSON()
		s1Check(t, e)
		// Existing public native physical-time scheduler; no parameter writes,
		// sequence edits, changed decay rule, operation targets or model fitting.
		r.brain.ApplyMemoryDynamicsWithEncoding(at, r.dynamicsEncoding)
		after, e := r.brain.CanonicalJSON()
		s1Check(t, e)
		if r.LastSequence() != 75331 {
			t.Fatal("passive scheduler advanced revision")
		}
		rows = append(rows, map[string]any{"at": at, "before": json.RawMessage(before), "after": json.RawMessage(after), "sequence_before": 75331, "sequence_after": r.LastSequence()})
	}
	s1Write(t, filepath.Join(d, "passive-time.json"), rows)
}
