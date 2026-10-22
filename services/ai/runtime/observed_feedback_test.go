package runtime

import (
	"fmt"
	"math/rand"
	"strings"
	"testing"
	"time"

	"github.com/project-horizon/horizon-core/services/ai/knowledge"
	"github.com/project-horizon/horizon-core/services/ai/learning"
)

func reversalRuntime(t *testing.T, seed int64) (*BrainRuntime, time.Time) {
	t.Helper()
	r := NewBrainRuntime(knowledge.NewBrain())
	at := time.Date(2026, 10, 20, 4, 0, 0, 0, time.UTC)
	r.SetClock(NewFixedClock(at))
	for _, i := range rand.New(rand.NewSource(seed)).Perm(3) {
		r.brain.StoreSurfaceAt([]string{"a", "b", "c"}[i], at, knowledge.WrittenTextEncoding)
	}
	return r, at
}

func TestObservedFeedbackKeepsValidOldContext(t *testing.T) {
	r, at := reversalRuntime(t, 7)
	for _, word := range []string{"stable", "changing"} {
		r.brain.StoreSurfaceAt(word, at, knowledge.WrittenTextEncoding)
	}
	for i := 0; i < 12; i++ {
		for _, context := range []string{"stable", "changing"} {
			if _, err := r.LearnExperience(learning.Experience{ContextSequence: []string{context}, Sequence: []string{"a", "b"}, Weight: .8, Confidence: .8}, at); err != nil {
				t.Fatal(err)
			}
		}
	}
	for i := 0; i < 12; i++ {
		for step, word := range []string{"a", "c"} {
			_, _, err := NewCognitiveOrchestrator(r).ProcessObservation(Event{ID: fmt.Sprintf("context-%d-%d", i, step), EpisodeID: fmt.Sprintf("context-%d", i), Timestamp: at.Add(time.Duration(2*i+step) * time.Second), Cycles: 1}, ObservationInput{Source: "context-reversal", Modality: "text", Tokens: []string{word}, ContextTokens: []string{"changing"}}, nil)
			if err != nil {
				t.Fatal(err)
			}
		}
	}
	for _, context := range []string{"stable", "changing"} {
		probe := orderedFixtureClone(t, orderedSave(t, r))
		_, _, err := NewCognitiveOrchestrator(probe).ProcessObservation(Event{ID: "probe", EpisodeID: "probe", Timestamp: at.Add(25 * time.Second), Cycles: 1}, ObservationInput{Source: "context-reversal", Modality: "text", Tokens: strings.Fields("a"), ContextTokens: []string{context}}, nil)
		if err != nil {
			t.Fatal(err)
		}
		p := probe.activation.PredictionSnapshot().State
		b, c := p[probe.brain.Registry.Get("b").ID], p[probe.brain.Registry.Get("c").ID]
		t.Logf("context=%s B=%f C=%f", context, b, c)
		if context == "stable" && b < c+.1 || context == "changing" && c < b+.1 {
			t.Fatal("reversal damaged valid context or failed to adapt changed context")
		}
	}
}

func reversalEpisode(t *testing.T, r *BrainRuntime, episode int, outcome string, at time.Time) {
	t.Helper()
	for i, token := range []string{"a", outcome} {
		_, _, err := NewCognitiveOrchestrator(r).ProcessObservation(Event{ID: fmt.Sprintf("episode-%d-%d", episode, i), EpisodeID: fmt.Sprintf("episode-%d", episode), Timestamp: at.Add(time.Duration(i) * time.Second), Cycles: 1}, ObservationInput{Source: "reversal-probe", Modality: "text", Tokens: []string{token}}, nil)
		if err != nil {
			t.Fatal(err)
		}
	}
}

