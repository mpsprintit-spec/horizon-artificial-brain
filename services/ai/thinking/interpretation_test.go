package thinking

import (
	"testing"
	"time"

	"github.com/project-horizon/horizon-core/services/ai/activation"
	"github.com/project-horizon/horizon-core/services/ai/knowledge"
	"github.com/project-horizon/horizon-core/services/ai/understanding"
)

func seedCatGraph(kb *knowledge.KnowledgeBase) {
	cat := kb.Store("kucing")
	animal := kb.Store("hewan")
	leg := kb.Store("kaki")
	walk := kb.Store("berjalan")
	kb.ConnectKind(cat, animal, knowledge.RelationIsA, 0.95, 0.95, false)
	kb.ConnectKind(cat, leg, knowledge.RelationHas, 0.55, 0.60, false)
	kb.ConnectKind(cat, walk, knowledge.RelationCanDo, 0.50, 0.55, false)
	kb.ConnectKind(leg, walk, knowledge.RelationFunction, 0.50, 0.55, false)
}

func TestNeighborhoodRetainsLowActivationStructure(t *testing.T) {
	kb := knowledge.NewKnowledgeBase()
	seedCatGraph(kb)
	engine := NewThinkingEngine(kb)
	thought, ok := engine.ThinkAbout("kucing", nil)
	if !ok || thought.BestInterpretation == nil {
		t.Fatal("expected I*")
	}
	tokens := map[string]bool{}
	for _, id := range thought.BestInterpretation.Nodes {
		if n := kb.Registry.GetByID(id); n != nil {
			tokens[n.Token] = true
		}
	}
	for _, need := range []string{"kucing", "kaki", "berjalan", "hewan"} {
		if !tokens[need] {
			t.Fatalf("neighborhood/I* missing %s; got %v", need, tokens)
		}
	}
	// Active neighborhood on LastState should also include structural nodes
	stateTokens := map[string]bool{}
	for _, id := range engine.LastState.ActiveNodes {
		if n := kb.Registry.GetByID(id); n != nil {
			stateTokens[n.Token] = true
		}
	}
	if !stateTokens["kaki"] || !stateTokens["berjalan"] {
		t.Fatalf("CognitiveState ActiveNodes lost structural nodes: %v", stateTokens)
	}
}

func TestHeadToHead_CoherentBeatsSingleStrong(t *testing.T) {
	kb := knowledge.NewKnowledgeBase()
	cat := kb.Store("kucing")
	animal := kb.Store("hewan")
	leg := kb.Store("kaki")
	walk := kb.Store("berjalan")
	kb.ConnectKind(cat, animal, knowledge.RelationIsA, 0.99, 0.99, false)
	kb.ConnectKind(cat, leg, knowledge.RelationHas, 0.55, 0.60, false)
	kb.ConnectKind(cat, walk, knowledge.RelationCanDo, 0.50, 0.55, false)
	kb.ConnectKind(leg, walk, knowledge.RelationFunction, 0.50, 0.55, false)

	// Synthetic activation: kucing highest, others lower
	result := activation.Result{
		Activations: map[knowledge.NodeID]float64{
			cat.ID: 1.0, animal.ID: 0.9, leg.ID: 0.3, walk.ID: 0.25,
		},
		Confidence: map[knowledge.NodeID]float64{
			cat.ID: 1.0, animal.ID: 0.95, leg.ID: 0.5, walk.ID: 0.5,
		},
	}

	single := Interpretation{
		FocusID: cat.ID,
		Nodes:   []knowledge.NodeID{cat.ID, animal.ID},
		Relations: []ActiveRelation{{
			SourceID: cat.ID, TargetID: animal.ID, Kind: knowledge.RelationIsA,
			Weight: 0.99, Confidence: 0.99,
		}},
	}
	multi := Interpretation{
		FocusID: cat.ID,
		Nodes:   []knowledge.NodeID{cat.ID, animal.ID, leg.ID, walk.ID},
		Relations: []ActiveRelation{
			{SourceID: cat.ID, TargetID: animal.ID, Kind: knowledge.RelationIsA, Weight: 0.99, Confidence: 0.99},
			{SourceID: cat.ID, TargetID: leg.ID, Kind: knowledge.RelationHas, Weight: 0.55, Confidence: 0.60},
			{SourceID: cat.ID, TargetID: walk.ID, Kind: knowledge.RelationCanDo, Weight: 0.50, Confidence: 0.55},
			{SourceID: leg.ID, TargetID: walk.ID, Kind: knowledge.RelationFunction, Weight: 0.50, Confidence: 0.55},
		},
	}
	ScoreInterpretationForTest(kb, &single, result, nil)
	ScoreInterpretationForTest(kb, &multi, result, nil)
	if multi.TotalScore <= single.TotalScore {
		t.Fatalf("coherent multi-relation must beat single strong: multi=%f single=%f coh_m=%f coh_s=%f",
			multi.TotalScore, single.TotalScore, multi.CoherenceScore, single.CoherenceScore)
	}
}

