package thinking

import (
	"fmt"
	"sort"

	"github.com/project-horizon/horizon-core/services/ai/activation"
	"github.com/project-horizon/horizon-core/services/ai/knowledge"
	"github.com/project-horizon/horizon-core/services/ai/understanding"
)

const (
	maxCandidateInterpretations = 12
	maxExpansionDepth           = 3
)

// BuildInterpretations builds multiple candidate cognitive states from available
// neighborhood knowledge. Provenance is entry evidence; coherence decides relevance.
// Strategies (not fixed answers):
//   - focus-anchored: expand from candidate focus along outbound/near structure
//   - connected: allow limited reverse when it bridges focus-local structure
//   - broad/union: full neighborhood (siblings stay available for comparison)
func BuildInterpretations(
	kb *knowledge.KnowledgeBase,
	rep understanding.CognitiveRepresentation,
	result activation.Result,
	neighborhood understanding.SemanticNeighborhood,
	contextTokens []string,
) []Interpretation {
	available := neighborhood.Nodes
	edges := neighborhood.Edges
	if len(available) == 0 {
		available = append(append([]knowledge.NodeID{}, rep.DominantNodes...), rep.SupportingNodes...)
	}
	if len(available) == 0 {
		return nil
	}

	availSet := map[knowledge.NodeID]bool{}
	for _, id := range available {
		availSet[id] = true
	}

	bySource := map[knowledge.NodeID][]understanding.ActiveEdge{}
	byTarget := map[knowledge.NodeID][]understanding.ActiveEdge{}
	for _, e := range edges {
		bySource[e.SourceID] = append(bySource[e.SourceID], e)
		byTarget[e.TargetID] = append(byTarget[e.TargetID], e)
	}
	if len(edges) == 0 && kb != nil {
		for id := range availSet {
			n := kb.Registry.GetByID(id)
			if n == nil {
				continue
			}
			for _, s := range n.OutboundAll() {
			tid := s.TargetID
				if !availSet[tid] {
					continue
				}
				e := understanding.ActiveEdge{
					SourceID: id, TargetID: tid, Kind: s.Kind,
					Weight: s.Weight, Confidence: s.Confidence, Inhibitory: s.Inhibitory,
					Provenance: understanding.ProvUnknown,
				}
				bySource[id] = append(bySource[id], e)
				byTarget[tid] = append(byTarget[tid], e)
			}
		}
	}

	stimSet := map[knowledge.NodeID]bool{}
	for _, id := range neighborhood.Stimulus {
		stimSet[id] = true
	}
	// Also treat contextTokens matched to nodes as soft stimulus anchors.
	for _, tok := range contextTokens {
		if n := kb.Fetch(tok); n != nil {
			stimSet[n.ID] = true
		}
	}

	var candidates []Interpretation
	seenSig := map[string]bool{}
	addCand := func(interp Interpretation) {
		if len(interp.Nodes) == 0 {
			return
		}
		sig := interpretationSignature(interp)
		if seenSig[sig] {
			return
		}
		seenSig[sig] = true
		// Focus emergent from structure of THIS candidate (not all input tokens equal).
		interp.FocusID = chooseFocusFromStructure(kb, &interp, result, contextTokens)
		scoreInterpretation(kb, &interp, result, contextTokens, stimSet)
		candidates = append(candidates, interp)
	}

	// Candidate focuses: stimulus nodes first (if they exist in available), then other seeds.
	// Not every input token is equal: association-only stimulus tokens are weak seeds only.
	focusSeeds := selectFocusSeeds(kb, available, result, bySource, stimSet)
	for _, seed := range focusSeeds {
		// Narrow / connected: outbound + bridges from this focus seed.
		addCand(buildFocusAnchored(kb, seed, bySource, byTarget, availSet, result, false))
		// Connected with limited reverse for true bridges.
		addCand(buildFocusAnchored(kb, seed, bySource, byTarget, availSet, result, true))
	}

	// Broad union — siblings remain available for comparison, not auto-selected.
	addCand(buildConnectedUnion(kb, available, bySource, byTarget, result))

	sort.Slice(candidates, func(i, j int) bool {
		return candidates[i].TotalScore > candidates[j].TotalScore
	})
	if len(candidates) > maxCandidateInterpretations {
		candidates = candidates[:maxCandidateInterpretations]
	}
	return candidates
}

