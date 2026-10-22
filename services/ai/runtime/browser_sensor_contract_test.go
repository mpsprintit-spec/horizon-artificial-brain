package runtime

import (
	"encoding/json"
	"fmt"
	"os"
	"reflect"
	"slices"
	"testing"
	"time"

	"github.com/project-horizon/horizon-core/services/ai/knowledge"
)

func browserMeasurements(t *testing.T) []SensorObservation {
	t.Helper()
	data, err := os.ReadFile("testdata/browser_sensors_v1.json")
	if err != nil {
		t.Fatal(err)
	}
	var sensors []SensorObservation
	if err = json.Unmarshal(data, &sensors); err != nil {
		t.Fatal(err)
	}
	for _, s := range sensors {
		if err = s.Validate(); err != nil {
			t.Fatal(err)
		}
	}
	return sensors
}

func browserBinding(t *testing.T, bindings []knowledge.SensoryFieldBinding, id string) knowledge.SensoryFieldBinding {
	t.Helper()
	for _, b := range bindings {
		if b.FieldID == id {
			return b
		}
	}
	t.Fatalf("missing field %s", id)
	return knowledge.SensoryFieldBinding{}
}

func TestBrowserMeasurementsKeepLocalFactorsCoordinatesAndCaptureMetadata(t *testing.T) {
	r := NewBrainRuntime(knowledge.NewBrain())
	sensors := browserMeasurements(t)
	results := make([]CognitiveInterpretation, len(sensors))
	warm := 0
	for i, s := range sensors {
		out, err := r.ProcessSensorObservation(s)
		if err != nil {
			t.Fatal(err)
		}
		results[i] = out
		if i == 0 {
			warm = len(r.brain.Registry.Nodes())
		}
		if i == 1 && len(r.brain.Registry.Nodes()) != warm {
			t.Fatal("small IMU perturbation recruited new neurons")
		}
		if len(out.SensorBindings) != len(s.Fields) {
			t.Fatal("numeric fields collapsed before cognition")
		}
		for j, field := range s.Fields {
			b := out.SensorBindings[j]
			if b.FieldID != field.FieldID || b.Kind != field.Kind || !slices.Equal(b.Values, field.Values) || !slices.Equal(b.Coordinates, field.Coordinates) || b.TemporalOffsetNS != field.TemporalOffsetNS || b.TemporalScaleNS != field.TemporalScaleNS {
				t.Fatal("capture coordinates/timing were lost")
			}
		}
		trace, ok := r.brain.LastSensoryTrace(s.Source, s.Modality)
		if !ok || trace.Adapter != s.Provenance.Adapter || trace.CalibrationID != s.Provenance.CalibrationID || !trace.Timestamp.Equal(s.Timestamp) || !slices.Equal(trace.ParentSampleIDs, s.Provenance.ParentSampleIDs) {
			t.Fatal("capture provenance did not reach persistent sensory memory")
		}
	}
	first := browserBinding(t, results[0].SensorBindings, "acceleration:x")
	near := browserBinding(t, results[1].SensorBindings, "acceleration:x")
	reversed := browserBinding(t, results[2].SensorBindings, "acceleration:x")
	if !slices.Equal(first.Population, near.Population) || slices.Equal(first.Population, reversed.Population) {
		t.Fatal("small changes lost reuse or sign reversal lost separation")
	}
	for _, field := range sensors[0].Fields[1:] {
		if !slices.Equal(browserBinding(t, results[0].SensorBindings, field.FieldID).Population, browserBinding(t, results[2].SensorBindings, field.FieldID).Population) {
			t.Fatal("one-axis change rebuilt an unchanged physical factor")
		}
	}
	if slices.Equal(browserBinding(t, results[0].SensorBindings, "acceleration:y").Coordinates, first.Coordinates) {
		t.Fatal("physical axis bindings were merged")
	}
	for _, binding := range results[0].SensorBindings {
		if binding.FieldID == "acceleration:z" || binding.FieldID == "rotationRate:gamma" {
			t.Fatal("missing channel became a zero measurement")
		}
	}
	if !slices.Equal(first.Population, browserBinding(t, results[3].SensorBindings, "acceleration:x").Population) || sensors[0].Source == sensors[3].Source {
		t.Fatal("capture restart failed to separate lineage while reusing physical factors")
	}
	for _, kind := range []string{"audio.spectral-shape.125-2000hz-v1", "light.rgb-v1"} {
		for _, p := range r.brain.ProjectionPopulations {
			if p.Scope != nil && p.Scope.Field == kind {
				for _, unit := range p.Units {
					if slices.Contains(first.Population, unit.NodeID) {
						t.Fatal("incompatible modalities shared physical neurons")
					}
				}
			}
		}
	}
	if len(r.brain.SurfaceAnnotationsSnapshot()) != 0 {
		t.Fatal("physical observations invented lexical identities")
	}
}

