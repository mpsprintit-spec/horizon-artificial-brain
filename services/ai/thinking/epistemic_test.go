package thinking

import (
	"testing"

	"github.com/project-horizon/horizon-core/services/ai/knowledge"
)

func seedQueryTokens(kb *knowledge.KnowledgeBase) {
	kb.Store("apakah")
	kb.Store("bisa")
}

func TestEpistemic_A_WalkNotSupportFly(t *testing.T) {
	kb := knowledge.NewKnowledgeBase()
	cat := kb.Store("kucing")
	walk := kb.Store("berjalan")
	kb.Store("terbang")
	kb.ConnectKind(cat, walk, knowledge.RelationCanDo, 0.9, 0.9, false)
	seedQueryTokens(kb)
	eng := NewThinkingEngine(kb)
	th, _ := eng.ThinkAbout("apakah kucing bisa terbang", nil)
	if th.BestInterpretation == nil {
		t.Fatal("I*")
	}
	i := th.BestInterpretation
	if i.RequestedProposition.ObjectTok != "terbang" {
		t.Fatalf("requested object terbang, got %+v", i.RequestedProposition)
	}
	if i.EvalStatus == EvalSupported {
		t.Fatal("WALK must not support FLY")
	}
	if i.EvalStatus != EvalUnknown {
		t.Fatalf("expected UNKNOWN, got %s", i.EvalStatus)
	}
}

func TestEpistemic_B_HasLegsNotSupportFly(t *testing.T) {
	kb := knowledge.NewKnowledgeBase()
	cat := kb.Store("kucing")
	leg := kb.Store("kaki")
	kb.Store("terbang")
	kb.ConnectKind(cat, leg, knowledge.RelationHas, 0.9, 0.9, false)
	seedQueryTokens(kb)
	eng := NewThinkingEngine(kb)
	th, _ := eng.ThinkAbout("apakah kucing bisa terbang", nil)
	if th.BestInterpretation == nil {
		t.Fatal("I*")
	}
	if th.BestInterpretation.EvalStatus == EvalSupported {
		t.Fatal("HAS legs must not support FLY")
	}
	if th.BestInterpretation.EvalStatus != EvalUnknown {
		t.Fatalf("expected UNKNOWN got %s", th.BestInterpretation.EvalStatus)
	}
}

func TestEpistemic_C_EagleWingsNotSupportCatFly(t *testing.T) {
	kb := knowledge.NewKnowledgeBase()
	kb.Store("kucing")
	eagle := kb.Store("elang")
	wing := kb.Store("sayap")
	fly := kb.Store("terbang")
	kb.ConnectKind(eagle, wing, knowledge.RelationHas, 0.9, 0.9, false)
	kb.ConnectKind(wing, fly, knowledge.RelationFunction, 0.9, 0.9, false)
	seedQueryTokens(kb)
	eng := NewThinkingEngine(kb)
	th, _ := eng.ThinkAbout("apakah kucing bisa terbang", nil)
	if th.BestInterpretation == nil {
		t.Fatal("I*")
	}
	if th.BestInterpretation.EvalStatus == EvalSupported {
		t.Fatal("eagle wings must not support cat fly")
	}
}

func TestEpistemic_D_DirectSupport(t *testing.T) {
	kb := knowledge.NewKnowledgeBase()
	cat := kb.Store("kucing")
	fly := kb.Store("terbang")
	kb.ConnectKind(cat, fly, knowledge.RelationCanDo, 0.9, 0.9, false)
	seedQueryTokens(kb)
	eng := NewThinkingEngine(kb)
	th, _ := eng.ThinkAbout("apakah kucing bisa terbang", nil)
	if th.BestInterpretation == nil {
		t.Fatal("I*")
	}
	i := th.BestInterpretation
	if i.EvalStatus != EvalSupported {
		t.Fatalf("expected SUPPORTED got %s evals=%+v", i.EvalStatus, i.EvidenceEvals)
	}
	if i.RequestedProposition.TargetTok != "kucing" || i.RequestedProposition.ObjectTok != "terbang" {
		t.Fatalf("identity %+v", i.RequestedProposition)
	}
}

func TestEpistemic_E_InhibitoryContradicted(t *testing.T) {
	kb := knowledge.NewKnowledgeBase()
	cat := kb.Store("kucing")
	fly := kb.Store("terbang")
	kb.ConnectKind(cat, fly, knowledge.RelationCanDo, 0.9, 0.9, true)
	seedQueryTokens(kb)
	eng := NewThinkingEngine(kb)
	th, _ := eng.ThinkAbout("apakah kucing bisa terbang", nil)
	if th.BestInterpretation == nil {
		t.Fatal("I*")
	}
	st := th.BestInterpretation.EvalStatus
	if st != EvalContradicted && st != EvalConflicted {
		t.Fatalf("expected CONTRADICTED got %s", st)
	}
}

func TestEpistemic_G_WalkWithoutLegsConflicted(t *testing.T) {
	kb := knowledge.NewKnowledgeBase()
	cat := kb.Store("kucing")
	leg := kb.Store("kaki")
	walk := kb.Store("berjalan")
	kb.ConnectKind(cat, walk, knowledge.RelationCanDo, 0.9, 0.9, false)
	kb.ConnectKind(cat, leg, knowledge.RelationHas, 0.9, 0.9, false)
	kb.ConnectKind(leg, walk, knowledge.RelationFunction, 0.9, 0.9, false)
	tanpa := kb.Store("tanpa")
	kb.ConnectKind(tanpa, leg, knowledge.RelationContrast, 0.9, 0.9, false)
	seedQueryTokens(kb)
	eng := NewThinkingEngine(kb)
	th, _ := eng.ThinkAbout("apakah kucing bisa berjalan tanpa kaki", nil)
	if th.BestInterpretation == nil {
		t.Fatal("I*")
	}
	st := th.BestInterpretation.EvalStatus
	if st != EvalConflicted && st != EvalContradicted {
		t.Fatalf("expected CONFLICTED got %s constr=%+v", st, th.BestInterpretation.Constraints)
	}
	if th.BestInterpretation.RequestedProposition.ObjectTok != "berjalan" {
		t.Fatalf("requested walk, got %+v", th.BestInterpretation.RequestedProposition)
	}
}

func TestEpistemic_IdentityStable(t *testing.T) {
	kb := knowledge.NewKnowledgeBase()
	cat := kb.Store("kucing")
	walk := kb.Store("berjalan")
	kb.Store("terbang")
	kb.ConnectKind(cat, walk, knowledge.RelationCanDo, 0.9, 0.9, false)
	seedQueryTokens(kb)
	eng := NewThinkingEngine(kb)
	th, _ := eng.ThinkAbout("apakah kucing bisa terbang", nil)
	i := th.BestInterpretation
	if i == nil {
		t.Fatal("I*")
	}
	if i.RequestedProposition.ObjectTok != "terbang" {
		t.Fatalf("requested terbang got %+v", i.RequestedProposition)
	}
	if i.EvalStatus == EvalSupported {
		t.Fatal("must not support fly from walk")
	}
	// IRRELEVANT walk evidence must not count as support
	for _, ev := range i.EvidenceEvals {
		if ev.Judgement == EvidenceSupport && ev.Path.Tokens != nil {
			for _, tok := range ev.Path.Tokens {
				if tok == "berjalan" {
					t.Fatal("walk path must not SUPPORT fly proposition")
				}
			}
		}
	}
}
