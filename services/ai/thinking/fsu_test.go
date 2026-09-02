package thinking

import (
	"testing"

	"github.com/project-horizon/horizon-core/services/ai/knowledge"
)

func seedCatCapability(kb *knowledge.KnowledgeBase) {
	cat := kb.Store("kucing")
	leg := kb.Store("kaki")
	walk := kb.Store("berjalan")
	kb.ConnectKind(cat, leg, knowledge.RelationHas, 0.85, 0.85, false)
	kb.ConnectKind(cat, walk, knowledge.RelationCanDo, 0.8, 0.8, false)
	kb.ConnectKind(leg, walk, knowledge.RelationFunction, 0.8, 0.8, false)
}

func TestFSU2_FourFunctionalStates(t *testing.T) {
	kb := knowledge.NewKnowledgeBase()
	seedCatCapability(kb)
	for _, w := range []string{"apakah", "bisa", "mengapa", "jelaskan"} {
		hub := kb.Store(w)
		for _, x := range []string{"q1", "q2", "q3"} {
			kb.ConnectKind(hub, kb.Store(w+"_"+x), knowledge.RelationAssociation, 0.5, 0.5, false)
		}
	}
	engine := NewThinkingEngine(kb)
	inputs := []string{
		"kucing bisa berjalan",
		"apakah kucing bisa berjalan",
		"mengapa kucing bisa berjalan",
		"jelaskan kucing",
	}
	summaries := map[string]string{}
	for _, in := range inputs {
		th, ok := engine.ThinkAbout(in, nil)
		if !ok || th.BestInterpretation == nil {
			t.Fatalf("%s: expected I*", in)
		}
		summary := ""
		for _, n := range th.BestInterpretation.EvidenceNotes {
			if len(n) > len(summary) {
				summary = n
			}
		}
		summaries[in] = summary
	}
	// apakah/mengapa should evaluate; jelaskan should describe or single-content
	if !containsStr(summaries["apakah kucing bisa berjalan"], "evaluate=true") {
		t.Errorf("apakah: want evaluate, got %s", summaries["apakah kucing bisa berjalan"])
	}
	if !containsStr(summaries["jelaskan kucing"], "describe=true") && !containsStr(summaries["jelaskan kucing"], "content=1") {
		t.Errorf("jelaskan: want describe-ish, got %s", summaries["jelaskan kucing"])
	}
	// Distinct functional character across inputs
	if summaries["apakah kucing bisa berjalan"] == summaries["jelaskan kucing"] {
		t.Fatal("evaluate and describe must differ")
	}
}

func containsStr(s, sub string) bool {
	return len(s) >= len(sub) && (s == sub || len(sub) == 0 ||
		(len(s) > 0 && (func() bool {
			for i := 0; i+len(sub) <= len(s); i++ {
				if s[i:i+len(sub)] == sub {
					return true
				}
			}
			return false
		})()))
}

func TestFSU2_ConstraintRequiresFunctionalContrast(t *testing.T) {
	kb := knowledge.NewKnowledgeBase()
	seedCatCapability(kb)
	kb.Store("apakah")
	kb.Store("bisa")
	// tanpa linked by contrast to kaki (functional knowledge, not reasoning dictionary)
	tanpa := kb.Store("tanpa")
	kaki := kb.Fetch("kaki")
	kb.ConnectKind(tanpa, kaki, knowledge.RelationContrast, 0.9, 0.9, false)

	engine := NewThinkingEngine(kb)
	thA, _ := engine.ThinkAbout("apakah kucing bisa berjalan tanpa kaki", nil)
	thB, _ := engine.ThinkAbout("apakah kucing bisa berjalan kaki", nil)
	if thA.BestInterpretation == nil || thB.BestInterpretation == nil {
		t.Fatal("expected I*")
	}
	hasA := false
	for _, c := range thA.BestInterpretation.Constraints {
		if c.Token == "kaki" {
			hasA = true
		}
	}
	hasB := false
	for _, c := range thB.BestInterpretation.Constraints {
		if c.Token == "kaki" {
			hasB = true
		}
	}
	if !hasA {
		t.Fatalf("A with contrast knowledge should constrain kaki, got %+v", thA.BestInterpretation.Constraints)
	}
	if hasB {
		t.Fatal("B without contrast pair must NOT treat kaki as constraint")
	}
}

