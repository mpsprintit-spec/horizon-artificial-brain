package runtime

import (
	"compress/gzip"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"testing"
	"time"

	"github.com/project-horizon/horizon-core/services/ai/knowledge"
	"github.com/project-horizon/horizon-core/services/ai/learning"
)

// These are frozen outputs from the external ISF001 experiment, before this
// readout change. Targets are used only after native processing, for scoring.
type answerReadoutFixture struct {
	Schema           int    `json:"schema"`
	SourceCommit     string `json:"source_commit"`
	CheckpointSHA256 string `json:"checkpoint_sha256"`
	Cases            []struct {
		ID                     string             `json:"id"`
		Split                  string             `json:"split"`
		Events                 []Event            `json:"events"`
		TargetPopulation       []knowledge.NodeID `json:"target_population"`
		Outputs                []CognitiveOutput  `json:"outputs"`
		AfterNeuralSHA256      string             `json:"after_neural_sha256"`
		LegacyAnswerPopulation []knowledge.NodeID `json:"legacy_answer_population"`
	} `json:"cases"`
}

func readGzipFixture(t *testing.T, name string) []byte {
	t.Helper()
	f, err := os.Open(filepath.Join("testdata", "neural_answer_readout", name))
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	r, err := gzip.NewReader(f)
	if err != nil {
		t.Fatal(err)
	}
	defer r.Close()
	b, err := io.ReadAll(r)
	if err != nil {
		t.Fatal(err)
	}
	return b
}

func loadAnswerReadoutFixture(t *testing.T) (answerReadoutFixture, string) {
	t.Helper()
	var fixture answerReadoutFixture
	if err := json.Unmarshal(readGzipFixture(t, "isf001-cases.json.gz"), &fixture); err != nil {
		t.Fatal(err)
	}
	if fixture.Schema != 1 || len(fixture.Cases) != 42 || fixture.SourceCommit != "0b893e905742e27e3c9c4ad2a79c5c4fb4747cc0" {
		t.Fatal("unexpected frozen protocol")
	}
	checkpoint := readGzipFixture(t, "isf001-checkpoint.json.gz")
	digest := sha256.Sum256(checkpoint)
	if hex.EncodeToString(digest[:]) != fixture.CheckpointSHA256 || fixture.CheckpointSHA256 != "1d22349376aca4488f875b8f372fbddb9ea27eab7da0be08ff519e3b7961331b" {
		t.Fatal("frozen checkpoint changed")
	}
	path := filepath.Join(t.TempDir(), "checkpoint.json")
	if err := os.WriteFile(path, checkpoint, 0600); err != nil {
		t.Fatal(err)
	}
	return fixture, path
}