func selectFocusSeeds(
	kb *knowledge.KnowledgeBase,
	available []knowledge.NodeID,
	result activation.Result,
	bySource map[knowledge.NodeID][]understanding.ActiveEdge,
	stimSet map[knowledge.NodeID]bool,
) []knowledge.NodeID {
	// Inter-stimulus typed edges: A REL B where A and B are both in stimulus.
	// Endpoints of those edges are stronger focus seeds than a middle token that
	// only has external typed structure (e.g. a function word with many of its own links).
	interStimSource := map[knowledge.NodeID]int{}
	interStimTarget := map[knowledge.NodeID]int{}
	for id := range stimSet {
		for _, e := range bySource[id] {
			if e.Inhibitory {
				continue
			}
			if e.Kind == knowledge.RelationAssociation || e.Kind == knowledge.RelationAffix {
				continue
			}
			if stimSet[e.TargetID] {
				interStimSource[id]++
				interStimTarget[e.TargetID]++
			}
		}
	}

	type sc struct {
		id knowledge.NodeID
		s  float64
	}
	var scores []sc
	for _, id := range available {
		typedKinds := map[knowledge.RelationKind]bool{}
		assoc := 0
		typedToNonStim := 0
		for _, e := range bySource[id] {
			if e.Inhibitory {
				continue
			}
			if e.Kind == knowledge.RelationAssociation || e.Kind == knowledge.RelationAffix {
				assoc++
				continue
			}
			typedKinds[e.Kind] = true
			if !stimSet[e.TargetID] {
				typedToNonStim++
			}
		}
		s := float64(len(typedKinds))*0.2 + result.Activations[id]*0.1
		if interStimSource[id] > 0 {
			s += 1.2 // source of typed relation to another stimulus concept
		}
		if interStimTarget[id] > 0 {
			s += 0.6 // target of inter-stimulus typed relation
		}
		if stimSet[id] && interStimSource[id] == 0 && interStimTarget[id] == 0 {
			// In stimulus but not endpoint of inter-stimulus structure:
			// external typed knowledge does not make it the discourse center.
			s += 0.05
			if typedToNonStim >= 2 {
				s -= 0.35
			}
		} else if !stimSet[id] {
			s *= 0.4
		}
		if assoc >= 3 && len(typedKinds) == 0 {
			s -= 0.4
		}
		scores = append(scores, sc{id, s})
	}
	sort.Slice(scores, func(i, j int) bool { return scores[i].s > scores[j].s })
	limit := 6
	if len(scores) < limit {
		limit = len(scores)
	}
	var out []knowledge.NodeID
	for i := 0; i < limit; i++ {
		if scores[i].s > 0 {
			out = append(out, scores[i].id)
		}
	}
	if len(out) == 0 && len(available) > 0 {
		out = append(out, available[0])
	}
	return out
}