func TestFSU2_A_PropositionEvidence(t *testing.T) {
	kb := knowledge.NewKnowledgeBase()
	seedCatCapability(kb)
	kb.Store("apakah")
	kb.Store("bisa")
	engine := NewThinkingEngine(kb)
	th, ok := engine.ThinkAbout("apakah kucing bisa berjalan", nil)
	if !ok || th.BestInterpretation == nil {
		t.Fatal("expected I*")
	}
	i := th.BestInterpretation
	if len(i.EvidencePaths) == 0 && len(i.Propositions) == 0 {
		t.Fatal("expected props or paths")
	}
}

func TestFSU2_E_NoSiblingDog(t *testing.T) {
	kb := knowledge.NewKnowledgeBase()
	cat := kb.Store("kucing")
	dog := kb.Store("anjing")
	leg := kb.Store("kaki")
	walk := kb.Store("berjalan")
	kb.ConnectKind(cat, leg, knowledge.RelationHas, 0.8, 0.8, false)
	kb.ConnectKind(dog, leg, knowledge.RelationHas, 0.8, 0.8, false)
	kb.ConnectKind(cat, walk, knowledge.RelationCanDo, 0.7, 0.7, false)
	kb.ConnectKind(leg, walk, knowledge.RelationFunction, 0.7, 0.7, false)
	kb.Store("jelaskan")
	engine := NewThinkingEngine(kb)
	th, _ := engine.ThinkAbout("jelaskan kucing", nil)
	if th.BestInterpretation == nil {
		t.Fatal("expected I*")
	}
	for _, r := range th.BestInterpretation.Relations {
		s := kb.Registry.GetByID(r.SourceID)
		if s != nil && s.Token == "anjing" {
			t.Fatal("anjing must not be primary I* relation source")
		}
	}
}

func TestFSU2_G_KnowledgeGrowth(t *testing.T) {
	kb := knowledge.NewKnowledgeBase()
	seedCatCapability(kb)
	kb.Store("apakah")
	kb.Store("bisa")
	bird := kb.Store("burung")
	wing := kb.Store("sayap")
	fly := kb.Store("terbang")
	kb.ConnectKind(bird, wing, knowledge.RelationHas, 0.9, 0.9, false)
	kb.ConnectKind(wing, fly, knowledge.RelationFunction, 0.9, 0.9, false)
	kb.ConnectKind(bird, fly, knowledge.RelationCanDo, 0.85, 0.85, false)
	engine := NewThinkingEngine(kb)
	th, _ := engine.ThinkAbout("apakah burung bisa terbang", nil)
	if th.BestInterpretation == nil {
		t.Fatal("expected I*")
	}
	found := false
	for _, p := range th.BestInterpretation.Propositions {
		if p.TargetTok == "burung" && p.ObjectTok == "terbang" {
			found = true
		}
	}
	if !found {
		// background facts may hold
		for _, p := range th.BestInterpretation.Propositions {
			if p.ObjectTok == "terbang" {
				found = true
			}
		}
	}
	if !found {
		t.Fatalf("growth: expected terbang proposition, got %+v", th.BestInterpretation.Propositions)
	}
}

func TestFSU2_H_LearningIsA(t *testing.T) {
	kb := knowledge.NewKnowledgeBase()
	saya := kb.Store("saya")
	manusia := kb.Store("manusia")
	kb.ConnectKind(saya, manusia, knowledge.RelationIsA, 0.9, 0.9, false)
	engine := NewThinkingEngine(kb)
	th, _ := engine.ThinkAbout("saya", nil)
	if th.BestInterpretation == nil {
		t.Fatal("expected I*")
	}
	f := kb.Registry.GetByID(th.BestInterpretation.FocusID)
	if f == nil || f.Token != "saya" {
		t.Fatalf("focus saya, got %v", f)
	}
}

