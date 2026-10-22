package learning

import (
	"errors"
	"math"
	"reflect"
	"sort"

	"github.com/project-horizon/horizon-core/services/ai/knowledge"
)

// EvidenceLedgerSnapshot preserves accumulated evidence AND its dedup keys.
// It is runtime continuation state, not another world-knowledge store.
type EvidenceLedgerSnapshot struct {
	ByNode     map[knowledge.NodeID]Evidence          `json:"by_node"`
	Seen       map[knowledge.NodeID][]string          `json:"seen"`
	Provenance map[knowledge.NodeID]OutcomeProvenance `json:"provenance,omitempty"`
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
	if len(l.provenance) > 0 {
		out.Provenance = make(map[knowledge.NodeID]OutcomeProvenance, len(l.provenance))
		for id, p := range l.provenance {
			out.Provenance[id] = cloneOutcomeProvenance(p)
		}
	}
	return out
}

func RestoreEvidenceLedger(snapshot EvidenceLedgerSnapshot) (*EvidenceLedger, error) {
	l := NewEvidenceLedger()
	if len(snapshot.ByNode) != len(snapshot.Seen) {
		return nil, errors.New("evidence values and dedup keys disagree")
	}
	for id, value := range snapshot.ByNode {
		_, typed := snapshot.Provenance[id]
		if id == 0 || value.IndependentSources < 0 || !typed && value.IndependentSources < 1 || value.Contradictions < 0 || len(snapshot.Seen[id]) == 0 {
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
	for id, p := range snapshot.Provenance {
		value, ok := snapshot.ByNode[id]
		if !ok || len(p.Records) == 0 || (p.LegacyEvidence == nil) != (len(p.LegacySeen) == 0) {
			return nil, errors.New("invalid typed evidence continuation")
		}
		expected := make(map[string]struct{})
		legacyKeys := make(map[string]struct{})
		if p.LegacyEvidence != nil {
			if err := validateEvidenceValue(*p.LegacyEvidence, true); err != nil {
				return nil, err
			}
		}
		for _, key := range p.LegacySeen {
			if key == "" {
				return nil, errors.New("empty legacy evidence key")
			}
			if _, exists := legacyKeys[key]; exists {
				return nil, errors.New("duplicate legacy evidence key")
			}
			legacyKeys[key] = struct{}{}
		}
		for _, o := range p.Records {
			normalized, err := normalizeOutcomeEvidence(o)
			if err != nil || normalized != o || o.NodeID != id {
				return nil, errors.New("invalid typed outcome record")
			}
			for _, key := range p.LegacySeen {
				if key == o.RequestID+"|"+o.Source {
					return nil, errors.New("typed evidence duplicates legacy observation")
				}
			}
			key := outcomeKey(o)
			if _, exists := expected[key]; exists {
				return nil, errors.New("duplicate typed outcome identity")
			}
			expected[key] = struct{}{}
		}
		combined, _ := summarizeOutcomes(p.Records)
		if combined != value || !reflect.DeepEqual(expected, l.seen[id]) {
			return nil, errors.New("typed evidence aggregate or dedup keys disagree")
		}
		l.provenance[id] = cloneOutcomeProvenance(p)
	}
	return l, nil
}
