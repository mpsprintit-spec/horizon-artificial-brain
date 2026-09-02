package thinking

import (
	"fmt"
	"strings"

	"github.com/project-horizon/horizon-core/services/ai/knowledge"
	"github.com/project-horizon/horizon-core/services/ai/understanding"
)

// Proposition is a temporary cognitive claim (not a Synapse write).
// Requested marks whether this proposition is being asked about / asserted
// from stimulus structure (vs a plain fact attached to focus).
type Proposition struct {
	ID         string // stable identity within one cognitive cycle
	TargetID   knowledge.NodeID
	ObjectID   knowledge.NodeID
	Relation   knowledge.RelationKind
	TargetTok  string
	ObjectTok  string
	Strength   float64
	Requested  bool // true = from stimulus-linked claim, false = background fact
}

// EvidenceJudgement is the epistemic relation of one candidate path to one proposition.
type EvidenceJudgement string

const (
	EvidenceSupport       EvidenceJudgement = "SUPPORT"
	EvidenceContradiction EvidenceJudgement = "CONTRADICTION"
	EvidenceIrrelevant    EvidenceJudgement = "IRRELEVANT"
)

// EvidenceEvaluation binds a candidate path to a specific proposition identity.
// IRRELEVANT does not delete knowledge; it only fails to raise that proposition.
type EvidenceEvaluation struct {
	PropositionID string
	Path          EvidencePath
	Judgement     EvidenceJudgement
	Confidence    float64
	Note          string
}

type Constraint struct {
	ConceptID knowledge.NodeID
	Token     string
	Role      string // absence | condition | contrast
	Note      string
}

type EvidencePath struct {
	NodeIDs    []knowledge.NodeID
	Tokens     []string
	Relations  []knowledge.RelationKind
	Support    bool
	Conflict   bool
	Length     int
	Confidence float64
	Note       string
}

type EvalStatus string

const (
	EvalSupported     EvalStatus = "SUPPORTED"
	EvalUnsupported   EvalStatus = "UNSUPPORTED" // legacy alias; prefer UNKNOWN for no evidence
	EvalUnknown       EvalStatus = "UNKNOWN"
	EvalConflicted    EvalStatus = "CONFLICTED"
	EvalContradicted  EvalStatus = "CONTRADICTED"
)

// FunctionalState is built BEFORE candidate ranking / I* selection.
// It is structural evidence about what the input is doing with knowledge,
// not a word→function dictionary.
type FunctionalState struct {
	Stimulus       []string
	ContentIDs     []knowledge.NodeID // stimulus nodes with typed structure or typed edge endpoints
	FramingIDs     []knowledge.NodeID // stimulus nodes without typed link to content (query framing)
	RequestedProps []Proposition
	BackgroundFacts []Proposition
	Constraints    []Constraint
	EvidencePaths  []EvidencePath
	Signals        []FunctionalSignal
	// Structural goal descriptors (not word triggers): what kind of cognitive work is indicated.
	GoalDescribe  bool // single content focus, build understanding
	GoalEvaluate  bool // framing + content proposition → evaluate claim
	GoalExplain   bool // evaluate-like + multi-path evidence centrality
	GoalAssert    bool // content-only proposition, little framing
}

const maxEvidencePathLen = 3
const maxEvidencePaths = 8

