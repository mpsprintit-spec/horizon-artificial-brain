package understanding

import (
	"sort"
	"strings"

	"github.com/project-horizon/horizon-core/services/ai/activation"
	"github.com/project-horizon/horizon-core/services/ai/knowledge"
)

type CognitiveRepresentation struct {
	DominantNodes       []knowledge.NodeID
	SupportingNodes     []knowledge.NodeID
	CompetingNodes      []knowledge.NodeID
	UnknownTokens       []string
	ConflictNodes       []knowledge.NodeID
	HiddenRelations     [][2]knowledge.NodeID
	Confidence          float64
	Resonance           float64
	ActivationSignature map[knowledge.NodeID]float64
}

type Trace interface {
	Add(stage string, node *knowledge.ConceptNode, confidence float64, note string)
}

type Engine struct{ Memory *knowledge.KnowledgeBase }

func NewEngine(memory *knowledge.KnowledgeBase) *Engine {
	return &Engine{Memory: memory}
}

func (u *Engine) Understand(result activation.Result, stimulus []string, trace Trace, includeStimulus bool) CognitiveRepresentation {
	rep := CognitiveRepresentation{ActivationSignature: result.Activations, Resonance: result.Resonance}
	for _, token := range stimulus {
		if u.Memory.Fetch(token) == nil {
			rep.UnknownTokens = append(rep.UnknownTokens, token)
		}
	}
	for _, node := range result.RankedNodes {
		level := result.Activations[node.ID]
		conf := result.Confidence[node.ID]
		if isAffixMarker(node.Token) {
			continue
		}
		// Horizon 2 P0: Dominant/Supporting are computational organization labels,
		// not semantic relevance boundaries. Low local activation nodes stay available
		// so bridges can participate in interpretation. Final relevance is decided by coherence.
		if level >= 0.40 && len(rep.DominantNodes) < 12 {
			rep.DominantNodes = append(rep.DominantNodes, node.ID)
			trace.Add("Understanding", node, conf, "high activation in available subgraph")
		} else if level >= 0.08 && len(rep.SupportingNodes) < 32 {
			rep.SupportingNodes = append(rep.SupportingNodes, node.ID)
			trace.Add("Understanding", node, conf, "available subgraph member")
		}
		if hasStrongInhibition(node) {
			rep.ConflictNodes = append(rep.ConflictNodes, node.ID)
		}
		for _, syn := range node.OutboundAll() {
			if syn.Inhibitory && result.Activations[syn.TargetID] > 0.18 {
				rep.CompetingNodes = appendUniqueID(rep.CompetingNodes, syn.TargetID)
			}
		}
	}
	if includeStimulus {
		for _, token := range stimulus {
			if n := u.Memory.Fetch(token); n != nil && !isAffixMarker(n.Token) {
				if !ContainsID(rep.DominantNodes, n.ID) && !ContainsID(rep.SupportingNodes, n.ID) {
					rep.SupportingNodes = append(rep.SupportingNodes, n.ID)
					trace.Add("Understanding", n, result.Confidence[n.ID], "stimulus retention")
				}
			}
		}
	}
	rep.HiddenRelations = u.hiddenRelations(rep.DominantNodes, rep.SupportingNodes)
	rep.Confidence = clamp01((result.Resonance + averageConfidence(result, rep.DominantNodes, rep.SupportingNodes)) / 2)
	return rep
}

// EdgeProvenance is runtime evidence of how an edge entered the neighborhood.
// It is not a permanent role and not a relevance decision.
type EdgeProvenance string

const (
	ProvDirect     EdgeProvenance = "direct"     // incident on stimulus structural core
	ProvNear       EdgeProvenance = "near"       // outbound chain from stimulus core
	ProvCollateral EdgeProvenance = "collateral" // reverse fan-in / siblings of a reached node
	ProvBridge     EdgeProvenance = "bridge"     // connects two parts of stimulus-local structure
	ProvUnknown    EdgeProvenance = "unknown"
	// Legacy aliases kept so older call sites compile during transition.
	ProvOutbound   = ProvNear
	ProvReverse    = ProvCollateral
	ProvActivation = ProvUnknown
)

