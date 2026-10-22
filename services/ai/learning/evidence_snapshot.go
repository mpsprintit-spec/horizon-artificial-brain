package learning

import (
	"errors"
	"math"
	"sort"

	"github.com/project-horizon/horizon-core/services/ai/knowledge"
)

// EvidenceLedgerSnapshot preserves accumulated evidence AND its dedup keys.
// It is runtime continuation state, not another world-knowledge store.
type EvidenceLedgerSnapshot struct {
	ByNode map[knowledge.NodeID]Evidence `json:"by_node"`
	Seen   map[knowledge.NodeID][]string `json:"seen"`
}

func (l *EvidenceLedger) Snapshot() EvidenceLedgerSnapshot {
	out := EvidenceLedgerSnapshot{ByNode: make(map[knowledge.NodeID]Evidence), Seen: make(map[knowledge.NodeID][]string)}
	if l == nil {
		return out
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	for id, value := range l.byNode {
		out.ByNode[id] = value
	}
	for id, keys := range l.seen {
		for key := range keys {
			out.Seen[id] = append(out.Seen[id], key)
		}
		sort.Strings(out.Seen[id])
	}
	return out
}

func RestoreEvidenceLedger(snapshot EvidenceLedgerSnapshot) (*EvidenceLedger, error) {
	l := NewEvidenceLedger()
	if len(snapshot.ByNode) != len(snapshot.Seen) {
		return nil, errors.New("evidence values and dedup keys disagree")
	}
	for id, value := range snapshot.ByNode {
		if id == 0 || value.IndependentSources < 1 || value.Contradictions < 0 || len(snapshot.Seen[id]) == 0 {
			return nil, errors.New("invalid evidence ledger entry")
		}
		for _, number := range []float64{value.Weight, value.Confidence, value.Reliability} {
			if math.IsNaN(number) || math.IsInf(number, 0) || number < 0 || number > 1 {
				return nil, errors.New("invalid evidence ledger value")
			}
		}
		l.byNode[id] = value
		l.seen[id] = make(map[string]struct{})
		for _, key := range snapshot.Seen[id] {
			if key == "" {
				return nil, errors.New("empty evidence dedup key")
			}
			if _, exists := l.seen[id][key]; exists {
				return nil, errors.New("duplicate evidence dedup key")
			}
			l.seen[id][key] = struct{}{}
		}
	}
	return l, nil
}
