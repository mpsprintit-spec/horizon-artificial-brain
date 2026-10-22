package runtime

import (
	"bytes"
	"encoding/json"
	"fmt"
	"math"
	"math/rand"
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/project-horizon/horizon-core/services/ai/knowledge"
	"github.com/project-horizon/horizon-core/services/ai/learning"
)

type continuationFixture struct {
	r    *BrainRuntime
	pops map[string][]knowledge.NodeID // Teacher/test metadata only.
	at   time.Time
	zero string
}

func newContinuationFixture(t *testing.T, seed int64, journal bool) continuationFixture {
	t.Helper()
	r := NewBrainRuntime(knowledge.NewBrain())
	at := time.Unix(100, 0).UTC()
	r.SetClock(NewFixedClock(at))
	f := continuationFixture{r: r, at: at, pops: map[string][]knowledge.NodeID{}}
	if journal {
		r.SetEventLog(openKnowledgeJournal(t))
		f.zero = filepath.Join(t.TempDir(), "zero.json")
		if err := r.Checkpoint(f.zero); err != nil {
			t.Fatal(err)
		}
	}
	coordinates := rand.New(rand.NewSource(seed)).Perm(32)
	for i := 0; i < map[int64]int{7: 0, 17: 1, 47: 2}[seed]; i++ {
		vector := make([]float64, 32)
		vector[coordinates[31-i]] = 1
		if _, err := r.GroundVectorAt(knowledge.NewNeuralVector(vector), at); err != nil {
			t.Fatal(err)
		}
	}
	names := []string{"seed", "object-a", "object-b", "motion-a", "motion-b", "Bola", "Gelas", " ", "diam.", "jatuh.", ""}
	for i, name := range names {
		vector := make([]float64, 32)
		vector[coordinates[i]] = 1
		p, err := r.GroundVectorAt(knowledge.NewNeuralVector(vector), at)
		if err != nil {
			t.Fatal(err)
		}
		f.pops[name] = p.Population
		if i >= 5 {
			e, err := knowledge.NewSpellingExposure(p.Population, name, .9, .9)
			if err != nil {
				t.Fatal(err)
			}
			if _, err := r.LearnSpelling(e, at); err != nil {
				t.Fatal(err)
			}
		}
	}
	return f
}

func (f continuationFixture) teach(t *testing.T, rounds int) {
	t.Helper()
	// Local paired transitions only. No whole sentence or joint object/motion
	// context is ever taught. The teacher supplies this factorization; the
	// experiment does not claim that Horizon discovered grammatical roles.
	for round := 0; round < rounds; round++ {
		for _, lesson := range [][3]string{
			{"seed", "Bola", "object-a"}, {"seed", "Gelas", "object-b"},
			{"Bola", " ", ""}, {"Gelas", " ", ""},
			{" ", "diam.", "motion-a"}, {" ", "jatuh.", "motion-b"},
			{"diam.", "", ""}, {"jatuh.", "", ""},
		} {
			e := learning.Experience{NeuralSequence: [][]knowledge.NodeID{f.pops[lesson[0]], f.pops[lesson[1]]}, Weight: .9, Confidence: .9}
			if lesson[2] != "" {
				e.NeuralContext = [][]knowledge.NodeID{f.pops[lesson[2]]}
			}
			if _, err := f.r.LearnExperience(e, f.at); err != nil {
				t.Fatal(err)
			}
		}
	}
}

func (f continuationFixture) context(t *testing.T, object, motion string) {
	t.Helper()
	_, err := f.r.CognitiveProcess(Event{ID: fmt.Sprintf("numeric-context-%d", f.r.LastSequence()), Timestamp: f.at.Add(time.Second), Cycles: 1, EpisodeID: "continuation", ResetContext: true, NeuralTrace: &knowledge.ObservationTrace{Stimulus: [][]knowledge.NodeID{f.pops[object], f.pops[motion]}}})
	if err != nil {
		t.Fatal(err)
	}
}

