package learning

import (
	"errors"
	"time"

	"github.com/project-horizon/horizon-core/services/ai/knowledge"
)

// PromotionEngine is the mutation boundary for learning policy decisions.
type PromotionEngine struct {
	Policy LearningPolicy
	Brain  *knowledge.Brain
}

func NewPromotionEngine(brain *knowledge.Brain, policy LearningPolicy) *PromotionEngine {
	return &PromotionEngine{Brain: brain, Policy: policy}
}

func (p *PromotionEngine) Apply(nodeID knowledge.NodeID, evidence Evidence, now time.Time) (PromotionDecision, error) {
	if p == nil || p.Brain == nil || p.Brain.Registry == nil {
		return PromotionCandidate, errors.New("promotion engine is not initialized")
	}
	decision := p.Policy.Evaluate(evidence)
	p.Brain.Lock()
	defer p.Brain.Unlock()
	node := p.Brain.Registry.GetByID(nodeID)
	if node == nil {
		return decision, errors.New("promotion target node not found")
	}

	switch decision {
	case PromotionAccepted:
		node.Importance = clampPromotion(node.Importance + 0.10*evidence.Confidence)
		node.Plasticity = clampPromotion(node.Plasticity + 0.05*evidence.Reliability)
		node.Frequency++
		node.LastActivation = now.UTC()
	case PromotionRejected:
		node.Importance = clampPromotion(node.Importance * 0.70)
		node.Plasticity = clampPromotion(node.Plasticity * 0.85)
	}
	return decision, nil
}

func clampPromotion(v float64) float64 {
	if v < 0 { return 0 }
	if v > 1 { return 1 }
	return v
}
