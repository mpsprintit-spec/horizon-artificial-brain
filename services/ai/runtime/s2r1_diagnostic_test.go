//go:build horizon_s2r1

package runtime

// Exact historical replay and detached counterfactual evaluation. No new
// lesson is admitted, no source file is opened as a writer, and final holdout
// values are never loaded by this diagnostic.
import (
	"bytes"
	"encoding/json"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	k "github.com/project-horizon/horizon-core/services/ai/knowledge"
)

func s2r1Paths(t *testing.T) (string, string) {
	t.Helper()
	a, b := os.Getenv("HORIZON_S2R1_SOURCE"), os.Getenv("HORIZON_S2R1_WORK")
	if a == "" || b == "" {
		t.Skip("requires verified immutable S2 archive root and isolated diagnostic work")
	}
	if a == b {
		t.Fatal("source must be isolated from diagnostic work")
	}
	s1Check(t, os.MkdirAll(b, 0700))
	return a, b
}

func s2r1Write(t *testing.T, work, name string, v any) {
	t.Helper()
	s1Write(t, filepath.Join(work, name), v)
}

func s2r1CloneFile(t *testing.T, from, to string) {
	t.Helper()
	b, e := os.ReadFile(from)
	s1Check(t, e)
	s1Check(t, os.WriteFile(to, b, 0600))
}

func s2r1Open(t *testing.T, root, work string) (*BrainRuntime, *EventLog) {
	t.Helper()
	cp := filepath.Join(work, "source-continuation.json")
	wal := filepath.Join(work, "source-events.jsonl")
	s2r1CloneFile(t, filepath.Join(root, "s2/lineage/s2-continuation.json"), cp)
	s2r1CloneFile(t, filepath.Join(root, "s2/lineage/events.jsonl"), wal)
	return s1Open(t, cp, wal)
}

func TestS2R1Source(t *testing.T) {
	root, work := s2r1Paths(t)
	var manifest map[string]struct {
		Bytes int    `json:"bytes"`
		SHA   string `json:"sha256"`
	}
	s1Read(t, filepath.Join(root, "MANIFEST.json"), &manifest)
	for name, want := range manifest {
		b, e := os.ReadFile(filepath.Join(root, name))
		s1Check(t, e)
		if len(b) != want.Bytes || s1Digest(b) != want.SHA {
			t.Fatal("source payload mismatch", name)
		}
	}
	r, wal := s2r1Open(t, root, work)
	s := s1Snapshot(t, filepath.Join(work, "source-continuation.json"))
	sum, e := checkpointChecksumForSchema(s)
	s1Check(t, e)
	if sum != s.Checksum || sum != "b87285e38a321d45e4da920f219b388d1a2d061554042d2dd0f109883b38ab9d" || s.Sequence != 76617 {
		t.Fatal("S2 native lineage mismatch")
	}
	var b struct {
		Units    []any `json:"neural_units"`
		Synapses []any `json:"synapses"`
		Patterns []any `json:"temporal_patterns"`
	}
	s1Check(t, json.Unmarshal(s.Brain, &b))
	if len(b.Units) != 264 || len(b.Synapses) != 6244 || len(b.Patterns) != 24 || len(r.brain.BrainState.NResolution.Features) != 30 || r.brain.NativeControlSnapshot().Updates != 10752 {
		t.Fatal("source inventory mismatch")
	}
	before := continuationBytes(t, r)
	pop := s1Pop(t, filepath.Join(root, "source/source"))
	var dev []s2Trial
	s1Read(t, filepath.Join(root, "s2/development.json"), &dev)
	for _, row := range dev {
		in := s2Input(row.Pre, pop)
		o, e := r.ReadComposition(in, row.Pre.At)
		s1Check(t, e)
		d, e := r.brain.S2R1Inspect(in)
		s1Check(t, e)
		if o.NResolution.Value != d["value"].(float64) {
			t.Fatal("diagnostic model/readout mismatch")
		}
		// Diagnostic descriptors must be detached even if a caller edits them.
		d["sources"].([]k.NodeID)[56] = 999999
		for _, f := range d["features"].([]map[string]any) {
			if parents, ok := f["parents"].([]k.NodeID); ok && len(parents) > 0 {
				parents[0] = 999999
				break
			}
		}
	}
	if !bytes.Equal(before, continuationBytes(t, r)) {
		t.Fatal("read-only verification mutated state")
	}
	s2r1Write(t, work, "source-validation.json", map[string]any{"payloads_verified": len(manifest), "sequence": s.Sequence, "native_checksum": sum, "neurons": len(b.Units), "synapses": len(b.Synapses), "patterns": len(b.Patterns), "features": len(r.brain.BrainState.NResolution.Features), "policy_updates": r.brain.NativeControlSnapshot().Updates, "development_readout_checks": len(dev), "state_unchanged": true, "final_holdout_used": false})
	s1Check(t, wal.Close())
	t.Log("verified immutable S2 continuation and 64 development reads; no holdout evaluation")
}