func (f continuationFixture) read(t *testing.T, steps int) ContinuationReadout {
	t.Helper()
	before := continuationBytes(t, f.r)
	var journal []byte
	var err error
	if f.r.eventLog != nil {
		journal, err = os.ReadFile(f.r.eventLog.file.Name())
		if err != nil {
			t.Fatal(err)
		}
	}
	out, err := f.r.ReadLearnedContinuation(&knowledge.ObservationTrace{Stimulus: [][]knowledge.NodeID{f.pops["seed"]}}, steps, f.at.Add(time.Second))
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(before, continuationBytes(t, f.r)) {
		t.Fatal("hypothetical output changed the Brain, real forecast, context, sequence or evidence")
	}
	if out.Sequence != f.r.LastSequence() || out.BrainIdentity != BrainIdentity || !out.Hypothetical {
		t.Fatal("hypothesis lost its identity/revision boundary")
	}
	if f.r.eventLog != nil {
		after, err := os.ReadFile(f.r.eventLog.file.Name())
		if err != nil || !bytes.Equal(journal, after) {
			t.Fatal("hypothetical output appended real experience")
		}
	}
	return out
}

func TestLearnedContinuationComposesNumericLocalTransitions(t *testing.T) {
	for _, seed := range []int64{7, 17, 47} {
		t.Run(fmt.Sprint(seed), func(t *testing.T) {
			f := newContinuationFixture(t, seed, false)
			f.teach(t, 6)
			canonical, err := f.r.brain.CanonicalJSON()
			if err != nil {
				t.Fatal(err)
			}
			for _, text := range []string{"Bola", "Gelas", "diam.", "jatuh.", "Bola jatuh.", "Gelas diam."} {
				quoted, _ := json.Marshal(text)
				if bytes.Contains(canonical, quoted) {
					t.Fatal("teacher words/sentences leaked into canonical knowledge strings")
				}
			}
			synapses := 0
			for _, node := range f.r.brain.Registry.Nodes() {
				synapses += len(node.OutboundAll())
			}
			t.Logf("fixture seed=%d units=%d synapses=%d patterns=%d pretty-canonical-bytes=%d", seed, len(f.r.brain.Registry.Nodes()), synapses, len(f.r.brain.Patterns.All()), len(canonical))
			cp := filepath.Join(t.TempDir(), "teacher.json")
			if err := f.r.Checkpoint(cp); err != nil {
				t.Fatal(err)
			}
			for _, object := range []string{"object-a", "object-b"} {
				for _, motion := range []string{"motion-a", "motion-b"} {
					r := NewBrainRuntime(knowledge.NewBrain())
					if err := r.RestoreCheckpoint(cp); err != nil {
						t.Fatal(err)
					}
					candidate := f
					candidate.r = r
					candidate.context(t, object, motion)
					out := candidate.read(t, 8)
					want := map[string]string{"object-a": "Bola", "object-b": "Gelas"}[object] + " " + map[string]string{"motion-a": "diam.", "motion-b": "jatuh."}[motion]
					if out.Status != ContinuationComplete || out.Text != want || len(out.Steps) != 4 || out.MinConfidence <= ContinuationConfidenceGate {
						t.Fatalf("numeric factors %s/%s: got %+v, want %q", object, motion, out, want)
					}
					t.Logf("seed=%d context=%s/%s text=%q weakest-step=%f", seed, object, motion, out.Text, out.MinConfidence)
				}
			}
		})
	}
}

func TestLearnedContinuationRequiresLearningAndPreservesUncertainty(t *testing.T) {
	for _, rounds := range []int{0, 1, 6} {
		f := newContinuationFixture(t, 7, false)
		f.teach(t, rounds)
		f.context(t, "object-a", "motion-b")
		out := f.read(t, 8)
		if rounds == 0 && out.Status != ContinuationUnknown || rounds == 1 && out.Status != ContinuationUncertain || rounds == 6 && out.Status != ContinuationComplete {
			t.Fatalf("unexpected learning curve at %d rounds: %+v", rounds, out)
		}
		if rounds < 6 && (out.Text != "" || len(out.Scalars) != 0) {
			t.Fatal("uncertain/incomplete continuation published final text")
		}
	}
}

