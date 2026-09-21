package learning

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/project-horizon/horizon-core/services/ai/knowledge"
)

//go:embed data/basic_experiences.json
var foundationalExperienceCorpus []byte

type Experience struct {
	Sequence          []string  `json:"sequence"`
	Weight            float64   `json:"weight"`
	Confidence        float64   `json:"confidence"`
	ExperienceID      string    `json:"experience_id,omitempty"`
	Source            string    `json:"source,omitempty"`
	Modality          string    `json:"modality,omitempty"`
	Timestamp         time.Time `json:"timestamp,omitempty"`
	Reliability       float64   `json:"reliability,omitempty"`
	IndependenceGroup string    `json:"independence_group,omitempty"`
	CausalLink        string    `json:"causal_link,omitempty"`
	ContradictionSet  string    `json:"contradiction_set,omitempty"`
}

func clamp01(v float64) float64 {
	if v < 0 { return 0 }
	if v > 1 { return 1 }
	return v
}

func (e Experience) evidence(now time.Time) knowledge.ExperienceEvidence {
	stamp := e.Timestamp
	if stamp.IsZero() { stamp = now }
	return knowledge.ExperienceEvidence{
		ExperienceID: e.ExperienceID, Source: e.Source, Modality: e.Modality,
		Timestamp: stamp, Reliability: clamp01(e.Reliability),
		IndependenceGroup: e.IndependenceGroup, CausalLink: e.CausalLink,
		ContradictionSet: e.ContradictionSet,
	}
}

func (l *LearningUnit) LearnExperience(experience Experience, now time.Time) {
	if l == nil || l.Kb == nil || len(experience.Sequence) == 0 { return }
	if now.IsZero() { now = time.Now().UTC() }
	now = now.UTC()
	weight := clamp01(experience.Weight)
	confidence := clamp01(experience.Confidence)
	if weight == 0 { weight = 0.5 }
	if confidence == 0 { confidence = 0.5 }

	steps := make([]knowledge.PatternStep, 0, len(experience.Sequence))
	var previous *knowledge.ConceptNode
	for i, token := range experience.Sequence {
		node := l.Kb.StoreAt(token, now)
		if node == nil { continue }
		steps = append(steps, knowledge.PatternStep{NodeID: node.ID, Position: i, Activation: 1})
		if previous != nil { l.Kb.ConnectAt(previous, node, weight, confidence, false, now) }
		previous = node
		node.LastActivation = now
	}
	if len(steps) == 0 { return }
	result := steps[len(steps)-1].NodeID
	pattern := l.Kb.Patterns.LearnTraceWithEvidence(steps, nil, result, weight, confidence, experience.evidence(now))
	if pattern != nil {
		// Recompute confidence from the retained evidence rather than repeatedly
		// compounding the prior confidence. This makes repeated same-source
		// observations saturate instead of becoming pseudo-independent evidence.
		pattern.Confidence = knowledge.CalibratedConfidence(confidence, pattern.Evidence)
	}
}

func (l *LearningUnit) LoadExperiences(path string, now time.Time) (int, error) {
	if l == nil || l.Kb == nil { return 0, fmt.Errorf("learning unit or knowledge base is nil") }
	data, err := os.ReadFile(path)
	if err != nil { return 0, err }
	return l.loadExperienceData(data, now)
}

func (l *LearningUnit) loadExperienceData(data []byte, now time.Time) (int, error) {
	var experiences []Experience
	if err := json.Unmarshal(data, &experiences); err != nil { return 0, err }
	for _, experience := range experiences { l.LearnExperience(experience, now) }
	return len(experiences), nil
}

func (l *LearningUnit) BootstrapBasicExperiences(now time.Time) (int, error) {
	return l.loadExperienceData(foundationalExperienceCorpus, now)
}


type FoundationalVectorExperience struct {
	ExperienceID string          `json:"experience_id"`
	Modality     string          `json:"modality"`
	Vectors      [][]float64     `json:"vectors"`
	Weight       float64         `json:"weight"`
	Confidence   float64         `json:"confidence"`
	Reliability  float64         `json:"reliability"`
	CausalLink   string          `json:"causal_link,omitempty"`
}

func (l *LearningUnit) LearnFoundationalVectorExperience(experience FoundationalVectorExperience, now time.Time) {
	if l == nil || l.Kb == nil || len(experience.Vectors) == 0 {
		return
	}
	if now.IsZero() {
		now = time.Now().UTC()
	}
	weight := clamp01(experience.Weight)
	if weight == 0 {
		weight = 0.6
	}
	confidence := clamp01(experience.Confidence)
	if confidence == 0 {
		confidence = 0.75
	}
	reliability := clamp01(experience.Reliability)
	if reliability == 0 {
		reliability = confidence
	}

	var previous *knowledge.ConceptNode
	steps := make([]knowledge.PatternStep, 0, len(experience.Vectors))
	for index, values := range experience.Vectors {
		vector := knowledge.NewNeuralVector(values)
		if vector.Empty() {
			continue
		}
		population, err := l.Kb.ProjectVectorPopulationAt(vector, 0.90, 4, now)
		if err != nil || len(population.Units) == 0 {
			continue
		}
		anchor := l.Kb.Registry.GetByID(population.Units[0].NodeID)
		if anchor == nil {
			continue
		}
		steps = append(steps, knowledge.PatternStep{NodeID: anchor.ID, Position: index, Activation: population.Units[0].Activation})
		if previous != nil {
			l.Kb.ConnectAt(previous, anchor, weight, confidence, false, now)
		}
		previous = anchor
	}
	if len(steps) == 0 {
		return
	}
	result := steps[len(steps)-1].NodeID
	evidence := knowledge.ExperienceEvidence{
		ExperienceID: experience.ExperienceID,
		Source: "foundational-bootstrap",
		Modality: experience.Modality,
		Timestamp: now,
		Reliability: reliability,
		IndependenceGroup: experience.ExperienceID,
		CausalLink: experience.CausalLink,
	}
	pattern := l.Kb.Patterns.LearnTraceWithEvidence(steps, nil, result, weight, confidence, evidence)
	if pattern != nil {
		pattern.Confidence = knowledge.CalibratedConfidence(confidence, pattern.Evidence)
	}
}

func (l *LearningUnit) LoadFoundationalVectorExperiences(path string, now time.Time) (int, error) {
	if l == nil || l.Kb == nil {
		return 0, fmt.Errorf("learning unit or knowledge base is nil")
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return 0, err
	}
	var experiences []FoundationalVectorExperience
	if err := json.Unmarshal(data, &experiences); err != nil {
		return 0, err
	}
	for _, experience := range experiences {
		l.LearnFoundationalVectorExperience(experience, now)
	}
	return len(experiences), nil
}