type ActiveEdge struct {
	SourceID   knowledge.NodeID
	TargetID   knowledge.NodeID
	Kind       knowledge.RelationKind
	Weight     float64
	Confidence float64
	Inhibitory bool
	Provenance EdgeProvenance // how this edge entered available knowledge
}

func (u *Engine) CollectActiveRelations(rep CognitiveRepresentation, result activation.Result) []ActiveEdge {
	nodeSet := map[knowledge.NodeID]bool{}
	for _, id := range rep.DominantNodes {
		nodeSet[id] = true
	}
	for _, id := range rep.SupportingNodes {
		nodeSet[id] = true
	}
	var out []ActiveEdge
	seen := map[[2]knowledge.NodeID]bool{}
	for id := range nodeSet {
		n := u.Memory.Registry.GetByID(id)
		if n == nil {
			continue
		}
		for _, s := range n.OutboundAll() {
			tid := s.TargetID
			// Keep inhibitory edges among available nodes as conflict evidence.
			// Keep weak supportive edges if either end is available (bridge rule).
			if !nodeSet[tid] && !s.Inhibitory && result.Activations[tid] < 0.05 && s.Confidence < 0.20 {
				continue
			}
			key := [2]knowledge.NodeID{id, tid}
			if seen[key] {
				continue
			}
			seen[key] = true
			out = append(out, ActiveEdge{
				SourceID: id, TargetID: tid, Kind: s.Kind,
				Weight: s.Weight, Confidence: s.Confidence, Inhibitory: s.Inhibitory,
			})
			nodeSet[tid] = true
		}
	}
	return out
}


// SemanticNeighborhood is the structurally relevant active knowledge available
// for reasoning — not the full KnowledgeBase and not top-activation-only.
// Dominant/Supporting remain organizational metadata only.
type SemanticNeighborhood struct {
	Nodes     []knowledge.NodeID
	Edges     []ActiveEdge
	Stimulus  []knowledge.NodeID
	HopDepth  int
}

const (
	maxNeighborhoodHops  = 2
	maxNeighborhoodNodes = 48 // computational ceiling only
)

