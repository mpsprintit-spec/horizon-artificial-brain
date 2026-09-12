package learning

import (
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/project-horizon/horizon-core/services/ai/knowledge"
)

// Experience is a source-neutral observation. The brain receives only the
// observed activation sequence; the source is deliberately not part of the
// neural representation.
type Experience struct {
	Sequence   []string `json:"sequence"`
	Weight     float64  `json:"weight"`
	Confidence float64  `json:"confidence"`
}

// LearnExperience converts an experience into changes in the same persistent
// neural substrate used by every other experience source. It does not assign
// semantic relation types or execute cognitive rules.
func (l *LearningUnit) LearnExperience(experience Experience, now time.Time) {
	if l == nil || l.Kb == nil || len(experience.Sequence) == 0 {
		return
	}
	if now.IsZero() {
		now = time.Now().UTC()
	}
	weight := clamp01(experience.Weight)
	confidence := clamp01(experience.Confidence)
	if weight == 0 {
		weight = 0.5
	}
	if confidence == 0 {
		confidence = 0.5
	}

	steps := make([]knowledge.PatternStep, 0, len(experience.Sequence))
	var previous *knowledge.ConceptNode
	for i, token := range experience.Sequence {
		node := l.Kb.Store(token)
		if node == nil {
			continue
		}
		steps = append(steps, knowledge.PatternStep{
			NodeID: node.ID,
			Position: i,
			Activation: 1,
		})
		if previous != nil {
			// Direction records observed temporal succession, not semantic
			// meaning. Repeated experiences reinforce the same substrate edge.
			l.Kb.Connect(previous, node, weight, confidence, false)
		}
		previous = node
		node.LastActivation = now
		node.Frequency++
	}
	if len(steps) == 0 {
		return
	}
	result := steps[len(steps)-1].NodeID
	l.Kb.Patterns.LearnTrace(steps, nil, result, weight, confidence)
}

// LoadExperiences reads a source-neutral experience corpus and teaches it to
// the brain. The file is training experience, not persisted brain memory.
func (l *LearningUnit) LoadExperiences(path string, now time.Time) (int, error) {
	if l == nil || l.Kb == nil {
		return 0, fmt.Errorf("learning unit or knowledge base is nil")
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return 0, err
	}
	var experiences []Experience
	if err := json.Unmarshal(data, &experiences); err != nil {
		return 0, err
	}
	for _, experience := range experiences {
		l.LearnExperience(experience, now)
	}
	return len(experiences), nil
}

// BootstrapBasicExperiences teaches a compact set of foundational regularities
// intended to give the substrate an initial structured experience. These are
// data, not hard-coded semantic rules.
func (l *LearningUnit) BootstrapBasicExperiences(now time.Time) (int, error) {
	return l.LoadExperiences("services/ai/learning/data/basic_experiences.json", now)
}