// BuildFunctionalState constructs FSU state from neighborhood + stimulus BEFORE I*.
func BuildFunctionalState(
	kb *knowledge.KnowledgeBase,
	stimulus []string,
	neighborhood understanding.SemanticNeighborhood,
	activeRels []understanding.ActiveEdge,
	resultActivations map[knowledge.NodeID]float64,
) FunctionalState {
	fs := FunctionalState{Stimulus: append([]string{}, stimulus...)}
	if kb == nil {
		return fs
	}
	stimIDs := map[knowledge.NodeID]bool{}
	stimSet := map[string]bool{}
	for _, t := range stimulus {
		stimSet[t] = true
		if n := kb.Fetch(t); n != nil {
			stimIDs[n.ID] = true
		}
	}

	// Classify stimulus nodes: content vs framing (structure only).
	typedOut := map[knowledge.NodeID]int{}
	assocOut := map[knowledge.NodeID]int{}
	for id := range stimIDs {
		n := kb.Registry.GetByID(id)
		if n == nil {
			continue
		}
		for _, s := range n.OutboundAll() {
			if s.Kind == knowledge.RelationAssociation || s.Kind == knowledge.RelationAffix {
				if !s.Inhibitory {
					assocOut[id]++
				}
				continue
			}
			// Typed structure includes inhibitory (agent still "about" that object)
			typedOut[id]++
		}
	}
	// Endpoints of typed edges among neighborhood that touch stimulus
	endpoint := map[knowledge.NodeID]bool{}
	for _, e := range activeRels {
		if e.Inhibitory || e.Kind == knowledge.RelationAssociation || e.Kind == knowledge.RelationAffix {
			continue
		}
		if stimIDs[e.SourceID] || stimIDs[e.TargetID] {
			endpoint[e.SourceID] = true
			endpoint[e.TargetID] = true
		}
	}
	for id := range stimIDs {
		if typedOut[id] > 0 || endpoint[id] {
			fs.ContentIDs = append(fs.ContentIDs, id)
		} else if assocOut[id] == 0 && hasInboundCapabilityRole(kb, id) {
			// Bare concept that is a capability/function object in memory.
			fs.ContentIDs = append(fs.ContentIDs, id)
		} else if assocOut[id] == 0 {
			// Bare surface token with no structural role yet → framing (not content claim endpoint).
			fs.FramingIDs = append(fs.FramingIDs, id)
		} else {
			fs.FramingIDs = append(fs.FramingIDs, id)
		}
	}

	// Requested propositions: typed edges between stimulus content concepts (or content→content via KB).
	seen := map[string]bool{}
	addReq := func(src, tgt knowledge.NodeID, kind knowledge.RelationKind, strength float64) {
		if kind == knowledge.RelationAssociation || kind == knowledge.RelationAffix {
			return
		}
		key := fmt.Sprintf("%d-%s-%d", src, kind, tgt)
		if seen[key] {
			return
		}
		seen[key] = true
		fs.RequestedProps = append(fs.RequestedProps, Proposition{
			ID: propID(src, kind, tgt),
			TargetID: src, ObjectID: tgt, Relation: kind,
			TargetTok: tokenOf(kb, src), ObjectTok: tokenOf(kb, tgt),
			Strength: strength, Requested: true,
		})
	}
	contentSet := map[knowledge.NodeID]bool{}
	for _, id := range fs.ContentIDs {
		contentSet[id] = true
	}
	for _, e := range activeRels {
		if e.Inhibitory {
			continue
		}
		if contentSet[e.SourceID] && (contentSet[e.TargetID] || stimIDs[e.TargetID]) {
			addReq(e.SourceID, e.TargetID, e.Kind, e.Weight*e.Confidence)
		}
	}
	// KB edges between stimulus content nodes
	for sid := range contentSet {
		n := kb.Registry.GetByID(sid)
		if n == nil {
			continue
		}
		for _, s := range n.OutboundAll() {
			tid := s.TargetID
			if s.Inhibitory {
				continue
			}
			if stimIDs[tid] {
				addReq(sid, tid, s.Kind, s.Weight*s.Confidence)
			}
		}
	}

	// Stimulus-linked claim binding: pairs of content concepts co-present in stimulus
	// form requested propositions even without a direct stored edge.
	// Relation kind is inferred from structural evidence paths (not word dictionary).
	contentList := append([]knowledge.NodeID{}, fs.ContentIDs...)
	for i := 0; i < len(contentList); i++ {
		for j := 0; j < len(contentList); j++ {
			if i == j {
				continue
			}
			src, tgt := contentList[i], contentList[j]
			// Prefer agent = richer typed structure as source
			if typedOut[src] < typedOut[tgt] {
				continue
			}
			kind, strength, ok := inferRequestedRelation(kb, src, tgt, activeRels)
			if !ok {
				continue
			}
			addReq(src, tgt, kind, strength)
		}
	}

	// Evaluation claims: content agent + bare stimulus tokens (potential capability objects)
	// even when object not yet in memory as function target — may be UNSUPPORTED.
	primaryGuess := primaryContent(kb, fs.ContentIDs, typedOut, resultActivations)
	if primaryGuess != 0 && len(fs.FramingIDs) > 0 {
		for _, fid := range append(append([]knowledge.NodeID{}, fs.FramingIDs...), fs.ContentIDs...) {
			if fid == primaryGuess {
				continue
			}
			// Skip association hubs (rich framing)
			n := kb.Registry.GetByID(fid)
			if n == nil {
				continue
			}
			assoc := 0
			typed := 0
			for _, s := range n.OutboundAll() {
				if s.Inhibitory {
					continue
				}
				if s.Kind == knowledge.RelationAssociation || s.Kind == knowledge.RelationAffix {
					assoc++
				} else {
					typed++
				}
			}
			if assoc >= 2 {
				continue // query hub, not capability object
			}
			if typed > typedOut[primaryGuess] {
				continue
			}
			kind, strength, ok := inferRequestedRelation(kb, primaryGuess, fid, activeRels)
			if ok {
				addReq(primaryGuess, fid, kind, strength)
				continue
			}
			// Bare/low structure stimulus object → can_do claim under evaluation
			if assoc == 0 && typed == 0 {
				addReq(primaryGuess, fid, knowledge.RelationCanDo, 0.25)
			}
		}
	}

	// Background facts: typed outbound from primary content not both-in-stimulus as requested edge object-only
	primary := primaryContent(kb, fs.ContentIDs, typedOut, resultActivations)
	if primary != 0 {
		if n := kb.Registry.GetByID(primary); n != nil {
			for _, s := range n.OutboundAll() {
			tid := s.TargetID
				if s.Inhibitory || s.Kind == knowledge.RelationAssociation || s.Kind == knowledge.RelationAffix {
					continue
				}
				key := fmt.Sprintf("%d-%s-%d", primary, s.Kind, tid)
				if seen[key] {
					continue
				}
				fs.BackgroundFacts = append(fs.BackgroundFacts, Proposition{
					ID: propID(primary, s.Kind, tid),
					TargetID: primary, ObjectID: tid, Relation: s.Kind,
					TargetTok: n.Token, ObjectTok: tokenOf(kb, tid),
					Strength: s.Weight * s.Confidence, Requested: false,
				})
			}
		}
	}

	// Evidence paths for requested props from neighborhood structure only
	edgeIndex := map[knowledge.NodeID][]understanding.ActiveEdge{}
	for _, e := range activeRels {
		edgeIndex[e.SourceID] = append(edgeIndex[e.SourceID], e)
	}
	for _, p := range fs.RequestedProps {
		paths := findEvidencePathsFromActive(kb, p.TargetID, p.ObjectID, edgeIndex, activeRels)
		fs.EvidencePaths = append(fs.EvidencePaths, paths...)
		if len(fs.EvidencePaths) >= maxEvidencePaths {
			break
		}
	}
	// Also paths for primary can_do-like background when evaluate
	for _, p := range fs.BackgroundFacts {
		if p.Relation != knowledge.RelationCanDo {
			continue
		}
		paths := findEvidencePathsFromActive(kb, p.TargetID, p.ObjectID, edgeIndex, activeRels)
		fs.EvidencePaths = append(fs.EvidencePaths, paths...)
	}

	// Constraints ONLY from functional contrast/condition structure in active state,
	// not from "node lies on path".
	fs.Constraints = constraintsFromFunctionalStructure(kb, stimIDs, activeRels, fs.RequestedProps)

	// Goal structure from graph roles only (no word dictionary).
	// framing "empty" = framing tokens with no synapses yet (unknown surface words)
	emptyFraming := 0
	richFraming := 0
	for _, id := range fs.FramingIDs {
		n := kb.Registry.GetByID(id)
		if n == nil || len(n.Synapses) == 0 {
			emptyFraming++
		} else {
			richFraming++
		}
	}
	multi := 0
	for _, ep := range fs.EvidencePaths {
		if ep.Length >= 2 {
			multi++
		}
	}
	fs.GoalDescribe = len(fs.ContentIDs) == 1 && len(fs.RequestedProps) == 0 && primary != 0
	fs.GoalAssert = len(fs.RequestedProps) > 0 && richFraming == 0 && emptyFraming <= 1
	fs.GoalEvaluate = len(fs.FramingIDs) >= 1 && len(fs.ContentIDs) >= 1 && (len(fs.RequestedProps) > 0 || hasCanDoFact(fs.BackgroundFacts) || len(fs.ContentIDs) >= 2) && !fs.GoalAssert
	// Explain: multi-path evidence central under multi-framing evaluate
	fs.GoalExplain = fs.GoalEvaluate && multi > 0 && len(fs.FramingIDs) >= 2

	fs.Signals = append(fs.Signals, FunctionalSignal{
		Kind: "functional_state", Source: "active_structure",
		Strength: 0.5,
		Note: fmt.Sprintf("content=%d framing=%d reqProps=%d constr=%d describe=%v eval=%v assert=%v explain=%v",
			len(fs.ContentIDs), len(fs.FramingIDs), len(fs.RequestedProps), len(fs.Constraints),
			fs.GoalDescribe, fs.GoalEvaluate, fs.GoalAssert, fs.GoalExplain),
	})
	_ = neighborhood
	return fs
}


