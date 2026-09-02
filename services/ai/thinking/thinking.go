package thinking

import (
	"strings"

	"github.com/project-horizon/horizon-core/services/ai/activation"
	"github.com/project-horizon/horizon-core/services/ai/hfcc"
	contextengine "github.com/project-horizon/horizon-core/services/ai/context"
	"github.com/project-horizon/horizon-core/services/ai/knowledge"
	"github.com/project-horizon/horizon-core/services/ai/understanding"
)

type ThinkingEngine struct {
	Activation    *activation.Engine
	Context       *contextengine.Engine
	Understanding *understanding.Engine
	MetaCognition *MetaCognitiveLayer
	LastState     CognitiveState
	LastTrace     PathTrace
}

type Thought struct {
	Concepts           []string
	Confidence         float64
	Resonance          float64
	Conflicts          []string
	Hypotheses         []Hypothesis
	NeedsWebSearch     bool
	Inferences         []InferredFact
	PatternEvidence    []*knowledge.PatternSynapse
	BestInterpretation *Interpretation
	Interpretations    []Interpretation
	FunctionalSignals  []FunctionalSignal // FSU Phase 1 observability
	HFCCCandidateSet   *hfcc.CandidateSet // preserved at boundary (hypothesis)
	FormationState     *CandidateFormationState
	PreservedState     *PreservedCandidateState
	PreservationReport *PreservationReport
	HFCCConsumerReport *PreservationReport // Audit B: PCS → HFCC
}

func NewThinkingEngine(kb *knowledge.KnowledgeBase) *ThinkingEngine {
	return &ThinkingEngine{
		Activation:    activation.NewEngine(kb),
		Context:       contextengine.NewEngine(kb),
		Understanding: understanding.NewEngine(kb),
		MetaCognition: NewMetaCognitiveLayer(),
	}
}

func splitPrompt(prompt string) []string {
	return strings.Fields(strings.ToLower(strings.TrimSpace(prompt)))
}