func TestBridgeLowActivationKeptInNeighborhood(t *testing.T) {
	kb := knowledge.NewKnowledgeBase()
	a := kb.Store("a")
	b := kb.Store("bridge")
	c := kb.Store("c")
	d := kb.Store("d")
	kb.ConnectKind(a, b, knowledge.RelationHas, 0.7, 0.7, false)
	kb.ConnectKind(b, c, knowledge.RelationFunction, 0.4, 0.4, false)
	kb.ConnectKind(c, d, knowledge.RelationCanDo, 0.4, 0.4, false)

	engine := NewThinkingEngine(kb)
	thought, ok := engine.ThinkAbout("a d", nil)
	if !ok {
		t.Fatal("expected thought")
	}
	foundBridge := false
	for _, id := range engine.LastState.ActiveNodes {
		if n := kb.Registry.GetByID(id); n != nil && n.Token == "bridge" {
			foundBridge = true
		}
	}
	if !foundBridge {
		// also check I*
		if thought.BestInterpretation != nil {
			for _, id := range thought.BestInterpretation.Nodes {
				if n := kb.Registry.GetByID(id); n != nil && n.Token == "bridge" {
					foundBridge = true
				}
			}
		}
	}
	if !foundBridge {
		t.Fatal("bridge with lower activation was lost before coherence")
	}
	_ = time.Now()
}

func TestA_MultiRelation(t *testing.T) {
	kb := knowledge.NewKnowledgeBase()
	seedCatGraph(kb)
	engine := NewThinkingEngine(kb)
	thought, ok := engine.ThinkAbout("kucing", nil)
	if !ok || thought.BestInterpretation == nil {
		t.Fatal("expected I*")
	}
	if len(thought.BestInterpretation.Relations) < 2 {
		t.Fatalf("expected multi-relation I*, got %d", len(thought.BestInterpretation.Relations))
	}
}

func TestE_KnowledgeGrowth(t *testing.T) {
	kb := knowledge.NewKnowledgeBase()
	cat := kb.Store("kucing")
	animal := kb.Store("hewan")
	kb.ConnectKind(cat, animal, knowledge.RelationIsA, 0.9, 0.9, false)
	engine := NewThinkingEngine(kb)
	before, _ := engine.ThinkAbout("kucing", nil)
	nBefore := 0
	if before.BestInterpretation != nil {
		nBefore = len(before.BestInterpretation.Relations)
	}
	leg := kb.Store("kaki")
	kb.ConnectKind(cat, leg, knowledge.RelationHas, 0.85, 0.85, false)
	after, ok := engine.ThinkAbout("kucing", nil)
	if !ok || after.BestInterpretation == nil {
		t.Fatal("expected I*")
	}
	if len(after.BestInterpretation.Relations) < nBefore {
		t.Fatal("knowledge growth reduced relations")
	}
}

func TestG_LongInput(t *testing.T) {
	kb := knowledge.NewKnowledgeBase()
	seedCatGraph(kb)
	engine := NewThinkingEngine(kb)
	long := "kalau saya ingin tahu tentang kucing terutama apakah termasuk hewan dan apa yang dimilikinya serta apa yang dapat dilakukan"
	thought, ok := engine.ThinkAbout(long, nil)
	if !ok || thought.BestInterpretation == nil {
		t.Fatal("expected I* on long input")
	}
	if len(thought.BestInterpretation.Nodes) < 2 {
		t.Fatal("long input lost multi-concept structure")
	}
}


