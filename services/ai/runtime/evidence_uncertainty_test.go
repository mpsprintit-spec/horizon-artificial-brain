package runtime

import (
	"fmt"
	"math"
	"math/rand"
	"reflect"
	"slices"
	"testing"
	"time"

	"github.com/project-horizon/horizon-core/services/ai/knowledge"
	"github.com/project-horizon/horizon-core/services/ai/learning"
)

func uncertaintyProbe(t *testing.T, r *BrainRuntime, at time.Time) CognitiveInterpretation {
	t.Helper()
	probe := orderedFixtureClone(t, orderedSave(t, r))
	got, _, err := NewCognitiveOrchestrator(probe).ProcessObservation(Event{ID: "heldout-cue", EpisodeID: "heldout", Timestamp: at, Cycles: 1}, ObservationInput{Source: "calibration-probe", Modality: "text", Tokens: []string{"a"}}, nil)
	if err != nil {
		t.Fatal(err)
	}
	return got
}

func TestEvidenceUncertaintyNoEvidenceAndPopulationSize(t *testing.T) {
	for _, width := range []int{1, 4, 32} {
		r := NewBrainRuntime(knowledge.NewBrain())
		at := time.Unix(100, 0).UTC()
		var population []knowledge.NodeID
		for i := 0; i < width; i++ {
			population = append(population, r.brain.StoreAt(fmt.Sprintf("unit-%d", i), at).ID)
		}
		out, err := r.CognitiveProcess(Event{ID: "untrained", Timestamp: at, StimulusNodeIDs: population, Cycles: 1})
		if err != nil {
			t.Fatal(err)
		}
		got, err := r.InterpretCognitive(out, Observation{})
		if err != nil {
			t.Fatal(err)
		}
		if got.Answer.Confidence != 0 || got.Answer.Uncertainty.Level != 1 || got.Answer.PredictionAssessment.EvidenceMass != 0 {
			t.Fatal("population size fabricated certainty without outcome evidence")
		}
		if width == 32 && out.Resonance <= 1 {
			t.Fatal("control did not saturate legacy resonance uncertainty")
		}
	}
}