func s2r1Condition(p s2Pre, name string, index int) s2Pre {
	p.Forecast = append([]float64(nil), p.Forecast...)
	p.History = append([]float64(nil), p.History...)
	switch name {
	case "B-no-forecast":
		p.Forecast = []float64{0, 0, 0}
	case "C-no-history":
		p.History = nil
		p.HistoryAction = 0
	case "D-controlled-both":
		p.Forecast = []float64{.25, -.25, .125}
		p.History = []float64{.04, .01, .02}
		p.HistoryAction = 1
	case "E-legal-variation":
		v := -.8 + 1.6*float64(index%17)/16
		p.Forecast = []float64{v, -v, v / 2}
		p.History = []float64{math.Abs(v) / 2, math.Abs(v) / 3, math.Abs(v) / 4}
		p.HistoryAction = index % 2
	}
	return p
}

func TestS2R1InputAndCausal(t *testing.T) {
	root, work := s2r1Paths(t)
	r, wal := s2r1Open(t, root, work)
	defer wal.Close()
	pop := s1Pop(t, filepath.Join(root, "source/source"))
	var rows []s2Trial
	s1Read(t, filepath.Join(root, "s2/development.json"), &rows)
	before := continuationBytes(t, r)
	conditions := []map[string]any{}
	inputs := []map[string]any{}
	observations := []k.NObservation{}
	var graph map[string]any
	for _, name := range []string{"A-original", "B-no-forecast", "C-no-history", "D-controlled-both", "E-legal-variation"} {
		maxDelta := 0.
		changes := 0
		detail := []map[string]any{}
		for i, row := range rows {
			in := s2Input(s2r1Condition(row.Pre, name, i), pop)
			s1Check(t, in.Validate())
			d, e := r.brain.S2R1Inspect(in)
			s1Check(t, e)
			o, e := r.ReadComposition(in, row.Pre.At)
			s1Check(t, e)
			base := s2Predict(t, r, row.Pre, pop)
			delta := math.Abs(base - o.NResolution.Value)
			maxDelta = math.Max(maxDelta, delta)
			if delta > 0 {
				changes++
			}
			if o.NResolution.Value != d["value"].(float64) {
				t.Fatal("trace/actual mismatch")
			}
			if graph == nil {
				graph, e = r.brain.S2R1Inspect(in)
				s1Check(t, e)
			}
			delete(d, "model")
			for _, feature := range d["features"].([]map[string]any) {
				delete(feature, "axis")
			}
			if name == "A-original" {
				x := d["x"].([]float64)
				old := r.brain.S2R1Input(s1Input(3, s1Sample{Before: row.Pre.Before, Previous: row.Pre.Previous, Action: row.Pre.Action, PreviousAction: row.Pre.PreviousAction, BeforeAt: row.Pre.At}, pop))
				if !reflect.DeepEqual(x[:56], old[:56]) || !reflect.DeepEqual(x[56:59], row.Pre.Forecast) {
					t.Fatal("reverse lag mapping mismatch")
				}
				if len(row.Pre.History) == 3 && !reflect.DeepEqual(x[59:62], row.Pre.History) {
					t.Fatal("history mapping mismatch")
				}
				inputs = append(inputs, map[string]any{"id": row.ID, "forecast": row.Pre.Forecast, "history": row.Pre.History, "action": row.Pre.Action, "target": row.Error, "inspection": d})
				observations = append(observations, k.NObservation{Input: in, Outcome: row.Error, EpisodeID: row.ID, ObservedAt: row.AfterAt})
			}
			detail = append(detail, map[string]any{"id": row.ID, "value": o.NResolution.Value, "delta": delta, "x_extra": d["x"].([]float64)[56:70], "features": d["features"]})
		}
		conditions = append(conditions, map[string]any{"condition": name, "n": len(rows), "changed_predictions": changes, "max_prediction_delta": maxDelta, "rows": detail})
	}
	causal, e := r.brain.S2R1Causal(observations)
	s1Check(t, e)
	if !bytes.Equal(before, continuationBytes(t, r)) {
		t.Fatal("input/causal interventions mutated runtime")
	}
	s2r1Write(t, work, "input-path.json", map[string]any{"source_graph": graph, "rows": inputs, "conditions": conditions, "read_only_state_unchanged": true, "scope": "historical development data; no holdout or generalization claim"})
	s2r1Write(t, work, "causal-interventions.json", causal)
	t.Log("encoded coordinates checked; five input conditions and readout/source parameter interventions evaluated without state mutation")
}