// inferRequestedRelation chooses a relation kind for a stimulus content pair from structure:
// direct typed edge, or has→function / can_do paths. No word→relation dictionary.
func inferRequestedRelation(
	kb *knowledge.KnowledgeBase,
	src, tgt knowledge.NodeID,
	active []understanding.ActiveEdge,
) (knowledge.RelationKind, float64, bool) {
	if n := kb.Registry.GetByID(src); n != nil {
		for _, s := range n.SynapsesTo(tgt) {
			if s.Inhibitory {
				continue
			}
			if s.Kind != knowledge.RelationAssociation && s.Kind != knowledge.RelationAffix {
				return s.Kind, s.Weight * s.Confidence, true
			}
		}
	}
	// Structural capability: src -has-> mid -function-> tgt  ⇒ requested can_do
	if n := kb.Registry.GetByID(src); n != nil {
		for _, s1 := range n.OutboundAll() {
			mid := s1.TargetID
			if s1.Inhibitory || s1.Kind != knowledge.RelationHas {
				continue
			}
			mn := kb.Registry.GetByID(mid)
			if mn == nil {
				continue
			}
			for _, s2 := range mn.SynapsesTo(tgt) {
				if !s2.Inhibitory && s2.Kind == knowledge.RelationFunction {
					return knowledge.RelationCanDo, (s1.Confidence + s2.Confidence) / 2, true
				}
			}
		}
	}
	// Active neighborhood 2-hop has→function
	for _, e1 := range active {
		if e1.SourceID != src || e1.Inhibitory || e1.Kind != knowledge.RelationHas {
			continue
		}
		for _, e2 := range active {
			if e2.SourceID == e1.TargetID && e2.TargetID == tgt && !e2.Inhibitory && e2.Kind == knowledge.RelationFunction {
				return knowledge.RelationCanDo, (e1.Confidence + e2.Confidence) / 2, true
			}
		}
	}
	// Unsupported capability claim: tgt is capability-shaped in memory (function/can_do inbound)
	// or path exists; never invent claims to arbitrary bare framing tokens.
	if hasInboundCapabilityRole(kb, tgt) {
		sn := kb.Registry.GetByID(src)
		if sn != nil {
			srcTyped := 0
			for _, s := range sn.OutboundAll() {
				if !s.Inhibitory && s.Kind != knowledge.RelationAssociation && s.Kind != knowledge.RelationAffix {
					srcTyped++
				}
			}
			if srcTyped > 0 {
				return knowledge.RelationCanDo, 0.35, true
			}
		}
	}
	return "", 0, false
}