func TestLearnedContinuationLabelsAblationsAndLimits(t *testing.T) {
	f := newContinuationFixture(t, 7, false)
	f.teach(t, 6)
	f.context(t, "object-a", "motion-b")
	want := f.read(t, 8)
	if got := f.read(t, 3); got.Status != ContinuationStepLimit || got.Text != "" || len(got.Scalars) != 0 {
		t.Fatalf("missing learned terminator published partial output: %+v", got)
	}
	for _, label := range []string{"label", "forged", ""} {
		var annotations []knowledge.SurfaceAnnotation
		if label != "" {
			for name, ids := range f.pops {
				annotations = append(annotations, knowledge.SurfaceAnnotation{Surface: label + name, Modality: "language", NodeID: ids[0], Population: ids, LastSeen: f.at})
			}
		}
		if err := f.r.brain.RestoreSurfaceAnnotations(annotations); err != nil {
			t.Fatal(err)
		}
		if !reflect.DeepEqual(want, f.read(t, 8)) {
			t.Fatal("surface metadata changed numeric continuation")
		}
	}
	cp := filepath.Join(t.TempDir(), "ablation-control.json")
	if err := f.r.Checkpoint(cp); err != nil {
		t.Fatal(err)
	}
	bitCells := map[knowledge.NodeID]bool{}
	for _, population := range f.r.brain.ProjectionPopulations {
		if population.Scope != nil && population.Scope.Modality == "written-output" && population.Scope.Field == knowledge.SpellingCodecV1 {
			for _, unit := range population.Units {
				bitCells[unit.NodeID] = true
			}
		}
	}
	for _, id := range f.pops["Bola"] {
		// Isolated conductance intervention. Keep coordinates, connections,
		// connection frequency/history, and all temporal channels intact.
		for target, list := range f.r.brain.Registry.GetByID(id).Synapses {
			if bitCells[target] {
				for _, synapse := range list {
					synapse.Dynamic.Weight, synapse.Dynamic.Confidence = 0, 0
					synapse.Weight, synapse.Confidence = 0, 0
				}
			}
		}
	}
	if got := f.read(t, 8); got.Status != ContinuationUndecodable || got.Text != "" {
		t.Fatalf("word spelling survived conductance ablation: %+v", got)
	}
	g := f
	g.r = NewBrainRuntime(knowledge.NewBrain())
	if err := g.r.RestoreCheckpoint(cp); err != nil {
		t.Fatal(err)
	}
	if spelling, err := g.r.ReadLearnedSpelling(g.pops["Bola"]); err != nil || spelling.Text != "Bola" {
		t.Fatal("temporal ablation must start with the spelling still learned")
	}
	g.r.brain.Patterns = knowledge.NewPatternIndex()
	if got := g.read(t, 8); got.Status != ContinuationUnknown || got.Text != "" {
		t.Fatalf("continuation survived temporal-channel ablation: %+v", got)
	}
}

func TestLearnedContinuationBoundsBeforeCopyAndUnknownReferences(t *testing.T) {
	f := newContinuationFixture(t, 7, true)
	before := continuationBytes(t, f.r)
	good := &knowledge.ObservationTrace{Stimulus: [][]knowledge.NodeID{f.pops["seed"]}}
	for _, steps := range []int{0, -1, MaxContinuationSteps + 1} {
		if _, err := f.r.ReadLearnedContinuation(good, steps, f.at); err == nil {
			t.Fatal("invalid step budget accepted")
		}
	}
	for _, seed := range []*knowledge.ObservationTrace{
		nil, {}, {Stimulus: [][]knowledge.NodeID{{}}},
		{Stimulus: [][]knowledge.NodeID{{999999}}},
		{Stimulus: [][]knowledge.NodeID{{-1}}},
		{Stimulus: [][]knowledge.NodeID{{f.pops["seed"][0], f.pops["seed"][0]}}},
		{Stimulus: [][]knowledge.NodeID{make([]knowledge.NodeID, MaxContinuationMembers+1)}},
		{Stimulus: make([][]knowledge.NodeID, MaxContinuationInputFrames+1)},
	} {
		if _, err := f.r.ReadLearnedContinuation(seed, 4, f.at); err == nil {
			t.Fatal("invalid neural seed accepted")
		}
	}
	if !bytes.Equal(before, continuationBytes(t, f.r)) || f.r.RecoveryRequired() {
		t.Fatal("validation failure changed runtime state")
	}
	packet, err := json.Marshal(good)
	if err != nil {
		t.Fatal(err)
	}
	_, _ = f.r.ReadLearnedContinuation(good, 4, f.at)
	after, _ := json.Marshal(good)
	if !bytes.Equal(packet, after) || !slices.Equal(good.Stimulus[0], f.pops["seed"]) {
		t.Fatal("preview changed caller-owned frames")
	}
}

