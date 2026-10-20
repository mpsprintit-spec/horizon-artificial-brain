package learning

import (
	"errors"
	"strings"
	"sync"

	"github.com/project-horizon/horizon-core/services/ai/knowledge"
)

type OutcomeEvidence struct {
	RequestID         string
	NodeID            knowledge.NodeID
	Source            string
	Evidence          Evidence
	EpisodeID         string `json:"episode_id,omitempty"`
	IndependenceGroup string `json:"independence_group,omitempty"`
	CausalGroup       string `json:"causal_group,omitempty"`
}

type EvidenceLedger struct {
	mu         sync.Mutex
	byNode     map[knowledge.NodeID]Evidence
	seen       map[knowledge.NodeID]map[string]struct{}
	provenance map[knowledge.NodeID]OutcomeProvenance
}

func NewEvidenceLedger() *EvidenceLedger {
	return &EvidenceLedger{byNode: make(map[knowledge.NodeID]Evidence), seen: make(map[knowledge.NodeID]map[string]struct{}), provenance: make(map[knowledge.NodeID]OutcomeProvenance)}
}

// RecordLegacy is the immutable pre-H15 request/source accounting law. It is
// used only by historical dynamics, never by current serving.
func (l *EvidenceLedger) RecordLegacy(outcome OutcomeEvidence) (Evidence, bool, error) {
	if l == nil {
		return Evidence{}, false, errors.New("evidence ledger is not initialized")
	}
	if outcome.RequestID == "" || outcome.NodeID == 0 {
		return Evidence{}, false, errors.New("outcome evidence requires request ID and node ID")
	}
	if strings.TrimSpace(outcome.Source) == "" {
		return Evidence{}, false, errors.New("outcome evidence source is required")
	}

	l.mu.Lock()
	defer l.mu.Unlock()
	if _, exists := l.provenance[outcome.NodeID]; exists {
		return Evidence{}, false, errors.New("cannot downgrade typed outcome provenance to legacy dynamics")
	}
	if l.seen[outcome.NodeID] == nil {
		l.seen[outcome.NodeID] = make(map[string]struct{})
	}
	key := outcome.RequestID + "|" + strings.TrimSpace(outcome.Source)
	if _, exists := l.seen[outcome.NodeID][key]; exists {
		return l.byNode[outcome.NodeID], false, nil
	}
	l.seen[outcome.NodeID][key] = struct{}{}

	current := l.byNode[outcome.NodeID]
	if current.IndependentSources == 0 {
		current = outcome.Evidence
		if current.IndependentSources == 0 {
			current.IndependentSources = 1
		}
	} else {
		current = MergeOutcomeEvidence(current, outcome.Evidence)
		current.IndependentSources++
	}
	l.byNode[outcome.NodeID] = current
	return current, true, nil
}

func MergeOutcomeEvidence(a, b Evidence) Evidence {
	independent := a.IndependentSources
	if independent < 1 {
		independent = 1
	}
	confidence := max(a.Confidence, b.Confidence) + 0.10
	if confidence > 1 {
		confidence = 1
	}
	return Evidence{Weight: max(a.Weight, b.Weight), Confidence: confidence, Reliability: max(a.Reliability, b.Reliability), IndependentSources: independent, Contradictions: a.Contradictions + b.Contradictions}
}