func hasInboundCapabilityRole(kb *knowledge.KnowledgeBase, id knowledge.NodeID) bool {
	for _, n := range kb.Registry.Nodes() {
		if s := n.SynapsesTo(id).Find("", false); false {
		_ = s
	} else if list := n.SynapsesTo(id); len(list) > 0 {
		s := list[0]
			if s.Kind == knowledge.RelationFunction || s.Kind == knowledge.RelationCanDo {
				return true // includes inhibitory can_do (contradiction evidence)
			}
		}
	}
	return false
}

func primaryContent(kb *knowledge.KnowledgeBase, content []knowledge.NodeID, typed map[knowledge.NodeID]int, act map[knowledge.NodeID]float64) knowledge.NodeID {
	var best knowledge.NodeID
	bestS := -1.0
	for _, id := range content {
		s := float64(typed[id]) + act[id]
		if s > bestS {
			bestS = s
			best = id
		}
	}
	return best
}

func hasCanDoFact(facts []Proposition) bool {
	for _, f := range facts {
		if f.Relation == knowledge.RelationCanDo {
			return true
		}
	}
	return false
}

// constraintsFromFunctionalStructure: only Contrast/Condition relations among stimulus nodes.
func constraintsFromFunctionalStructure(
	kb *knowledge.KnowledgeBase,
	stimIDs map[knowledge.NodeID]bool,
	active []understanding.ActiveEdge,
	req []Proposition,
) []Constraint {
	var out []Constraint
	seen := map[knowledge.NodeID]bool{}
	add := func(id knowledge.NodeID, role, note string) {
		if seen[id] || !stimIDs[id] {
			return
		}
		seen[id] = true
		out = append(out, Constraint{ConceptID: id, Token: tokenOf(kb, id), Role: role, Note: note})
	}
	for _, e := range active {
		// Constraint edge requires BOTH ends in current stimulus (not merely available neighborhood).
		if !(stimIDs[e.SourceID] && stimIDs[e.TargetID]) {
			continue
		}
		switch e.Kind {
		case knowledge.RelationContrast:
			add(e.TargetID, "contrast", "stimulus contrast pair")
		case knowledge.RelationCondition:
			add(e.TargetID, "condition", "stimulus condition pair")
		}
	}
	// Also KB contrast/condition between stimulus tokens
	for sid := range stimIDs {
		n := kb.Registry.GetByID(sid)
		if n == nil {
			continue
		}
		for _, s := range n.OutboundAll() {
			tid := s.TargetID
			if !stimIDs[tid] {
				continue
			}
			if s.Kind == knowledge.RelationContrast {
				add(tid, "contrast", "stimulus pair linked by contrast")
			}
			if s.Kind == knowledge.RelationCondition {
				add(tid, "condition", "stimulus pair linked by condition")
			}
		}
	}
	_ = req
	return out
}