func TestLearnedContinuationMissingAmbiguousAndExpiredContext(t *testing.T) {
	f := newContinuationFixture(t, 7, false)
	f.teach(t, 6)
	f.context(t, "object-a", "seed") // No learned motion context.
	if got := f.read(t, 8); got.Status != ContinuationUnknown || got.Text != "" {
		t.Fatalf("missing context was filled with a plausible sentence: %+v", got)
	}
	f.context(t, "object-a", "motion-b")
	seed := &knowledge.ObservationTrace{Stimulus: [][]knowledge.NodeID{f.pops["seed"]}}
	for _, at := range []time.Time{f.at.Add(-time.Second), f.at.Add(knowledge.ContextMaxAge + 2*time.Second)} {
		before := continuationBytes(t, f.r)
		out, err := f.r.ReadLearnedContinuation(seed, 8, at)
		if err != nil || out.Status != ContinuationUnknown || out.Text != "" || !bytes.Equal(before, continuationBytes(t, f.r)) {
			t.Fatalf("ineligible context supplied evidence: %+v, %v", out, err)
		}
	}
	both := append(slices.Clone(f.pops["object-a"]), f.pops["object-b"]...)
	_, err := f.r.CognitiveProcess(Event{ID: "ambiguous-context", Timestamp: f.at.Add(time.Second), Cycles: 1, EpisodeID: "ambiguous", ResetContext: true, NeuralTrace: &knowledge.ObservationTrace{Stimulus: [][]knowledge.NodeID{both, f.pops["motion-b"]}}})
	if err != nil {
		t.Fatal(err)
	}
	if got := f.read(t, 8); got.Status != ContinuationUncertain || got.Text != "" || len(got.Steps) != 1 {
		t.Fatalf("competing contexts became a confident sentence: %+v", got)
	}
}

func TestLearnedContinuationRecoveryAndNativeRoundoffBound(t *testing.T) {
	f := newContinuationFixture(t, 7, true)
	f.teach(t, 3)
	middle := filepath.Join(t.TempDir(), "middle.json")
	if err := f.r.Checkpoint(middle); err != nil {
		t.Fatal(err)
	}
	f.teach(t, 3)
	f.context(t, "object-a", "motion-b")
	want := f.read(t, 8)
	for i := 0; i < 4; i++ {
		if !reflect.DeepEqual(want, f.read(t, 8)) {
			t.Fatal("repeated imagination increased its own evidence or changed its outcome")
		}
	}
	full := filepath.Join(t.TempDir(), "full.json")
	if err := f.r.Checkpoint(full); err != nil {
		t.Fatal(err)
	}
	log := f.r.eventLog
	genesis, err := RebuildRuntimeFromCommittedJournal(log, knowledge.NewBrain)
	if err != nil {
		t.Fatal(err)
	}
	zero, err := RebuildRuntimeFromCheckpointAndJournal(log, f.zero)
	if err != nil {
		t.Fatal(err)
	}
	suffix, err := RebuildRuntimeFromCheckpointAndJournal(log, middle)
	if err != nil {
		t.Fatal(err)
	}
	restored := NewBrainRuntime(knowledge.NewBrain())
	if err := restored.RestoreCheckpoint(full); err != nil {
		t.Fatal(err)
	}
	for _, recovered := range []*BrainRuntime{genesis, zero, suffix, restored} {
		assertCompleteContinuation(t, f.r, recovered)
		candidate := f
		candidate.r = recovered
		if !reflect.DeepEqual(want, candidate.read(t, 8)) {
			t.Fatal("recovery changed the learned hypothetical output")
		}
	}
	// A preview neither replaces nor trains the next real recurrent transition.
	f.r.SetEventLog(nil)
	var thought CognitiveOutput
	for i, r := range []*BrainRuntime{f.r, genesis, zero, suffix, restored} {
		r.SetClock(NewFixedClock(f.at.Add(2 * time.Second)))
		out, err := r.CognitiveThink(1)
		if err != nil {
			t.Fatal(err)
		}
		if i == 0 {
			thought = out
		}
		// Historical real propagation orders equal-target opponents only by
		// target, so canonical restore can reorder floating-point additions.
		// This exists on the unmodified parent with NO preview (see probe).
		// Only this inherited next-real-thought comparison has a declared
		// floating roundoff bound. Recovery BEFORE thought, preview outputs,
		// and the before/after-preview continuation remain byte-exact above.
		a, err := json.Marshal(thought)
		if err != nil {
			t.Fatal(err)
		}
		b, err := json.Marshal(out)
		if err != nil {
			t.Fatal(err)
		}
		assertNativeJSONRoundoff(t, a, b)
		assertNativeJSONRoundoff(t, continuationBytes(t, f.r), continuationBytes(t, r))
	}
}

