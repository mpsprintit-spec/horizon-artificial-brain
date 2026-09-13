package knowledge

// ReconsolidateEvidence adjusts a pattern's confidence when evidence assigned
// to the same contradiction set is present. History is retained in Evidence;
// only the current confidence state is adapted.
func (p *PatternSynapse) ReconsolidateEvidence() {
	if p == nil || len(p.Evidence) == 0 {
		return
	}
	sets := make(map[string]struct{})
	for _, e := range p.Evidence {
		if e.ContradictionSet != "" {
			sets[e.ContradictionSet] = struct{}{}
		}
	}
	if len(sets) == 0 {
		return
	}

	var load float64
	var count float64
	for _, e := range p.Evidence {
		if e.ContradictionSet == "" {
			continue
		}
		reliability := e.Reliability
		if reliability <= 0 {
			reliability = 0.5
		}
		if reliability > 1 {
			reliability = 1
		}
		load += reliability
		count++
	}
	if count == 0 {
		return
	}

	// Contradiction is adaptation, not deletion. A bounded reduction prevents
	// one contradictory observation from erasing an established pattern.
	load /= count
	p.Confidence = clamp01(p.Confidence * (1 - 0.35*load))
}

// ReconsolidateAll applies contradiction adaptation to every learned pattern
// while preserving its complete evidence history.
func (p *PatternIndex) ReconsolidateAll() {
	if p == nil {
		return
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	for _, pattern := range p.patterns {
		pattern.ReconsolidateEvidence()
	}
}
