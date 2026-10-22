package runtime

import (
	"encoding/json"
	"github.com/project-horizon/horizon-core/services/ai/knowledge"
	"github.com/project-horizon/horizon-core/services/ai/learning"
	"os"
	"path/filepath"
	"reflect"
	"testing"
	"time"
)

func TestEvidenceUncertaintyPreservesActualPR35ReplayAndNextHistoricalThought(t *testing.T) {
	const fixture = "testdata/observed_feedback_v1"
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
	if ref.Commit != "e8a7b03682c6cf10177efb068194ece6300f0a55" {
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
			t.Fatal("PR35 complete historical state or serving default changed")
		}
		r.SetClock(NewFixedClock(time.Date(2026, 10, 19, 11, 0, 0, 0, time.UTC)))
		r.dynamicsEncoding = knowledge.ObservedFeedbackDynamics
		next, err := r.CognitiveThink(1)
		r.dynamicsEncoding = knowledge.CurrentDynamics
		if err != nil {
			t.Fatal(err)
		}
		if !reflect.DeepEqual(next, ref.Next) || checksum(r) != ref.After {
			t.Fatal("PR35 historical next thought/state changed")
		}
	}
}

func TestEvidenceUncertaintyInquiryGenesisSuffixAndNextReadout(t *testing.T) {
	r := NewBrainRuntime(knowledge.NewBrain())
	at := time.Date(2026, 10, 20, 5, 0, 0, 0, time.UTC)
	r.SetClock(NewFixedClock(at))
	log := openKnowledgeJournal(t)
	r.SetEventLog(log)
	for i := 0; i < 12; i++ {
		for _, target := range []string{"b", "c"} {
			if _, err := r.LearnExperience(learning.Experience{Sequence: []string{"a", target}, Weight: .8, Confidence: .8}, at); err != nil {
				t.Fatal(err)
			}
		}
	}
	orch := NewCognitiveOrchestrator(r)
	cue, _, err := orch.ProcessObservation(Event{ID: "cue", EpisodeID: "one", Timestamp: at, Cycles: 1}, ObservationInput{Source: "probe", Modality: "text", Tokens: []string{"a"}}, nil)
	if err != nil {
		t.Fatal(err)
	}
	request, err := r.BuildInquiryRequest(*cue.InquiryAgenda)
	if err != nil {
		t.Fatal(err)
	}
	prefix := orderedSave(t, r)
	r.SetClock(NewFixedClock(at))
	if _, err = r.CognitiveThink(2); err != nil {
		t.Fatal(err)
	}
	if _, _, err = orch.ProcessInquiryObservation(request, Event{ID: "result", EpisodeID: "one", Timestamp: at.Add(time.Second), Cycles: 1}, ObservationInput{Source: "probe", Modality: "text", Tokens: []string{"b"}}, nil); err != nil {
		t.Fatal(err)
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
	want := r.activation.PredictionAssessment(at.Add(time.Second), knowledge.CurrentDynamics)
	for _, other := range []*BrainRuntime{genesis, suffix, standalone} {
		if !reflect.DeepEqual(want, other.activation.PredictionAssessment(at.Add(time.Second), knowledge.CurrentDynamics)) {
			t.Fatal("derived forecast differs after recovery")
		}
	}
	events, err := ReadEventLog(log.file.Name())
	if err != nil {
		t.Fatal(err)
	}
	for _, event := range events {
		if event.DynamicsEncoding != knowledge.CurrentDynamics {
			t.Fatal("new readout/inquiry policy lost journal version")
		}
	}
	last := events[len(events)-1]
	if last.ObservationCommand == nil || last.ObservationCommand.Inquiry == nil || !reflect.DeepEqual(last.ObservationCommand.Inquiry.PredictionAssessment, request.PredictionAssessment) {
		t.Fatal("complete command lost captured inquiry forecast")
	}
	r.SetEventLog(nil)
	var next CognitiveOutput
	for i, candidate := range []*BrainRuntime{r, genesis, suffix, standalone} {
		candidate.SetClock(NewFixedClock(at.Add(2 * time.Second)))
		out, err := candidate.CognitiveThink(1)
		if err != nil {
			t.Fatal(err)
		}
		if i == 0 {
			next = out
		} else if !reflect.DeepEqual(out, next) {
			t.Fatal("next thought/assessment differs after recovery")
		}
		contextEpisode(t, candidate, "next-cue", "a", "next", false, at.Add(3*time.Second))
	}
	for _, other := range []*BrainRuntime{genesis, suffix, standalone} {
		assertCompleteContinuation(t, r, other)
		if !reflect.DeepEqual(r.activation.PredictionAssessment(at.Add(3*time.Second), knowledge.CurrentDynamics), other.activation.PredictionAssessment(at.Add(3*time.Second), knowledge.CurrentDynamics)) {
			t.Fatal("next learned readout differs after restart")
		}
	}
}
