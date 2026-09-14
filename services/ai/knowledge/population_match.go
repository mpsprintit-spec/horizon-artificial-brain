package knowledge

import "sort"

// PopulationMatch is a tolerant structural match between an observed sparse
// population and a learned distributed pattern. Coverage makes the pattern
// robust to partial observation or ablation of individual units.
type PopulationMatch struct {
	Pattern  *PatternSynapse
	Coverage float64
	Score    float64
}

// MatchPopulation returns patterns whose distributed membership is covered by
// the observed population at or above minCoverage. It deliberately uses only
// substrate membership and learned strength; it does not attach semantic
// relation labels to the population.
func (p *PatternIndex) MatchPopulation(candidateIDs []NodeID, minCoverage float64) []PopulationMatch {
	if p == nil || len(candidateIDs) == 0 {
		return nil
	}
	minCoverage = clamp01(minCoverage)
	observed := make(map[NodeID]struct{}, len(candidateIDs))
	for _, id := range candidateIDs {
		observed[id] = struct{}{}
	}

	p.mu.RLock()
	defer p.mu.RUnlock()
	matches := make([]PopulationMatch, 0)
	for _, pattern := range p.patterns {
		if pattern == nil || len(pattern.Members) == 0 {
			continue
		}
		covered := 0
		for _, id := range pattern.Members {
			if _, ok := observed[id]; ok {
				covered++
			}
		}
		coverage := float64(covered) / float64(len(pattern.Members))
		if coverage < minCoverage {
			continue
		}
		strength := clamp01(pattern.Weight) * clamp01(pattern.Confidence)
		matches = append(matches, PopulationMatch{
			Pattern:  pattern,
			Coverage: coverage,
			Score:    coverage*.8 + strength*.2,
		})
	}

	sort.Slice(matches, func(i, j int) bool {
		if matches[i].Score == matches[j].Score {
			return matches[i].Pattern.ID < matches[j].Pattern.ID
		}
		return matches[i].Score > matches[j].Score
	})
	return matches
}