func findEvidencePathsFromActive(
	kb *knowledge.KnowledgeBase,
	from, to knowledge.NodeID,
	edgeIndex map[knowledge.NodeID][]understanding.ActiveEdge,
	active []understanding.ActiveEdge,
) []EvidencePath {
	var out []EvidencePath
	// Direct in active edges
	for _, e := range edgeIndex[from] {
		if e.TargetID == to && !e.Inhibitory {
			out = append(out, EvidencePath{
				NodeIDs: []knowledge.NodeID{from, to},
				Tokens:  []string{tokenOf(kb, from), tokenOf(kb, to)},
				Relations: []knowledge.RelationKind{e.Kind}, Support: true, Length: 1,
				Confidence: e.Confidence, Note: "direct_active",
			})
		}
	}
	// 2-hop via active edges only
	for _, e1 := range edgeIndex[from] {
		if e1.Inhibitory || e1.TargetID == to {
			continue
		}
		if e1.Kind == knowledge.RelationAssociation || e1.Kind == knowledge.RelationAffix {
			continue
		}
		for _, e2 := range edgeIndex[e1.TargetID] {
			if e2.Inhibitory || e2.TargetID != to {
				continue
			}
			if e2.Kind == knowledge.RelationAssociation || e2.Kind == knowledge.RelationAffix {
				continue
			}
			out = append(out, EvidencePath{
				NodeIDs: []knowledge.NodeID{from, e1.TargetID, to},
				Tokens:  []string{tokenOf(kb, from), tokenOf(kb, e1.TargetID), tokenOf(kb, to)},
				Relations: []knowledge.RelationKind{e1.Kind, e2.Kind}, Support: true, Length: 2,
				Confidence: (e1.Confidence + e2.Confidence) / 2, Note: "derived_active",
			})
			if len(out) >= maxEvidencePaths {
				return out
			}
		}
	}
	_ = active
	return out
}

