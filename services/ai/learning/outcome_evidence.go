package learning

import (
	"errors"
	"strings"
	"sync"

	"github.com/project-horizon/horizon-core/services/ai/knowledge"
)

// OutcomeEvidence binds an externally observed result to a neural target.
// The ledger is deliberately separate from the event log: it is an evidence
// accumulator used by LearningPolicy, not a second memory store.
type OutcomeEvidence struct {
	RequestID string
	NodeID    knowledge.NodeID
	Source    string
	Evidence  Evidence
}

type EvidenceLedger struct {
	mu     sync.Mutex
	byNode map[knowledge.NodeID]Evidence
	seen   map[knowledge.NodeID]map[string]struct{}
}

func NewEvidenceLedger() *EvidenceLedger {
	return &EvidenceLedger{
		byNode: make(map[knowledge.NodeID]Evidence),
		seen:   make(map[knowledge.NodeID]map[string]struct{}),
	}
}

// Record merges an outcome exactly once per node/request/source tuple.
// Repeated delivery of the same outcome cannot manufacture independent
// evidence.
func (l *EvidenceLedger) Record(outcome OutcomeEvidence) (Evidence, error) {
	if l == nil {
		return Evidence{}, errors.New("evidence ledger is not initialized")
	}
	if outcome.RequestID == "" || outcome.NodeID == "" {
		return Evidence{}, errors.New("outcome evidence requires request ID and node ID")
	}
	if strings.TrimSpace(outcome.Source) == "" {
		return Evidence{}, errors.New("outcome evidence source is required")
	}

	l.mu.Lock()
	defer l.mu.Unlock()

	if l.seen[outcome.NodeID] == nil {
		l.seen[outcome.NodeID] = make(map[string]struct{})
	}
	key := outcome.RequestID + "|" + strings.TrimSpace(outcome.Source)
	if _, exists := l.seen[outcome.NodeID][key]; exists {
		return l.byNode[outcome.NodeID], nil
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
	return current, nil
}

// MergeOutcomeEvidence keeps the strongest direct evidence while allowing
// independent corroboration to raise confidence in a bounded, explicit way.
func MergeOutcomeEvidence(a, b Evidence) Evidence {
	independent := a.IndependentSources
	if independent < 1 {
		independent = 1
	}
	confidence := max(a.Confidence, b.Confidence)
	if independent >= 1 {
		confidence += 0.10
		if confidence > 1 {
			confidence = 1
		}
	}
	reliability := max(a.Reliability, b.Reliability)
	return Evidence{
		Weight:              max(a.Weight, b.Weight),
		Confidence:          confidence,
		Reliability:         reliability,
		IndependentSources:  independent,
		Contradictions:      a.Contradictions + b.Contradictions,
	}
}