func TestS2R1HistoricalAdmission(t *testing.T) {
	root, work := s2r1Paths(t)
	verified, wal := s2r1Open(t, root, work)
	defer wal.Close()
	events, e := ReadEventLog(filepath.Join(work, "source-events.jsonl"))
	s1Check(t, e)
	var receipts []k.NDiagnosis
	s1Read(t, filepath.Join(root, "s2/learning.json"), &receipts)
	r := NewBrainRuntime(k.NewBootstrapBrain())
	s1Check(t, r.RestoreCheckpoint(filepath.Join(root, "s2/01-observed-untrained.json")))
	r.SetClock(NewFixedClock(s1Base))
	lessons := []map[string]any{}
	probes := []map[string]any{}
	positive := []map[string]any{}
	count := 0
	start := r.LastSequence()
	for _, event := range events[start:] {
		if event.CompositionExposure != nil && len(event.CompositionExposure.ID) >= 18 && event.CompositionExposure.ID[:18] == "s2-error-learning-" {
			count++
			before := continuationBytes(t, r)
			brain, e := r.brain.CanonicalJSON()
			s1Check(t, e)
			tr, e := r.brain.S2R1Diagnose(r.composition.Config, *event.CompositionExposure, event.Timestamp)
			s1Check(t, e)
			if !bytes.Equal(before, continuationBytes(t, r)) {
				t.Fatal("candidate trace mutated state", count)
			}
			want := receipts[count-1]
			gotTrials, _ := json.Marshal(tr.Diagnosis.Trials)
			wantTrials, _ := json.Marshal(want.Trials)
			if tr.Diagnosis.Committed != want.Committed || tr.Diagnosis.BeforeError != want.BeforeError || tr.Diagnosis.AfterError != want.AfterError || !bytes.Equal(gotTrials, wantTrials) {
				s2r1Write(t, work, "trace-receipt-mismatch.json", map[string]any{"lesson": count, "got": tr.Diagnosis, "want": want})
				t.Fatalf("native candidate trace differs from historical receipt at lesson %d", count)
			}
			lessons = append(lessons, map[string]any{"lesson": count, "event_id": event.CompositionExposure.ID, "sequence_before": r.LastSequence(), "brain_sha256_before": s1Digest(brain), "native_control_before": r.brain.NativeControlSnapshot(), "activation_before": r.activation.CurrentState(), "trace": tr, "historical_receipt": want, "exact_trial_match": true, "state_unchanged_by_trace": true})
			if count == 1 || count == 5 || count == 9 {
				probes = append(probes, map[string]any{"lesson": count, "scope": "counterfactual uninstalled single-coordinate feature candidates", "trials": tr.S2R1BasisProbes()})
			}
			if count == 9 {
				for _, seed := range []int64{3101, 3109, 3121} {
					for _, coordinate := range []int{56, 59} {
						for _, vary := range []bool{false, true} {
							diagnostic := k.S2R1DiagnosticLesson(*event.CompositionExposure, seed, coordinate, vary)
							p, e := r.brain.S2R1Diagnose(r.composition.Config, diagnostic, event.Timestamp)
							s1Check(t, e)
							positive = append(positive, map[string]any{"seed": seed, "coordinate": coordinate, "other_coordinates_vary": vary, "scope": "synthetic reachability diagnostic, not S2 error prediction or new training", "trace": p})
						}
					}
				}
			}
			if !bytes.Equal(before, continuationBytes(t, r)) {
				t.Fatal("counterfactual probes mutated state")
			}
		}
		// Replay only already committed historical events through the original
		// native recovery path. This acquires no experience and writes no WAL.
		s1Check(t, replayEventLogFromSequence(r, []LoggedEvent{event}, event.Sequence))
	}
	if count != 16 {
		t.Fatal("did not reconstruct 16 historical lessons", count)
	}
	if !bytes.Equal(continuationBytes(t, r), continuationBytes(t, verified)) {
		t.Fatal("historical suffix recovery differs from continuation")
	}
	s2r1Write(t, work, "admission-traces.json", map[string]any{"lessons": lessons, "exact_16_receipt_matches": true, "replayed_sequence_from": start, "replayed_sequence_to": r.LastSequence(), "continuation_exact": true, "new_learning_commands": 0})
	s2r1Write(t, work, "basis-probes.json", probes)
	s2r1Write(t, work, "positive-controls.json", positive)
	t.Logf("exact historical receipt match for %d lessons; recovered original continuation at %d; all counterfactuals uninstalled", count, r.LastSequence())
}

func TestS2R1PositiveControlLabels(t *testing.T) {
	// Ordinary CI can verify the diagnostic generator even without artifacts.
	seed := int64(3101)
	p := s2Pre{Before: map[k.NodeID]float64{}, Forecast: []float64{0, 0, 0}, Action: 0, PreviousAction: -1, At: s1Base}
	pop := [][]k.NodeID{}
	for i := 0; i < 9; i++ {
		pop = append(pop, []k.NodeID{k.NodeID(i*4 + 1), k.NodeID(i*4 + 2), k.NodeID(i*4 + 3), k.NodeID(i*4 + 4)})
	}
	rows := []s2Trial{}
	for i := 0; i < 4; i++ {
		rows = append(rows, s2Trial{ID: fmt.Sprint(i), Pre: p, Error: .01, AfterAt: s1Base})
	}
	e := s2Lesson(1, rows, rows, pop, s1Base)
	for _, coordinate := range []int{56, 59} {
		d := k.S2R1DiagnosticLesson(e, seed, coordinate, true)
		for _, r := range d.Resolution.Fit {
			lag, part := coordinate/7, coordinate%7
			v := r.Input.Sequence[len(r.Input.Sequence)-1-lag][part]
			if r.Outcome != .1+.2*v {
				t.Fatal("diagnostic label mismatch")
			}
		}
	}
}