func reversalScores(t *testing.T, r *BrainRuntime, at time.Time) (float64, float64) {
	t.Helper()
	probe := orderedFixtureClone(t, orderedSave(t, r))
	_, _, err := NewCognitiveOrchestrator(probe).ProcessObservation(Event{ID: "probe", EpisodeID: "probe", Timestamp: at, Cycles: 1}, ObservationInput{Source: "reversal-probe", Modality: "text", Tokens: []string{"a"}}, nil)
	if err != nil {
		t.Fatal(err)
	}
	p := probe.activation.PredictionSnapshot().State
	return p[probe.brain.Registry.Get("b").ID], p[probe.brain.Registry.Get("c").ID]
}

func TestObservedFeedbackReversalCurve(t *testing.T) {
	for seed := int64(1); seed <= 20; seed++ {
		for _, initial := range []string{"b", "c"} {
			t.Run(fmt.Sprintf("seed-%d/%s", seed, initial), func(t *testing.T) {
				r, at := reversalRuntime(t, seed)
				target := "c"
				if initial == "c" {
					target = "b"
				}
				for i := 0; i < 12; i++ {
					reversalEpisode(t, r, i, initial, at.Add(time.Duration(2*i)*time.Second))
				}
				first := -1
				lastMargin := 0.0
				for i := 0; i < 12; i++ {
					reversalEpisode(t, r, 12+i, target, at.Add(time.Duration(24+2*i)*time.Second))
					b, c := reversalScores(t, r, at.Add(time.Duration(26+2*i)*time.Second))
					margin := c - b
					if target == "b" {
						margin = -margin
					}
					if margin > 0 && first < 0 {
						first = i + 1
					}
					if i >= 5 && margin < lastMargin {
						t.Fatal("post-change prediction regressed")
					}
					lastMargin = margin
				}
				t.Logf("first=%d margin12=%.6f", first, lastMargin)
				if first < 0 || first > 6 || lastMargin < .15 {
					t.Fatalf("no supported reversal within budget: first=%d margin=%v", first, lastMargin)
				}
			})
		}
	}
}

func TestObservedFeedbackUnchangedWorldAndFeedbackAblation(t *testing.T) {
	for _, initial := range []string{"b", "c"} {
		r, at := reversalRuntime(t, 17)
		for i := 0; i < 24; i++ {
			reversalEpisode(t, r, i, initial, at.Add(time.Duration(2*i)*time.Second))
		}
		b, c := reversalScores(t, r, at.Add(48*time.Second))
		margin := b - c
		if initial == "c" {
			margin = -margin
		}
		if margin < .25 {
			t.Fatal("stable mapping lost support without a world change")
		}
	}
	for _, thoughts := range []int{0, 5} {
		r, at := reversalRuntime(t, 17)
		for i := 0; i < 12; i++ {
			reversalEpisode(t, r, i, "b", at.Add(time.Duration(2*i)*time.Second))
		}
		for i := 0; i < 12; i++ {
			tick := at.Add(time.Duration(24+2*i) * time.Second)
			contextEpisode(t, r, fmt.Sprintf("cue-%d", i), "a", fmt.Sprintf("new-%d", i), false, tick)
			for j := 0; j < thoughts; j++ {
				r.SetClock(NewFixedClock(tick))
				if _, err := r.CognitiveThink(1); err != nil {
					t.Fatal(err)
				}
			}
			contextEpisode(t, r, fmt.Sprintf("outcome-%d", i), "c", fmt.Sprintf("new-%d", i), false, tick.Add(time.Second))
		}
		b, c := reversalScores(t, r, at.Add(48*time.Second))
		t.Logf("thoughts=%d margin=%.6f", thoughts, c-b)
		if c-b < .15 {
			t.Fatal("simulation blocked real outcome adaptation")
		}
	}
	r, at := reversalRuntime(t, 17)
	for i := 0; i < 12; i++ {
		reversalEpisode(t, r, i, "b", at.Add(time.Duration(2*i)*time.Second))
	}
	for i := 0; i < 12; i++ {
		tick := at.Add(time.Duration(24+2*i) * time.Second)
		contextEpisode(t, r, fmt.Sprintf("ablate-cue-%d", i), "a", fmt.Sprintf("new-%d", i), false, tick)
		snapshot := r.activation.SnapshotState()
		snapshot.ObservationPrediction = nil
		r.activation.RestoreState(snapshot)
		contextEpisode(t, r, fmt.Sprintf("ablate-outcome-%d", i), "c", fmt.Sprintf("new-%d", i), false, tick.Add(time.Second))
	}
	b, c := reversalScores(t, r, at.Add(48*time.Second))
	t.Logf("feedback ablated margin=%.6f", c-b)
	if c-b > .1 {
		t.Fatal("reversal advantage survived removal of observed feedback")
	}
}

