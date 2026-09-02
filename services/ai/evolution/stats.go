package evolution

import "github.com/project-horizon/horizon-core/services/ai/knowledge"

type NodeStats struct {
	IsAIncoming           int
	IsAOutgoing           int
	DescriptiveIncoming   int
	DistinctRelationKinds int
	OutDegree             int
	InDegree              int
	AverageConfidence     float64
}

func ComputeStats(kb *knowledge.KnowledgeBase, target *knowledge.ConceptNode) NodeStats {
	var stats NodeStats
	kinds := map[knowledge.RelationKind]bool{}
	var confSum float64
	var confCount int

	for _, s := range target.OutboundAll() {
		stats.OutDegree++
		kinds[s.Kind] = true
		confSum += s.Confidence
		confCount++
		if s.Kind == knowledge.RelationIsA {
			stats.IsAOutgoing++
		}
	}

	seenIsA := map[knowledge.NodeID]bool{}
	seenDescriptive := map[knowledge.NodeID]bool{}
	for _, node := range kb.Registry.Nodes() {
		if node.ID == target.ID {
			continue
		}
		list := node.SynapsesTo(target.ID)
		if len(list) == 0 {
			continue
		}
		for _, s := range list {
			stats.InDegree++
			kinds[s.Kind] = true
			confSum += s.Confidence
			confCount++
			switch s.Kind {
			case knowledge.RelationIsA:
				if !seenIsA[node.ID] {
					seenIsA[node.ID] = true
					stats.IsAIncoming++
				}
			case knowledge.RelationHas, knowledge.RelationPartWhole:
				if !seenDescriptive[node.ID] {
					seenDescriptive[node.ID] = true
					stats.DescriptiveIncoming++
				}
			}
		}
	}

	stats.DistinctRelationKinds = len(kinds)
	if confCount > 0 {
		stats.AverageConfidence = confSum / float64(confCount)
	}
	return stats
}
