package runtime

import (
	"bytes"
	"errors"
	"math"
	"os"
	"path/filepath"
	"reflect"
	"testing"
	"time"
)

func factorizedSensor() SensorObservation {
	return SensorObservation{
		Modality: "vision", Source: "camera-local", SampleID: "frame-1",
		Timestamp: time.Date(2026, 10, 18, 14, 0, 0, 0, time.UTC), Confidence: .5,
		Provenance: SensorProvenance{Adapter: "test-light", CalibrationID: "relative-v1"},
		Fields: []SensorField{
			{FieldID: "left:color", GroupID: "left", Kind: "light.rgb-v1", Values: []float64{.8, -.4, .2}, Coordinates: []float64{-1, -1, 0, 1}},
			{FieldID: "right:color", GroupID: "right", Kind: "light.rgb-v1", Values: []float64{.8, -.4, .2}, Coordinates: []float64{0, -1, 1, 1}},
			{FieldID: "left:motion", GroupID: "left", Kind: "light.temporal-delta-v1", Values: []float64{0}, TemporalOffsetNS: -int64(time.Second), TemporalScaleNS: int64(time.Second)},
		},
	}
}

func TestFactorizedSensorSharesFactorsWithoutMergingLocalBindings(t *testing.T) {
	rt := NewBrainRuntime(nil)
	sensor := factorizedSensor()
	first, err := rt.ProcessSensorObservation(sensor)
	if err != nil {
		t.Fatal(err)
	}
	if len(first.SensorBindings) != 3 || len(first.GroundedRepresentations) != 3 {
		t.Fatal("fields collapsed into one grounding")
	}
	left, right := first.SensorBindings[0], first.SensorBindings[1]
	if !reflect.DeepEqual(left.Population, right.Population) {
		t.Fatal("same light factor did not reuse one population")
	}
	if left.GroupID == right.GroupID || reflect.DeepEqual(left.Coordinates, right.Coordinates) {
		t.Fatal("shared light factor merged spatial bindings")
	}
	changed := sensor.Clone()
	changed.SampleID = "frame-2"
	changed.Timestamp = changed.Timestamp.Add(time.Second)
	changed.Fields[2].Values[0] = .9
	second, err := rt.ProcessSensorObservation(changed)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(left.Population, second.SensorBindings[0].Population) {
		t.Fatal("motion change recreated static light representation")
	}
	if reflect.DeepEqual(first.SensorBindings[2].Population, second.SensorBindings[2].Population) {
		t.Fatal("large motion change did not separate its receptive field")
	}
	trace, ok := rt.brain.LastSensoryTrace(sensor.Source, sensor.Modality)
	if !ok || trace.SampleID != "frame-2" || len(trace.Fields) != 3 {
		t.Fatal("missing persistent local sensor trace")
	}
	trace.Fields[0].Population[0] = 0
	trace.Fields[0].Values[0] = -1
	second.SensorBindings[0].Coordinates[0] = 1
	stored, _ := rt.brain.LastSensoryTrace(sensor.Source, sensor.Modality)
	if stored.Fields[0].Population[0] == 0 || stored.Fields[0].Values[0] != .8 || stored.Fields[0].Coordinates[0] != -1 {
		t.Fatal("inspection aliases live sensor memory")
	}
}

func TestFactorizedSensorCaptureJitterReusesScaleAndKeepsExactTiming(t *testing.T) {
	rt := NewBrainRuntime(nil)
	sensor := factorizedSensor()
	first, err := rt.ProcessSensorObservation(sensor)
	if err != nil {
		t.Fatal(err)
	}
	sensor.SampleID = "jitter-frame"
	sensor.Timestamp = sensor.Timestamp.Add(1050 * time.Millisecond)
	sensor.Fields[2].TemporalScaleNS = int64(1050 * time.Millisecond)
	sensor.Fields[2].TemporalOffsetNS = -sensor.Fields[2].TemporalScaleNS
	second, err := rt.ProcessSensorObservation(sensor)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(first.SensorBindings[2].Population, second.SensorBindings[2].Population) {
		t.Fatal("capture jitter allocated a duplicate temporal factor")
	}
	trace, _ := rt.brain.LastSensoryTrace(sensor.Source, sensor.Modality)
	if trace.Fields[2].TemporalScaleNS != int64(1050*time.Millisecond) || trace.Fields[2].TemporalOffsetNS != -int64(1050*time.Millisecond) {
		t.Fatal("receptive temporal scale replaced exact capture timing")
	}
	sensor.SampleID = "short-scale-frame"
	sensor.Timestamp = sensor.Timestamp.Add(time.Second)
	sensor.Fields[2].TemporalScaleNS = int64(100 * time.Millisecond)
	sensor.Fields[2].TemporalOffsetNS = -sensor.Fields[2].TemporalScaleNS
	third, err := rt.ProcessSensorObservation(sensor)
	if err != nil {
		t.Fatal(err)
	}
	if reflect.DeepEqual(first.SensorBindings[2].Population, third.SensorBindings[2].Population) {
		t.Fatal("different temporal scales lost their distinct receptive fields")
	}
}

