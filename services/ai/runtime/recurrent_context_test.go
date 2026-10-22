package runtime

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/project-horizon/horizon-core/services/ai/learning"
)

func contextObservation(t *testing.T, r *BrainRuntime, id, text string, at time.Time) {
	t.Helper()
	_, _, err := NewCognitiveOrchestrator(r).ProcessObservation(Event{ID: id, Timestamp: at, Cycles: 1}, ObservationInput{Source: "context-probe", Modality: "text", Tokens: strings.Fields(text)}, nil)
	if err != nil {
		t.Fatal(err)
	}
}

func contextTraining(t *testing.T, seed int64) string {
	t.Helper()
	foundation := orderedFixtureCheckpoint(t, seed, "xa ya cue left right distractor other")
	r := orderedFixtureClone(t, foundation)
	for exposure := 0; exposure < 5; exposure++ {
		for i := 0; i < 2; i++ {
			side := (exposure + i + int(seed)) % 2
			sequence := []string{"xa cue left", "ya cue right"}[side]
			if _, err := r.LearnExperience(learning.Experience{Sequence: strings.Fields(sequence), Weight: .9, Confidence: .9, ExperienceID: fmt.Sprintf("%d-%d", exposure, side)}, time.Unix(100, 0)); err != nil {
				t.Fatal(err)
			}
		}
	}
	return orderedSave(t, r)
}

func TestRecurrentContextDistinguishesLearnedPrefixesAcrossObservations(t *testing.T) {
	for _, seed := range []int64{7, 17, 47} {
		checkpoint := contextTraining(t, seed)
		for _, distractors := range [][]string{nil, {"distractor", "other"}} {
			for side, context := range []string{"xa", "ya"} {
				t.Run(fmt.Sprintf("seed-%d/noise-%d/%s", seed, len(distractors), context), func(t *testing.T) {
					r := orderedFixtureClone(t, checkpoint)
					at := time.Unix(101, 0).UTC()
					contextObservation(t, r, "context", context, at)
					for i, noise := range distractors {
						contextObservation(t, r, fmt.Sprintf("noise-%d", i), noise, at.Add(time.Duration(i+1)*time.Second))
					}
					contextObservation(t, r, "question", "cue", at.Add(time.Duration(len(distractors)+1)*time.Second))
					margin := orderedMargin(r, "left", "right")
					if side == 0 && margin < .2 || side == 1 && margin > -.2 {
						t.Fatalf("X,A and Y,A do not predict their different learned outcomes: margin=%g", margin)
					}
				})
			}
		}
	}
}

func TestRecurrentContextReconstructsMultiObservationPrefixWithInterruption(t *testing.T) {
	foundation := orderedFixtureCheckpoint(t, 17, "xa ya middle cue left right distractor")
	r := orderedFixtureClone(t, foundation)
	for _, sequence := range []string{"xa middle cue left", "ya middle cue right"} {
		for i := 0; i < 5; i++ {
			if _, err := r.LearnExperience(learning.Experience{Sequence: strings.Fields(sequence), Weight: .9, Confidence: .9}, time.Unix(100, 0)); err != nil {
				t.Fatal(err)
			}
		}
	}
	path := orderedSave(t, r)
	for side, context := range []string{"xa", "ya"} {
		clone := orderedFixtureClone(t, path)
		for i, text := range []string{context, "middle", "distractor", "cue"} {
			contextObservation(t, clone, fmt.Sprintf("step-%d", i), text, time.Unix(int64(101+i), 0))
		}
		margin := orderedMargin(clone, "left", "right")
		if side == 0 && margin < .2 || side == 1 && margin > -.2 {
			t.Fatalf("multi-observation prefix lost its earlier context: %g", margin)
		}
	}
}
