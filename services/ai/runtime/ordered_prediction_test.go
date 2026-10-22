package runtime

import (
	"fmt"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/project-horizon/horizon-core/services/ai/knowledge"
	"github.com/project-horizon/horizon-core/services/ai/learning"
)

func orderedSave(t *testing.T, r *BrainRuntime) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "trained.json")
	if err := r.Checkpoint(path); err != nil {
		t.Fatal(err)
	}
	return path
}

func orderedMargin(r *BrainRuntime, a, b string) float64 {
	prediction := r.activation.PredictionSnapshot().State
	return prediction[r.brain.Registry.Get(a).ID] - prediction[r.brain.Registry.Get(b).ID]
}

// These are supervised prediction probes, not an assertion that the generic
// answer ranker understands grammar or transfers unseen actor/role bindings.
func TestOrderedLearnedAuditPredictionsAndCausalControls(t *testing.T) {
	answers := [][2]string{{"rina", "budi"}, {"sari", "doni"}, {"meja", "rak"}, {"kotak", "tas"}, {"budi", "rina"}, {"pintu", "lampu"}}
	for i, pair := range orderedAuditPairs {
		for _, seed := range []int64{7, 17, 47} {
			t.Run(fmt.Sprintf("%s/seed-%d", pair.name, seed), func(t *testing.T) {
				foundation := orderedFixtureCheckpoint(t, seed, pair.a, pair.b, pair.question)
				a, b := answers[i][0], answers[i][1]
				margins := func(path string, ablate bool, separate bool) [2]float64 {
					var out [2]float64
					for side, scene := range []string{pair.a, pair.b} {
						r := orderedFixtureClone(t, path)
						if ablate {
							r.brain.Patterns = knowledge.NewPatternIndex()
						}
						if separate {
							_, _, err := NewCognitiveOrchestrator(r).ProcessObservation(Event{ID: "context", Timestamp: time.Unix(101, 0), Cycles: 1}, ObservationInput{Source: "audit", Modality: "text", Tokens: strings.Fields(scene)}, nil)
							if err != nil {
								t.Fatal(err)
							}
							// Resume between context and question to test persisted recurrent context.
							r = orderedFixtureClone(t, orderedSave(t, r))
							orderedAuditObservation(t, r, "", pair.question)
						} else {
							orderedAuditObservation(t, r, scene, pair.question)
						}
						out[side] = orderedMargin(r, a, b)
					}
					return out
				}
				untrained := margins(foundation, false, false)
				var trainedPath string
				var trained, shuffled [2]float64
				for shuffle := 0; shuffle < 2; shuffle++ {
					r := orderedFixtureClone(t, foundation)
					scenes := []string{pair.a, pair.b}
					labels := []string{a, b}
					if shuffle == 1 {
						labels[0], labels[1] = labels[1], labels[0]
					}
					for exposure := 0; exposure < 6; exposure++ {
						for order := 0; order < 2; order++ {
							side := (order + exposure + int(seed)) % 2
							_, err := r.LearnExperience(learning.Experience{ContextSequence: strings.Fields(scenes[side]), Sequence: append(strings.Fields(pair.question), labels[side]), Weight: .9, Confidence: .9, Source: "teacher", ExperienceID: fmt.Sprintf("%d-%d", exposure, side)}, time.Unix(100, 0))
							if err != nil {
								t.Fatal(err)
							}
						}
					}
					path := orderedSave(t, r)
					if shuffle == 0 {
						trainedPath = path
						trained = margins(path, false, false)
					} else {
						shuffled = margins(path, false, false)
					}
				}
				ablated := margins(trainedPath, true, false)
				separate := margins(trainedPath, false, true)
				if trained[0] < .2 || trained[1] > -.2 || separate[0] < .2 || separate[1] > -.2 {
					t.Fatalf("learned predictions do not follow ordered context: together=%v separate=%v", trained, separate)
				}
				if shuffled[0] > -.2 || shuffled[1] < .2 {
					t.Fatalf("prediction did not follow shuffled supervision: %v", shuffled)
				}
				effect := trained[0] - trained[1]
				if effect <= ablated[0]-ablated[1]+.1 || effect <= untrained[0]-untrained[1]+.1 {
					t.Fatalf("conditional advantage survives ablation or predates training: trained=%v ablated=%v untrained=%v", trained, ablated, untrained)
				}
			})
		}
	}
}

func TestOrderedRepetitionAndNovelPrefixComposition(t *testing.T) {
	foundation := orderedFixtureCheckpoint(t, 4, "na na lu end left lu na na end right novel")
	r := orderedFixtureClone(t, foundation)
	for _, sequence := range []string{"na na lu end left", "na lu na end right"} {
		if _, err := r.LearnExperience(learning.Experience{Sequence: strings.Fields(sequence), Weight: .9, Confidence: .9}, time.Unix(100, 0)); err != nil {
			t.Fatal(err)
		}
	}
	path := orderedSave(t, r)
	for i, cue := range []string{"na na lu end", "na lu na end", "novel na na lu end", "novel na lu na end"} {
		clone := orderedFixtureClone(t, path)
		orderedAuditObservation(t, clone, "", cue)
		margin := orderedMargin(clone, "left", "right")
		if i%2 == 0 && margin < .2 || i%2 == 1 && margin > -.2 {
			t.Fatalf("repetition or novel prefix lost: cue=%q margin=%v", cue, margin)
		}
	}
	// Equal outcomes must not cause member-set deduplication of distinct order.
	for _, sequence := range []string{"na na lu end left", "na lu na end left"} {
		if _, err := r.LearnExperience(learning.Experience{Sequence: strings.Fields(sequence)}, time.Unix(100, 0)); err != nil {
			t.Fatal(err)
		}
	}
	if len(r.brain.Patterns.All()) != 3 {
		t.Fatal("different repeated traces with the same result were merged")
	}
}

func TestOrderedContextAndDataStreamsConditionDifferentPredictions(t *testing.T) {
	foundation := orderedFixtureCheckpoint(t, 3, "na lu cue left right")
	r := orderedFixtureClone(t, foundation)
	for i, groups := range [][2]string{{"na lu", "lu na"}, {"lu na", "na lu"}} {
		label := []string{"left", "right"}[i]
		_, err := r.LearnExperience(learning.Experience{ContextSequence: strings.Fields(groups[0]), DataSequence: strings.Fields(groups[1]), Sequence: []string{"cue", label}, Weight: .9, Confidence: .9}, time.Unix(100, 0))
		if err != nil {
			t.Fatal(err)
		}
	}
	path := orderedSave(t, r)
	for i, groups := range [][2]string{{"na lu", "lu na"}, {"lu na", "na lu"}} {
		clone := orderedFixtureClone(t, path)
		_, _, err := NewCognitiveOrchestrator(clone).ProcessObservation(Event{ID: "streams", Timestamp: time.Unix(101, 0), Cycles: 1}, ObservationInput{Source: "audit", Modality: "text", Tokens: []string{"cue"}, ContextTokens: strings.Fields(groups[0]), DataTokens: strings.Fields(groups[1])}, nil)
		if err != nil {
			t.Fatal(err)
		}
		margin := orderedMargin(clone, "left", "right")
		if i == 0 && margin < .2 || i == 1 && margin > -.2 {
			t.Fatalf("context/data boundaries lost: margin=%v", margin)
		}
	}
}
