package learning

import (
	"testing"
	"time"

	"github.com/project-horizon/horizon-core/services/ai/knowledge"
)

func TestAssimilateLabelsCausalRelations(t *testing.T) {
	kb := knowledge.NewKnowledgeBase()
	NewLearningUnit(kb).Assimilate("api menyebabkan panas", 0.9, 0.8)
	api := kb.Fetch("api")
	panas := kb.Fetch("panas")
	if api.FindSynapse(panas.ID, knowledge.RelationCause, false).Kind != knowledge.RelationCause {
		t.Fatalf("expected cause relation, got %s", api.FindSynapse(panas.ID, knowledge.RelationCause, false).Kind)
	}
}

func TestOptimizeWeakensStaleSynapse(t *testing.T) {
	kb := knowledge.NewKnowledgeBase()
	api := kb.Store("api")
	panas := kb.Store("panas")
	kb.Connect(api, panas, 1, 1, false)
	api.FindSynapse(panas.ID, knowledge.RelationAssociation, false).LastActivation = time.Now().Add(-60 * 24 * time.Hour)
	NewLearningUnit(kb).Optimize(time.Now())
	if api.FindSynapse(panas.ID, knowledge.RelationAssociation, false).Weight >= 1 {
		t.Fatal("expected stale synapse weight to decay")
	}
}

func TestConfirmTargetsExactRelationChannel(t *testing.T) {
	kb := knowledge.NewKnowledgeBase()
	cat := kb.Store("kucing")
	leg := kb.Store("kaki")
	kb.ConnectKind(cat, leg, knowledge.RelationHas, 0.5, 0.5, false)
	kb.ConnectKind(cat, leg, knowledge.RelationCause, 0.5, 0.5, false)
	lu := NewLearningUnit(kb)
	lu.LastTouched = []TouchedRelation{{
		SourceID: cat.ID, TargetID: leg.ID, Kind: knowledge.RelationHas, Inhibitory: false,
	}}
	beforeCause := cat.FindSynapse(leg.ID, knowledge.RelationCause, false).Confidence
	lu.Confirm()
	has := cat.FindSynapse(leg.ID, knowledge.RelationHas, false)
	cause := cat.FindSynapse(leg.ID, knowledge.RelationCause, false)
	if has.Confidence <= 0.5 {
		t.Fatal("HAS should be confirmed")
	}
	if cause.Confidence != beforeCause {
		t.Fatal("CAUSE must not change when confirming HAS channel")
	}
}
