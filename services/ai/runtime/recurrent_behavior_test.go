package runtime

import (
	"bytes"
	"fmt"
	"math"
	"reflect"
	"testing"
	"time"

	"github.com/project-horizon/horizon-core/services/ai/knowledge"
	"github.com/project-horizon/horizon-core/services/ai/learning"
)

func contextEpisode(t *testing.T, r *BrainRuntime, id, token, episode string, reset bool, at time.Time) {
	t.Helper()
	_, _, err := NewCognitiveOrchestrator(r).ProcessObservation(Event{ID: id, EpisodeID: episode, ResetContext: reset, Timestamp: at, Cycles: 1}, ObservationInput{Source: "probe", Modality: "text", Tokens: []string{token}}, nil)
	if err != nil {
		t.Fatal(err)
	}
}

func TestRecurrentContextDelayCapacityAndHistoryAblation(t *testing.T) {
	checkpoint := contextTraining(t, 17)
	for _, test := range []struct {
		name   string
		delay  time.Duration
		noise  int
		ablate bool
		expect bool
	}{
		{"delay-1s", time.Second, 0, false, true},
		{"delay-20s", 20 * time.Second, 0, false, true},
		{"expired", knowledge.ContextMaxAge + time.Second, 0, false, false},
		{"evicted", time.Second, knowledge.ContextMaxFrames + 1, false, false},
		{"history-ablation", time.Second, 0, true, false},
	} {
		t.Run(test.name, func(t *testing.T) {
			var margins [2]float64
			for side, context := range []string{"xa", "ya"} {
				r := orderedFixtureClone(t, checkpoint)
				at := time.Unix(101, 0).UTC()
				contextObservation(t, r, "context", context, at)
				for i := 0; i < test.noise; i++ {
					contextObservation(t, r, fmt.Sprintf("noise-%d", i), "distractor", at)
				}
				if test.ablate {
					s := r.activation.SnapshotState()
					s.ContextState = nil
					r.activation.RestoreState(s)
				}
				contextObservation(t, r, "question", "cue", at.Add(test.delay))
				margins[side] = orderedMargin(r, "left", "right")
				s := r.activation.SnapshotState().ContextState
				if s == nil || len(s.Frames) > knowledge.ContextMaxFrames {
					t.Fatal("context exceeded capacity")
				}
			}
			if test.expect {
				if margins[0] < .2 || margins[1] > -.2 {
					t.Fatalf("context did not survive bounded delay: %v", margins)
				}
			} else if math.Abs(margins[0]-margins[1]) > .03 {
				t.Fatalf("expired/ablated context still controls prediction: %v", margins)
			}
		})
	}
}

func TestRecurrentContextFollowsLatestEvidenceAndShuffledLabels(t *testing.T) {
	checkpoint := contextTraining(t, 47)
	for _, contexts := range [][]string{{"xa", "ya"}, {"ya", "xa"}} {
		r := orderedFixtureClone(t, checkpoint)
		at := time.Unix(101, 0).UTC()
		for i, context := range contexts {
			contextObservation(t, r, fmt.Sprintf("context-%d", i), context, at)
		}
		contextObservation(t, r, "question", "cue", at.Add(time.Second))
		margin := orderedMargin(r, "left", "right")
		if contexts[1] == "xa" && margin < .1 || contexts[1] == "ya" && margin > -.1 {
			t.Fatalf("older context dominated recent context: %v margin=%g", contexts, margin)
		}
	}
	foundation := orderedFixtureCheckpoint(t, 47, "xa ya cue left right")
	for train := false; ; train = true {
		r := orderedFixtureClone(t, foundation)
		if train {
			for _, sequence := range [][]string{{"xa", "cue", "right"}, {"ya", "cue", "left"}} {
				for i := 0; i < 5; i++ {
					if _, err := r.LearnExperience(learning.Experience{Sequence: sequence, Weight: .9, Confidence: .9}, time.Unix(100, 0)); err != nil {
						t.Fatal(err)
					}
				}
			}
		}
		contextObservation(t, r, "context", "xa", time.Unix(101, 0))
		contextObservation(t, r, "question", "cue", time.Unix(102, 0))
		margin := orderedMargin(r, "left", "right")
		if !train && math.Abs(margin) > .02 {
			t.Fatalf("untrained context invented a target: %g", margin)
		}
		if train && margin > -.2 {
			t.Fatalf("prediction did not follow shuffled supervision: %g", margin)
		}
		if train {
			break
		}
	}
}

