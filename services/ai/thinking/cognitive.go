package thinking

import (
	"time"

	"github.com/project-horizon/horizon-core/services/ai/activation"
	"github.com/project-horizon/horizon-core/services/ai/knowledge"
	"github.com/project-horizon/horizon-core/services/ai/understanding"
)

// CognitiveState is the multi-concept understanding state retained before Decision.
// It represents the relevant semantic subgraph that is active for the current input,
// not a second database and not a list of ranked single nodes.
type CognitiveState struct {
	CurrentContext    []string
	FocusedNodes      []knowledge.NodeID
	SupportingNodes   []knowledge.NodeID
	CompetingNodes    []knowledge.NodeID
	UnknownNodes      []string
	ConflictNodes     []knowledge.NodeID
	ReasoningDepth    int
	ActivationHistory []activation.Result
	Confidence        float64
	CurrentGoal       string

	// Horizon 2: richer subgraph retained for interpretation building.
	ActiveNodes     []knowledge.NodeID
	Activations     map[knowledge.NodeID]float64
	Confidences     map[knowledge.NodeID]float64
	ActiveRelations []ActiveRelation
	ActivePatterns  []*knowledge.PatternSynapse
	StimulusTokens  []string

	// FSU Phase 1: functional evidence derived from active structure (not triggers).
	FunctionalSignals []FunctionalSignal
	Propositions       []Proposition
	Constraints        []Constraint
	EvidencePaths      []EvidencePath
}

// ActiveRelation is a synapse that participates in the current cognitive state.
// RelationKind is evidence only; it does not permanently assign a role.
type ActiveRelation struct {
	SourceID   knowledge.NodeID
	TargetID   knowledge.NodeID
	Kind       knowledge.RelationKind
	Weight     float64
	Confidence float64
	Inhibitory bool
	Provenance understanding.EdgeProvenance // entry evidence; not final relevance
}

// FunctionalSignal is runtime evidence derived from activated structure
// (relations, patterns, paths). It is NOT a word→function dictionary entry
// and NOT a permanent role. Kind is a descriptive label of structure found,
// not a closed set of language-act triggers.
type FunctionalSignal struct {
	Kind          string
	AnchorIDs     []knowledge.NodeID
	RelationKinds []knowledge.RelationKind
	Strength      float64
	Source        string // active_structure | interpretation | pattern
	Note          string
}


// Interpretation is a candidate internal understanding state (I).
// It is a relational structure, not a list of pre-assigned roles and not a textual answer.
type Interpretation struct {
	FocusID knowledge.NodeID

	Nodes     []knowledge.NodeID
	Relations []ActiveRelation
	Patterns  []*knowledge.PatternSynapse

	SemanticScore  float64
	RelationScore  float64
	PatternScore   float64
	HistoryScore   float64
	ContextScore   float64
	CoherenceScore float64
	ConflictScore  float64
	TotalScore     float64

	EvidenceNotes []string

	// FSU Phase 1: signals attached to this candidate understanding state.
	FunctionalSignals []FunctionalSignal

	// FSU Phase 2: internal understanding of what is asked / known.
	Propositions  []Proposition
	Constraints   []Constraint
	EvidencePaths []EvidencePath
	EvalStatus    EvalStatus

	// Epistemic isolation: the single requested claim under evaluation this cycle.
	RequestedProposition Proposition
	EvidenceEvals        []EvidenceEvaluation
}


type WorkingMemory struct {
	Activation          map[knowledge.NodeID]float64
	Hypotheses          []Hypothesis
	Candidates          []knowledge.NodeID
	Context             []string
	TemporaryRelations  map[knowledge.NodeID][]knowledge.NodeID
	TemporaryConfidence map[knowledge.NodeID]float64
	Interpretations     []Interpretation
	BestInterpretation  *Interpretation
}

type Hypothesis struct {
	Nodes      []knowledge.NodeID
	Confidence float64
	Evidence   int
	Conflicts  int
}

type PathTrace struct{ Steps []TraceStep }
type TraceStep struct {
	Stage      string
	NodeID     knowledge.NodeID
	Token      string
	Confidence float64
	Note       string
	Time       time.Time
}

func (p *PathTrace) Add(stage string, node *knowledge.ConceptNode, confidence float64, note string) {
	step := TraceStep{Stage: stage, Confidence: confidence, Note: note, Time: time.Now().UTC()}
	if node != nil {
		step.NodeID = node.ID
		step.Token = node.Token
	}
	p.Steps = append(p.Steps, step)
}

func newWorkingMemory(context []string) *WorkingMemory {
	return &WorkingMemory{
		Activation:          map[knowledge.NodeID]float64{},
		Context:             context,
		TemporaryRelations:  map[knowledge.NodeID][]knowledge.NodeID{},
		TemporaryConfidence: map[knowledge.NodeID]float64{},
	}
}

func (w *WorkingMemory) Clear() { *w = WorkingMemory{} }
