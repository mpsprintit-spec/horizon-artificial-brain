package runtime

import (
	"errors"
	"sort"
	"time"
)

type InquiryAction string

const (
	InquiryReobserve InquiryAction = "reobserve"
	InquiryChangeView InquiryAction = "change_view"
	InquiryFocus InquiryAction = "focus"
	InquiryPoint InquiryAction = "point"
	InquiryVocalize InquiryAction = "vocalize"
	InquiryGesture InquiryAction = "gesture"
	InquiryWait InquiryAction = "wait"
	InquiryImitate InquiryAction = "imitate"
	InquirySafeManipulation InquiryAction = "safe_manipulation"
	InquiryVerbalQuestion InquiryAction = "verbal_question"
)

type InquiryCandidate struct {
	ID string
	Action InquiryAction
	ExpectedInformationGain float64
	Cost float64
	Reversibility float64
	SocialFit float64
	PriorExperience float64
	RequiresAuthorization bool
}

type InquiryEvaluation struct {
	CandidateID string
	Action InquiryAction
	InformationValue float64
	Score float64
	RequiresAuthorization bool
}

type InquiryAgenda struct {
	BrainIdentity string
	Sequence uint64
	CreatedAt time.Time
	Candidates []InquiryCandidate
	Evaluations []InquiryEvaluation
	Selected *InquiryEvaluation
}

type InquiryPolicy struct {
	InformationWeight float64
	CostWeight float64
	ReversibilityWeight float64
	SocialFitWeight float64
	PriorExperienceWeight float64
}

func DefaultInquiryPolicy() InquiryPolicy {
	return InquiryPolicy{
		InformationWeight: 1.0,
		CostWeight: .30,
		ReversibilityWeight: .20,
		SocialFitWeight: .20,
		PriorExperienceWeight: .15,
	}
}

// BuildInquiryAgenda evaluates possible information-seeking actions only.
// It never authorizes or executes an action. Verbal questioning is therefore
// one candidate among alternatives, not the default response to uncertainty.
func BuildInquiryAgenda(brainIdentity string, sequence uint64, at time.Time, candidates []InquiryCandidate, policy InquiryPolicy) (InquiryAgenda, error) {
	if brainIdentity == "" {
		return InquiryAgenda{}, errors.New("brain identity is required")
	}
	if len(candidates) == 0 {
		return InquiryAgenda{}, errors.New("inquiry candidates are required")
	}
	if at.IsZero() {
		at = time.Now().UTC()
	}
	agenda := InquiryAgenda{
		BrainIdentity: brainIdentity,
		Sequence: sequence,
		CreatedAt: at.UTC(),
		Candidates: append([]InquiryCandidate(nil), candidates...),
		Evaluations: make([]InquiryEvaluation, 0, len(candidates)),
	}
	for _, candidate := range candidates {
		if candidate.ID == "" || candidate.Action == "" {
			continue
		}
		information := clamp01(candidate.ExpectedInformationGain)
		cost := clamp01(candidate.Cost)
		reversibility := clamp01(candidate.Reversibility)
		socialFit := clamp01(candidate.SocialFit)
		prior := clamp01(candidate.PriorExperience)
		score := policy.InformationWeight*information -
			policy.CostWeight*cost +
			policy.ReversibilityWeight*reversibility +
			policy.SocialFitWeight*socialFit +
			policy.PriorExperienceWeight*prior
		agenda.Evaluations = append(agenda.Evaluations, InquiryEvaluation{
			CandidateID: candidate.ID,
			Action: candidate.Action,
			InformationValue: information,
			Score: score,
			RequiresAuthorization: candidate.RequiresAuthorization,
		})
	}
	if len(agenda.Evaluations) == 0 {
		return InquiryAgenda{}, errors.New("no valid inquiry candidates")
	}
	sort.SliceStable(agenda.Evaluations, func(i, j int) bool {
		if agenda.Evaluations[i].Score == agenda.Evaluations[j].Score {
			return agenda.Evaluations[i].Action < agenda.Evaluations[j].Action
		}
		return agenda.Evaluations[i].Score > agenda.Evaluations[j].Score
	})
	selected := agenda.Evaluations[0]
	agenda.Selected = &selected
	return agenda, nil
}

// DefaultInquiryCandidates derives candidate actions from uncertainty and
// avoids making language the only mechanism for acquiring information.
// BuildInquiryAgendaFromCognition derives inquiry candidates from Horizon's
// current cognitive boundary. It does not create a second uncertainty model:
// the Answer.Uncertainty value is the uncertainty already produced by the
// neural interpretation layer. Candidate evaluation remains separate from
// authorization and execution.
func BuildInquiryAgendaFromCognition(interpretation CognitiveInterpretation, at time.Time) (InquiryAgenda, error) {
	uncertainty := clamp01(interpretation.Answer.Uncertainty.Level)
	candidates := DefaultInquiryCandidates(uncertainty)
	return BuildInquiryAgenda(interpretation.State.BrainIdentity, interpretation.State.Sequence, at, candidates, DefaultInquiryPolicy())
}

func DefaultInquiryCandidates(uncertainty float64) []InquiryCandidate {
	u := clamp01(uncertainty)
	return []InquiryCandidate{
		{ID:"reobserve", Action:InquiryReobserve, ExpectedInformationGain:u*.65, Cost:.10, Reversibility:1, SocialFit:.95, PriorExperience:.50},
		{ID:"change-view", Action:InquiryChangeView, ExpectedInformationGain:u*.65, Cost:.20, Reversibility:1, SocialFit:.90, PriorExperience:.50},
		{ID:"focus", Action:InquiryFocus, ExpectedInformationGain:u*.75, Cost:.10, Reversibility:1, SocialFit:1, PriorExperience:.50},
		{ID:"point", Action:InquiryPoint, ExpectedInformationGain:u*.45, Cost:.15, Reversibility:1, SocialFit:.85, PriorExperience:.30},
		{ID:"vocalize", Action:InquiryVocalize, ExpectedInformationGain:u*.35, Cost:.25, Reversibility:1, SocialFit:.70, PriorExperience:.30},
		{ID:"gesture", Action:InquiryGesture, ExpectedInformationGain:u*.40, Cost:.15, Reversibility:1, SocialFit:.85, PriorExperience:.35},
		{ID:"wait", Action:InquiryWait, ExpectedInformationGain:u*.25, Cost:.05, Reversibility:1, SocialFit:.95, PriorExperience:.50},
		{ID:"imitate", Action:InquiryImitate, ExpectedInformationGain:u*.55, Cost:.35, Reversibility:.80, SocialFit:.65, PriorExperience:.30},
		{ID:"safe-manipulation", Action:InquirySafeManipulation, ExpectedInformationGain:u*.70, Cost:.55, Reversibility:.65, SocialFit:.55, PriorExperience:.25, RequiresAuthorization:true},
		{ID:"verbal-question", Action:InquiryVerbalQuestion, ExpectedInformationGain:u*.65, Cost:.30, Reversibility:1, SocialFit:.80, PriorExperience:.35},
	}
}