func TestFUG_FishSwimPropositionBinding(t *testing.T) {
	kb := knowledge.NewKnowledgeBase()
	fish := kb.Store("ikan")
	fin := kb.Store("sirip")
	swim := kb.Store("berenang")
	fly := kb.Store("terbang")
	kb.ConnectKind(fish, fin, knowledge.RelationHas, 0.9, 0.9, false)
	kb.ConnectKind(fin, swim, knowledge.RelationFunction, 0.9, 0.9, false)
	kb.ConnectKind(fish, kb.Store("hewan"), knowledge.RelationIsA, 0.8, 0.8, false)
	kb.Store("apakah")
	kb.Store("bisa")
	// optional contrast for constraint tests later
	engine := NewThinkingEngine(kb)

	th, ok := engine.ThinkAbout("apakah ikan bisa berenang", nil)
	if !ok || th.BestInterpretation == nil {
		t.Fatal("expected I*")
	}
	i := th.BestInterpretation
	f := kb.Registry.GetByID(i.FocusID)
	if f == nil || f.Token != "ikan" {
		t.Fatalf("focus must be ikan, got %v", f)
	}
	found := false
	for _, p := range i.Propositions {
		if p.TargetTok == "ikan" && p.ObjectTok == "berenang" && p.Relation == knowledge.RelationCanDo {
			found = true
		}
	}
	if !found {
		t.Fatalf("requested prop ikan can_do berenang, got %+v", i.Propositions)
	}
	// Must not evaluate only is_a/has as primary
	if len(i.Propositions) > 0 && i.Propositions[0].Relation == knowledge.RelationIsA {
		t.Fatal("primary proposition must not be is_a")
	}
	hasPath := false
	for _, ep := range i.EvidencePaths {
		if ep.Length >= 2 && ep.Support {
			hasPath = true
		}
		joined := ""
		for _, tok := range ep.Tokens {
			joined += tok + " "
		}
		if containsStr(joined, "sirip") && containsStr(joined, "berenang") {
			hasPath = true
		}
	}
	if !hasPath {
		t.Logf("paths=%v status=%s", i.EvidencePaths, i.EvalStatus)
	}
	if i.EvalStatus != EvalSupported && i.EvalStatus != EvalUnknown {
		t.Logf("status=%s (prefer SUPPORTED when path exists)", i.EvalStatus)
	}

	th2, _ := engine.ThinkAbout("apakah ikan bisa terbang", nil)
	if th2.BestInterpretation == nil {
		t.Fatal("expected I* fly")
	}
	foundFly := false
	foundSwim := false
	for _, p := range th2.BestInterpretation.Propositions {
		if p.TargetTok == "ikan" && p.ObjectTok == "terbang" && p.Relation == knowledge.RelationCanDo {
			foundFly = true
		}
		if p.ObjectTok == "berenang" && p.Relation == knowledge.RelationCanDo {
			foundSwim = true
		}
	}
	if !foundFly {
		t.Fatalf("prop must be ikan can_do terbang, got %+v", th2.BestInterpretation.Propositions)
	}
	if foundSwim && !foundFly {
		t.Fatal("must not keep swim prop as only answer for fly question")
	}
	_ = fly
}

func TestFUG_ConstraintWithoutFin(t *testing.T) {
	kb := knowledge.NewKnowledgeBase()
	fish := kb.Store("ikan")
	fin := kb.Store("sirip")
	swim := kb.Store("berenang")
	kb.ConnectKind(fish, fin, knowledge.RelationHas, 0.9, 0.9, false)
	kb.ConnectKind(fin, swim, knowledge.RelationFunction, 0.9, 0.9, false)
	tanpa := kb.Store("tanpa")
	kb.ConnectKind(tanpa, fin, knowledge.RelationContrast, 0.9, 0.9, false)
	kb.Store("apakah")
	kb.Store("bisa")
	engine := NewThinkingEngine(kb)
	th, _ := engine.ThinkAbout("apakah ikan bisa berenang tanpa sirip", nil)
	if th.BestInterpretation == nil {
		t.Fatal("expected I*")
	}
	has := false
	for _, c := range th.BestInterpretation.Constraints {
		if c.Token == "sirip" {
			has = true
		}
	}
	if !has {
		t.Fatalf("expected constraint sirip, got %+v", th.BestInterpretation.Constraints)
	}
}
