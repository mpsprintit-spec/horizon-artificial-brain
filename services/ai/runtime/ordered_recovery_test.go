package runtime

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/project-horizon/horizon-core/services/ai/knowledge"
	"github.com/project-horizon/horizon-core/services/ai/learning"
)

func TestOrderedLearningAndImplicitContextRecoverFromGenesisAndCheckpointSuffix(t *testing.T) {
	r := NewBrainRuntime(knowledge.NewBrain())
	log := openKnowledgeJournal(t)
	r.SetEventLog(log)
	at := time.Unix(100, 0).UTC()
	for _, example := range [][2]string{{"na lu", "left"}, {"lu na", "right"}} {
		_, err := r.LearnExperience(learning.Experience{ContextSequence: strings.Fields(example[0]), Sequence: []string{"cue", example[1]}, Weight: .9, Confidence: .9}, at)
		if err != nil {
			t.Fatal(err)
		}
	}
	pop, err := r.GroundVectorAt(knowledge.NewNeuralVector([]float64{0, 1, 0}), at)
	if err != nil {
		t.Fatal(err)
	}
	_, err = r.LearnExperience(learning.Experience{NeuralSequence: [][]knowledge.NodeID{pop.Population, pop.Population}}, at)
	if err != nil {
		t.Fatal(err)
	}
	_, _, err = NewCognitiveOrchestrator(r).ProcessObservation(Event{ID: "context", Timestamp: at.Add(time.Second), Cycles: 1}, ObservationInput{Source: "audit", Modality: "text", Tokens: []string{"na", "lu"}}, nil)
	if err != nil {
		t.Fatal(err)
	}
	checkpoint := filepath.Join(t.TempDir(), "prefix.json")
	if err = r.Checkpoint(checkpoint); err != nil {
		t.Fatal(err)
	}
	orderedAuditObservation(t, r, "", "cue")
	_, _, err = r.ThinkAt(1, at.Add(2*time.Second))
	if err != nil {
		t.Fatal(err)
	}
	genesis, err := RebuildRuntimeFromCommittedJournal(log, knowledge.NewBrain)
	if err != nil {
		t.Fatal(err)
	}
	suffix, err := RebuildRuntimeFromCheckpointAndJournal(log, checkpoint)
	if err != nil {
		t.Fatal(err)
	}
	for _, restored := range []*BrainRuntime{genesis, suffix} {
		assertCompleteContinuation(t, r, restored)
		if orderedMargin(restored, "left", "right") < .2 {
			t.Fatal("recovered trace lost conditional prediction")
		}
		// Compare the next full recurrent transition as well as serialized state.
		live := orderedFixtureClone(t, orderedSave(t, r))
		expected, err := live.CognitiveThink(1)
		if err != nil {
			t.Fatal(err)
		}
		restored.SetClock(NewFixedClock(time.Unix(101, 0)))
		got, err := restored.CognitiveThink(1)
		if err != nil {
			t.Fatal(err)
		}
		if !reflect.DeepEqual(expected, got) {
			t.Fatal("recovery changed next ordered recurrent prediction")
		}
		assertCompleteContinuation(t, live, restored)
	}
}

func TestSensorCommitCannotChangeRecordedDynamics(t *testing.T) {
	r := NewBrainRuntime(knowledge.NewBrain())
	log := openKnowledgeJournal(t)
	r.SetEventLog(log)
	if _, err := r.ProcessSensorObservation(testSensorObservation()); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(log.file.Name())
	if err != nil {
		t.Fatal(err)
	}
	lines := strings.Split(strings.TrimSpace(string(data)), "\n")
	if len(lines) != 2 {
		t.Fatal("sensor did not produce intent/commit pair")
	}
	lines[1] = strings.ReplaceAll(lines[1], `"dynamics_encoding":"`+string(knowledge.CurrentDynamics)+`",`, "")
	bad := filepath.Join(t.TempDir(), "bad-sensor.jsonl")
	if err = os.WriteFile(bad, []byte(strings.Join(lines, "\n")+"\n"), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err = ReadEventLog(bad); err == nil {
		t.Fatal("sensor commit changed the intent's dynamics contract")
	}
}