// buildFocusAnchored expands from a candidate focus.
// allowReverse: if true, reverse edges only when they bridge two focus-local nodes.
func buildFocusAnchored(
	kb *knowledge.KnowledgeBase,
	seed knowledge.NodeID,
	bySource, byTarget map[knowledge.NodeID][]understanding.ActiveEdge,
	availSet map[knowledge.NodeID]bool,
	result activation.Result,
	allowReverse bool,
) Interpretation {
	interp := Interpretation{Nodes: []knowledge.NodeID{seed}, FocusID: seed}
	visited := map[knowledge.NodeID]bool{seed: true}
	frontier := []knowledge.NodeID{seed}

	for d := 0; d < maxExpansionDepth && len(frontier) > 0; d++ {
		var next []knowledge.NodeID
		for _, cur := range frontier {
			for _, e := range bySource[cur] {
				appendEdge(&interp, e)
				if e.Inhibitory {
					continue
				}
				if !visited[e.TargetID] && availSet[e.TargetID] {
					visited[e.TargetID] = true
					interp.Nodes = append(interp.Nodes, e.TargetID)
					next = append(next, e.TargetID)
				}
			}
			if !allowReverse {
				continue
			}
			// Reverse only as bridge: both endpoints already focus-local, or
			// reverse node connects two nodes already in interpretation.
			for _, e := range byTarget[cur] {
				other := e.SourceID
				if other == cur {
					other = e.TargetID
				}
				if visited[other] {
					appendEdge(&interp, e)
					continue
				}
				// Bridge promotion: reverse node has outbound into another visited node.
				bridges := false
				for _, out := range bySource[other] {
					if visited[out.TargetID] && out.TargetID != cur {
						bridges = true
						break
					}
				}
				if bridges && availSet[other] {
					appendEdge(&interp, e)
					visited[other] = true
					interp.Nodes = append(interp.Nodes, other)
					next = append(next, other)
				}
			}
		}
		frontier = next
	}
	if kb != nil && kb.Patterns != nil {
		interp.Patterns = kb.Patterns.Match(interp.Nodes)
	}
	return interp
}

func appendEdge(interp *Interpretation, e understanding.ActiveEdge) {
	for _, r := range interp.Relations {
		if r.SourceID == e.SourceID && r.TargetID == e.TargetID && r.Kind == e.Kind {
			return
		}
	}
	interp.Relations = append(interp.Relations, ActiveRelation{
		SourceID: e.SourceID, TargetID: e.TargetID, Kind: e.Kind,
		Weight: e.Weight, Confidence: e.Confidence, Inhibitory: e.Inhibitory,
		Provenance: e.Provenance,
	})
}

func buildConnectedUnion(
	kb *knowledge.KnowledgeBase,
	available []knowledge.NodeID,
	bySource, byTarget map[knowledge.NodeID][]understanding.ActiveEdge,
	result activation.Result,
) Interpretation {
	nodeSet := map[knowledge.NodeID]bool{}
	interp := Interpretation{}
	for _, id := range available {
		nodeSet[id] = true
		interp.Nodes = append(interp.Nodes, id)
	}
	seen := map[string]bool{}
	add := func(e understanding.ActiveEdge) {
		key := fmt.Sprintf("%d-%s-%d-%v", e.SourceID, e.Kind, e.TargetID, e.Inhibitory)
		if seen[key] {
			return
		}
		if !nodeSet[e.SourceID] && !nodeSet[e.TargetID] {
			return
		}
		seen[key] = true
		interp.Relations = append(interp.Relations, ActiveRelation{
			SourceID: e.SourceID, TargetID: e.TargetID, Kind: e.Kind,
			Weight: e.Weight, Confidence: e.Confidence, Inhibitory: e.Inhibitory,
			Provenance: e.Provenance,
		})
	}
	for _, edges := range bySource {
		for _, e := range edges {
			add(e)
		}
	}
	for _, edges := range byTarget {
		for _, e := range edges {
			add(e)
		}
	}
	if kb != nil && kb.Patterns != nil {
		interp.Patterns = kb.Patterns.Match(interp.Nodes)
	}
	return interp
}