func TestNeuralAnswerReadoutFrozenISF001(t *testing.T) {
	fixture, checkpoint := loadAnswerReadoutFixture(t)
	for _, mode := range []string{"normal", "labels-hidden", "labels-swapped"} {
		t.Run(mode, func(t *testing.T) {
			correct := map[string]int{}
			counts := map[string]int{}
			legacyCorrect := map[string]int{}
			for _, c := range fixture.Cases {
				r := NewBrainRuntime(knowledge.NewBrain())
				if err := r.RestoreCheckpoint(checkpoint); err != nil {
					t.Fatal(err)
				}
				// Isolated synthetic fixture only, never a managed user's Brain.
				annotations := r.brain.SurfaceAnnotationsSnapshot()
				switch mode {
				case "labels-hidden":
					annotations = nil
				case "labels-swapped":
					permutation := map[string]string{"lantai": "bola", "bola": "di", "di": "gelas", "gelas": "lantai"}
					for i := range annotations {
						if label, ok := permutation[annotations[i].Surface]; ok {
							annotations[i].Surface = label
						}
					}
				}
				if err := r.brain.RestoreSurfaceAnnotations(annotations); err != nil {
					t.Fatal(err)
				}
				if len(c.Events) != 2 || len(c.Outputs) != 2 {
					t.Fatalf("incomplete case %s", c.ID)
				}
				var last CognitiveOutput
				for i, event := range c.Events {
					var err error
					last, err = r.CognitiveProcess(event)
					if err != nil {
						t.Fatalf("%s: %v", c.ID, err)
					}
					if !reflect.DeepEqual(last, c.Outputs[i]) {
						t.Fatalf("%s: native dynamics changed at input %d", c.ID, i)
					}
				}
				result, err := r.InterpretCognitive(last, Observation{Source: "controlled-numeric-replay", Modality: "numeric"})
				if err != nil {
					t.Fatal(err)
				}
				assessment := last.PredictionAssessment
				if assessment == nil || !slices.Equal(result.Answer.NodeIDs, assessment.NodeIDs) || result.Answer.Confidence != assessment.Confidence || !reflect.DeepEqual(result.Answer.PredictionAssessment, assessment) {
					t.Fatalf("%s: answer does not expose the native forecast", c.ID)
				}
				if len(assessment.NodeIDs) == 0 && result.Answer.Status != AnswerUninterpreted || len(assessment.NodeIDs) > 0 && result.Answer.Status != AnswerNeuralForecast {
					t.Fatalf("%s: incorrect readout status", c.ID)
				}
				if !slices.Equal(result.State.ActiveNodeIDs, c.LegacyAnswerPopulation) || !slices.Equal(result.Interpretation.RankedNodeIDs, c.LegacyAnswerPopulation) {
					t.Fatalf("%s: diagnostic ranking was lost or reranked", c.ID)
				}
				canonical, err := r.brain.CanonicalJSON()
				if err != nil {
					t.Fatal(err)
				}
				got := sha256.Sum256(canonical)
				if hex.EncodeToString(got[:]) != c.AfterNeuralSHA256 {
					t.Fatalf("%s: interpretation changed the persistent Brain", c.ID)
				}
				if mode == "labels-hidden" && len(r.brain.SurfaceAnnotationsSnapshot()) != 0 {
					t.Fatal("numeric readout reconstructed labels")
				}
				counts[c.Split]++
				if slices.Equal(result.Answer.NodeIDs, c.TargetPopulation) && result.Answer.Confidence >= .5 {
					correct[c.Split]++
				}
				if len(c.LegacyAnswerPopulation) > 0 && slices.Equal(c.LegacyAnswerPopulation[:1], c.TargetPopulation) {
					legacyCorrect[c.Split]++
				}
			}
			wantCounts := map[string]int{"terlatih": 18, "kombinasi-pelaku-baru": 18, "kombinasi-kondisi-baru": 6}
			wantCorrect := map[string]int{"terlatih": 18, "kombinasi-pelaku-baru": 0, "kombinasi-kondisi-baru": 4}
			for _, split := range []string{"terlatih", "kombinasi-pelaku-baru", "kombinasi-kondisi-baru"} {
				count := wantCounts[split]
				if counts[split] != count || correct[split] != wantCorrect[split] || legacyCorrect[split] != 0 {
					t.Fatalf("%s: count=%d new=%d legacy=%d", split, counts[split], correct[split], legacyCorrect[split])
				}
				t.Logf("%s: forecast target with confidence >= .5 = %d/%d; legacy activity top1 = %d/%d", split, correct[split], count, legacyCorrect[split], count)
			}
		})
	}
}

func TestNeuralAnswerReadoutPopulationAndUnknown(t *testing.T) {
	for _, assessment := range []*knowledge.PredictionAssessment{
		nil,
		knowledge.UnknownPredictionAssessment(),
		{Scope: "learned-next-population", NodeIDs: []knowledge.NodeID{7, 9}, Confidence: .1, EvidenceMass: 1, ContextSupport: .2, UnknownProbability: .9},
	} {
		input := Interpretation{BrainIdentity: BrainIdentity, RankedNodeIDs: []knowledge.NodeID{1, 2}, Activations: map[knowledge.NodeID]float64{1: 10, 2: 8}, PredictionAssessment: assessment}
		result := ToCognitiveInterpretation(Observation{}, input)
		if assessment == nil || len(assessment.NodeIDs) == 0 {
			if result.Answer.Status != AnswerUninterpreted || len(result.Answer.NodeIDs) != 0 {
				t.Fatal("unknown forecast fell back to activity")
			}
			continue
		}
		if result.Answer.Status != AnswerNeuralForecast || !slices.Equal(result.Answer.NodeIDs, []knowledge.NodeID{7, 9}) || result.Answer.Confidence != .1 || result.Answer.Uncertainty.Level != .9 {
			t.Fatal("weak population forecast was truncated, thresholded or reranked")
		}
		result.Answer.NodeIDs[0] = 100
		if assessment.NodeIDs[0] != 7 || result.Answer.PredictionAssessment.NodeIDs[0] != 7 {
			t.Fatal("answer aliases the assessment")
		}
		result.Answer.PredictionAssessment.NodeIDs[1] = 200
		if assessment.NodeIDs[1] != 9 || result.Answer.NodeIDs[1] != 9 {
			t.Fatal("assessment aliases the answer or source")
		}
	}
}

