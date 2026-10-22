package runtime

import (
	"math/rand"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/project-horizon/horizon-core/services/ai/knowledge"
)

var orderedAuditPairs = []struct{ name, a, b, question string }{
	{"actor-push", "rina mendorong budi", "budi mendorong rina", "siapa yang mendorong"},
	{"actor-follow", "sari mengikuti doni", "doni mengikuti sari", "siapa yang mengikuti"},
	{"negation-place", "gelas di meja bukan di rak", "gelas di rak bukan di meja", "di mana gelas"},
	{"negation-container", "bola di kotak bukan di tas", "bola di tas bukan di kotak", "di mana bola"},
	{"time-people", "budi datang sebelum rina pergi", "rina pergi sebelum budi datang", "apa terjadi lebih dahulu"},
	{"time-events", "pintu dibuka sebelum lampu dinyalakan", "lampu dinyalakan sebelum pintu dibuka", "apa terjadi lebih dahulu"},
}

func orderedFixtureCheckpoint(t *testing.T, seed int64, texts ...string) string {
	t.Helper()
	set := make(map[string]bool)
	for _, text := range texts {
		for _, token := range strings.Fields(text) {
			set[token] = true
		}
	}
	var tokens []string
	for token := range set {
		tokens = append(tokens, token)
	}
	sort.Strings(tokens)
	rand.New(rand.NewSource(seed)).Shuffle(len(tokens), func(i, j int) { tokens[i], tokens[j] = tokens[j], tokens[i] })
	brain := knowledge.NewBrain()
	for _, token := range tokens {
		brain.Store(token)
	}
	r := NewBrainRuntime(brain)
	r.SetClock(NewFixedClock(time.Unix(100, 0).UTC()))
	path := filepath.Join(t.TempDir(), "shared-foundation.json")
	if err := r.Checkpoint(path); err != nil {
		t.Fatal(err)
	}
	return path
}

func orderedFixtureClone(t *testing.T, path string) *BrainRuntime {
	t.Helper()
	r := NewBrainRuntime(knowledge.NewBrain())
	if err := r.RestoreCheckpoint(path); err != nil {
		t.Fatal(err)
	}
	r.SetClock(NewFixedClock(time.Unix(101, 0).UTC()))
	return r
}

func orderedAuditObservation(t *testing.T, r *BrainRuntime, context, question string) CognitiveInterpretation {
	t.Helper()
	out, _, err := NewCognitiveOrchestrator(r).ProcessObservation(Event{ID: "audit-query", Timestamp: time.Unix(101, 0).UTC(), Cycles: 1}, ObservationInput{Source: "audit", Modality: "text", ContextTokens: strings.Fields(context), Tokens: strings.Fields(question)}, nil)
	if err != nil {
		t.Fatal(err)
	}
	return out
}

func TestOrderedAuditPairsReachDistinctNeuralStatesWithStableIdentities(t *testing.T) {
	for _, pair := range orderedAuditPairs {
		t.Run(pair.name, func(t *testing.T) {
			checkpoint := orderedFixtureCheckpoint(t, 17, pair.a, pair.b, pair.question)
			a, b, control := orderedFixtureClone(t, checkpoint), orderedFixtureClone(t, checkpoint), orderedFixtureClone(t, checkpoint)
			left := orderedAuditObservation(t, a, pair.a, pair.question)
			right := orderedAuditObservation(t, b, pair.b, pair.question)
			repeat := orderedAuditObservation(t, control, pair.a, pair.question)
			if !reflect.DeepEqual(left.State, repeat.State) || !reflect.DeepEqual(a.activation.PredictionSnapshot(), control.activation.PredictionSnapshot()) {
				t.Fatal("identical input is not reproducible")
			}
			if reflect.DeepEqual(left.State.Activations, right.State.Activations) && reflect.DeepEqual(a.activation.PredictionSnapshot(), b.activation.PredictionSnapshot()) {
				t.Fatal("order vanished after grounding: neural state and prediction are identical")
			}
		})
	}
}
