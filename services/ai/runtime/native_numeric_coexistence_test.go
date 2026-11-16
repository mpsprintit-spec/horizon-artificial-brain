package runtime

import (
	"bytes"
	"errors"
	"path/filepath"
	"testing"

	k "github.com/project-horizon/horizon-core/services/ai/knowledge"
)

// This is an API/storage compatibility proof, not S1 acquisition. All test
// states are disposable. No special transition predictor is added to serving.
func TestNativeNumericRegistrationCoexistence(t *testing.T) {
	r, ids := nativeFixture(t)
	dir := t.TempDir()
	journal := filepath.Join(dir, "events.jsonl")
	wal, err := OpenEventLog(journal)
	s1Check(t, err)
	defer wal.Close()
	r.SetEventLog(wal)
	cfg := nativeControlConfig(ids)
	nativeApply(t, r, NativeControlCommand{ID: "numeric-native-init", Initialize: &cfg})
	r.SetClock(NewFixedClock(s1Base))
	initial := filepath.Join(dir, "before.json")
	s1Check(t, r.Checkpoint(initial))
	originalPolicy := k.NativeDigest(r.brain.NativeControlSnapshot())
	w, err := r.PlanNeuralWrittenGrounding()
	s1Check(t, err)
	if len(w.Projections) != 0 {
		t.Fatal("expected annotation-free native fixture")
	}
	first := filepath.Join(dir, "before-registration.json")
	s1Check(t, r.ApplyNeuralWrittenGrounding(w, first, s1Base))
	c := k.DefaultCompositionConfig()
	c.Version = k.NeuralCompositionMetricV6
	c.Initialization = "zero"
	p, err := r.PlanNeuralComposition(c)
	s1Check(t, err)
	second := filepath.Join(dir, "before-banks.json")
	s1Check(t, r.ApplyNeuralComposition(p, second, s1Base))
	for _, path := range []string{first, second} {
		if s1Snapshot(t, path).SchemaVersion != nativeCheckpointSchemaVersion {
			t.Fatal("native rollback was downgraded")
		}
	}
	if originalPolicy != k.NativeDigest(r.brain.NativeControlSnapshot()) {
		t.Fatal("numeric registration replaced native policy")
	}
	fit, check := nGrid([]float64{-.8, -.4, .4, .8}, []float64{-.6, .6}), nGrid([]float64{-.7, -.2, .2, .7}, []float64{-.5, .5})
	lesson := nLesson(0, 0, fit, check, func(p nXY) float64 { return p.x })
	lesson.Motor = []float64{0, 0, 0, 0}
	lesson.After = nScene(s1Base)
	_, receipt, err := r.LearnCompositionWithDiagnosis(lesson, s1Base)
	s1Check(t, err)
	if receipt == nil || !receipt.Diagnosis.Committed {
		t.Fatal("existing numeric synaptic learner did not run")
	}
	probe := nInputFixture(0, nXY{.43, -.37}, s1Base)
	read, err := r.ReadComposition(probe, s1Base)
	s1Check(t, err)
	if read.NResolution == nil || read.NResolution.Value < .42 || read.NResolution.Value > .44 {
		t.Fatal("native numeric readout lost continuous evidence")
	}
	final := filepath.Join(dir, "after.json")
	s1Check(t, r.Checkpoint(final))
	expected := continuationBytes(t, r)
	for _, cp := range []string{initial, final} {
		resumed, err := RebuildRuntimeFromCheckpointAndJournal(wal, cp)
		s1Check(t, err)
		if !bytes.Equal(expected, continuationBytes(t, resumed)) {
			t.Fatal("numeric/native mixed suffix replay differs")
		}
	}
	if originalPolicy != k.NativeDigest(r.brain.NativeControlSnapshot()) {
		t.Fatal("numeric acquisition changed native policy")
	}
}

func TestNativeNumericRegistrationRejectsInvalidCoexistence(t *testing.T) {
	for _, mode := range []string{"nonempty-written", "active-written", "unsupported-numeric", "active-numeric"} {
		t.Run(mode, func(t *testing.T) {
			r, ids := nativeFixture(t)
			dir := t.TempDir()
			wal, err := OpenEventLog(filepath.Join(dir, "events.jsonl"))
			s1Check(t, err)
			defer wal.Close()
			r.SetEventLog(wal)
			if mode == "nonempty-written" {
				r.brain.Store("existing-language-binding")
			}
			cfg := nativeControlConfig(ids)
			nativeApply(t, r, NativeControlCommand{ID: "init", Initialize: &cfg})
			w, err := r.PlanNeuralWrittenGrounding()
			s1Check(t, err)
			if mode == "unsupported-numeric" || mode == "active-numeric" {
				s1Check(t, r.ApplyNeuralWrittenGrounding(w, filepath.Join(dir, "before-written.json"), s1Base))
			}
			if mode == "active-written" || mode == "active-numeric" {
				nativeApply(t, r, NativeControlCommand{ID: "begin", Begin: &NativeBegin{EpisodeID: "active", Observation: &NativeObservation{Source: "test", SampleID: "test", State: map[k.NodeID]float64{ids[0]: .5}}}})
			}
			before := continuationBytes(t, r)
			journal, err := ReadEventLog(wal.file.Name())
			s1Check(t, err)
			if mode == "nonempty-written" || mode == "active-written" {
				err = r.ApplyNeuralWrittenGrounding(w, filepath.Join(dir, "forbidden.json"), s1Base)
			} else {
				c := k.DefaultCompositionConfig()
				if mode == "active-numeric" {
					c.Version = k.NeuralCompositionMetricV6
				}
				p, e := r.PlanNeuralComposition(c)
				s1Check(t, e)
				err = r.ApplyNeuralComposition(p, filepath.Join(dir, "forbidden.json"), s1Base)
			}
			if err == nil {
				t.Fatal("unsafe numeric/native coexistence admitted")
			}
			if mode != "unsupported-numeric" && !errors.Is(err, ErrCommandAdmission) {
				t.Fatal("expected native transaction/annotation admission rejection", err)
			}
			after, e := ReadEventLog(wal.file.Name())
			s1Check(t, e)
			if !bytes.Equal(before, continuationBytes(t, r)) || len(after) != len(journal) {
				t.Fatal("rejection mutated neural state or durable journal")
			}
		})
	}
}
