package thinking

import (
	"sort"
	"time"

	"github.com/project-horizon/horizon-core/services/ai/activation"
	"github.com/project-horizon/horizon-core/services/ai/hfcc"
	"github.com/project-horizon/horizon-core/services/ai/knowledge"
	"github.com/project-horizon/horizon-core/services/ai/understanding"
)

func (t *ThinkingEngine) think(prompt string, contextTokens []string, includeStimulus bool) (Thought, bool) {
	stimulus := splitPrompt(prompt)
	wm := newWorkingMemory(contextTokens)
	defer wm.Clear()
	trace := PathTrace{}
	focus := t.Context.Focus(contextTokens)
	result := t.Activation.ActivateWith(activation.Request{StimulusTokens: stimulus, ContextBoosts: focus.Boosts, Cycles: 8})
	for _, node := range result.RankedNodes {
		trace.Add("Activation", node, result.Confidence[node.ID], "ranked activation")
	}
	if !result.Converged {
		t.LastState = CognitiveState{CurrentContext: contextTokens, UnknownNodes: stimulus, ReasoningDepth: 1}
		return Thought{NeedsWebSearch: true}, false
	}

	rep := t.Understanding.Understand(result, stimulus, &trace, includeStimulus)
	wm.Candidates = append(append([]knowledge.NodeID{}, rep.DominantNodes...), rep.SupportingNodes...)
	for id, c := range result.Confidence {
		wm.TemporaryConfidence[id] = c
	}

	// Semantic Neighborhood = available knowledge for reasoning (structural).
	neighborhood := t.Understanding.BuildSemanticNeighborhood(result, stimulus, contextTokens)
	focusContext := append([]string{}, contextTokens...)
	focusContext = append(focusContext, stimulus...)
	// Collect active edges for FSU (available neighborhood only).
	var uEdges []understanding.ActiveEdge
	if neighborhood.Edges != nil {
		uEdges = neighborhood.Edges
	}

	// FSU Phase 2 correction: Functional State BEFORE candidate ranking / I*.
	fstate := BuildFunctionalState(t.Activation.Memory, stimulus, neighborhood, uEdges, result.Activations)

	candidates := BuildInterpretations(t.Activation.Memory, rep, result, neighborhood, focusContext)
	// Apply functional state to each candidate before selecting I*.
	for i := range candidates {
		ApplyFunctionalStateToInterpretation(t.Activation.Memory, &candidates[i], fstate)
	}

	// ★ PRESERVATION BOUNDARY A + HFCC BOUNDARY B (before irreversible selection)
	obsRef := ObservationRef{ID: "cycle", Version: "1", SourceRef: "prompt"}
	formation := BuildCandidateFormationState(obsRef, t.Activation.Memory, stimulus, candidates)
	preserved := PreserveCandidateState(formation, nil, nil)
	// Audit A: Formation → PCS
	presReport := AuditPreservationIID(formation, preserved)
	obs := hfcc.Observation{ID: obsRef.ID, RawText: prompt, Surface: stimulus, Timestamp: time.Now().UTC()}
	// Single source: PCS only (no []Interpretation fallback)
	proj := BuildHFCCFromPreservedWithMappings(obs, preserved)
	cs := proj.CandidateSet
	// Audit B: PCS → HFCC via explicit ProjectionMapping
	hfccReport := AuditHFCCConsumerWithMappings(preserved, proj)
	if domain, notes := AuditTanpaKakiCase(formation, stimulus); domain != "" {
		presReport.Notes = append(presReport.Notes, notes...)
		if presReport.FailureDomain == "" {
			presReport.FailureDomain = domain
		}
		presReport.NotFormed = append(presReport.NotFormed, notes...)
	}

	// Selection AFTER preservation — alternatives remain in candidates / HFCC set
	bestInterp := SelectBestInterpretation(candidates)
	if bestInterp != nil {
		bestInterp.EvalStatus = DecideEvaluation(bestInterp, fstate, t.Activation.Memory)
		bestInterp.EvidenceNotes = append(bestInterp.EvidenceNotes, FormatFunctionalStateSummary(fstate))
	}
	wm.Interpretations = candidates
	wm.BestInterpretation = bestInterp

	hypotheses := BuildHypotheses(rep, result)
	wm.Hypotheses = hypotheses

	activeNodes := neighborhood.Nodes
	if len(activeNodes) == 0 {
		activeNodes = append(append([]knowledge.NodeID{}, rep.DominantNodes...), rep.SupportingNodes...)
	}
	activeRels := make([]ActiveRelation, 0, len(neighborhood.Edges))
	for _, e := range neighborhood.Edges {
		activeRels = append(activeRels, ActiveRelation{
			SourceID: e.SourceID, TargetID: e.TargetID,
			Kind: e.Kind, Weight: e.Weight, Confidence: e.Confidence, Inhibitory: e.Inhibitory,
			Provenance: e.Provenance,
		})
	}
	var activePatterns []*knowledge.PatternSynapse
	if t.Activation.Memory.Patterns != nil {
		activePatterns = t.Activation.Memory.Patterns.Match(activeNodes)
	}

	state := CognitiveState{
		CurrentContext:    contextTokens,
		FocusedNodes:      rep.DominantNodes,
		SupportingNodes:   rep.SupportingNodes,
		CompetingNodes:    rep.CompetingNodes,
		UnknownNodes:      rep.UnknownTokens,
		ConflictNodes:     rep.ConflictNodes,
		ReasoningDepth:    3,
		ActivationHistory: []activation.Result{result},
		Confidence:        rep.Confidence,
		ActiveNodes:       activeNodes,
		Activations:       result.Activations,
		Confidences:       result.Confidence,
		ActiveRelations:   activeRels,
		ActivePatterns:    activePatterns,
		StimulusTokens:    stimulus,
		FunctionalSignals: fstate.Signals,
		Propositions:      fstate.RequestedProps,
		Constraints:       fstate.Constraints,
		EvidencePaths:     fstate.EvidencePaths,
	}
	if bestInterp != nil {
		state.Propositions = bestInterp.Propositions
		state.Constraints = bestInterp.Constraints
		state.EvidencePaths = bestInterp.EvidencePaths
		state.FunctionalSignals = append(state.FunctionalSignals, bestInterp.FunctionalSignals...)
	}

	assessment := t.MetaCognition.Evaluate(state, rep)

	thought := Thought{
		Resonance:          rep.Resonance,
		Confidence:         rep.Confidence,
		Hypotheses:         hypotheses,
		NeedsWebSearch:     assessment.NeedWebSearch,
		BestInterpretation: bestInterp,
		Interpretations:    candidates,
		FunctionalSignals:  state.FunctionalSignals,
	}
	thought.HFCCCandidateSet = &cs
	thought.FormationState = &formation
	thought.PreservedState = &preserved
	thought.PreservationReport = &presReport
	thought.HFCCConsumerReport = &hfccReport

	if bestInterp != nil {
		thought.Confidence = clamp01((thought.Confidence + bestInterp.TotalScore) / 2)
		for _, id := range bestInterp.Nodes {
			if n := t.Activation.Memory.Registry.GetByID(id); n != nil && !contains(thought.Concepts, n.Token) {
				thought.Concepts = append(thought.Concepts, n.Token)
				trace.Add("Interpretation", n, bestInterp.TotalScore, "member of I*")
			}
		}
		for _, note := range bestInterp.EvidenceNotes {
			trace.Add("Coherence", nil, bestInterp.TotalScore, note)
		}
	} else {
		for _, h := range hypotheses {
			if h.Confidence < 0.2 {
				continue
			}
			for _, id := range h.Nodes {
				if n := t.Activation.Memory.Registry.GetByID(id); n != nil && !contains(thought.Concepts, n.Token) {
					thought.Concepts = append(thought.Concepts, n.Token)
					trace.Add("Hypothesis", n, h.Confidence, "candidate hypothesis (fallback)")
				}
			}
		}
	}

	for _, id := range rep.ConflictNodes {
		if n := t.Activation.Memory.Registry.GetByID(id); n != nil {
			thought.Conflicts = append(thought.Conflicts, n.Token)
		}
	}
	if thought.Confidence < t.MetaCognition.MinConfidence {
		thought.NeedsWebSearch = true
	}

	var focusNode *knowledge.ConceptNode
	if bestInterp != nil {
		focusNode = t.Activation.Memory.Registry.GetByID(bestInterp.FocusID)
	} else if len(hypotheses) > 0 && len(hypotheses[0].Nodes) > 0 {
		focusNode = t.Activation.Memory.Registry.GetByID(hypotheses[0].Nodes[0])
	}
	if focusNode != nil {
		thought.Inferences = InferFromIsAChain(t.Activation.Memory, focusNode)
	}

	var stimulusIDs []knowledge.NodeID
	for _, tok := range stimulus {
		if n := t.Activation.Memory.Fetch(tok); n != nil {
			stimulusIDs = append(stimulusIDs, n.ID)
		}
	}
	evidence := t.Activation.Memory.Patterns.Match(stimulusIDs)
	seen := map[knowledge.PatternID]bool{}
	for _, ps := range evidence {
		seen[ps.ID] = true
	}
	for _, id := range stimulusIDs {
		for _, ps := range t.Activation.Memory.Patterns.ResultsFor(id) {
			if !seen[ps.ID] {
				evidence = append(evidence, ps)
				seen[ps.ID] = true
			}
		}
	}
	thought.PatternEvidence = evidence
	trace.Add("Decision", nil, thought.Confidence, "interpretation selection completed")
	t.LastState, t.LastTrace = state, trace
	return thought, len(thought.Concepts) > 0 && thought.Confidence >= 0.2
}