func TestObservedFeedbackNumericSensorReversal(t *testing.T) {
	for _, seed := range []int64{7, 17, 47} {
		r := NewBrainRuntime(knowledge.NewBrain())
		at := time.Date(2026, 10, 20, 4, 0, 0, 0, time.UTC)
		axes := rand.New(rand.NewSource(seed)).Perm(8)
		vectors := make([][]float64, 3)
		for i := range vectors {
			vectors[i] = make([]float64, 8)
			vectors[i][axes[i]] = 1
		}
		var bPop, cPop []knowledge.NodeID
		for episode := 0; episode < 24; episode++ {
			outcome := 1
			if episode >= 12 {
				outcome = 2
			}
			for step, index := range []int{0, outcome} {
				sensor := SensorObservation{SampleID: fmt.Sprintf("sensor-%d-%d", episode, step), SessionID: fmt.Sprintf("sensor-%d", episode), Source: "numeric-probe", Modality: "vision", Timestamp: at.Add(time.Duration(2*episode+step) * time.Second), ChannelStates: vectors[index], Confidence: .9, Provenance: SensorProvenance{Adapter: "reversal-probe"}}
				result, err := r.ProcessSensorObservation(sensor)
				if err != nil {
					t.Fatal(err)
				}
				if index == 1 {
					bPop = result.GroundedRepresentations[0].Population
				}
				if index == 2 {
					cPop = result.GroundedRepresentations[0].Population
				}
			}
		}
		probe := orderedFixtureClone(t, orderedSave(t, r))
		_, err := probe.ProcessSensorObservation(SensorObservation{SampleID: "probe", SessionID: "probe", Source: "numeric-probe", Modality: "vision", Timestamp: at.Add(48 * time.Second), ChannelStates: vectors[0], Confidence: .9, Provenance: SensorProvenance{Adapter: "reversal-probe"}})
		if err != nil {
			t.Fatal(err)
		}
		prediction := probe.activation.PredictionSnapshot().State
		mean := func(ids []knowledge.NodeID) float64 {
			v := 0.0
			for _, id := range ids {
				v += prediction[id]
			}
			return v / float64(len(ids))
		}
		b, c := mean(bPop), mean(cPop)
		t.Logf("numeric seed=%d B=%f C=%f", seed, b, c)
		if c-b < .1 {
			t.Fatal("numeric observed population did not adapt to changed outcome")
		}
		if len(r.brain.SurfaceAnnotationsSnapshot()) != 0 {
			t.Fatal("numeric feedback invented lexical identity")
		}
	}
}