// BuildSemanticNeighborhood expands from stimulus tokens along synapses
// (exploration is bidirectional; edge direction remains evidence).
// Low-activation bridge nodes reachable by structure are retained.
// Ceiling ranking uses connectivity + relation support + activation + context —
// not activation alone. This is candidate availability, not I* scoring.
func (u *Engine) BuildSemanticNeighborhood(
	result activation.Result,
	stimulus []string,
	contextTokens []string,
) SemanticNeighborhood {
	nb := SemanticNeighborhood{HopDepth: maxNeighborhoodHops}
	nodeSet := map[knowledge.NodeID]bool{}
	var frontier []knowledge.NodeID

	for _, tok := range stimulus {
		if n := u.Memory.Fetch(tok); n != nil {
			if !nodeSet[n.ID] {
				nodeSet[n.ID] = true
				nb.Stimulus = append(nb.Stimulus, n.ID)
				frontier = append(frontier, n.ID)
			}
		}
	}
	for _, tok := range contextTokens {
		if n := u.Memory.Fetch(tok); n != nil {
			if !nodeSet[n.ID] {
				nodeSet[n.ID] = true
				frontier = append(frontier, n.ID)
			}
		}
	}
	// Ranked activation is NOT an automatic neighborhood seed.
	// Unrelated high-activation nodes must not enter available structure.

	// Reach kind: stimulus = direct origin; outbound expansion = near; reverse = collateral.
	reach := map[knowledge.NodeID]EdgeProvenance{}
	for _, id := range nb.Stimulus {
		reach[id] = ProvDirect
	}

	reverse := map[knowledge.NodeID][]knowledge.NodeID{}
	for _, n := range u.Memory.Registry.Nodes() {
		for tid := range n.Synapses { // targets only
			reverse[tid] = append(reverse[tid], n.ID)
		}
	}
	for hop := 0; hop < maxNeighborhoodHops && len(frontier) > 0; hop++ {
		var next []knowledge.NodeID
		for _, id := range frontier {
			n := u.Memory.Registry.GetByID(id)
			if n == nil {
				continue
			}
			for tid := range n.Synapses { // targets only
				if !nodeSet[tid] {
					nodeSet[tid] = true
					if reach[id] == ProvDirect || reach[id] == ProvNear {
						reach[tid] = ProvNear
					} else {
						reach[tid] = ProvNear
					}
					next = append(next, tid)
				}
			}
			for _, src := range reverse[id] {
				if !nodeSet[src] {
					nodeSet[src] = true
					reach[src] = ProvCollateral
					next = append(next, src)
				}
			}
		}
		frontier = next
	}

	// Edge collection: ONLY edges among nodes already in nodeSet.
	// Exploration has finished; max hop is meaningful. Do not grow the set here.
	stimCore := map[knowledge.NodeID]bool{}
	for _, id := range nb.Stimulus {
		stimCore[id] = true
	}
	// Bridge nodes: on a path between two distinct stimulus nodes (structural necessity).
	bridgeNode := map[knowledge.NodeID]bool{}
	stimList := append([]knowledge.NodeID{}, nb.Stimulus...)
	if len(stimList) >= 2 {
		// BFS parents within nodeSet from each stimulus; nodes that appear on paths between stimuli.
		for _, s0 := range stimList {
			parent := map[knowledge.NodeID]knowledge.NodeID{}
			q := []knowledge.NodeID{s0}
			seenN := map[knowledge.NodeID]bool{s0: true}
			for len(q) > 0 {
				cur := q[0]
				q = q[1:]
				n := u.Memory.Registry.GetByID(cur)
				if n == nil {
					continue
				}
				for tid := range n.Synapses { // targets only
					if !nodeSet[tid] || seenN[tid] {
						continue
					}
					seenN[tid] = true
					parent[tid] = cur
					q = append(q, tid)
				}
				// also reverse within set
				for _, n2 := range u.Memory.Registry.Nodes() {
					if !nodeSet[n2.ID] {
						continue
					}
					if _, ok := n2.Synapses[cur]; ok && !seenN[n2.ID] {
						seenN[n2.ID] = true
						parent[n2.ID] = cur
						q = append(q, n2.ID)
					}
				}
			}
			for _, s1 := range stimList {
				if s1 == s0 || !seenN[s1] {
					continue
				}
				// walk parents from s1 to s0
				x := s1
				for x != s0 {
					p := parent[x]
					if p == 0 && x != s0 {
						break
					}
					if p != s0 && p != s1 {
						bridgeNode[p] = true
					}
					x = p
					if x == s0 {
						break
					}
				}
			}
		}
	}
	for id := range bridgeNode {
		reach[id] = ProvBridge
	}

	seen := map[[2]knowledge.NodeID]bool{}
	for id := range nodeSet {
		n := u.Memory.Registry.GetByID(id)
		if n == nil {
			continue
		}
		for _, s := range n.OutboundAll() {
			tid := s.TargetID
			if !nodeSet[tid] {
				continue // do not expand neighborhood during edge collection
			}
			key := [2]knowledge.NodeID{id, tid}
			if seen[key] {
				continue
			}
			seen[key] = true
			prov := ProvNear
			if stimCore[id] || stimCore[tid] {
				prov = ProvDirect
			} else if reach[id] == ProvCollateral || reach[tid] == ProvCollateral {
				prov = ProvCollateral
			}
			if bridgeNode[id] || bridgeNode[tid] {
				prov = ProvBridge
			} else if prov != ProvCollateral && !stimCore[id] && !stimCore[tid] {
				if (reach[id] == ProvNear || reach[id] == ProvDirect || reach[id] == ProvBridge) &&
					(reach[tid] == ProvNear || reach[tid] == ProvDirect || reach[tid] == ProvBridge) {
					// intermediate hop on stimulus-local chain
					if reach[id] == ProvBridge || reach[tid] == ProvBridge {
						prov = ProvBridge
					}
				}
			}
			nb.Edges = append(nb.Edges, ActiveEdge{
				SourceID: id, TargetID: tid, Kind: s.Kind,
				Weight: s.Weight, Confidence: s.Confidence, Inhibitory: s.Inhibitory,
				Provenance: prov,
			})
		}
	}

	// Rebuild node list; computational ceiling with structural priority.
	type ns struct {
		id knowledge.NodeID
		sc float64
	}
	ctxSet := map[string]bool{}
	for _, tok := range contextTokens {
		ctxSet[tok] = true
	}
	degree := map[knowledge.NodeID]int{}
	for _, e := range nb.Edges {
		if !e.Inhibitory {
			degree[e.SourceID]++
			degree[e.TargetID]++
		}
	}
	var ranked []ns
	for id := range nodeSet {
		act := result.Activations[id] * result.Confidence[id]
		if act == 0 {
			act = 0.05
		}
		conn := clamp01(float64(degree[id]) / 6.0)
		ctx := 0.0
		if n := u.Memory.Registry.GetByID(id); n != nil && ctxSet[n.Token] {
			ctx = 0.2
		}
		// Bridge protection: nodes that sit on paths between stimulus nodes get a floor score
		// so low-activation bridges are not dropped by high-degree distractors at the ceiling.
		bridgeBoost := 0.0
		if reach[id] == ProvBridge || reach[id] == ProvNear {
			if degree[id] >= 2 && act < 0.3 {
				bridgeBoost = 0.35
			}
		}
		if stimCore[id] {
			bridgeBoost = 0.5
		}
		sc := 0.30*act + 0.35*conn + 0.15*ctx + bridgeBoost
		ranked = append(ranked, ns{id, sc})
	}
	sort.Slice(ranked, func(i, j int) bool { return ranked[i].sc > ranked[j].sc })
	if len(ranked) > maxNeighborhoodNodes {
		ranked = ranked[:maxNeighborhoodNodes]
		keep := map[knowledge.NodeID]bool{}
		for _, r := range ranked {
			keep[r.id] = true
		}
		var edges []ActiveEdge
		for _, e := range nb.Edges {
			if keep[e.SourceID] && keep[e.TargetID] {
				edges = append(edges, e)
			}
		}
		nb.Edges = edges
		nodeSet = keep
	}
	for _, r := range ranked {
		if nodeSet[r.id] {
			nb.Nodes = append(nb.Nodes, r.id)
		}
	}
	return nb
}

