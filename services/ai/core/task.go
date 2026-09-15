package core

import (
	"github.com/project-horizon/horizon-core/services/ai/runtime"
	"github.com/project-horizon/horizon-core/services/ai/thinking"
)

type TaskPulse struct {
	Stimulus string
	Context  string
	Data     string
}

// PulseResult adalah hasil satu siklus berpikir penuh — dikembalikan, bukan
// dicetak langsung, supaya cangkang apa pun (CLI, suara, robot) bisa
// menampilkannya dengan caranya sendiri.
type PulseResult struct {
	Answer         string
	Concepts       []string
	Confidence     float64
	NeedsWebSearch bool
	Learned        bool
	Success        bool
	Hypotheses     []thinking.Hypothesis
	PatternEvidence []string

	// Cognitive is the typed neural result produced before presentation.
	// It is intentionally distinct from Answer and does not grant action
	// authority. Recommendation remains nil until an explicit recommendation
	// stage exists.
	Cognitive *runtime.CognitiveInterpretation

	// FSU Phase 1 observability (not answers).
	Intent              string
	Path                string
	InterpretationSource string // neural | legacy-compatibility
	FocusToken          string
	FunctionalSignals   []string
	InterpretationNotes []string
	Propositions        []string
	Constraints         []string
	EvidencePaths       []string
	EvalStatus           string
}