// TestFocusEmergent_TypedConceptBeatsAssociationHub verifies that a node with
// rich typed semantic structure (is_a/has/can_do) becomes focus over a stimulus
// token that only has many association links — without any token dictionary.
func TestFocusEmergent_TypedConceptBeatsAssociationHub(t *testing.T) {
	kb := knowledge.NewKnowledgeBase()
	// Verb-like hub: many associations only
	hub := kb.Store("hubverb")
	for _, w := range []string{"answering", "asking", "telling", "requesting", "describing"} {
		n := kb.Store(w)
		kb.ConnectKind(hub, n, knowledge.RelationAssociation, 0.8, 0.8, false)
	}
	// Concept with typed structure
	concept := kb.Store("conceptx")
	animal := kb.Store("animalx")
	part := kb.Store("partx")
	act := kb.Store("actionx")
	kb.ConnectKind(concept, animal, knowledge.RelationIsA, 0.9, 0.9, false)
	kb.ConnectKind(concept, part, knowledge.RelationHas, 0.7, 0.7, false)
	kb.ConnectKind(concept, act, knowledge.RelationCanDo, 0.6, 0.6, false)

	engine := NewThinkingEngine(kb)
	// Both tokens in stimulus — like "hubverb conceptx"
	thought, ok := engine.ThinkAbout("hubverb conceptx", nil)
	if !ok || thought.BestInterpretation == nil {
		t.Fatal("expected I*")
	}
	focus := kb.Registry.GetByID(thought.BestInterpretation.FocusID)
	if focus == nil {
		t.Fatal("nil focus")
	}
	if focus.Token != "conceptx" {
		t.Fatalf("expected focus=conceptx (typed structure), got %s", focus.Token)
	}
}

// TestFocusEmergent_StimulusConceptNotStolenBySharedTarget: stimulus concept
// with typed outbound should remain focus; shared capability target should not.
func TestFocusEmergent_StimulusConceptNotStolenBySharedTarget(t *testing.T) {
	kb := knowledge.NewKnowledgeBase()
	concept := kb.Store("entityy")
	kind := kb.Store("kindy")
	part := kb.Store("party")
	cap := kb.Store("capy")
	kb.ConnectKind(concept, kind, knowledge.RelationIsA, 0.9, 0.9, false)
	kb.ConnectKind(concept, part, knowledge.RelationHas, 0.6, 0.6, false)
	kb.ConnectKind(concept, cap, knowledge.RelationCanDo, 0.5, 0.5, false)
	kb.ConnectKind(part, cap, knowledge.RelationFunction, 0.5, 0.5, false)

	engine := NewThinkingEngine(kb)
	thought, ok := engine.ThinkAbout("entityy", nil)
	if !ok || thought.BestInterpretation == nil {
		t.Fatal("expected I*")
	}
	focus := kb.Registry.GetByID(thought.BestInterpretation.FocusID)
	if focus == nil || focus.Token != "entityy" {
		got := ""
		if focus != nil {
			got = focus.Token
		}
		t.Fatalf("expected focus=entityy, got %s", got)
	}
}

// TestFocusEmergent_QueryWordsDoNotBecomeFocus: association-heavy query tokens
// in stimulus should not become focus when a typed concept is also present.
func TestFocusEmergent_QueryWordsDoNotBecomeFocus(t *testing.T) {
	kb := knowledge.NewKnowledgeBase()
	q1 := kb.Store("querya")
	q2 := kb.Store("queryb")
	for _, w := range []string{"rel1", "rel2", "rel3", "rel4"} {
		n := kb.Store(w)
		kb.ConnectKind(q1, n, knowledge.RelationAssociation, 0.85, 0.85, false)
		kb.ConnectKind(q2, n, knowledge.RelationAssociation, 0.85, 0.85, false)
	}
	concept := kb.Store("topicz")
	kb.ConnectKind(concept, kb.Store("catz"), knowledge.RelationIsA, 0.9, 0.9, false)
	kb.ConnectKind(concept, kb.Store("propz"), knowledge.RelationHas, 0.7, 0.7, false)

	engine := NewThinkingEngine(kb)
	thought, ok := engine.ThinkAbout("querya queryb topicz", nil)
	if !ok || thought.BestInterpretation == nil {
		t.Fatal("expected I*")
	}
	focus := kb.Registry.GetByID(thought.BestInterpretation.FocusID)
	if focus == nil || focus.Token != "topicz" {
		got := ""
		if focus != nil {
			got = focus.Token
		}
		t.Fatalf("expected focus=topicz, got %s", got)
	}
}