func chooseFocusFromStructure(kb *knowledge.KnowledgeBase, interp *Interpretation, result activation.Result, stimulus []string) knowledge.NodeID {
	if len(interp.Nodes) == 0 {
		return 0
	}
	stimSet := map[string]bool{}
	for _, t := range stimulus {
		stimSet[t] = true
	}
	typedOutKinds := map[knowledge.NodeID]map[knowledge.RelationKind]bool{}
	assocOut := map[knowledge.NodeID]int{}
	typedOut := map[knowledge.NodeID]int{}
	typedIn := map[knowledge.NodeID]int{}
	for _, r := range interp.Relations {
		if r.Inhibitory {
			continue
		}
		if r.Kind == knowledge.RelationAssociation || r.Kind == knowledge.RelationAffix {
			assocOut[r.SourceID]++
			continue
		}
		if typedOutKinds[r.SourceID] == nil {
			typedOutKinds[r.SourceID] = map[knowledge.RelationKind]bool{}
		}
		typedOutKinds[r.SourceID][r.Kind] = true
		typedOut[r.SourceID]++
		typedIn[r.TargetID]++
	}
	bestID := interp.Nodes[0]
	bestScore := -1.0
	for _, id := range interp.Nodes {
		act := result.Activations[id] * result.Confidence[id]
		if act == 0 {
			act = 0.05
		}
		kindVariety := float64(len(typedOutKinds[id]))
		typedSource := clamp01(kindVariety / 3.0)
		totalOut := float64(typedOut[id] + assocOut[id])
		typedRatio := 0.0
		if totalOut > 0 {
			typedRatio = float64(typedOut[id]) / totalOut
		}
		assocHubPenalty := 0.0
		if assocOut[id] >= 2 && typedOut[id] == 0 {
			assocHubPenalty = 0.45
		}
		sinkPenalty := 0.0
		if typedIn[id] >= 2 && typedOut[id] == 0 && assocOut[id] == 0 {
			sinkPenalty = 0.35
		}
		// Inter-stimulus participation: typed edge to/from another stimulus token.
		interStim := 0.0
		if kb != nil {
			for _, r := range interp.Relations {
				if r.Inhibitory || r.Kind == knowledge.RelationAssociation || r.Kind == knowledge.RelationAffix {
					continue
				}
				if r.SourceID == id {
					if tn := kb.Registry.GetByID(r.TargetID); tn != nil && stimSet[tn.Token] {
						interStim += 0.55
					}
				}
				if r.TargetID == id {
					if sn := kb.Registry.GetByID(r.SourceID); sn != nil && stimSet[sn.Token] {
						interStim += 0.25
					}
				}
			}
		}
		stimBonus := 0.0
		inStim := false
		if kb != nil {
			if n := kb.Registry.GetByID(id); n != nil && stimSet[n.Token] {
				inStim = true
				stimBonus = 0.15
			}
		}
		// Token with only external typed structure while other stimulus concepts exist:
		// do not win focus merely by having its own knowledge graph.
		externalOnlyPenalty := 0.0
		if inStim && interStim == 0 && typedOut[id] > 0 {
			for _, oid := range interp.Nodes {
				if kb == nil {
					break
				}
				if on := kb.Registry.GetByID(oid); on != nil && stimSet[on.Token] && oid != id {
					externalOnlyPenalty = 0.4
					break
				}
			}
		}
		nonStimPenalty := 0.0
		if !inStim && typedOut[id] > 0 {
			for _, oid := range interp.Nodes {
				if kb == nil {
					break
				}
				if on := kb.Registry.GetByID(oid); on != nil && stimSet[on.Token] {
					nonStimPenalty = 0.25
					break
				}
			}
		}
		s := 0.12*act + 0.20*typedSource + 0.15*typedRatio + stimBonus + interStim - assocHubPenalty - sinkPenalty - nonStimPenalty - externalOnlyPenalty
		if s > bestScore {
			bestScore = s
			bestID = id
		}
	}
	return bestID
}

func interpretationSignature(interp Interpretation) string {
	ids := append([]knowledge.NodeID{}, interp.Nodes...)
	sort.Slice(ids, func(i, j int) bool { return ids[i] < ids[j] })
	s := ""
	for _, id := range ids {
		s += fmt.Sprintf("%d,", id)
	}
	s += "|"
	type rk struct {
		a, b knowledge.NodeID
		k    knowledge.RelationKind
	}
	var rels []rk
	for _, r := range interp.Relations {
		rels = append(rels, rk{r.SourceID, r.TargetID, r.Kind})
	}
	sort.Slice(rels, func(i, j int) bool {
		if rels[i].a != rels[j].a {
			return rels[i].a < rels[j].a
		}
		if rels[i].b != rels[j].b {
			return rels[i].b < rels[j].b
		}
		return rels[i].k < rels[j].k
	})
	for _, r := range rels {
		s += fmt.Sprintf("%d-%s-%d,", r.a, r.k, r.b)
	}
	return s
}

