package learning

import (
	"encoding/json"
	"errors"
	"math"
	"sort"
	"strings"

	"github.com/project-horizon/horizon-core/services/ai/knowledge"
)

// OutcomeProvenance retains typed dedup identities and reported lineage.
// Legacy values remain inspectable but cannot be reclassified as independent
// evidence because historical delimiter keys did not retain that information.
type OutcomeProvenance struct {
	Records        []OutcomeEvidence `json:"records"`
	LegacyEvidence *Evidence         `json:"legacy_evidence,omitempty"`
	LegacySeen     []string          `json:"legacy_seen,omitempty"`
}

func normalizeOutcomeEvidence(o OutcomeEvidence) (OutcomeEvidence, error) {
	o.Source = strings.TrimSpace(o.Source)
	o.EpisodeID = strings.TrimSpace(o.EpisodeID)
	o.IndependenceGroup = strings.TrimSpace(o.IndependenceGroup)
	o.CausalGroup = strings.TrimSpace(o.CausalGroup)
	if o.NodeID <= 0 || strings.TrimSpace(o.RequestID) == "" || o.Source == "" {
		return o, errors.New("outcome evidence requires node, request and source")
	}
	if err := validateEvidenceValue(o.Evidence, false); err != nil {
		return o, err
	}
	// Counts supplied by a caller never establish independent support.
	o.Evidence.IndependentSources = 0
	return o, nil
}

func validateEvidenceValue(e Evidence, legacy bool) error {
	if e.Contradictions < 0 || e.IndependentSources < 0 || legacy && e.IndependentSources < 1 {
		return errors.New("invalid evidence counts")
	}
	for _, v := range []float64{e.Weight, e.Confidence, e.Reliability} {
		if math.IsNaN(v) || math.IsInf(v, 0) || v < 0 || v > 1 {
			return errors.New("invalid evidence ledger value")
		}
	}
	return nil
}

func outcomeKey(o OutcomeEvidence) string {
	b, _ := json.Marshal([2]string{o.RequestID, o.Source})
	return string(b)
}

func summarizeOutcomes(records []OutcomeEvidence) (Evidence, knowledge.EvidenceCounts) {
	var e Evidence
	lineage := make([]knowledge.ExperienceEvidence, 0, len(records))
	for _, o := range records {
		e.Weight = max(e.Weight, o.Evidence.Weight)
		e.Confidence = max(e.Confidence, o.Evidence.Confidence)
		e.Reliability = max(e.Reliability, o.Evidence.Reliability)
		e.Contradictions += o.Evidence.Contradictions
		lineage = append(lineage, knowledge.ExperienceEvidence{Source: o.Source, EpisodeID: o.EpisodeID, IndependenceGroup: o.IndependenceGroup, CausalGroup: o.CausalGroup})
	}
	counts := knowledge.CountEvidenceProvenance(lineage)
	e.IndependentSources = counts.IndependentGroups
	if e.IndependentSources > 1 {
		e.Confidence = math.Min(1, e.Confidence+.1*float64(e.IndependentSources-1))
	}
	return e, counts
}

// Record accepts one observation per typed node/request/source identity. New
// request IDs increase frequency; independent support comes only from declared
// provenance components. The first accepted identity is immutable on retry.
func (l *EvidenceLedger) Record(outcome OutcomeEvidence) (Evidence, bool, error) {
	if l == nil {
		return Evidence{}, false, errors.New("evidence ledger is not initialized")
	}
	o, err := normalizeOutcomeEvidence(outcome)
	if err != nil {
		return Evidence{}, false, err
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	p, typed := l.provenance[o.NodeID]
	if !typed {
		if len(l.seen[o.NodeID]) > 0 {
			previous := l.byNode[o.NodeID]
			p.LegacyEvidence = &previous
			for key := range l.seen[o.NodeID] {
				p.LegacySeen = append(p.LegacySeen, key)
			}
			sort.Strings(p.LegacySeen)
		}
	}
	for _, key := range p.LegacySeen {
		if key == o.RequestID+"|"+o.Source {
			return l.byNode[o.NodeID], false, nil
		}
	}
	for _, prior := range p.Records {
		if prior.RequestID == o.RequestID && prior.Source == o.Source {
			return l.byNode[o.NodeID], false, nil
		}
	}
	p.Records = append(p.Records, o)
	combined, _ := summarizeOutcomes(p.Records)
	if !typed || l.seen[o.NodeID] == nil {
		l.seen[o.NodeID] = make(map[string]struct{})
	}
	l.seen[o.NodeID][outcomeKey(o)] = struct{}{}
	l.byNode[o.NodeID] = combined
	l.provenance[o.NodeID] = p
	return combined, true, nil
}

// Counts reports typed observations separately from unclassified legacy ones.
func (l *EvidenceLedger) Counts(id knowledge.NodeID) knowledge.EvidenceCounts {
	if l == nil {
		return knowledge.EvidenceCounts{}
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	p, ok := l.provenance[id]
	if !ok {
		return knowledge.EvidenceCounts{LegacyObservations: len(l.seen[id])}
	}
	_, counts := summarizeOutcomes(p.Records)
	counts.LegacyObservations = len(p.LegacySeen)
	return counts
}

// HasObservation checks request idempotency without changing continuation.
func (l *EvidenceLedger) HasObservation(id knowledge.NodeID, request, source string) bool {
	if l == nil {
		return false
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	source = strings.TrimSpace(source)
	p, typed := l.provenance[id]
	if !typed {
		_, ok := l.seen[id][request+"|"+source]
		return ok
	}
	for _, k := range p.LegacySeen {
		if k == request+"|"+source {
			return true
		}
	}
	for _, o := range p.Records {
		if o.RequestID == request && o.Source == source {
			return true
		}
	}
	return false
}

func cloneOutcomeProvenance(p OutcomeProvenance) OutcomeProvenance {
	p.Records = append([]OutcomeEvidence(nil), p.Records...)
	sort.Slice(p.Records, func(i, j int) bool {
		a, b := p.Records[i], p.Records[j]
		if a.RequestID != b.RequestID {
			return a.RequestID < b.RequestID
		}
		return a.Source < b.Source
	})
	p.LegacySeen = append([]string(nil), p.LegacySeen...)
	sort.Strings(p.LegacySeen)
	if p.LegacyEvidence != nil {
		e := *p.LegacyEvidence
		p.LegacyEvidence = &e
	}
	return p
}