func TestFactorizedSensorLearnsAcousticWindowsInCaptureOrder(t *testing.T) {
	rt := NewBrainRuntime(nil)
	sensor := factorizedSensor()
	sensor.Modality, sensor.Source = "audio", "microphone"
	sensor.Fields = []SensorField{
		{FieldID: "late:rms", GroupID: "late", Kind: "audio.rms-v1", Values: []float64{.9}, TemporalOffsetNS: -int64(100 * time.Millisecond), TemporalScaleNS: int64(100 * time.Millisecond)},
		{FieldID: "early:rms", GroupID: "early", Kind: "audio.rms-v1", Values: []float64{-.9}, TemporalOffsetNS: -int64(200 * time.Millisecond), TemporalScaleNS: int64(100 * time.Millisecond)},
	}
	result, err := rt.ProcessSensorObservation(sensor)
	if err != nil {
		t.Fatal(err)
	}
	traces := rt.brain.BrainState.ExperienceTraces
	if len(traces) != 1 || !reflect.DeepEqual(traces[0].PreviousPopulation, result.SensorBindings[1].Population) || !reflect.DeepEqual(traces[0].CurrentPopulation, result.SensorBindings[0].Population) || !traces[0].CurrentAt.After(traces[0].PreviousAt) {
		t.Fatal("temporal learning followed field-array order instead of measured capture order")
	}
}

func TestFactorizedSensorReplayAndCheckpointSuffixPreserveCompleteState(t *testing.T) {
	journal, err := OpenEventLog(filepath.Join(t.TempDir(), "events.jsonl"))
	if err != nil {
		t.Fatal(err)
	}
	defer journal.Close()
	rt := NewBrainRuntime(nil)
	rt.SetEventLog(journal)
	sensor := factorizedSensor()
	if _, err := rt.ProcessSensorObservation(sensor); err != nil {
		t.Fatal(err)
	}
	checkpoint := filepath.Join(t.TempDir(), "checkpoint.json")
	if err := rt.Checkpoint(checkpoint); err != nil {
		t.Fatal(err)
	}
	sensor.SampleID = "frame-2"
	sensor.Timestamp = sensor.Timestamp.Add(time.Second)
	sensor.Fields[2].Values[0] = .9
	if _, err := rt.ProcessSensorObservation(sensor); err != nil {
		t.Fatal(err)
	}
	want, err := rt.brain.CanonicalJSON()
	if err != nil {
		t.Fatal(err)
	}
	events, err := ReadEventLog(journal.file.Name())
	if err != nil || len(events) != 2 {
		t.Fatalf("sensor journal events=%d err=%v", len(events), err)
	}
	for _, event := range events {
		if event.Event == nil || event.Event.SensorObservation == nil || len(event.Event.SensorObservation.Fields) != 3 || len(event.Event.Stimulus) != 0 {
			t.Fatal("journal replaced numeric fields with token input")
		}
	}
	replayed := NewBrainRuntime(nil)
	if err := ReplayEventLog(replayed, events); err != nil {
		t.Fatal(err)
	}
	rebuilt, err := RebuildRuntimeFromCheckpointAndJournal(journal, checkpoint)
	if err != nil {
		t.Fatal(err)
	}
	for _, candidate := range []*BrainRuntime{replayed, rebuilt} {
		got, err := candidate.brain.CanonicalJSON()
		if err != nil {
			t.Fatal(err)
		}
		if !bytes.Equal(want, got) || candidate.LastSequence() != 2 {
			t.Fatal("factorized sensory state diverged after replay/restart")
		}
		trace, ok := candidate.brain.LastSensoryTrace(sensor.Source, sensor.Modality)
		if !ok || trace.SampleID != "frame-2" {
			t.Fatal("recovery lost sensor provenance and binding history")
		}
	}
	sensor.SampleID = "frame-3"
	sensor.Timestamp = sensor.Timestamp.Add(time.Second)
	third, err := rebuilt.ProcessSensorObservation(sensor)
	if err != nil {
		t.Fatal(err)
	}
	if third.State.Sequence != 3 || len(third.SensorBindings) != 3 {
		t.Fatal("recovered brain could not continue sensor learning")
	}
}