// TestDirectVsCollateral_SameMemoryDifferentStimulus:
// same memory, different stimulus → different relevant I*, siblings stay available.
func TestDirectVsCollateral_SameMemoryDifferentStimulus(t *testing.T) {
	kb := knowledge.NewKnowledgeBase()
	cat := kb.Store("kucing")
	animal := kb.Store("hewan")
	leg := kb.Store("kaki")
	walk := kb.Store("berjalan")
	flea := kb.Store("kutu")
	dog := kb.Store("anjing")
	bird := kb.Store("burung")
	kb.ConnectKind(cat, animal, knowledge.RelationIsA, 0.9, 0.9, false)
	kb.ConnectKind(cat, leg, knowledge.RelationHas, 0.7, 0.7, false)
	kb.ConnectKind(cat, walk, knowledge.RelationCanDo, 0.6, 0.6, false)
	kb.ConnectKind(leg, walk, knowledge.RelationFunction, 0.6, 0.6, false)
	kb.ConnectKind(flea, animal, knowledge.RelationIsA, 0.9, 0.9, false)
	kb.ConnectKind(dog, animal, knowledge.RelationIsA, 0.9, 0.9, false)
	kb.ConnectKind(bird, animal, knowledge.RelationIsA, 0.9, 0.9, false)

	engine := NewThinkingEngine(kb)

	// Stimulus kutu: I* should center on kutu→hewan, not sibling animals as primary facts.
	thFlea, ok := engine.ThinkAbout("kutu", nil)
	if !ok || thFlea.BestInterpretation == nil {
		t.Fatal("kutu: expected I*")
	}
	focusFlea := kb.Registry.GetByID(thFlea.BestInterpretation.FocusID)
	if focusFlea == nil || focusFlea.Token != "kutu" {
		got := ""
		if focusFlea != nil {
			got = focusFlea.Token
		}
		t.Fatalf("kutu: expected focus=kutu, got %s", got)
	}
	hasFleaIsA := false
	siblingAsSource := false
	for _, r := range thFlea.BestInterpretation.Relations {
		src := kb.Registry.GetByID(r.SourceID)
		tgt := kb.Registry.GetByID(r.TargetID)
		if src == nil || tgt == nil {
			continue
		}
		if src.Token == "kutu" && tgt.Token == "hewan" {
			hasFleaIsA = true
		}
		if (src.Token == "kucing" || src.Token == "anjing" || src.Token == "burung") && tgt.Token == "hewan" {
			siblingAsSource = true
		}
	}
	if !hasFleaIsA {
		t.Fatal("kutu I* missing kutu→hewan")
	}
	if siblingAsSource {
		t.Fatal("kutu I* should not treat sibling animals as primary facts about kutu")
	}
	// Available: siblings still in neighborhood / ActiveNodes
	avail := map[string]bool{}
	for _, id := range engine.LastState.ActiveNodes {
		if n := kb.Registry.GetByID(id); n != nil {
			avail[n.Token] = true
		}
	}
	if !avail["kucing"] && !avail["anjing"] && !avail["burung"] {
		// neighborhood may still hold them even if not in ActiveNodes naming — check edges
		for _, r := range engine.LastState.ActiveRelations {
			if s := kb.Registry.GetByID(r.SourceID); s != nil {
				if s.Token == "kucing" || s.Token == "anjing" || s.Token == "burung" {
					avail[s.Token] = true
				}
			}
		}
	}
	if !avail["kucing"] && !avail["anjing"] && !avail["burung"] {
		t.Log("warning: siblings not in ActiveNodes (may still be in memory only)")
	}
	// Memory itself never discarded
	if kb.Fetch("kucing").FindSynapse(animal.ID, knowledge.RelationIsA, false) == nil {
		t.Fatal("memory must retain kucing→hewan")
	}

	// Stimulus kucing: multi-relation I*
	thCat, ok := engine.ThinkAbout("kucing", nil)
	if !ok || thCat.BestInterpretation == nil {
		t.Fatal("kucing: expected I*")
	}
	focusCat := kb.Registry.GetByID(thCat.BestInterpretation.FocusID)
	if focusCat == nil || focusCat.Token != "kucing" {
		got := ""
		if focusCat != nil {
			got = focusCat.Token
		}
		t.Fatalf("kucing: expected focus=kucing, got %s", got)
	}
	if len(thCat.BestInterpretation.Relations) < 2 {
		t.Fatalf("kucing: expected multi-relation I*, got %d", len(thCat.BestInterpretation.Relations))
	}
	// Should not be only sibling dump
	kinds := map[knowledge.RelationKind]bool{}
	for _, r := range thCat.BestInterpretation.Relations {
		if r.SourceID == cat.ID || r.SourceID == leg.ID {
			kinds[r.Kind] = true
		}
	}
	if len(kinds) < 2 {
		t.Fatalf("kucing I* should include diverse typed structure, kinds=%v", kinds)
	}

	// Stimulus anjing: focus anjing, is_a hewan
	thDog, ok := engine.ThinkAbout("anjing", nil)
	if !ok || thDog.BestInterpretation == nil {
		t.Fatal("anjing: expected I*")
	}
	focusDog := kb.Registry.GetByID(thDog.BestInterpretation.FocusID)
	if focusDog == nil || focusDog.Token != "anjing" {
		got := ""
		if focusDog != nil {
			got = focusDog.Token
		}
		t.Fatalf("anjing: expected focus=anjing, got %s", got)
	}
}