func (t *ThinkingEngine) Think(prompt string, contextTokens []string) (Thought, bool) {
	return t.think(prompt, contextTokens, false)
}

func (t *ThinkingEngine) ThinkAbout(prompt string, contextTokens []string) (Thought, bool) {
	return t.think(prompt, contextTokens, true)
}

func (t *ThinkingEngine) Reason(prompt string) ([]string, bool) {
	thought, ok := t.Think(prompt, nil)
	return thought.Concepts, ok
}

func BuildHypotheses(rep understanding.CognitiveRepresentation, result activation.Result) []Hypothesis {
	ids := append(append([]knowledge.NodeID{}, rep.DominantNodes...), rep.SupportingNodes...)
	sort.Slice(ids, func(i, j int) bool {
		return result.Confidence[ids[i]]*result.Activations[ids[i]] > result.Confidence[ids[j]]*result.Activations[ids[j]]
	})
	var hs []Hypothesis
	for i, id := range ids {
		if i >= 6 {
			break
		}
		conflicts := 0
		if understanding.ContainsID(rep.ConflictNodes, id) || understanding.ContainsID(rep.CompetingNodes, id) {
			conflicts = 1
		}
		hs = append(hs, Hypothesis{
			Nodes:      []knowledge.NodeID{id},
			Confidence: clamp01(result.Confidence[id]*result.Activations[id] - float64(conflicts)*0.2),
			Evidence:   len(rep.HiddenRelations) + 1,
			Conflicts:  conflicts,
		})
	}
	return hs
}