// ApplyFunctionalStateToInterpretation attaches FSU fields and scores relevance to request.
func ApplyFunctionalStateToInterpretation(kb *knowledge.KnowledgeBase, interp *Interpretation, fs FunctionalState) {
	if interp == nil {
		return
	}
	interp.FunctionalSignals = append(interp.FunctionalSignals, fs.Signals...)
	// Prefer requested props; fall back to background for describe
	if len(fs.RequestedProps) > 0 {
		interp.Propositions = preferredRequestedProps(fs.RequestedProps)
	} else if fs.GoalDescribe {
		interp.Propositions = append([]Proposition{}, fs.BackgroundFacts...)
	} else {
		interp.Propositions = preferredRequestedProps(append(append([]Proposition{}, fs.RequestedProps...), fs.BackgroundFacts...))
	}
	interp.Constraints = append([]Constraint{}, fs.Constraints...)
	interp.EvidencePaths = filterPathsForInterp(fs.EvidencePaths, interp)
	// Boost coherence when focus matches primary requested target
	for _, p := range fs.RequestedProps {
		if p.Relation == knowledge.RelationCanDo && fs.GoalEvaluate {
			// Under evaluation, focus is the agent of the requested capability claim.
			interp.FocusID = p.TargetID
			interp.ContextScore = clamp01(interp.ContextScore + 0.2)
			interp.TotalScore = clamp01(interp.TotalScore + 0.12)
			break
		}
		if p.TargetID == interp.FocusID {
			interp.ContextScore = clamp01(interp.ContextScore + 0.15)
			interp.TotalScore = clamp01(interp.TotalScore + 0.08)
		}
	}
	if fs.GoalDescribe && len(fs.ContentIDs) > 0 && interp.FocusID == fs.ContentIDs[0] {
		interp.TotalScore = clamp01(interp.TotalScore + 0.05)
	}
	// Phase-1 structural signals remain available on the interpretation.
	st := CognitiveState{StimulusTokens: nil}
	for _, r := range interp.Relations {
		st.ActiveRelations = append(st.ActiveRelations, r)
	}
	p1 := deriveFunctionalSignals(st, interp, kb)
	interp.FunctionalSignals = append(interp.FunctionalSignals, p1...)
}

func filterPathsForInterp(paths []EvidencePath, interp *Interpretation) []EvidencePath {
	if interp == nil || len(paths) == 0 {
		return paths
	}
	nodeSet := map[knowledge.NodeID]bool{}
	for _, id := range interp.Nodes {
		nodeSet[id] = true
	}
	var out []EvidencePath
	for _, p := range paths {
		ok := true
		for _, id := range p.NodeIDs {
			if !nodeSet[id] && len(nodeSet) > 0 {
				// allow paths that start at focus even if intermediate not in node list
				if id != interp.FocusID {
					// still keep if any endpoint in interp
				}
			}
		}
		if ok {
			out = append(out, p)
		}
	}
	if len(out) == 0 {
		return paths
	}
	return out
}

// DecideEvaluation computes evaluation status from I* + functional state (Decision layer logic).
func propID(src knowledge.NodeID, kind knowledge.RelationKind, tgt knowledge.NodeID) string {
	return fmt.Sprintf("%d|%s|%d", src, kind, tgt)
}

// EvaluateEvidenceForProposition scores candidate paths only against the given proposition.
// Paths about other objects/relations are IRRELEVANT (not SUPPORT).
func EvaluateEvidenceForProposition(
	kb *knowledge.KnowledgeBase,
	prop Proposition,
	paths []EvidencePath,
	constraints []Constraint,
) []EvidenceEvaluation {
	var out []EvidenceEvaluation
	seen := map[string]bool{}
	for _, ep := range paths {
		key := FormatEvidencePath(ep)
		if seen[key] {
			continue
		}
		seen[key] = true
		ev := EvidenceEvaluation{PropositionID: prop.ID, Path: ep, Confidence: ep.Confidence}
		j := judgePathAgainstProposition(kb, prop, ep)
		ev.Judgement = j
		switch j {
		case EvidenceSupport:
			ev.Note = "path supports requested proposition"
		case EvidenceContradiction:
			ev.Note = "path contradicts requested proposition"
		default:
			ev.Note = "path not about requested proposition"
		}
		out = append(out, ev)
	}
	// Direct KB check for inhibitory (contradiction) and matching edge (support)
	if kb != nil {
		if n := kb.Registry.GetByID(prop.TargetID); n != nil {
			for _, s := range n.SynapsesTo(prop.ObjectID) {
				if s.Inhibitory {
					out = append(out, EvidenceEvaluation{
						PropositionID: prop.ID,
						Path: EvidencePath{
							NodeIDs:   []knowledge.NodeID{prop.TargetID, prop.ObjectID},
							Tokens:    []string{prop.TargetTok, prop.ObjectTok},
							Relations: []knowledge.RelationKind{s.Kind}, Length: 1, Confidence: s.Confidence,
							Note: "inhibitory_direct", Conflict: true,
						},
						Judgement:  EvidenceContradiction,
						Confidence: s.Confidence,
						Note:       "inhibitory synapse contradicts proposition",
					})
				} else if s.Kind == prop.Relation {
					out = append(out, EvidenceEvaluation{
						PropositionID: prop.ID,
						Path: EvidencePath{
							NodeIDs:   []knowledge.NodeID{prop.TargetID, prop.ObjectID},
							Tokens:    []string{prop.TargetTok, prop.ObjectTok},
							Relations: []knowledge.RelationKind{s.Kind}, Support: true, Length: 1, Confidence: s.Confidence,
							Note: "direct_match",
						},
						Judgement:  EvidenceSupport,
						Confidence: s.Confidence,
						Note:       "direct matching relation",
					})
				}
			}
		}
	}
	// Constraint on intermediate of a support-shaped path for this prop → contradiction/conflict signal
	for _, c := range constraints {
		for _, ep := range paths {
			if ep.Length < 2 {
				continue
			}
			// only if path is about this prop's object
			if len(ep.NodeIDs) == 0 || ep.NodeIDs[len(ep.NodeIDs)-1] != prop.ObjectID {
				continue
			}
			if ep.NodeIDs[0] != prop.TargetID {
				continue
			}
			for _, id := range ep.NodeIDs[1 : len(ep.NodeIDs)-1] {
				if id == c.ConceptID {
					out = append(out, EvidenceEvaluation{
						PropositionID: prop.ID,
						Path:          ep,
						Judgement:     EvidenceContradiction,
						Confidence:    ep.Confidence,
						Note:          "constraint blocks support path for this proposition",
					})
				}
			}
		}
	}
	return out
}