func assertNativeJSONRoundoff(t *testing.T, expected, actual []byte) {
	t.Helper()
	decode := func(data []byte) any {
		decoder := json.NewDecoder(bytes.NewReader(data))
		decoder.UseNumber()
		var value any
		if err := decoder.Decode(&value); err != nil {
			t.Fatal(err)
		}
		return value
	}
	maxDelta := 0.0
	var compare func(any, any, string)
	compare = func(a, b any, path string) {
		switch x := a.(type) {
		case json.Number:
			y, ok := b.(json.Number)
			if !ok {
				t.Fatalf("native type changed at %s", path)
			}
			if x == y {
				return
			}
			if !strings.ContainsAny(string(x)+string(y), ".eE") {
				t.Fatalf("integer/ID/count changed at %s", path)
			}
			av, err := x.Float64()
			if err != nil {
				t.Fatal(err)
			}
			bv, err := y.Float64()
			if err != nil {
				t.Fatal(err)
			}
			delta := math.Abs(av-bv) / max(1, math.Abs(av))
			if math.IsNaN(bv) || delta > 1e-14 {
				t.Fatalf("native value changed beyond roundoff at %s: %s vs %s", path, x, y)
			}
			maxDelta = max(maxDelta, delta)
		case map[string]any:
			y, ok := b.(map[string]any)
			if !ok || len(x) != len(y) {
				t.Fatalf("native object changed at %s", path)
			}
			for key, value := range x {
				v, ok := y[key]
				if !ok {
					t.Fatalf("native field disappeared at %s/%s", path, key)
				}
				compare(value, v, path+"/"+key)
			}
		case []any:
			y, ok := b.([]any)
			if !ok || len(x) != len(y) {
				t.Fatalf("native array/population changed at %s", path)
			}
			for i, value := range x {
				compare(value, y[i], fmt.Sprintf("%s/%d", path, i))
			}
		default:
			if !reflect.DeepEqual(a, b) {
				t.Fatalf("native identity/structure changed at %s", path)
			}
		}
	}
	compare(decode(expected), decode(actual), "")
	t.Logf("next native transition maximum relative roundoff=%g (limit 1e-14)", maxDelta)
}

func TestLearnedContinuationConcurrentOwnerReadsAndLearning(t *testing.T) {
	f := newContinuationFixture(t, 7, true)
	f.teach(t, 6)
	f.context(t, "object-a", "motion-b")
	owner, err := NewRuntimeSupervisor(f.r)
	if err != nil {
		t.Fatal(err)
	}
	defer owner.Close()
	before := f.r.LastSequence()
	var wg sync.WaitGroup
	for i := 0; i < 4; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 4; j++ {
				if err := owner.With(func(r *BrainRuntime) error {
					candidate := f
					candidate.r = r
					if out := candidate.read(t, 8); out.Status != ContinuationComplete || out.Text != "Bola jatuh." {
						t.Errorf("concurrent preview lost learned output: %+v", out)
					}
					return nil
				}); err != nil {
					t.Error(err)
				}
			}
		}()
	}
	if err := owner.WithMutation(func(r *BrainRuntime) error {
		e, err := knowledge.NewSpellingExposure(f.pops["Bola"], "Bola", .9, .9)
		if err != nil {
			return err
		}
		_, err = r.LearnSpelling(e, f.at.Add(time.Second))
		return err
	}); err != nil {
		t.Fatal(err)
	}
	wg.Wait()
	if f.r.LastSequence() != before+1 {
		t.Fatal("concurrent previews created experience revisions")
	}
}

