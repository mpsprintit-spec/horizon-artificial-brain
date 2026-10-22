package runtime

import (
	"bytes"
	"path/filepath"
	"reflect"
	"testing"
	"time"

	"github.com/project-horizon/horizon-core/services/ai/knowledge"
)

func visualEvidence(at time.Time, id, parent string, direction float64, supported bool) SensorObservation {
	timing := int64(time.Second)
	sensor := SensorObservation{Modality: "vision", Source: "visual-evidence-camera", Timestamp: at, SampleID: id, Confidence: .5,
		Provenance: SensorProvenance{Adapter: "browser-light-motion-v2", CalibrationID: "uncalibrated-relative-v1", ParentSampleIDs: []string{parent}},
		Fields:     []SensorField{{FieldID: "patch-0:rgb", GroupID: "patch-0", Kind: "light.rgb-v1", Values: []float64{.4, -.2, .1}, Coordinates: []float64{-1, -1, -.5, -.5}}}}
	quality := []float64{0, 1, 0, 1, 1, 0, 0}
	if supported {
		quality = []float64{.3, 0, 1, 0, .5, 1, 1}
	}
	for _, factor := range []struct {
		kind   string
		values []float64
	}{{"quality", quality}, {"displacement", []float64{direction, 0}}, {"relative", []float64{direction / 2, 0}}} {
		if !supported && factor.kind != "quality" {
			continue
		}
		kind := "light.flow." + factor.kind + ".r6px.120x120-v1"
		sensor.Fields = append(sensor.Fields, SensorField{FieldID: "patch-0:" + kind, GroupID: "patch-0", Kind: kind, Values: factor.values, Coordinates: []float64{-1, -1, -.5, -.5}, TemporalOffsetNS: -timing, TemporalScaleNS: timing})
	}
	return sensor
}

func TestVisualEvidenceReplayRetainsChangingMotionAndUnknownCorrespondence(t *testing.T) {
	journalPath := filepath.Join(t.TempDir(), "visual.jsonl")
	journal, err := OpenEventLog(journalPath)
	if err != nil {
		t.Fatal(err)
	}
	defer journal.Close()
	rt := NewBrainRuntime(nil)
	rt.SetEventLog(journal)
	at := time.Date(2026, 10, 19, 8, 0, 0, 0, time.UTC)
	checkpoint := filepath.Join(t.TempDir(), "checkpoint.json")
	var initial, reversed CognitiveInterpretation
	for i, direction := range []float64{.5, -.5, 0} {
		id, parent := []string{"flow-1", "flow-2", "flow-3"}[i], []string{"frame-0", "flow-1", "flow-2"}[i]
		result, err := rt.ProcessSensorObservation(visualEvidence(at.Add(time.Duration(i)*time.Second), id, parent, direction, i != 2))
		if err != nil {
			t.Fatal(err)
		}
		if i == 0 {
			initial = result
			if err := rt.Checkpoint(checkpoint); err != nil {
				t.Fatal(err)
			}
		} else if i == 1 {
			reversed = result
		} else if len(result.SensorBindings) != 2 {
			t.Fatal("unknown correspondence fabricated a zero-motion binding")
		}
	}
	if !reflect.DeepEqual(initial.SensorBindings[0].Population, reversed.SensorBindings[0].Population) || reflect.DeepEqual(initial.SensorBindings[2].Population, reversed.SensorBindings[2].Population) {
		t.Fatal("motion evidence reversal lost separation or recreated unchanged light state")
	}
	want, err := rt.brain.CanonicalJSON()
	if err != nil {
		t.Fatal(err)
	}
	events, err := ReadEventLog(journalPath)
	if err != nil || len(events) != 3 {
		t.Fatalf("visual history: events=%d err=%v", len(events), err)
	}
	genesis := NewBrainRuntime(nil)
	if err := ReplayEventLog(genesis, events); err != nil {
		t.Fatal(err)
	}
	restored, err := RebuildRuntimeFromCheckpointAndJournal(journal, checkpoint)
	if err != nil {
		t.Fatal(err)
	}
	for _, candidate := range []*BrainRuntime{genesis, restored} {
		got, err := candidate.brain.CanonicalJSON()
		if err != nil || !bytes.Equal(got, want) {
			t.Fatal("motion/uncertainty evidence diverged during replay or checkpoint recovery")
		}
		trace, found := candidate.brain.LastSensoryTrace("visual-evidence-camera", "vision")
		if !found || len(trace.Fields) != 2 || trace.Fields[1].Values[5] != 0 || trace.ParentSampleIDs[0] != "flow-2" {
			t.Fatal("recovery lost unknown correspondence or its causal parent")
		}
	}
	continued := visualEvidence(at.Add(3*time.Second), "flow-4", "flow-3", .5, true)
	beforeTransitions := len(restored.brain.BrainState.ExperienceTraces)
	result, err := restored.ProcessSensorObservation(continued)
	if err != nil {
		t.Fatal(err)
	}
	if result.State.Sequence != 4 || !reflect.DeepEqual(initial.SensorBindings[2].Population, result.SensorBindings[2].Population) {
		t.Fatal("recovered sensor could not reuse an earlier supported motion factor")
	}
	for _, binding := range result.SensorBindings[1:] {
		if binding.TemporalOffsetNS != -int64(time.Second) || binding.TemporalScaleNS != int64(time.Second) || len(binding.Population) != 4 || binding.Population[0] == knowledge.NodeID(0) {
			t.Fatal("captured motion timing or neural binding was lost")
		}
	}
	for _, trace := range restored.brain.BrainState.ExperienceTraces[beforeTransitions:] {
		if reflect.DeepEqual(trace.PreviousPopulation, initial.SensorBindings[2].Population) || reflect.DeepEqual(trace.PreviousPopulation, reversed.SensorBindings[2].Population) {
			t.Fatal("unsupported intermediate observation fabricated uninterrupted motion continuity")
		}
	}
}