func (u *Engine) hiddenRelations(dominant, supporting []knowledge.NodeID) [][2]knowledge.NodeID {
	var out [][2]knowledge.NodeID
	for _, a := range dominant {
		na := u.Memory.Registry.GetByID(a)
		if na == nil {
			continue
		}
		for _, b := range supporting {
			if a == b {
				continue
			}
			if _, ok := na.Synapses[b]; ok {
				out = append(out, [2]knowledge.NodeID{a, b})
			}
		}
	}
	return out
}

func hasStrongInhibition(node *knowledge.ConceptNode) bool {
	for _, synapse := range node.OutboundAll() {
		if synapse.Inhibitory && synapse.Weight >= 0.6 {
			return true
		}
	}
	return false
}

func appendUniqueID(ids []knowledge.NodeID, id knowledge.NodeID) []knowledge.NodeID {
	if ContainsID(ids, id) {
		return ids
	}
	return append(ids, id)
}

func ContainsID(ids []knowledge.NodeID, id knowledge.NodeID) bool {
	for _, v := range ids {
		if v == id {
			return true
		}
	}
	return false
}

func averageConfidence(result activation.Result, groups ...[]knowledge.NodeID) float64 {
	var sum float64
	var count int
	for _, g := range groups {
		for _, id := range g {
			sum += result.Confidence[id] * result.Activations[id]
			count++
		}
	}
	if count == 0 {
		return 0
	}
	return sum / float64(count)
}

func clamp01(v float64) float64 {
	if v < 0 {
		return 0
	}
	if v > 1 {
		return 1
	}
	return v
}

func isAffixMarker(token string) bool {
	return strings.HasPrefix(token, "-") || strings.HasSuffix(token, "-")
}