func TestFactorizedSensorValidatesAllFieldsBeforeJournalAndMutation(t *testing.T) {
	for _, scenario := range []string{"nan", "out-of-range", "duplicate-id", "mixed-flat", "future-window", "empty-field", "too-many-fields"} {
		t.Run(scenario, func(t *testing.T) {
			journal, err := OpenEventLog(filepath.Join(t.TempDir(), "events.jsonl"))
			if err != nil {
				t.Fatal(err)
			}
			defer journal.Close()
			rt := NewBrainRuntime(nil)
			rt.SetEventLog(journal)
			before, _ := rt.brain.CanonicalJSON()
			sensor := factorizedSensor()
			switch scenario {
			case "nan":
				sensor.Fields[2].Values[0] = math.NaN()
			case "out-of-range":
				sensor.Fields[2].Values[0] = 2
			case "duplicate-id":
				sensor.Fields[2].FieldID = sensor.Fields[0].FieldID
			case "mixed-flat":
				sensor.ChannelStates = []float64{.1}
			case "future-window":
				sensor.Fields[2].TemporalOffsetNS = 0
			case "empty-field":
				sensor.Fields[2].Values = nil
			case "too-many-fields":
				sensor.Fields = make([]SensorField, 129)
			}
			if _, err := rt.ProcessSensorObservation(sensor); !errors.Is(err, ErrInvalidSensorObservation) {
				t.Fatalf("invalid state accepted: %v", err)
			}
			after, _ := rt.brain.CanonicalJSON()
			raw, _ := os.ReadFile(journal.file.Name())
			if !bytes.Equal(before, after) || len(raw) != 0 || rt.LastSequence() != 0 || rt.RecoveryRequired() {
				t.Fatal("invalid field partially changed committed state")
			}
		})
	}
}

func TestFactorizedSensorUncertainCommitStopsBeforePublishing(t *testing.T) {
	for _, failType := range []string{EventTypeSensorIntent, EventTypeSensorCommit} {
		t.Run(failType, func(t *testing.T) {
			journal, err := OpenEventLog(filepath.Join(t.TempDir(), "events.jsonl"))
			if err != nil {
				t.Fatal(err)
			}
			defer journal.Close()
			journal.beforeAppend = func(event LoggedEvent) error {
				if event.Type == failType {
					return errors.New("injected write uncertainty")
				}
				return nil
			}
			rt := NewBrainRuntime(nil)
			rt.SetEventLog(journal)
			hub := rt.Telemetry()
			sub := hub.Subscribe()
			defer hub.Unsubscribe(sub)
			before, _ := rt.brain.CanonicalJSON()
			if _, err := rt.ProcessSensorObservation(factorizedSensor()); !errors.Is(err, ErrRuntimeRecoveryRequired) {
				t.Fatalf("uncertain transaction remained available: %v", err)
			}
			if failType == EventTypeSensorIntent {
				after, _ := rt.brain.CanonicalJSON()
				if !bytes.Equal(before, after) {
					t.Fatal("failed intent mutated physical factors")
				}
			}
			if _, err := rt.MonitorSnapshot(); !errors.Is(err, ErrRuntimeRecoveryRequired) {
				t.Fatal("uncertain field projection was exposed to monitor")
			}
			select {
			case _, open := <-sub.C():
				if open {
					t.Fatal("half-applied field transaction published telemetry")
				}
			default:
				t.Fatal("uncertain commit did not close telemetry")
			}
		})
	}
}