func TestNeuralAnswerReadoutCheckpointAndSuffix(t *testing.T) {
	r := NewBrainRuntime(knowledge.NewBrain())
	at := time.Date(2026, 10, 21, 5, 0, 0, 0, time.UTC)
	r.SetClock(NewFixedClock(at))
	log, err := OpenEventLog(filepath.Join(t.TempDir(), "events.jsonl"))
	if err != nil {
		t.Fatal(err)
	}
	defer log.Close()
	r.SetEventLog(log)
	for i := 0; i < 4; i++ {
		if _, err := r.LearnExperience(learning.Experience{Sequence: []string{"naku", "melo", "tiva"}, Weight: .8, Confidence: .8}, at); err != nil {
			t.Fatal(err)
		}
	}
	mid := filepath.Join(t.TempDir(), "mid-query.json")
	if err := r.Checkpoint(mid); err != nil {
		t.Fatal(err)
	}
	event := Event{ID: "query", Timestamp: at.Add(time.Second), Stimulus: []string{"naku", "melo"}, Cycles: 1}
	wantOutput, err := r.CognitiveProcess(event)
	if err != nil {
		t.Fatal(err)
	}
	want, err := r.InterpretCognitive(wantOutput, Observation{})
	if err != nil {
		t.Fatal(err)
	}
	if want.Answer.Status != AnswerNeuralForecast {
		t.Fatal("fixture did not produce a selected forecast")
	}
	// Restoring the pre-query checkpoint must produce the same native output
	// and selected answer when given the query again.
	resumed := NewBrainRuntime(knowledge.NewBrain())
	if err := resumed.RestoreCheckpoint(mid); err != nil {
		t.Fatal(err)
	}
	resumedOutput, err := resumed.CognitiveProcess(event)
	if err != nil || !reflect.DeepEqual(resumedOutput, wantOutput) {
		t.Fatalf("checkpoint changed the query output: %v", err)
	}
	resumedResult, err := resumed.InterpretCognitive(resumedOutput, Observation{})
	if err != nil || !reflect.DeepEqual(resumedResult.Answer, want.Answer) {
		t.Fatalf("checkpoint changed the numeric answer: %v", err)
	}
	recovered, err := RebuildRuntimeFromCheckpointAndJournal(log, mid)
	if err != nil {
		t.Fatal(err)
	}
	assertCompleteContinuation(t, r, recovered)
	if !reflect.DeepEqual(recovered.activation.PredictionAssessment(event.Timestamp, recovered.dynamicsEncoding), want.Answer.PredictionAssessment) {
		t.Fatal("suffix replay lost the selected forecast")
	}
	recovered.SetClock(NewFixedClock(at.Add(2 * time.Second)))
	r.SetClock(NewFixedClock(at.Add(2 * time.Second)))
	wantThought, err := r.CognitiveThink(1)
	if err != nil {
		t.Fatal(err)
	}
	gotThought, err := recovered.CognitiveThink(1)
	if err != nil || !reflect.DeepEqual(wantThought, gotThought) {
		t.Fatalf("readout/recovery changed the next internal thought: %v", err)
	}
	wantNext, err := r.InterpretCognitive(wantThought, Observation{})
	if err != nil {
		t.Fatal(err)
	}
	gotNext, err := recovered.InterpretCognitive(gotThought, Observation{})
	if err != nil || !reflect.DeepEqual(wantNext.Answer, gotNext.Answer) {
		t.Fatalf("next thought lost the numeric readout: %v", err)
	}
}
