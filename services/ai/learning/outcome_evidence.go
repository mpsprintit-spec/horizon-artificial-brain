package learning

import (
	"errors"
	"strings"
	"sync"

	"github.com/project-horizon/horizon-core/services/ai/knowledge"
)

type OutcomeEvidence struct {
	RequestID string
	NodeID knowledge.NodeID
	Source string
	Evidence Evidence
}

type EvidenceLedger struct {
	mu sync.Mutex
	byNode map[knowledge.NodeID]Evidence
	seen map[knowledge.NodeID]map[string]struct{}
}

func NewEvidenceLedger() *EvidenceLedger {
	return &EvidenceLedger{byNode: make(map[knowledge.NodeID]Evidence), seen: make(map[knowledge.NodeID]map[string]struct{})}
}

// Record merges an outcome exactly once per node/request/source tuple. The
// boolean reports whether new evidence was accepted into the ledger.
func (l *EvidenceLedger) Record(outcome OutcomeEvidence) (Evidence, bool, error) {
	if l == nil { return Evidence{}, false, errors.New("evidence ledger is not initialized") }
	if outcome.RequestID == "" || outcome.NodeID == 0 { return Evidence{}, false, errors.New("outcome evidence requires request ID and node ID") }
	if strings.TrimSpace(outcome.Source) == "" { return Evidence{}, false, errors.New("outcome evidence source is required") }

	l.mu.Lock()
	defer l.mu.Unlock()
	if l.seen[outcome.NodeID] == nil { l.seen[outcome.NodeID] = make(map[string]struct{}) }
	key := outcome.RequestID + "|" + strings.TrimSpace(outcome.Source)
	if _, exists := l.seen[outcome.NodeID][key]; exists { return l.byNode[outcome.NodeID], false, nil }
	l.seen[outcome.NodeID][key] = struct{}{}

	current := l.byNode[outcome.NodeID]
	if current.IndependentSources == 0 {
		current = outcome.Evidence
		if current.IndependentSources == 0 { current.IndependentSources = 1 }
	} else {
		current = MergeOutcomeEvidence(current, outcome.Evidence)
		current.IndependentSources++
	}
	l.byNode[outcome.NodeID] = current
	return current, true, nil
}

func MergeOutcomeEvidence(a, b Evidence) Evidence {
	independent := a.IndependentSources
	if independent < 1 { independent = 1 }
	confidence := max(a.Confidence, b.Confidence) + 0.10
	if confidence > 1 { confidence = 1 }
	return Evidence{Weight: max(a.Weight, b.Weight), Confidence: confidence, Reliability: max(a.Reliability, b.Reliability), IndependentSources: independent, Contradictions: a.Contradictions + b.Contradictions}
}