func TestBrowserMeasurementsRecoverCompleteStateAndNextInput(t *testing.T) {
	sensors := browserMeasurements(t)
	r := NewBrainRuntime(knowledge.NewBrain())
	r.SetClock(NewFixedClock(sensors[0].Timestamp))
	log := openKnowledgeJournal(t)
	r.SetEventLog(log)
	var prefix string
	for i, s := range sensors {
		if _, err := r.ProcessSensorObservation(s); err != nil {
			t.Fatal(err)
		}
		if i == 2 {
			prefix = orderedSave(t, r)
		}
	}
	genesis, err := RebuildRuntimeFromCommittedJournal(log, knowledge.NewBrain)
	if err != nil {
		t.Fatal(err)
	}
	suffix, err := RebuildRuntimeFromCheckpointAndJournal(log, prefix)
	if err != nil {
		t.Fatal(err)
	}
	standalone := orderedFixtureClone(t, orderedSave(t, r))
	r.SetEventLog(nil)
	for _, other := range []*BrainRuntime{genesis, suffix, standalone} {
		assertCompleteContinuation(t, r, other)
	}
	next := sensors[1].Clone()
	next.SampleID = "continued-imu"
	next.Timestamp = sensors[len(sensors)-1].Timestamp.Add(time.Second)
	next.Provenance.ParentSampleIDs = []string{"motion-b"}
	var want CognitiveInterpretation
	for i, candidate := range []*BrainRuntime{r, genesis, suffix, standalone} {
		out, err := candidate.ProcessSensorObservation(next)
		if err != nil {
			t.Fatal(err)
		}
		if i == 0 {
			want = out
		} else if !reflect.DeepEqual(out, want) {
			t.Fatal("next physical input/readout changed after restart")
		}
		candidate.SetClock(NewFixedClock(next.Timestamp.Add(time.Second)))
		if _, err = candidate.CognitiveThink(1); err != nil {
			t.Fatal(err)
		}
	}
	for _, other := range []*BrainRuntime{genesis, suffix, standalone} {
		assertCompleteContinuation(t, r, other)
	}
	events, err := ReadEventLog(log.file.Name())
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range events {
		if e.Event == nil || e.Event.SensorObservation == nil || len(e.Event.Stimulus) > 0 || len(e.Event.SensorObservation.Fields) == 0 {
			t.Fatal("journal substituted a hash/token for physical measurements")
		}
	}
}

