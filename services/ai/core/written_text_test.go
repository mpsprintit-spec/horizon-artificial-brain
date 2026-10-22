package core

import (
	"bytes"
	"path/filepath"
	"reflect"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/project-horizon/horizon-core/services/ai/knowledge"
	"github.com/project-horizon/horizon-core/services/ai/learning"
	"github.com/project-horizon/horizon-core/services/ai/perception"
	"github.com/project-horizon/horizon-core/services/ai/runtime"
)

func writtenTrainingFixture(t *testing.T) (*HorizonEngine, map[string]knowledge.NodeID, *runtime.EventLog) {
	t.Helper()
	r := runtime.NewBrainRuntime(knowledge.NewBrain())
	owner, err := runtime.NewRuntimeSupervisor(r)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = owner.Close() })
	h := &HorizonEngine{supervisor: owner, Perception: perception.UserInputPerception{}}
	log, err := runtime.OpenEventLog(filepath.Join(t.TempDir(), "events.jsonl"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = log.Close() })
	if err := owner.AttachEventLog(log); err != nil {
		t.Fatal(err)
	}
	at := time.Now().UTC().Add(-time.Second)
	for _, sequence := range [][]string{{"naku", "melo", "tiva"}, {"sopo", "keri", "luma"}} {
		for i := 0; i < 3; i++ {
			err := owner.WithMutation(func(active *runtime.BrainRuntime) error {
				_, err := active.LearnExperience(learning.Experience{Sequence: sequence, Weight: .8, Confidence: .8, Source: "same-teacher", Modality: "language", IndependenceGroup: "same-teacher", Reliability: .9}, at)
				return err
			})
			if err != nil {
				t.Fatal(err)
			}
		}
	}
	graph := inspectForTest(t, h).Graph
	ids := make(map[string]knowledge.NodeID)
	for _, token := range []string{"naku", "melo", "tiva", "sopo", "keri", "luma"} {
		vector := knowledge.EncodeObservation(token, "language").Values
		for _, unit := range graph.NeuralUnits() {
			if reflect.DeepEqual(vector, unit.Representation) {
				ids[token] = unit.ID
				break
			}
		}
		if ids[token] == 0 {
			t.Fatalf("trained representation %q missing", token)
		}
	}
	return h, ids, log
}

func TestWrittenTextFacadeRetrievesLearnedPrefixesWithoutTargetInPrompt(t *testing.T) {
	for _, test := range []struct{ prompt, target string }{{"naku melo", "tiva"}, {"sopo keri", "luma"}, {"  NAKU   MeLo  ", "tiva"}} {
		t.Run(test.prompt, func(t *testing.T) {
			h, ids, _ := writtenTrainingFixture(t)
			if strings.Contains(strings.ToLower(test.prompt), test.target) {
				t.Fatal("target leaked into prompt")
			}
			out, labels, err := h.PulseWithLabels(test.prompt)
			if err != nil {
				t.Fatal(err)
			}
			for _, grounded := range out.GroundedRepresentations {
				id := ids[strings.ToLower(strings.TrimSpace(grounded.Token))]
				if id == 0 || !slices.Contains(grounded.Population, id) {
					t.Errorf("text %q did not use its trained language representation %d: %v", grounded.Token, id, grounded.Population)
				}
			}
			if out.State.Activations[ids[test.target]] <= 0 || !slices.Contains(out.Answer.NodeIDs, ids[test.target]) {
				t.Errorf("held-out target %q absent from neural readout (activation=%g)", test.target, out.State.Activations[ids[test.target]])
			}
			if !slices.Contains(labels, test.target) {
				t.Errorf("trained target %q has no readable label: %v", test.target, labels)
			}
		})
	}
}

func TestWrittenTextSourceAliasesAndCheckpointRestartReuseTrainedUnits(t *testing.T) {
	h, ids, log := writtenTrainingFixture(t)
	checkpoint := filepath.Join(t.TempDir(), "trained.json")
	if err := h.Supervisor().WithMutation(func(r *runtime.BrainRuntime) error { return r.Checkpoint(checkpoint) }); err != nil {
		t.Fatal(err)
	}
	for i, modality := range []string{"text", "language", " TEXT ", "web-search"} {
		h.Perception = fixedPerception{signal: perception.PerceptionSignal{Source: "independent-transport-" + modality, Modality: modality, Tokens: []string{"NAKU", "melo"}, ObservedAt: time.Now().UTC().Add(time.Duration(i) * time.Millisecond)}}
		out, labels, err := h.PulseWithLabels("naku melo")
		if err != nil {
			t.Fatal(err)
		}
		if len(out.GroundedRepresentations) != 2 || out.GroundedRepresentations[0].NodeID != ids["naku"] || out.GroundedRepresentations[1].NodeID != ids["melo"] || !slices.Contains(labels, "tiva") {
			t.Fatalf("%s did not retrieve the trained prefix and held-out target", modality)
		}
	}
	before := inspectForTest(t, h)
	if err := h.Supervisor().RecoverFromCheckpoint(log, checkpoint); err != nil {
		t.Fatal(err)
	}
	after := inspectForTest(t, h)
	if before.Revision != after.Revision || !bytes.Equal(before.Graph.CanonicalBytes(), after.Graph.CanonicalBytes()) {
		t.Fatal("written command replay changed neural state")
	}
	h.Perception = perception.UserInputPerception{}
	out, labels, err := h.PulseWithLabels("SOPO KERI")
	if err != nil {
		t.Fatal(err)
	}
	if out.GroundedRepresentations[0].NodeID != ids["sopo"] || out.GroundedRepresentations[1].NodeID != ids["keri"] || !slices.Contains(out.Answer.NodeIDs, ids["luma"]) || !slices.Contains(labels, "luma") {
		t.Fatal("production facade lost learned text after restart")
	}
}
