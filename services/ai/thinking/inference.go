package thinking

import "github.com/project-horizon/horizon-core/services/ai/knowledge"

const maxInferenceDepth = 3

// InferredFact adalah kesimpulan yang TIDAK pernah diajarkan langsung --
// ditemukan dengan menelusuri rantai IS_A. Confidence-nya selalu lebih
// rendah dari fakta yang diajarkan langsung, dan turun tiap langkah
// tambahan, supaya Horizon tidak mengaku yakin penuh atas sesuatu yang
// cuma disimpulkan tidak langsung.
type InferredFact struct {
	Kind       knowledge.RelationKind
	TargetID   knowledge.NodeID
	Confidence float64
	PathLength int
}

// InferFromIsAChain menelusuri rantai IS_A dari sebuah node, mengumpulkan:
// (a) leluhur IS_A tidak langsung (Kucing->Hewan->MakhlukHidup), dan
// (b) sifat yang diwarisi dari leluhur itu lewat relasi lain (Bernapas, dst).
func InferFromIsAChain(kb *knowledge.KnowledgeBase, start *knowledge.ConceptNode) []InferredFact {
	var out []InferredFact
	visited := map[knowledge.NodeID]bool{start.ID: true}
	current := start
	confidence := 1.0

	for depth := 1; depth <= maxInferenceDepth; depth++ {
		var next *knowledge.ConceptNode
		var nextSynapse *knowledge.Synapse
		for _, s := range current.OutboundAll() {
			id := s.TargetID
			if s.Kind == knowledge.RelationIsA && !visited[id] {
				if n := kb.Registry.GetByID(id); n != nil {
					next = n
					nextSynapse = s
					break
				}
			}
		}
		if next == nil {
			break
		}
		confidence *= nextSynapse.Confidence * 0.85 // tiap langkah tambahan mengurangi keyakinan
		visited[next.ID] = true

		if depth > 1 { // langkah pertama (leluhur langsung) sudah pasti diketahui, bukan hasil inferensi
			out = append(out, InferredFact{Kind: knowledge.RelationIsA, TargetID: next.ID, Confidence: confidence, PathLength: depth})
		}

		for _, s := range next.OutboundAll() {
			id := s.TargetID
			if s.Kind == knowledge.RelationIsA || s.Kind == knowledge.RelationAffix {
				continue
			}
			inheritedConf := confidence * s.Confidence
			out = append(out, InferredFact{Kind: s.Kind, TargetID: id, Confidence: inheritedConf, PathLength: depth})
		}

		current = next
	}
	return out
}