func TestBrowserMotionAxisExchangeChangesNeuralInput(t *testing.T) {
	s := browserMeasurements(t)[0]
	r := NewBrainRuntime(knowledge.NewBrain())
	first, err := r.ProcessSensorObservation(s)
	if err != nil {
		t.Fatal(err)
	}
	swapped := s.Clone()
	swapped.SampleID = "swapped-axis"
	swapped.Timestamp = swapped.Timestamp.Add(time.Second)
	swapped.Fields[0].Values[0], swapped.Fields[1].Values[0] = swapped.Fields[1].Values[0], swapped.Fields[0].Values[0]
	second, err := r.ProcessSensorObservation(swapped)
	if err != nil {
		t.Fatal(err)
	}
	a, b := groundedPopulationNodeIDs(first.GroundedRepresentations), groundedPopulationNodeIDs(second.GroundedRepresentations)
	slices.Sort(a)
	slices.Sort(b)
	if slices.Equal(a, b) {
		t.Fatal("axis exchange retained only metadata and lost neural channel identity")
	}
	for _, id := range []string{"acceleration:x", "acceleration:y"} {
		if slices.Equal(browserBinding(t, first.SensorBindings, id).Population, browserBinding(t, second.SensorBindings, id).Population) {
			t.Fatal("changed physical axis did not separate its receptive population")
		}
	}
}

func TestBrowserMotionContinuityPredictsFirstOutcomeFromUnseenPerturbation(t *testing.T) {
	sensors := browserMeasurements(t)
	r := NewBrainRuntime(knowledge.NewBrain())
	at := sensors[0].Timestamp
	// Isolate one physical axis: other modalities/constants cannot supply the
	// successor. A/B and the held-out perturbation come from the real adapter.
	axis := func(template SensorObservation, id string, stamp time.Time) SensorObservation {
		s := template.Clone()
		s.SampleID = id
		s.Source = "controlled-imu"
		s.Timestamp = stamp
		s.Fields = s.Fields[:1]
		s.Provenance.ParentSampleIDs = nil
		return s
	}
	zero := axis(sensors[0], "untrained-zero", at)
	zero.Source = "untrained-control"
	zero.Fields[0].Values = []float64{0}
	control, err := r.ProcessSensorObservation(zero)
	if err != nil {
		t.Fatal(err)
	}
	var source, outcome []knowledge.NodeID
	for i := 0; i < 16; i++ {
		for step, index := range []int{0, 2} {
			s := axis(sensors[index], fmt.Sprintf("train-%d-%d", i, step), at.Add(time.Duration(2*i+step+1)*time.Second))
			s.SessionID = fmt.Sprintf("episode-%d", i)
			out, err := r.ProcessSensorObservation(s)
			if err != nil {
				t.Fatal(err)
			}
			if i == 0 {
				if step == 0 {
					source = slices.Clone(out.SensorBindings[0].Population)
				} else {
					outcome = slices.Clone(out.SensorBindings[0].Population)
				}
			}
		}
	}
	path := orderedSave(t, r)
	probe := orderedFixtureClone(t, path)
	ablated := orderedFixtureClone(t, path)
	heldout := axis(sensors[1], "unseen-perturbation", at.Add(34*time.Second))
	heldout.SessionID = "heldout"
	out, err := probe.ProcessSensorObservation(heldout)
	if err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(out.SensorBindings[0].Population, source) {
		t.Fatal("perturbed input did not recover original source population")
	}
	margin := func(candidate *BrainRuntime) float64 {
		state := candidate.activation.PredictionSnapshot().State
		mean := func(ids []knowledge.NodeID) float64 {
			sum := 0.0
			for _, id := range ids {
				sum += state[id]
			}
			return sum / float64(len(ids))
		}
		return mean(outcome) - mean(control.SensorBindings[0].Population)
	}
	got := margin(probe)
	if got < .02 {
		t.Fatalf("first-outcome prediction advantage absent: %g", got)
	}
	ablated.brain.Lock()
	for _, from := range source {
		for _, to := range outcome {
			delete(ablated.brain.Registry.GetByID(from).Synapses, to)
		}
	}
	ablated.brain.Unlock()
	if _, err = ablated.ProcessSensorObservation(heldout); err != nil {
		t.Fatal(err)
	}
	ablatedMargin := margin(ablated)
	t.Logf("first-outcome margin=%f ablated=%f", got, ablatedMargin)
	if got < ablatedMargin+.02 {
		t.Fatal("prediction advantage survived deletion of learned source/outcome synapses")
	}
}