func contains(tokens []string, token string) bool {
	for _, t := range tokens {
		if t == token {
			return true
		}
	}
	return false
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

// deriveFunctionalSignals builds FSU Phase-1 signals from structure already active.
// No token dictionary: signals describe relational / path structure in the state.
func deriveFunctionalSignals(state CognitiveState, interp *Interpretation, kb *knowledge.KnowledgeBase) []FunctionalSignal {
	var out []FunctionalSignal
	rels := state.ActiveRelations
	if interp != nil && len(interp.Relations) > 0 {
		rels = interp.Relations
	}
	if len(rels) == 0 {
		return out
	}

	// Typed relation bundle around focus / stimulus-local sources.
	typedBySource := map[knowledge.NodeID][]knowledge.RelationKind{}
	targetSupport := map[knowledge.NodeID]int{} // how many distinct paths support a target
	for _, r := range rels {
		if r.Inhibitory {
			continue
		}
		if r.Kind == knowledge.RelationAssociation || r.Kind == knowledge.RelationAffix {
			continue
		}
		typedBySource[r.SourceID] = append(typedBySource[r.SourceID], r.Kind)
		targetSupport[r.TargetID]++
	}

	focus := knowledge.NodeID(0)
	if interp != nil {
		focus = interp.FocusID
	}
	if focus != 0 {
		if kinds := typedBySource[focus]; len(kinds) > 0 {
			uniq := uniqueKinds(kinds)
			out = append(out, FunctionalSignal{
				Kind:          "focus_typed_structure",
				AnchorIDs:     []knowledge.NodeID{focus},
				RelationKinds: uniq,
				Strength:      clamp01(float64(len(uniq)) / 4.0),
				Source:        "interpretation",
				Note:          "focus owns typed outbound structure",
			})
		}
	}

	// Multi-path support: same target reached more than once from structure.
	for tid, n := range targetSupport {
		if n < 2 {
			continue
		}
		out = append(out, FunctionalSignal{
			Kind:      "multi_path_support",
			AnchorIDs: []knowledge.NodeID{tid},
			Strength:  clamp01(float64(n) / 3.0),
			Source:    "active_structure",
			Note:      "target supported by multiple relational paths",
		})
	}

	// Pattern participation as functional evidence.
	if len(state.ActivePatterns) > 0 {
		var ids []knowledge.NodeID
		for _, ps := range state.ActivePatterns {
			ids = append(ids, ps.Members...)
			if ps.Result != 0 {
				ids = append(ids, ps.Result)
			}
		}
		out = append(out, FunctionalSignal{
			Kind:      "pattern_structure",
			AnchorIDs: ids,
			Strength:  clamp01(float64(len(state.ActivePatterns)) / 3.0),
			Source:    "pattern",
			Note:      "active pattern match in neighborhood",
		})
	}

	_ = kb
	return out
}

func uniqueKinds(kinds []knowledge.RelationKind) []knowledge.RelationKind {
	seen := map[knowledge.RelationKind]bool{}
	var out []knowledge.RelationKind
	for _, k := range kinds {
		if seen[k] {
			continue
		}
		seen[k] = true
		out = append(out, k)
	}
	return out
}