// scoreInterpretation: existing components only.
// Coherence emphasizes structure that explains the focus, not raw edge count.
// Reverse/collateral mass relative to focus reduces coherence contribution.
func scoreInterpretation(
	kb *knowledge.KnowledgeBase,
	interp *Interpretation,
	result activation.Result,
	contextTokens []string,
	stimSet map[knowledge.NodeID]bool,
) {
	if len(interp.Nodes) == 0 {
		return
	}
	focus := interp.FocusID
	if focus == 0 && len(interp.Nodes) > 0 {
		focus = interp.Nodes[0]
	}

	// Focus-local node set: focus + nodes reached by non-reverse edges from focus structure.
	focusLocal := map[knowledge.NodeID]bool{focus: true}
	for _, r := range interp.Relations {
		if r.Inhibitory {
			continue
		}
		if r.Provenance == understanding.ProvCollateral {
			continue
		}
		if focusLocal[r.SourceID] {
			focusLocal[r.TargetID] = true
		}
	}
	// Also mark direct incident on focus regardless of provenance.
	for _, r := range interp.Relations {
		if r.SourceID == focus {
			focusLocal[r.TargetID] = true
		}
		if r.TargetID == focus {
			focusLocal[r.SourceID] = true
		}
	}

	degree := map[knowledge.NodeID]int{}
	var supportAll []ActiveRelation
	var supportLocal []ActiveRelation
	var conflictSum float64
	for _, r := range interp.Relations {
		if r.Inhibitory {
			conflictSum += r.Weight * r.Confidence
			continue
		}
		supportAll = append(supportAll, r)
		degree[r.SourceID]++
		degree[r.TargetID]++
		// Local = explains focus structure (both ends focus-local, or source is focus).
		if r.SourceID == focus || (focusLocal[r.SourceID] && focusLocal[r.TargetID] && r.Provenance != understanding.ProvCollateral) {
			supportLocal = append(supportLocal, r)
		} else if r.SourceID == focus {
			supportLocal = append(supportLocal, r)
		}
	}
	// Ensure focus outbound always in local
	seenLocal := map[string]bool{}
	var localClean []ActiveRelation
	for _, r := range supportLocal {
		k := fmt.Sprintf("%d-%s-%d", r.SourceID, r.Kind, r.TargetID)
		if seenLocal[k] {
			continue
		}
		seenLocal[k] = true
		localClean = append(localClean, r)
	}
	supportLocal = localClean
	if len(supportLocal) == 0 {
		// Fallback: all outbound from focus
		for _, r := range supportAll {
			if r.SourceID == focus {
				supportLocal = append(supportLocal, r)
			}
		}
	}

	var contribSum, weightSum float64
	for _, id := range interp.Nodes {
		act := result.Activations[id] * result.Confidence[id]
		structW := 1.0 + float64(degree[id])
		if focusLocal[id] {
			structW += 1.0
		}
		contribSum += act * structW
		weightSum += structW
	}
	if weightSum > 0 {
		interp.SemanticScore = clamp01(contribSum / weightSum)
	}

	var relSum float64
	for _, r := range supportLocal {
		relSum += r.Weight * r.Confidence
	}
	if len(supportLocal) > 0 {
		interp.RelationScore = clamp01(relSum / float64(len(supportLocal)))
	}
	interp.ConflictScore = clamp01(conflictSum)

	var patSum float64
	for _, p := range interp.Patterns {
		patSum += p.Weight * p.Confidence * (1 + 0.1*float64(len(p.Members)))
	}
	if len(interp.Patterns) > 0 {
		interp.PatternScore = clamp01(patSum / float64(len(interp.Patterns)))
	}

	var histSum float64
	histCount := 0
	if kb != nil {
		for _, id := range interp.Nodes {
			n := kb.Registry.GetByID(id)
			if n == nil {
				continue
			}
			f := float64(n.Frequency)
			if f > 20 {
				f = 20
			}
			histSum += f / 20.0
			histCount++
		}
	}
	if histCount > 0 {
		interp.HistoryScore = clamp01(histSum / float64(histCount))
	}
	if len(contextTokens) > 0 && kb != nil {
		ctxSet := map[string]bool{}
		for _, t := range contextTokens {
			ctxSet[t] = true
		}
		hit := 0
		for _, id := range interp.Nodes {
			if n := kb.Registry.GetByID(id); n != nil && ctxSet[n.Token] {
				hit++
			}
		}
		interp.ContextScore = clamp01(float64(hit) / float64(len(interp.Nodes)))
	}

	// Coherence over focus-local structure only (not collateral mass).
	nodeLocal := 0
	for id := range focusLocal {
		if containsNode(interp.Nodes, id) {
			nodeLocal++
		}
	}
	relLocal := float64(len(supportLocal))
	density := 0.0
	if nodeLocal > 1 {
		density = relLocal / (float64(nodeLocal) * float64(nodeLocal-1))
	}
	kindSet := map[knowledge.RelationKind]bool{}
	for _, r := range supportLocal {
		kindSet[r.Kind] = true
	}
	variety := clamp01(float64(len(kindSet)) / 4.0)
	multi := 0
	localDeg := map[knowledge.NodeID]int{}
	for _, r := range supportLocal {
		localDeg[r.SourceID]++
		localDeg[r.TargetID]++
	}
	for _, d := range localDeg {
		if d >= 2 {
			multi++
		}
	}
	bridgeBonus := 0.0
	if nodeLocal > 0 {
		bridgeBonus = clamp01(float64(multi) / float64(nodeLocal))
	}
	breadth := clamp01(relLocal / 4.0)

	// Focus incidence: share of local support among all support — penalize collateral-heavy I.
	incidence := 1.0
	if len(supportAll) > 0 {
		incidence = float64(len(supportLocal)) / float64(len(supportAll))
	}
	interp.CoherenceScore = clamp01(
		(0.25*density + 0.20*variety + 0.20*bridgeBonus + 0.15*breadth + 0.20*interp.RelationScore) * (0.4 + 0.6*incidence),
	)

	const (
		wSem = 0.10
		wRel = 0.12
		wPat = 0.10
		wHis = 0.05
		wCtx = 0.05
		wCoh = 0.50
		wX   = 0.30
	)
	total := wSem*interp.SemanticScore +
		wRel*interp.RelationScore +
		wPat*interp.PatternScore +
		wHis*interp.HistoryScore +
		wCtx*interp.ContextScore +
		wCoh*interp.CoherenceScore -
		wX*interp.ConflictScore
	// Prefer interpretations whose focus is a stimulus concept that owns local structure.
	if stimSet[focus] && len(supportLocal) > 0 {
		total += 0.08
	}
	interp.TotalScore = clamp01(total)

	interp.EvidenceNotes = []string{
		fmt.Sprintf("nodes=%d local_rels=%d all_rels=%d incidence=%.2f",
			len(interp.Nodes), len(supportLocal), len(supportAll), incidence),
		fmt.Sprintf("sem=%.3f rel=%.3f coh=%.3f total=%.3f",
			interp.SemanticScore, interp.RelationScore, interp.CoherenceScore, interp.TotalScore),
	}
	if kb != nil {
		if f := kb.Registry.GetByID(focus); f != nil {
			interp.EvidenceNotes = append(interp.EvidenceNotes, "focus="+f.Token)
		}
	}
}

func containsNode(ids []knowledge.NodeID, id knowledge.NodeID) bool {
	for _, x := range ids {
		if x == id {
			return true
		}
	}
	return false
}

func SelectBestInterpretation(candidates []Interpretation) *Interpretation {
	if len(candidates) == 0 {
		return nil
	}
	best := &candidates[0]
	for i := 1; i < len(candidates); i++ {
		if candidates[i].TotalScore > best.TotalScore {
			best = &candidates[i]
		}
	}
	return best
}

func ScoreInterpretationForTest(kb *knowledge.KnowledgeBase, interp *Interpretation, result activation.Result, contextTokens []string) {
	stim := map[knowledge.NodeID]bool{}
	scoreInterpretation(kb, interp, result, contextTokens, stim)
}
