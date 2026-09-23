package runtime

import (
	"testing"
	"time"
)

func TestInquiryConsequenceUpdatesPriorExperience(t *testing.T) {
	runtime := NewBrainRuntime(nil)
	base := runtime.InquiryPriorExperience(InquiryFocus, 0.50)
	if base != 0.50 {
		t.Fatalf("unexpected baseline prior experience: %v", base)
	}
	runtime.RecordInquiryConsequence(InquiryFocus, 1)
	positive := runtime.InquiryPriorExperience(InquiryFocus, 0.50)
	if positive <= base {
		t.Fatalf("positive consequence did not increase prior experience: %v <= %v", positive, base)
	}

	runtime.RecordInquiryConsequence(InquiryFocus, -1)
	negative := runtime.InquiryPriorExperience(InquiryFocus, 0.50)
	if negative >= positive {
		t.Fatalf("negative consequence did not reduce learned preference: %v >= %v", negative, positive)
	}
}

func TestAdaptiveInquiryAgendaUsesLearnedPreference(t *testing.T) {
	runtime := NewBrainRuntime(nil)
	interpretation := CognitiveInterpretation{
		State: CognitiveState{BrainIdentity: BrainIdentity, Sequence: 1},
		Answer: Answer{Uncertainty: Uncertainty{Level: 0.9}},
	}
	at := time.Unix(1, 0).UTC()

	before, err := runtime.BuildAdaptiveInquiryAgenda(interpretation, at)
	if err != nil {
		t.Fatal(err)
	}
	for _, evaluation := range before.Evaluations {
		if evaluation.Action == InquiryFocus {
			if evaluation.Score <= 0 {
				t.Fatalf("unexpected focus score: %v", evaluation.Score)
			}
			break
		}
	}

	runtime.RecordInquiryConsequence(InquiryFocus, 1)
	after, err := runtime.BuildAdaptiveInquiryAgenda(interpretation, at.Add(time.Second))
	if err != nil {
		t.Fatal(err)
	}
	var beforeScore, afterScore float64
	for _, evaluation := range before.Evaluations {
		if evaluation.Action == InquiryFocus { beforeScore = evaluation.Score }
	}
	for _, evaluation := range after.Evaluations {
		if evaluation.Action == InquiryFocus { afterScore = evaluation.Score }
	}
	if afterScore <= beforeScore {
		t.Fatalf("learned positive valence did not increase focus score: %v <= %v", afterScore, beforeScore)
	}
}