func TestRecurrentEpisodeBoundariesPreventContextAndTemporalLinksFromCrossing(t *testing.T) {
	checkpoint := contextTraining(t, 17)
	for _, test := range []struct {
		name, first, next string
		reset             bool
		survives          bool
	}{
		{"same-session", "one", "one", false, true},
		{"new-session", "one", "two", false, false},
		{"explicit-reset", "one", "one", true, false},
		{"scoped-to-unscoped", "one", "", false, false},
		{"unscoped-to-scoped", "", "one", false, false},
	} {
		t.Run(test.name, func(t *testing.T) {
			var margins [2]float64
			for side, context := range []string{"xa", "ya"} {
				r := orderedFixtureClone(t, checkpoint)
				at := time.Unix(101, 0).UTC()
				contextEpisode(t, r, "context", context, test.first, false, at)
				patterns := len(r.brain.Patterns.All())
				contextEpisode(t, r, "question", "cue", test.next, test.reset, at.Add(time.Second))
				margins[side] = orderedMargin(r, "left", "right")
				if !test.survives && len(r.brain.Patterns.All()) != patterns {
					t.Fatal("separate episodes learned an accidental temporal edge")
				}
				if r.activation.SnapshotState().ContextState.EpisodeID != test.next {
					t.Fatal("wrong retained episode identity")
				}
			}
			if test.survives {
				if margins[0] < .2 || margins[1] > -.2 {
					t.Fatalf("same episode lost context: %v", margins)
				}
			} else if math.Abs(margins[0]-margins[1]) > .03 {
				t.Fatalf("cross-episode context leaked: %v", margins)
			}
		})
	}
}

func TestRecurrentContextClockRollbackAndThoughtDoNotInventFreshEvidence(t *testing.T) {
	checkpoint := contextTraining(t, 7)
	r := orderedFixtureClone(t, checkpoint)
	contextObservation(t, r, "context", "xa", time.Unix(101, 0))
	contextObservation(t, r, "question", "cue", time.Unix(102, 0))
	before := r.activation.SnapshotState().ContextState
	for i := 0; i < 3; i++ {
		if _, _, err := r.ThinkAt(1, time.Unix(103, 0)); err != nil {
			t.Fatal(err)
		}
	}
	if !reflect.DeepEqual(before, r.activation.SnapshotState().ContextState) {
		t.Fatal("thought refreshed or appended observed context")
	}
	detached := r.activation.SnapshotState()
	detached.ContextState.Frames[0].Population[0] = 999999
	if reflect.DeepEqual(detached.ContextState, r.activation.SnapshotState().ContextState) {
		t.Fatal("context snapshot aliases the engine")
	}
	contextObservation(t, r, "rollback", "cue", time.Unix(100, 0))
	state := r.activation.SnapshotState().ContextState
	if len(state.Frames) != 1 || state.Frames[0].At != time.Unix(100, 0).UTC() {
		t.Fatal("clock rollback retained future context")
	}
}

func TestRecurrentContextSurvivesRestartAndCompleteJournalRecovery(t *testing.T) {
	r := NewBrainRuntime(knowledge.NewBrain())
	at := time.Unix(100, 0).UTC()
	r.SetClock(NewFixedClock(at))
	log := openKnowledgeJournal(t)
	r.SetEventLog(log)
	for _, sequence := range [][]string{{"xa", "cue", "left"}, {"ya", "cue", "right"}} {
		if _, err := r.LearnExperience(learning.Experience{Sequence: sequence, Weight: .9, Confidence: .9}, at); err != nil {
			t.Fatal(err)
		}
	}
	contextEpisode(t, r, "context", "ya", "episode", false, at.Add(time.Second))
	prefix := orderedSave(t, r)
	contextEpisode(t, r, "noise", "distractor", "episode", false, at.Add(2*time.Second))
	contextEpisode(t, r, "question", "cue", "episode", false, at.Add(3*time.Second))
	if orderedMargin(r, "left", "right") > -.2 {
		t.Fatal("reference did not follow context")
	}
	genesis, err := RebuildRuntimeFromCommittedJournal(log, knowledge.NewBrain)
	if err != nil {
		t.Fatal(err)
	}
	suffix, err := RebuildRuntimeFromCheckpointAndJournal(log, prefix)
	if err != nil {
		t.Fatal(err)
	}
	for _, restored := range []*BrainRuntime{genesis, suffix} {
		assertCompleteContinuation(t, r, restored)
		live := orderedFixtureClone(t, orderedSave(t, r))
		live.SetClock(NewFixedClock(time.Unix(104, 0)))
		restored.SetClock(NewFixedClock(time.Unix(104, 0)))
		expected, err := live.CognitiveThink(1)
		if err != nil {
			t.Fatal(err)
		}
		got, err := restored.CognitiveThink(1)
		if err != nil {
			t.Fatal(err)
		}
		if !reflect.DeepEqual(expected, got) || !bytes.Equal(continuationBytes(t, live), continuationBytes(t, restored)) {
			t.Fatal("recovery changed the next full recurrent thought")
		}
	}
}