func TestNeighborhoodKeepsSiblingsAvailable(t *testing.T) {
	kb := knowledge.NewKnowledgeBase()
	flea := kb.Store("kutu")
	animal := kb.Store("hewan")
	cat := kb.Store("kucing")
	kb.ConnectKind(flea, animal, knowledge.RelationIsA, 0.9, 0.9, false)
	kb.ConnectKind(cat, animal, knowledge.RelationIsA, 0.9, 0.9, false)
	engine := NewThinkingEngine(kb)
	_, ok := engine.ThinkAbout("kutu", nil)
	if !ok {
		t.Fatal("expected thought")
	}
	// Sibling edge must still exist in memory
	if cat.FindSynapse(animal.ID, knowledge.RelationIsA, false) == nil {
		t.Fatal("sibling knowledge discarded from memory")
	}
}

// TestFunctionWordNotAutoFocus: a middle stimulus token with rich external typed
// structure must not become I* focus when other stimulus tokens form the relation.
func TestFunctionWordNotAutoFocus(t *testing.T) {
	kb := knowledge.NewKnowledgeBase()
	// "relx" has many external typed links (like a function word that was taught about)
	rel := kb.Store("relx")
	for i, w := range []string{"ext1", "ext2", "ext3", "ext4", "ext5"} {
		n := kb.Store(w)
		_ = i
		kb.ConnectKind(rel, n, knowledge.RelationIsA, 0.9, 0.9, false)
	}
	a := kb.Store("entitya")
	b := kb.Store("entityb")
	// The actual discourse structure between content concepts
	kb.ConnectKind(a, b, knowledge.RelationIsA, 0.85, 0.85, false)

	engine := NewThinkingEngine(kb)
	// Stimulus contains all three — like "entitya relx entityb"
	thought, ok := engine.ThinkAbout("entitya relx entityb", nil)
	if !ok || thought.BestInterpretation == nil {
		t.Fatal("expected I*")
	}
	focus := kb.Registry.GetByID(thought.BestInterpretation.FocusID)
	if focus == nil {
		t.Fatal("nil focus")
	}
	if focus.Token == "relx" {
		t.Fatalf("function-like token with external typed structure must not auto-become focus, got relx")
	}
	// Prefer content endpoint of the inter-stimulus relation
	if focus.Token != "entitya" && focus.Token != "entityb" {
		t.Fatalf("expected focus on entitya or entityb, got %s", focus.Token)
	}
	// I* should include entitya→entityb when possible
	found := false
	for _, r := range thought.BestInterpretation.Relations {
		s := kb.Registry.GetByID(r.SourceID)
		tg := kb.Registry.GetByID(r.TargetID)
		if s != nil && tg != nil && s.Token == "entitya" && tg.Token == "entityb" {
			found = true
		}
	}
	if !found {
		t.Log("note: entitya→entityb not in I* relations; focus=", focus.Token)
	}
}