func TestLearnedContinuationLoopsScalarAndMemberBudgets(t *testing.T) {
	f := newContinuationFixture(t, 7, false)
	seed := &knowledge.ObservationTrace{Stimulus: [][]knowledge.NodeID{f.pops["Bola"]}}
	for i := 0; i < 6; i++ {
		_, err := f.r.LearnExperience(learning.Experience{NeuralSequence: [][]knowledge.NodeID{f.pops["Bola"], f.pops["Bola"]}, Weight: .9, Confidence: .9}, f.at)
		if err != nil {
			t.Fatal(err)
		}
	}
	out, err := f.r.ReadLearnedContinuation(seed, MaxContinuationSteps, f.at)
	if err != nil || out.Status != ContinuationStepLimit || len(out.Steps) != MaxContinuationSteps || out.Text != "" {
		t.Fatalf("unterminated learned loop escaped its step budget: %+v, %v", out, err)
	}
	e, err := knowledge.NewSpellingExposure(f.pops["Bola"], strings.Repeat("x", knowledge.MaxSpellingScalars), .9, .9)
	if err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 4; i++ {
		if _, err := f.r.LearnSpelling(e, f.at); err != nil {
			t.Fatal(err)
		}
	}
	before := continuationBytes(t, f.r)
	out, err = f.r.ReadLearnedContinuation(seed, MaxContinuationSteps, f.at)
	if err != nil || out.Status != ContinuationScalarLimit || len(out.Steps) != MaxContinuationScalars/knowledge.MaxSpellingScalars+1 || out.Text != "" || len(out.Scalars) != 0 || !bytes.Equal(before, continuationBytes(t, f.r)) {
		t.Fatalf("scalar budget or read-only boundary failed: %s, %v", out.Status, err)
	}
	g := newContinuationFixture(t, 7, false)
	var big []knowledge.NodeID
	for _, node := range g.r.brain.Registry.Nodes() {
		big = append(big, node.ID)
	}
	slices.Sort(big)
	big = big[:MaxContinuationMembers+1]
	for i := 0; i < 6; i++ {
		if _, err := g.r.LearnExperience(learning.Experience{NeuralSequence: [][]knowledge.NodeID{g.pops["seed"], big}, Weight: .9, Confidence: .9}, g.at); err != nil {
			t.Fatal(err)
		}
	}
	out = g.read(t, 4)
	if out.Status != ContinuationMemberLimit || out.Steps[0].Forecast.MemberCount != len(big) || len(out.Steps[0].Forecast.NodeIDs) != 0 || out.Text != "" {
		t.Fatal("oversized selected population was copied/truncated as an output")
	}
}

func TestLearnedContinuationPreservesRepeatedPopulationOrder(t *testing.T) {
	f := newContinuationFixture(t, 7, false)
	e, err := knowledge.NewSpellingExposure(f.pops["Bola"], "ha", .9, .9)
	if err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 4; i++ {
		if _, err := f.r.LearnSpelling(e, f.at); err != nil {
			t.Fatal(err)
		}
	}
	for i := 0; i < 6; i++ {
		for _, sequence := range [][][]knowledge.NodeID{
			{f.pops["seed"], f.pops["Bola"]},
			{f.pops["seed"], f.pops["Bola"], f.pops["Bola"]},
			{f.pops["seed"], f.pops["Bola"], f.pops["Bola"], f.pops[""]},
		} {
			if _, err := f.r.LearnExperience(learning.Experience{NeuralSequence: sequence, Weight: .9, Confidence: .9}, f.at); err != nil {
				t.Fatal(err)
			}
		}
	}
	if out := f.read(t, 8); out.Status != ContinuationComplete || out.Text != "haha" || len(out.Steps) != 3 || !slices.Equal(out.Steps[0].Forecast.NodeIDs, out.Steps[1].Forecast.NodeIDs) {
		t.Fatalf("repeated population was collapsed or terminated as a cycle: %+v", out)
	}
}