func TestEvidenceUncertaintyExpiredForecastCannotClaimCurrentConfidence(t *testing.T) {
	r, at := reversalRuntime(t, 17)
	for i := 0; i < 12; i++ {
		if _, err := r.LearnExperience(learning.Experience{Sequence: []string{"a", "b"}, Weight: .8, Confidence: .8}, at); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := r.CognitiveProcess(Event{ID: "cue", EpisodeID: "one", Timestamp: at, Stimulus: []string{"a"}, Cycles: 1}); err != nil {
		t.Fatal(err)
	}
	if r.activation.PredictionAssessment(at, knowledge.CurrentDynamics).Confidence < .8 {
		t.Fatal("missing supported forecast")
	}
	r.SetClock(NewFixedClock(at.Add(knowledge.ContextMaxAge + time.Second)))
	out, err := r.CognitiveThink(1)
	if err != nil {
		t.Fatal(err)
	}
	if out.PredictionAssessment.Confidence != 0 || out.PredictionAssessment.UnknownProbability != 1 {
		t.Fatal("expired cue retained empirical certainty")
	}
}

func TestEvidenceUncertaintySingleRepeatedBalancedAndThought(t *testing.T) {
	var confidences []float64
	for _, count := range []int{1, 12} {
		r, at := reversalRuntime(t, 17)
		for i := 0; i < count; i++ {
			if _, err := r.LearnExperience(learning.Experience{Sequence: []string{"a", "b"}, Weight: .8, Confidence: .8}, at); err != nil {
				t.Fatal(err)
			}
		}
		got := uncertaintyProbe(t, r, at.Add(time.Second))
		confidences = append(confidences, got.Answer.Confidence)
		if got.Answer.PredictionAssessment.Scope != "learned-next-population" || !slices.Equal(got.Answer.PredictionAssessment.NodeIDs, []knowledge.NodeID{r.brain.Registry.Get("b").ID}) {
			t.Fatal("confidence scope or selected neural forecast missing")
		}
	}
	if confidences[0] <= 0 || confidences[0] >= .6 || confidences[1] < .8 || confidences[1] >= 1 {
		t.Fatalf("unsupported evidence response: %v", confidences)
	}
	r, at := reversalRuntime(t, 17)
	for i := 0; i < 12; i++ {
		for _, outcome := range []string{"b", "c"} {
			if _, err := r.LearnExperience(learning.Experience{Sequence: []string{"a", outcome}, Weight: .8, Confidence: .8}, at); err != nil {
				t.Fatal(err)
			}
		}
	}
	got, _, err := NewCognitiveOrchestrator(r).ProcessObservation(Event{ID: "ambiguous", EpisodeID: "heldout", Timestamp: at, Cycles: 1}, ObservationInput{Source: "probe", Modality: "text", Tokens: []string{"a"}}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if got.Answer.Uncertainty.Level < .4 || got.Answer.PredictionAssessment.Ambiguity < .95 {
		t.Fatal("balanced alternatives collapsed to certainty")
	}
	want := got.Answer.PredictionAssessment.Clone()
	r.SetClock(NewFixedClock(at))
	for i := 0; i < 20; i++ {
		out, err := r.CognitiveThink(1)
		if err != nil {
			t.Fatal(err)
		}
		if !reflect.DeepEqual(out.PredictionAssessment, want) {
			t.Fatal("thought manufactured outcome evidence or certainty")
		}
	}
	got.Answer.PredictionAssessment.NodeIDs[0] = 999999
	if reflect.DeepEqual(got.Answer.PredictionAssessment, r.activation.PredictionAssessment(at, knowledge.CurrentDynamics)) {
		t.Fatal("answer forecast aliases live continuation")
	}
}

func TestEvidenceUncertaintyHeldoutCalibration(t *testing.T) {
	type bin struct {
		count               int
		confidence, correct float64
	}
	bins := map[int]*bin{}
	brier, legacyBrier, observations := 0.0, 0.0, 0
	for seed := int64(1); seed <= 20; seed++ {
		for _, rate := range []int{5, 8, 10} {
			r, at := reversalRuntime(t, seed)
			train := make([]string, 40)
			for i := range train {
				train[i] = "c"
				if i < 4*rate {
					train[i] = "b"
				}
			}
			rand.New(rand.NewSource(seed*101+int64(rate))).Shuffle(len(train), func(i, j int) { train[i], train[j] = train[j], train[i] })
			for i, outcome := range train {
				if _, err := r.LearnExperience(learning.Experience{Sequence: []string{"a", outcome}, Weight: .8, Confidence: .8, Source: "training", ExperienceID: fmt.Sprintf("train-%d", i)}, at); err != nil {
					t.Fatal(err)
				}
			}
			got := uncertaintyProbe(t, r, at.Add(time.Second))
			assessment := got.Answer.PredictionAssessment
			var pb, pc float64
			for _, alternative := range assessment.Alternatives {
				if slices.Contains(alternative.NodeIDs, r.brain.Registry.Get("b").ID) {
					pb = alternative.Probability
				}
				if slices.Contains(alternative.NodeIDs, r.brain.Registry.Get("c").ID) {
					pc = alternative.Probability
				}
			}
			// Held-out outcomes remain in the evaluator, never in the runtime input.
			gold := make([]bool, 20)
			for i := range gold {
				gold[i] = i < 2*rate
			}
			rand.New(rand.NewSource(9000+seed)).Shuffle(len(gold), func(i, j int) { gold[i], gold[j] = gold[j], gold[i] })
			selectedB := slices.Contains(assessment.NodeIDs, r.brain.Registry.Get("b").ID)
			for _, actualB := range gold {
				yb, yc := 0.0, 1.0
				if actualB {
					yb, yc = 1, 0
				}
				brier += (pb-yb)*(pb-yb) + (pc-yc)*(pc-yc) + assessment.UnknownProbability*assessment.UnknownProbability
				// The old 1-uncertainty reports certainty when many input units saturate resonance.
				correct := selectedB == actualB
				if !correct {
					legacyBrier += 2
				}
				index := min(9, int(assessment.Confidence*10))
				if bins[index] == nil {
					bins[index] = &bin{}
				}
				b := bins[index]
				b.count++
				b.confidence += assessment.Confidence
				if correct {
					b.correct++
				}
				observations++
			}
		}
	}
	ece := 0.0
	for _, b := range bins {
		ece += math.Abs(b.confidence-b.correct) / float64(observations)
	}
	brier /= float64(observations)
	legacyBrier /= float64(observations)
	t.Logf("heldout=%d multiclass_Brier=%.6f ECE=%.6f certainty_ablation_Brier=%.6f", observations, brier, ece, legacyBrier)
	if ece > .15 || brier >= legacyBrier {
		t.Fatalf("readout lacks held-out calibration advantage: Brier=%f baseline=%f ECE=%f", brier, legacyBrier, ece)
	}
}

func TestEvidenceUncertaintyWorldChangeRaisesDoubt(t *testing.T) {
	r, at := reversalRuntime(t, 17)
	for i := 0; i < 12; i++ {
		reversalEpisode(t, r, i, "b", at.Add(time.Duration(2*i)*time.Second))
	}
	before := uncertaintyProbe(t, r, at.Add(24*time.Second)).Answer
	reversalEpisode(t, r, 12, "c", at.Add(24*time.Second))
	after := uncertaintyProbe(t, r, at.Add(26*time.Second)).Answer
	t.Logf("world change uncertainty before=%.6f after=%.6f", before.Uncertainty.Level, after.Uncertainty.Level)
	if after.Uncertainty.Level <= before.Uncertainty.Level+.01 {
		t.Fatal("contradictory actual outcome did not raise doubt")
	}
	for i := 1; i < 12; i++ {
		reversalEpisode(t, r, 12+i, "c", at.Add(time.Duration(24+2*i)*time.Second))
	}
	final := uncertaintyProbe(t, r, at.Add(48*time.Second)).Answer
	if !slices.Contains(final.PredictionAssessment.NodeIDs, r.brain.Registry.Get("c").ID) || final.Confidence >= 1 {
		t.Fatal("assessment lost neural adaptation or claimed certainty")
	}
}

func TestEvidenceUncertaintyNumericObservationsAndNoise(t *testing.T) {
	r := NewBrainRuntime(knowledge.NewBrain())
	at := time.Unix(100, 0).UTC()
	for i := 0; i < 12; i++ {
		for step, v := range [][]float64{{1, 0, 0, 0}, {0, 1, 0, 0}} {
			_, err := r.ProcessSensorObservation(SensorObservation{SampleID: fmt.Sprintf("learn-%d-%d", i, step), SessionID: fmt.Sprintf("episode-%d", i), Source: "camera", Modality: "vision", Timestamp: at.Add(time.Duration(2*i+step) * time.Second), ChannelStates: v, Confidence: .9, Provenance: SensorProvenance{Adapter: "numeric-calibration"}})
			if err != nil {
				t.Fatal(err)
			}
		}
	}
	path := orderedSave(t, r)
	for _, noise := range []float64{0, .01, .1} {
		probe := orderedFixtureClone(t, path)
		got, err := probe.ProcessSensorObservation(SensorObservation{SampleID: "heldout", SessionID: "heldout", Source: "camera", Modality: "vision", Timestamp: at.Add(24 * time.Second), ChannelStates: []float64{1, 0, noise, 0}, Confidence: .9, Provenance: SensorProvenance{Adapter: "numeric-calibration"}})
		if err != nil {
			t.Fatal(err)
		}
		t.Logf("numeric noise=%f confidence=%f unknown=%f populations=%d", noise, got.Answer.Confidence, got.Answer.PredictionAssessment.UnknownProbability, len(got.Answer.PredictionAssessment.Alternatives))
		if noise == 0 && (got.Answer.Confidence <= 0 || got.Answer.PredictionAssessment.EvidenceMass <= 0) {
			t.Fatal("actual numeric transition evidence was ignored")
		}
		if noise > 0 && got.Answer.PredictionAssessment.EvidenceMass == 0 && got.Answer.Confidence != 0 {
			t.Fatal("unrecognized noisy population claimed confidence")
		}
		if len(probe.brain.SurfaceAnnotationsSnapshot()) != 0 {
			t.Fatal("numeric assessment invented lexical identity")
		}
	}
}

func TestEvidenceUncertaintyChangesLearnedInquirySelection(t *testing.T) {
	r := NewBrainRuntime(knowledge.NewBrain())
	at := time.Unix(100, 0).UTC()
	var cue []knowledge.NodeID
	for i := 0; i < 32; i++ {
		cue = append(cue, r.brain.StoreAt(fmt.Sprintf("cue-%d", i), at).ID)
	}
	x := r.brain.StoreAt("x", at).ID
	y := r.brain.StoreAt("y", at).ID
	for i := 0; i < 12; i++ {
		for _, target := range []knowledge.NodeID{x, y} {
			if _, err := r.LearnExperience(learning.Experience{NeuralSequence: [][]knowledge.NodeID{cue, {target}}, Weight: .8, Confidence: .8}, at); err != nil {
				t.Fatal(err)
			}
		}
	}
	target := r.brain.StoreAt("action-target", at).ID
	if err := r.RegisterActionBinding(ActionBinding{RequestID: "view", BrainIdentity: BrainIdentity, Intent: "inquiry:change_view", TargetNodeIDs: []knowledge.NodeID{target}}); err != nil {
		t.Fatal(err)
	}
	for _, outcome := range []knowledge.NodeID{x, y} {
		if _, err := r.RecordInquiryConsequenceEvent(InquiryConsequenceEvent{Action: InquiryChangeView, InformationGain: .9, Reliability: 1, TargetNodeIDs: []knowledge.NodeID{target}, OutcomeNodeIDs: []knowledge.NodeID{outcome}, CausalLink: "learned-view"}, at); err != nil {
			t.Fatal(err)
		}
	}
	out, err := r.CognitiveProcess(Event{ID: "wide-cue", EpisodeID: "heldout", Timestamp: at, StimulusNodeIDs: cue, Cycles: 1})
	if err != nil {
		t.Fatal(err)
	}
	current, err := r.InterpretCognitive(out, Observation{})
	if err != nil {
		t.Fatal(err)
	}
	out.PredictionAssessment = nil
	legacy, err := r.InterpretCognitive(out, Observation{})
	if err != nil {
		t.Fatal(err)
	}
	if legacy.Answer.Uncertainty.Level != 0 || current.Answer.Uncertainty.Level < .4 {
		t.Fatal("uncertainty ablation did not isolate resonance saturation")
	}
	path := orderedSave(t, r)
	oldAgenda, err := orderedFixtureClone(t, path).PlanInquiry(legacy.Answer.Uncertainty.Level, at)
	if err != nil {
		t.Fatal(err)
	}
	newAgenda, err := orderedFixtureClone(t, path).PlanInquiry(current.Answer.Uncertainty.Level, at)
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("inquiry certainty-ablation=%s evidence-readout=%s", selectedAction(oldAgenda), selectedAction(newAgenda))
	if selectedAction(oldAgenda) != InquiryWait || selectedAction(newAgenda) != InquiryChangeView {
		t.Fatal("evidence uncertainty did not affect learned information seeking")
	}
	if current.Recommendation != nil || len(r.actions) != 1 {
		t.Fatal("uncertainty invented action authorization/execution")
	}
}

func TestEvidenceUncertaintyInquiryResolvesCapturedForecast(t *testing.T) {
	for _, observed := range []string{"b", "novel"} {
		r, at := reversalRuntime(t, 17)
		for i := 0; i < 12; i++ {
			for _, target := range []string{"b", "c"} {
				if _, err := r.LearnExperience(learning.Experience{Sequence: []string{"a", target}, Weight: .8, Confidence: .8}, at); err != nil {
					t.Fatal(err)
				}
			}
		}
		orch := NewCognitiveOrchestrator(r)
		cue, _, err := orch.ProcessObservation(Event{ID: "question", EpisodeID: "one", Timestamp: at, Cycles: 1}, ObservationInput{Source: "probe", Modality: "text", Tokens: []string{"a"}}, nil)
		if err != nil {
			t.Fatal(err)
		}
		request, err := r.BuildInquiryRequest(*cue.InquiryAgenda)
		if err != nil {
			t.Fatal(err)
		}
		if request.PredictionAssessment == nil {
			t.Fatal("request did not capture the forecast it asks to resolve")
		}
		// Inspect completion before a subsequent selection replaces InquiryState.
		returned, _, err := r.processObservationCommand(Event{ID: "answer", EpisodeID: "one", Timestamp: at.Add(time.Second), Cycles: 1}, ObservationCommand{Input: ObservationInput{Source: "probe", Modality: "text", Tokens: []string{observed}}, Inquiry: &request})
		if err != nil {
			t.Fatal(err)
		}
		gain := r.brain.BrainState.InquiryState.ObservedInformationGain
		t.Logf("observed=%s captured_uncertainty=%f covered=%f gain=%f next_uncertainty=%f", observed, request.Uncertainty, 1-request.PredictionAssessment.UnknownProbability, gain, returned.Answer.Uncertainty.Level)
		if observed == "b" && (gain < .3 || returned.Answer.Uncertainty.Level != 1) {
			t.Fatal("recognized prior result was confused with unrelated next-forecast uncertainty")
		}
		if observed == "novel" && gain != 0 {
			t.Fatal("unmodeled result invented calibrated information gain")
		}
	}
}