func judgePathAgainstProposition(kb *knowledge.KnowledgeBase, prop Proposition, ep EvidencePath) EvidenceJudgement {
	if len(ep.NodeIDs) < 2 {
		return EvidenceIrrelevant
	}
	start, end := ep.NodeIDs[0], ep.NodeIDs[len(ep.NodeIDs)-1]
	// Must concern same agent and object
	if start != prop.TargetID || end != prop.ObjectID {
		return EvidenceIrrelevant
	}
	if ep.Conflict {
		return EvidenceContradiction
	}
	// Direct same relation
	if ep.Length == 1 && len(ep.Relations) == 1 {
		if ep.Relations[0] == prop.Relation {
			return EvidenceSupport
		}
		// same endpoints different relation → irrelevant to this proposition identity
		return EvidenceIrrelevant
	}
	// Derived has→function supports can_do for same object
	if prop.Relation == knowledge.RelationCanDo && ep.Length >= 2 {
		ok := true
		for _, r := range ep.Relations {
			if r == knowledge.RelationAssociation || r == knowledge.RelationAffix {
				ok = false
			}
		}
		if ok {
			return EvidenceSupport
		}
	}
	_ = kb
	return EvidenceIrrelevant
}

// DecideEvaluation applies epistemic rules for the primary requested proposition only.
func DecideEvaluation(interp *Interpretation, fs FunctionalState, kb *knowledge.KnowledgeBase) EvalStatus {
	if interp == nil {
		return EvalUnknown
	}
	prop := selectPrimaryRequested(interp, fs)
	if prop.ID == "" && prop.TargetID == 0 {
		return EvalUnknown
	}
	if prop.ID == "" {
		prop.ID = propID(prop.TargetID, prop.Relation, prop.ObjectID)
	}
	// Bind requested proposition on interpretation for Language identity
	interp.RequestedProposition = prop
	if len(interp.Propositions) == 0 || !interp.Propositions[0].Requested {
		interp.Propositions = preferredRequestedProps(append([]Proposition{prop}, interp.Propositions...))
	}

	// Candidate paths: only those that could concern this proposition (same endpoints or derived).
	var paths []EvidencePath
	for _, ep := range append(append([]EvidencePath{}, interp.EvidencePaths...), fs.EvidencePaths...) {
		paths = append(paths, ep)
	}
	if kb != nil {
		// Fresh candidates from KB neighborhood structure for THIS proposition only
		edgeIndex := map[knowledge.NodeID][]understanding.ActiveEdge{}
		for _, r := range interp.Relations {
			edgeIndex[r.SourceID] = append(edgeIndex[r.SourceID], understanding.ActiveEdge{
				SourceID: r.SourceID, TargetID: r.TargetID, Kind: r.Kind,
				Weight: r.Weight, Confidence: r.Confidence, Inhibitory: r.Inhibitory,
			})
		}
		paths = append(paths, findEvidencePathsFromActive(kb, prop.TargetID, prop.ObjectID, edgeIndex, nil)...)
	}
	constr := interp.Constraints
	if len(constr) == 0 {
		constr = fs.Constraints
	}
	evals := EvaluateEvidenceForProposition(kb, prop, paths, constr)
	interp.EvidenceEvals = evals

	support, contradiction := 0, 0
	for _, ev := range evals {
		if ev.PropositionID != prop.ID {
			continue
		}
		switch ev.Judgement {
		case EvidenceSupport:
			support++
		case EvidenceContradiction:
			contradiction++
		}
	}
	switch {
	case support > 0 && contradiction == 0:
		return EvalSupported
	case contradiction > 0 && support == 0:
		return EvalContradicted
	case support > 0 && contradiction > 0:
		return EvalConflicted
	default:
		return EvalUnknown
	}
}