func TestObservedFeedbackSupervisedReversalAndContextRetention(t *testing.T) {
	for _, initial := range []string{"b", "c"} {
		r, at := reversalRuntime(t, 17)
		next := "c"
		if initial == "c" {
			next = "b"
		}
		for i := 0; i < 24; i++ {
			target := initial
			if i >= 12 {
				target = next
			}
			if _, err := r.LearnExperience(learning.Experience{Sequence: []string{"a", target}, Weight: .8, Confidence: .8}, at.Add(time.Duration(i)*time.Second)); err != nil {
				t.Fatal(err)
			}
		}
		b, c := reversalScores(t, r, at.Add(24*time.Second))
		margin := c - b
		if next == "b" {
			margin = -margin
		}
		t.Logf("supervised %s->%s margin=%f", initial, next, margin)
		if margin < .15 {
			t.Fatal("explicit supervised counterexamples did not revise the old mapping")
		}
	}
	r, at := reversalRuntime(t, 7)
	for _, context := range []string{"stable", "changing"} {
		for i := 0; i < 12; i++ {
			if _, err := r.LearnExperience(learning.Experience{ContextSequence: []string{context}, Sequence: []string{"a", "b"}, Weight: .8, Confidence: .8}, at); err != nil {
				t.Fatal(err)
			}
		}
	}
	for i := 0; i < 12; i++ {
		if _, err := r.LearnExperience(learning.Experience{ContextSequence: []string{"changing"}, Sequence: []string{"a", "c"}, Weight: .8, Confidence: .8}, at.Add(time.Duration(i+1)*time.Second)); err != nil {
			t.Fatal(err)
		}
	}
	for _, context := range []string{"stable", "changing"} {
		probe := orderedFixtureClone(t, orderedSave(t, r))
		_, _, err := NewCognitiveOrchestrator(probe).ProcessObservation(Event{ID: "probe", EpisodeID: "probe", Timestamp: at.Add(13 * time.Second), Cycles: 1}, ObservationInput{Source: "teacher-probe", Modality: "text", Tokens: []string{"a"}, ContextTokens: []string{context}}, nil)
		if err != nil {
			t.Fatal(err)
		}
		p := probe.activation.PredictionSnapshot().State
		b, c := p[probe.brain.Registry.Get("b").ID], p[probe.brain.Registry.Get("c").ID]
		if context == "stable" && b < c+.1 || context == "changing" && c < b+.1 {
			t.Fatal("supervised reversal damaged valid context")
		}
	}
}

func TestObservedFeedbackContextAndSimulationDoNotBecomeOutcomes(t *testing.T) {
	r, at := reversalRuntime(t, 7)
	for i, word := range []string{"a", "c"} {
		_, _, err := NewCognitiveOrchestrator(r).ProcessObservation(Event{ID: fmt.Sprintf("real-%d", i), EpisodeID: "one", Timestamp: at.Add(time.Duration(i) * time.Second), Cycles: 1}, ObservationInput{Source: "probe", Modality: "text", Tokens: []string{word}, ContextTokens: []string{"metadata"}, DataTokens: []string{"data"}}, nil)
		if err != nil {
			t.Fatal(err)
		}
	}
	traces := r.brain.BrainState.ExperienceTraces
	last := traces[len(traces)-1]
	if len(last.PreviousPopulation) != 1 || len(last.CurrentPopulation) != 1 || last.PreviousPopulation[0] != r.brain.Registry.Get("a").ID || last.CurrentPopulation[0] != r.brain.Registry.Get("c").ID {
		t.Fatal("context/data metadata entered observed temporal outcomes")
	}
	updated := r.brain.BrainState.PredictionState["last_update_unix"]
	errorSignal := r.brain.BrainState.PredictionState["last_error"]
	priority := r.brain.BrainState.MemoryState["last_priority"]
	r.SetClock(NewFixedClock(at.Add(2 * time.Second)))
	if _, err := r.CognitiveThink(1); err != nil {
		t.Fatal(err)
	}
	for i, tokens := range [][]string{nil, {" "}} {
		if _, err := r.CognitiveProcess(Event{ID: fmt.Sprintf("empty-%d", i), EpisodeID: "one", Stimulus: tokens, Timestamp: at.Add(3 * time.Second), Cycles: 1}); err != nil {
			t.Fatal(err)
		}
	}
	if r.brain.BrainState.PredictionState["last_update_unix"] != updated || r.brain.BrainState.PredictionState["last_error"] != errorSignal || r.brain.BrainState.MemoryState["last_priority"] != priority {
		t.Fatal("internal/empty activity overwrote observed learning evidence")
	}
}
