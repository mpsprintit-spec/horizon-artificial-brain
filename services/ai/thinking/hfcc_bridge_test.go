package thinking

import (
	"testing"

	"github.com/project-horizon/horizon-core/services/ai/hfcc"
	"github.com/project-horizon/horizon-core/services/ai/knowledge"
)

func TestHFCCBridgePreservesAllCandidates(t *testing.T) {
	kb := knowledge.NewKnowledgeBase()
	cat := kb.Store("kucing")
	leg := kb.Store("kaki")
	kb.ConnectKind(cat, leg, knowledge.RelationHas, 0.9, 0.9, false)
	kb.ConnectKind(cat, leg, knowledge.RelationCause, 0.5, 0.5, false) // multi-relation
	eng := NewThinkingEngine(kb)
	th, ok := eng.ThinkAbout("kucing punya kaki", nil)
	if !ok {
		t.Fatal("think failed")
	}
	if th.HFCCCandidateSet == nil {
		t.Fatal("HFCC CandidateSet not attached")
	}
	if len(th.Interpretations) > 0 && len(th.HFCCCandidateSet.Candidates) == 0 {
		t.Fatal("interpretations not mapped to HFCC candidates")
	}
	// Candidates are hypotheses
	for _, c := range th.HFCCCandidateSet.Candidates {
		if c.Status == hfcc.StatusGold {
			t.Fatal("must not auto-promote GOLD")
		}
		if c.Status != hfcc.StatusHypothesis && c.Status != hfcc.StatusRaw && c.Status != hfcc.StatusAnnotated {
			t.Fatalf("unexpected status %s", c.Status)
		}
	}
	// Multi-relation channels in knowledge survive
	if cat.FindSynapse(leg.ID, knowledge.RelationHas, false) == nil || cat.FindSynapse(leg.ID, knowledge.RelationCause, false) == nil {
		t.Fatal("multi-relation storage lost")
	}
}

func TestNoPositionalRoleInBridge(t *testing.T) {
	kb := knowledge.NewKnowledgeBase()
	kb.Store("kucing")
	kb.Store("anjing")
	eng := NewThinkingEngine(kb)
	th, _ := eng.ThinkAbout("kucing melihat anjing", nil)
	if th.HFCCCandidateSet == nil {
		return
	}
	for _, c := range th.HFCCCandidateSet.Candidates {
		var walk func(hfcc.StructureNode)
		walk = func(n hfcc.StructureNode) {
			for _, p := range n.Participants {
				if p.Participation != nil {
					t.Fatalf("bridge invented participation %s", *p.Participation)
				}
			}
			for _, ch := range n.Children {
				walk(ch)
			}
		}
		walk(c.Structure)
	}
}