func selectPrimaryRequested(interp *Interpretation, fs FunctionalState) Proposition {
	stimPos := map[string]int{}
	for i, tok := range fs.Stimulus {
		stimPos[tok] = i
	}
	var cands []Proposition
	for _, p := range interp.Propositions {
		if p.Requested {
			cands = append(cands, p)
		}
	}
	for _, p := range fs.RequestedProps {
		cands = append(cands, p)
	}
	if len(cands) == 0 {
		if len(interp.Propositions) > 0 {
			return interp.Propositions[0]
		}
		return Proposition{}
	}
	best := cands[0]
	for _, p := range cands[1:] {
		// Prefer can_do over other relations
		if p.Relation == knowledge.RelationCanDo && best.Relation != knowledge.RelationCanDo {
			best = p
			continue
		}
		if p.Relation != knowledge.RelationCanDo && best.Relation == knowledge.RelationCanDo {
			continue
		}
		// Prefer stronger evidence strength
		if p.Strength > best.Strength+0.05 {
			best = p
			continue
		}
		if best.Strength > p.Strength+0.05 {
			continue
		}
		// Tie-break: later object in stimulus (capability patient tends to be late)
		if stimPos[p.ObjectTok] > stimPos[best.ObjectTok] {
			best = p
		}
	}
	return best
}


func preferredRequestedProps(props []Proposition) []Proposition {
	if len(props) == 0 {
		return props
	}
	var canDo, other []Proposition
	maxCan := 0.0
	for _, p := range props {
		if p.Relation == knowledge.RelationCanDo {
			canDo = append(canDo, p)
			if p.Strength > maxCan {
				maxCan = p.Strength
			}
		} else {
			other = append(other, p)
		}
	}
	if len(canDo) > 0 {
		var strong []Proposition
		for _, p := range canDo {
			// Keep evidenced / strong claims; drop weak bare-token noise when stronger exists.
			if p.Strength >= 0.5 || p.Strength >= maxCan-0.05 {
				strong = append(strong, p)
			}
		}
		if len(strong) == 0 {
			strong = canDo
		}
		return append(strong, other...)
	}
	return props
}

func tokenOf(kb *knowledge.KnowledgeBase, id knowledge.NodeID) string {
	if n := kb.Registry.GetByID(id); n != nil {
		return n.Token
	}
	return ""
}

func FormatEvidencePath(p EvidencePath) string {
	if len(p.Tokens) == 0 {
		return p.Note
	}
	var b strings.Builder
	for i, tok := range p.Tokens {
		if i > 0 {
			rel := "?"
			if i-1 < len(p.Relations) {
				rel = string(p.Relations[i-1])
			}
			b.WriteString(" -[" + rel + "]-> ")
		}
		b.WriteString(tok)
	}
	if p.Note != "" {
		b.WriteString(" (" + p.Note + ")")
	}
	return b.String()
}

func FormatFunctionalStateSummary(fs FunctionalState) string {
	return fmt.Sprintf("describe=%v evaluate=%v assert=%v explain=%v content=%d framing=%d req=%d facts=%d constr=%d paths=%d",
		fs.GoalDescribe, fs.GoalEvaluate, fs.GoalAssert, fs.GoalExplain,
		len(fs.ContentIDs), len(fs.FramingIDs), len(fs.RequestedProps), len(fs.BackgroundFacts),
		len(fs.Constraints), len(fs.EvidencePaths))
}