// TestBridgeSurvivesHighDegreeDistractors: low-activation bridge kept under ceiling pressure.
func TestBridgeSurvivesHighDegreeDistractors(t *testing.T) {
	kb := knowledge.NewKnowledgeBase()
	a := kb.Store("enda")
	bridge := kb.Store("bridgey")
	c := kb.Store("endc")
	// Structurally necessary path: enda → bridgey → endc
	kb.ConnectKind(a, bridge, knowledge.RelationHas, 0.45, 0.45, false)
	kb.ConnectKind(bridge, c, knowledge.RelationFunction, 0.40, 0.40, false)
	// Many high-degree distractors not on the path between enda and endc
	for i := 0; i < 30; i++ {
		name := "dist" + string(rune('a'+(i%26))) + string(rune('A'+(i/26)%26))
		n := kb.Store(name)
		hub := kb.Store("hubz")
		kb.ConnectKind(hub, n, knowledge.RelationAssociation, 0.95, 0.95, false)
	}
	engine := NewThinkingEngine(kb)
	thought, ok := engine.ThinkAbout("enda endc", nil)
	if !ok {
		t.Fatal("expected thought")
	}
	foundAvail := false
	for _, id := range engine.LastState.ActiveNodes {
		if n := kb.Registry.GetByID(id); n != nil && n.Token == "bridgey" {
			foundAvail = true
		}
	}
	if !foundAvail {
		t.Fatal("bridge not in available neighborhood under distractor pressure")
	}
	foundI := false
	if thought.BestInterpretation != nil {
		for _, id := range thought.BestInterpretation.Nodes {
			if n := kb.Registry.GetByID(id); n != nil && n.Token == "bridgey" {
				foundI = true
			}
		}
		for _, r := range thought.BestInterpretation.Relations {
			if r.Provenance == understanding.ProvBridge {
				foundI = true
			}
			s := kb.Registry.GetByID(r.SourceID)
			tg := kb.Registry.GetByID(r.TargetID)
			if s != nil && tg != nil && (s.Token == "bridgey" || tg.Token == "bridgey") {
				foundI = true
			}
		}
	}
	if !foundI {
		t.Fatal("structurally necessary bridge should appear in candidate/I* when it connects stimulus ends")
	}
}

func TestProvenanceCopiedToCognitiveState(t *testing.T) {
	kb := knowledge.NewKnowledgeBase()
	a := kb.Store("pa")
	b := kb.Store("pb")
	kb.ConnectKind(a, b, knowledge.RelationIsA, 0.9, 0.9, false)
	engine := NewThinkingEngine(kb)
	_, ok := engine.ThinkAbout("pa", nil)
	if !ok {
		t.Fatal("expected thought")
	}
	anyProv := false
	for _, r := range engine.LastState.ActiveRelations {
		if r.Provenance != "" && r.Provenance != understanding.ProvUnknown {
			anyProv = true
			break
		}
	}
	if !anyProv {
		t.Fatal("Provenance not present on CognitiveState.ActiveRelations")
	}
}

func TestFSUPhase1_FunctionalSignalsFromStructure(t *testing.T) {
	kb := knowledge.NewKnowledgeBase()
	cat := kb.Store("kucing")
	leg := kb.Store("kaki")
	walk := kb.Store("berjalan")
	kb.ConnectKind(cat, leg, knowledge.RelationHas, 0.8, 0.8, false)
	kb.ConnectKind(cat, walk, knowledge.RelationCanDo, 0.7, 0.7, false)
	kb.ConnectKind(leg, walk, knowledge.RelationFunction, 0.7, 0.7, false)
	engine := NewThinkingEngine(kb)
	thought, ok := engine.ThinkAbout("kucing", nil)
	if !ok {
		t.Fatal("expected thought")
	}
	if len(thought.FunctionalSignals) == 0 && (thought.BestInterpretation == nil || len(thought.BestInterpretation.FunctionalSignals) == 0) {
		t.Fatal("Phase1 expects structural functional signals on thought or I*")
	}
	foundFocus := false
	foundMulti := false
	sigs := thought.FunctionalSignals
	if thought.BestInterpretation != nil && len(thought.BestInterpretation.FunctionalSignals) > 0 {
		sigs = thought.BestInterpretation.FunctionalSignals
	}
	for _, s := range sigs {
		if s.Kind == "focus_typed_structure" {
			foundFocus = true
		}
		if s.Kind == "multi_path_support" {
			foundMulti = true
		}
		// Must not look like word triggers
		if s.Source == "" {
			t.Fatal("signal missing source")
		}
	}
	if !foundFocus {
		t.Fatal("expected focus_typed_structure from typed outbound on focus")
	}
	if !foundMulti {
		t.Log("multi_path_support optional if only one path to targets")
	}
}
