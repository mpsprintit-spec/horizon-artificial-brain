package runtime

import (
	"encoding/json"
	"github.com/project-horizon/horizon-core/services/ai/knowledge"
	"os"
	"path/filepath"
	"reflect"
	"testing"
	"time"
)

func TestStableProjectionPreservesActualPR36ReplayAndNextHistoricalThought(t *testing.T) {
	const fixture = "testdata/evidence_uncertainty_v1"
	data, err := os.ReadFile(filepath.Join(fixture, "reference.json"))
	if err != nil {
		t.Fatal(err)
	}
	var ref struct {
		Commit   string          `json:"commit"`
		Checksum string          `json:"checksum"`
		Next     CognitiveOutput `json:"next"`
		After    string          `json:"after_next_checksum"`
	}
	if err = json.Unmarshal(data, &ref); err != nil {
		t.Fatal(err)
	}
	if ref.Commit != "2b5cca6891f43af8d57eda391b53b8fd300e5edf" {
		t.Fatal("wrong fixture provenance")
	}
	data, err = os.ReadFile(filepath.Join(fixture, "commands.jsonl"))
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), "commands.jsonl")
	if err = os.WriteFile(path, data, 0600); err != nil {
		t.Fatal(err)
	}
	log, err := OpenEventLog(path)
	if err != nil {
		t.Fatal(err)
	}
	defer log.Close()
	genesis, err := RebuildRuntimeFromCommittedJournal(log, knowledge.NewBrain)
	if err != nil {
		t.Fatal(err)
	}
	suffix, err := RebuildRuntimeFromCheckpointAndJournal(log, filepath.Join(fixture, "prefix-checkpoint.json"))
	if err != nil {
		t.Fatal(err)
	}
	checksum := func(r *BrainRuntime) string {
		s, e := readCheckpointForTest(orderedSave(t, r))
		if e != nil {
			t.Fatal(e)
		}
		s.SchemaVersion = observedFeedbackBrainSnapshotSchemaVersion
		sum, e := checkpointChecksumV7(s)
		if e != nil {
			t.Fatal(e)
		}
		return sum
	}
	for _, r := range []*BrainRuntime{genesis, suffix} {
		if r.dynamicsEncoding != knowledge.CurrentDynamics || checksum(r) != ref.Checksum {
			t.Fatal("PR36 complete historical state or serving default changed")
		}
		r.SetClock(NewFixedClock(time.Date(2026, 10, 19, 11, 0, 0, 0, time.UTC)))
		r.dynamicsEncoding = knowledge.EvidenceUncertaintyDynamics
		next, err := r.CognitiveThink(1)
		r.dynamicsEncoding = knowledge.CurrentDynamics
		if err != nil {
			t.Fatal(err)
		}
		if !reflect.DeepEqual(next, ref.Next) || checksum(r) != ref.After {
			t.Fatal("PR36 historical next thought/state changed")
		}
	}
}

func TestStableProjectionGenesisSuffixNoiseAndNextObservation(t *testing.T) {
	r := NewBrainRuntime(knowledge.NewBrain())
	at := time.Unix(100, 0).UTC()
	r.SetClock(NewFixedClock(at))
	log := openKnowledgeJournal(t)
	r.SetEventLog(log)
	if _, err := r.GroundVectorAt(knowledge.NewNeuralVector([]float64{.1, .3, .7}), at); err != nil {
		t.Fatal(err)
	}
	var prefix string
	for i := 0; i < 12; i++ {
		v := []float64{.2, .4, 0, 0}
		if i%2 == 1 {
			v = []float64{.8, 0, .4, 0}
		}
		if i >= 4 {
			v[3] = .005
		}
		if _, err := r.ProcessSensorObservation(stableSensor("observed-"+string(rune('a'+i)), at.Add(time.Duration(i+1)*time.Second), v, .02)); err != nil {
			t.Fatal(err)
		}
		if i == 5 {
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
	for _, other := range []*BrainRuntime{genesis, suffix, standalone} {
		assertCompleteContinuation(t, r, other)
	}
	events, err := ReadEventLog(log.file.Name())
	if err != nil {
		t.Fatal(err)
	}
	for _, event := range events {
		if event.DynamicsEncoding != knowledge.StableProjectionDynamics {
			t.Fatal("new projection policy was not versioned, including numeric grounding")
		}
	}
	r.SetEventLog(nil)
	var want CognitiveOutput
	var wantPopulation []knowledge.NodeID
	for i, candidate := range []*BrainRuntime{r, genesis, suffix, standalone} {
		candidate.SetClock(NewFixedClock(at.Add(13 * time.Second)))
		out, err := candidate.CognitiveThink(1)
		if err != nil {
			t.Fatal(err)
		}
		if i == 0 {
			want = out
		} else if !reflect.DeepEqual(out, want) {
			t.Fatal("next thought/readout changed after projection recovery")
		}
		result, err := candidate.ProcessSensorObservation(stableSensor("next-sensor", at.Add(14*time.Second), []float64{.2, .4, 0, -.005}, .02))
		if err != nil {
			t.Fatal(err)
		}
		population := result.GroundedRepresentations[0].Population
		if i == 0 {
			wantPopulation = population
		} else if !reflect.DeepEqual(population, wantPopulation) {
			t.Fatal("next noisy observation recruited different membership after restart")
		}
	}
	for _, other := range []*BrainRuntime{genesis, suffix, standalone} {
		assertCompleteContinuation(t, r, other)
	}
}
