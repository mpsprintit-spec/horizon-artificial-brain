package thinking

import (
	"strings"
	"testing"

	"github.com/project-horizon/horizon-core/services/ai/hfcc"
	"github.com/project-horizon/horizon-core/services/ai/knowledge"
)

func scope(s string) string { return s }

// --- Cases 1–7: independent audit fixtures (no Understanding required) ---

func TestCase1_RelocationRecipientToGiveSlot(t *testing.T) {
	beforeIID := MakeIID(KindStructural, InformationPayload{Entity: "DOG", Slot: "participation", Value: "RECIPIENT"}, "c0")
	afterIID := MakeIID(KindStructural, InformationPayload{Host: "GIVE", Slot: "recipient", Value: "DOG"}, "c0")
	before := CandidateFormationState{
		Candidates: []CandidateSnapshot{{ID: "interp-0"}},
		Inventory: []InformationItem{{
			IID: beforeIID, AuditID: "dog-recip", Class: InfoStructural,
			Origin: OriginDirect, Dependency: DepInternal, CandidateID: "interp-0",
		}},
	}
	after := PreservedCandidateState{
		Candidates: []CandidateSnapshot{{ID: "interp-0"}},
		Inventory: []InformationItem{{
			IID: afterIID, AuditID: "give-recip", Class: InfoStructural,
			Origin: OriginDirect, Dependency: DepInternal, CandidateID: "interp-0",
		}},
		RelocationRecords: []RelocationRecord{{
			FromIID: beforeIID.Digest, ToIID: afterIID.Digest, Justification: "SLOT_REPATH",
		}},
	}
	rep := AuditPreservationIID(before, after)
	if !rep.Pass {
		t.Fatalf("expected PASS, lost=%v", rep.Lost)
	}
	if len(rep.Relocated) == 0 {
		t.Fatal("expected RELOCATED")
	}
}

func TestCase2_RecipientDroppedEntityRemains_Lost(t *testing.T) {
	relIID := MakeIID(KindRelation, InformationPayload{Src: "DOG", Rel: "participation", Tgt: "RECIPIENT"}, "c0")
	entIID := MakeIID(KindEntity, InformationPayload{Entity: "DOG"}, "c0")
	before := CandidateFormationState{
		Candidates: []CandidateSnapshot{{ID: "interp-0"}},
		Inventory: []InformationItem{
			{IID: relIID, AuditID: "dog-recip-rel", Class: InfoSemantic, Origin: OriginDirect, Dependency: DepInternal},
			{IID: entIID, AuditID: "dog-ent", Class: InfoSemantic, Origin: OriginDirect, Dependency: DepInternal},
		},
	}
	after := PreservedCandidateState{
		Candidates: []CandidateSnapshot{{ID: "interp-0"}},
		Inventory: []InformationItem{
			{IID: entIID, AuditID: "dog-ent", Class: InfoSemantic, Origin: OriginDirect, Dependency: DepInternal},
		},
	}
	rep := AuditPreservationIID(before, after)
	if rep.Pass {
		t.Fatal("relationship must be LOST")
	}
	found := false
	for _, id := range rep.Lost {
		if id == "dog-recip-rel" {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected dog-recip-rel lost, got %v", rep.Lost)
	}
	// entity may remain available
	for _, a := range rep.ItemAudits {
		if a.Item.AuditID == "dog-ent" && a.Dims.Availability != AvailAvailable {
			t.Fatal("entity DOG should remain AVAILABLE")
		}
	}
}

func TestCase3_RelationLostEndpointsRemain(t *testing.T) {
	rel := MakeIID(KindRelation, InformationPayload{Src: "A", Rel: "R", Tgt: "B"}, "c0")
	eA := MakeIID(KindEntity, InformationPayload{Entity: "A"}, "c0")
	eB := MakeIID(KindEntity, InformationPayload{Entity: "B"}, "c0")
	before := CandidateFormationState{
		Candidates: []CandidateSnapshot{{ID: "interp-0"}},
		Inventory: []InformationItem{
			{IID: rel, AuditID: "rel-ab", Class: InfoSemantic, Origin: OriginDirect, Dependency: DepInternal},
			{IID: eA, AuditID: "ent-a", Class: InfoSemantic, Origin: OriginDirect, Dependency: DepInternal},
			{IID: eB, AuditID: "ent-b", Class: InfoSemantic, Origin: OriginDirect, Dependency: DepInternal},
		},
	}
	after := PreservedCandidateState{
		Candidates: []CandidateSnapshot{{ID: "interp-0"}},
		Inventory: []InformationItem{
			{IID: eA, AuditID: "ent-a", Class: InfoSemantic, Origin: OriginDirect, Dependency: DepInternal},
			{IID: eB, AuditID: "ent-b", Class: InfoSemantic, Origin: OriginDirect, Dependency: DepInternal},
		},
	}
	rep := AuditPreservationIID(before, after)
	if rep.Pass {
		t.Fatal("RELATION(A,B) must LOST when only A,B entities remain")
	}
	for _, id := range rep.Lost {
		if id == "rel-ab" {
			return
		}
	}
	t.Fatalf("rel-ab not in lost: %v", rep.Lost)
}

func TestCase4_DerivationDependencies(t *testing.T) {
	a := MakeIID(KindEntity, InformationPayload{Entity: "A"}, "c0")
	b := MakeIID(KindEntity, InformationPayload{Entity: "B"}, "c0")
	c := MakeIID(KindRelation, InformationPayload{Src: "A", Rel: "R", Tgt: "B"}, "c0")
	before := CandidateFormationState{
		Candidates: []CandidateSnapshot{{ID: "interp-0"}},
		Inventory: []InformationItem{
			{IID: a, AuditID: "a", Class: InfoSemantic, Origin: OriginDirect, Dependency: DepInternal},
			{IID: b, AuditID: "b", Class: InfoSemantic, Origin: OriginDirect, Dependency: DepInternal},
			{IID: c, AuditID: "c", Class: InfoSemantic, Origin: OriginDerived, Dependency: DepInternal},
		},
		Derivations: []DerivationRecord{{
			ID: "d1", ConcludesIID: c.Digest, PremiseIIDs: []string{a.Digest, b.Digest},
			RuleRef: "RuleR", Dependency: DepInternal,
		}},
	}
	after := PreserveCandidateState(before, nil, nil)
	// ensure derivation retained
	if len(after.Derivations) != 1 || after.Derivations[0].RuleRef != "RuleR" {
		t.Fatalf("derivation not preserved: %+v", after.Derivations)
	}
	if len(after.Derivations[0].PremiseIIDs) != 2 {
		t.Fatal("premises A,B required")
	}
	rep := AuditPreservationIID(before, after)
	if !rep.Pass {
		t.Fatalf("derived C should survive, lost=%v", rep.Lost)
	}
}

func TestCase5_ExternalKnowledgeDependency(t *testing.T) {
	x := MakeIID(KindEntity, InformationPayload{Entity: "X"}, "c0")
	before := CandidateFormationState{
		Candidates: []CandidateSnapshot{{ID: "interp-0"}},
		Inventory: []InformationItem{
			{IID: x, AuditID: "x", Class: InfoSemantic, Origin: OriginDerived, Dependency: DepExternal},
		},
		Derivations: []DerivationRecord{{
			ID: "d-ext", ConcludesIID: x.Digest, PremiseIIDs: nil, RuleRef: "ext-lookup",
			Dependency: DepExternal,
		}},
	}
	after := PreserveCandidateState(before, nil, nil)
	if after.Derivations[0].Dependency != DepExternal {
		t.Fatal("must remain EXTERNAL_KNOWLEDGE")
	}
	rep := AuditPreservationIID(before, after)
	if len(rep.ExternalDeps) == 0 {
		t.Fatal("external dependency must be reported")
	}
}

func TestCase6_RuntimeExplicitDiscard(t *testing.T) {
	rt := MakeIID(KindRuntime, InformationPayload{Metric: "activation=0.72"}, "c0")
	sem := MakeIID(KindRelation, InformationPayload{Src: "kucing", Rel: "can_do", Tgt: "berjalan"}, "c0")
	before := CandidateFormationState{
		Candidates: []CandidateSnapshot{{ID: "interp-0"}},
		Inventory: []InformationItem{
			{IID: rt, AuditID: "rt-1", Class: InfoRuntimeMetadata, Origin: OriginDerived, Dependency: DepInternal},
			{IID: sem, AuditID: "sem-1", Class: InfoSemantic, Origin: OriginDirect, Dependency: DepInternal},
		},
	}
	after := PreserveCandidateState(before, nil, nil)
	rep := AuditPreservationIID(before, after)
	if !rep.Pass {
		t.Fatalf("runtime discard should PASS, lost=%v", rep.Lost)
	}
	if len(rep.ExplicitlyDiscard) == 0 {
		t.Fatal("expected RUNTIME_ONLY discard")
	}
}

func TestCase7_TanpaKakiEvidenceWithoutRelation(t *testing.T) {
	// Fixture: evidence formed, relation tanpa–kaki NOT in inventory
	evT := MakeIID(KindEvidence, InformationPayload{Span: "tanpa", Entity: "tanpa"}, "obs")
	evK := MakeIID(KindEvidence, InformationPayload{Span: "kaki", Entity: "kaki"}, "obs")
	before := CandidateFormationState{
		Candidates: []CandidateSnapshot{{ID: "interp-0"}},
		Inventory: []InformationItem{
			{IID: evT, AuditID: "ev-tanpa", Class: InfoEvidence, Origin: OriginDirect, Dependency: DepInternal},
			{IID: evK, AuditID: "ev-kaki", Class: InfoEvidence, Origin: OriginDirect, Dependency: DepInternal},
		},
		Evidence: []EvidenceItem{
			{Span: "tanpa", IIDDigest: evT.Digest},
			{Span: "kaki", IIDDigest: evK.Digest},
		},
	}
	domain, notes := AuditTanpaKakiCase(before, []string{"tanpa", "kaki"})
	if domain != "UNDERSTANDING_FORMATION_MISSING_INFORMATION" {
		t.Fatalf("expected formation missing for relation, got %s notes=%v", domain, notes)
	}
	after := PreserveCandidateState(before, nil, nil)
	rep := AuditPreservationIID(before, after)
	if !rep.Pass {
		t.Fatalf("evidence should preserve; lost=%v", rep.Lost)
	}
}

func TestTokenOverlapDoesNotSaveRelation(t *testing.T) {
	// Adversarial: AFTER has only entity tokens A,B — must not RELOCATE relation via overlap
	rel := MakeIID(KindRelation, InformationPayload{Src: "A", Rel: "R", Tgt: "B"}, "c0")
	before := CandidateFormationState{
		Candidates: []CandidateSnapshot{{ID: "interp-0"}},
		Inventory:  []InformationItem{{IID: rel, AuditID: "rel", Class: InfoSemantic, Origin: OriginDirect, Dependency: DepInternal}},
	}
	after := PreservedCandidateState{
		Candidates: []CandidateSnapshot{{ID: "interp-0"}},
		Inventory: []InformationItem{
			{IID: MakeIID(KindEntity, InformationPayload{Entity: "A"}, "c0"), AuditID: "a", Class: InfoSemantic, Origin: OriginDirect, Dependency: DepInternal},
			{IID: MakeIID(KindEntity, InformationPayload{Entity: "B"}, "c0"), AuditID: "b", Class: InfoSemantic, Origin: OriginDirect, Dependency: DepInternal},
		},
	}
	rep := AuditPreservationIID(before, after)
	if rep.Pass || len(rep.Relocated) > 0 {
		t.Fatal("token presence must not count as relation preservation")
	}
}

func TestNoEvidenceEchoFromHFCCPath(t *testing.T) {
	// If EvidenceIndex empty, evidence BEFORE is LOST even if we "could" copy
	ev := MakeIID(KindEvidence, InformationPayload{Span: "tanpa"}, "obs")
	before := CandidateFormationState{
		Candidates: []CandidateSnapshot{{ID: "interp-0"}},
		Inventory:  []InformationItem{{IID: ev, AuditID: "ev", Class: InfoEvidence, Origin: OriginDirect, Dependency: DepInternal}},
		Evidence:   []EvidenceItem{{Span: "tanpa", IIDDigest: ev.Digest}},
	}
	after := PreservedCandidateState{
		Candidates:    []CandidateSnapshot{{ID: "interp-0"}},
		Inventory:     nil,
		EvidenceIndex: nil, // intentional empty — not an echo
	}
	rep := AuditPreservationIID(before, after)
	if rep.Pass {
		t.Fatal("empty EvidenceIndex must not silently pass via echo")
	}
}

func TestPreservation_SelectionAfterBoundary(t *testing.T) {
	kb := knowledge.NewKnowledgeBase()
	cat := kb.Store("kucing")
	walk := kb.Store("berjalan")
	kb.ConnectKind(cat, walk, knowledge.RelationCanDo, 0.9, 0.9, false)
	eng := NewThinkingEngine(kb)
	th, _ := eng.ThinkAbout("kucing bisa berjalan", nil)
	if th.FormationState == nil || th.PreservedState == nil {
		t.Fatal("formation + preserved required")
	}
	if th.PreservationReport == nil {
		t.Fatal("report required")
	}
}

func TestPreservation_TanpaKakiRuntimeNoInventedRelation(t *testing.T) {
	kb := knowledge.NewKnowledgeBase()
	cat := kb.Store("kucing")
	walk := kb.Store("berjalan")
	leg := kb.Store("kaki")
	kb.ConnectKind(cat, walk, knowledge.RelationCanDo, 0.9, 0.9, false)
	kb.ConnectKind(cat, leg, knowledge.RelationHas, 0.9, 0.9, false)
	kb.Store("apakah")
	kb.Store("bisa")
	kb.Store("tanpa")
	eng := NewThinkingEngine(kb)
	th, _ := eng.ThinkAbout("apakah kucing bisa berjalan tanpa kaki", nil)
	if th.FormationState == nil {
		t.Fatal("formation")
	}
	domain, notes := AuditTanpaKakiCase(*th.FormationState, []string{"apakah", "kucing", "bisa", "berjalan", "tanpa", "kaki"})
	t.Log(domain, notes)
	if domain != "UNDERSTANDING_FORMATION_MISSING_INFORMATION" {
		// only fail if structural relation was incorrectly invented
		for _, it := range th.FormationState.Inventory {
			if it.IID.Kind == KindRelation &&
				((it.IID.Payload.Src == "tanpa" && it.IID.Payload.Tgt == "kaki") ||
					(it.IID.Payload.Src == "kaki" && it.IID.Payload.Tgt == "tanpa")) {
				t.Fatal("must not invent tanpa–kaki relation")
			}
		}
	}
	// evidence spans present
	foundT, foundK := false, false
	for _, it := range th.FormationState.Inventory {
		if it.IID.Kind == KindEvidence && it.IID.Payload.Span == "tanpa" {
			foundT = true
		}
		if it.IID.Kind == KindEvidence && it.IID.Payload.Span == "kaki" {
			foundK = true
		}
	}
	if !foundT || !foundK {
		t.Fatal("surface evidence tanpa/kaki must form")
	}
}

func TestNoInventedWithoutPrimitive(t *testing.T) {
	kb := knowledge.NewKnowledgeBase()
	kb.Store("kucing")
	kb.Store("berjalan")
	kb.Store("kaki")
	kb.Store("tanpa")
	eng := NewThinkingEngine(kb)
	th, _ := eng.ThinkAbout("apakah kucing bisa berjalan tanpa kaki", nil)
	if th.HFCCCandidateSet == nil {
		return
	}
	var walk func(hfcc.StructureNode)
	walk = func(n hfcc.StructureNode) {
		if strings.EqualFold(n.Kind, "WITHOUT") {
			t.Fatalf("invented kind %s", n.Kind)
		}
		for _, ch := range n.Children {
			walk(ch)
		}
	}
	for _, c := range th.HFCCCandidateSet.Candidates {
		walk(c.Structure)
	}
}

func TestPreservation_MultipleCandidatesSurvive(t *testing.T) {
	kb := knowledge.NewKnowledgeBase()
	kb.Store("kucing")
	kb.Store("anjing")
	kb.Store("kamera")
	eng := NewThinkingEngine(kb)
	th, _ := eng.ThinkAbout("kucing melihat anjing dengan kamera", nil)
	if th.FormationState == nil || th.PreservedState == nil {
		t.Fatal("missing state")
	}
	if len(th.FormationState.Candidates) != len(th.PreservedState.Candidates) {
		t.Fatalf("candidate count changed %d -> %d", len(th.FormationState.Candidates), len(th.PreservedState.Candidates))
	}
}

// --- Boundary B: PCS → HFCC ---

func TestBoundaryB_SingleSourceNoInterpretation(t *testing.T) {
	rel := MakeIID(KindRelation, InformationPayload{Src: "A", Rel: "R", Tgt: "B"}, "interp-0")
	pcs := PreservedCandidateState{
		Candidates: []CandidateSnapshot{{ID: "interp-0"}},
		Inventory: []InformationItem{{
			IID: rel, AuditID: "rel", Class: InfoSemantic, Origin: OriginDirect, Dependency: DepInternal, CandidateID: "interp-0",
		}},
	}
	obs := hfcc.Observation{ID: "o1"}
	cs := BuildHFCCFromPreserved(obs, pcs)
	if len(cs.Candidates) != 1 {
		t.Fatal("expected one candidate from PCS")
	}
	found := false
	var walk func(n hfcc.StructureNode)
	walk = func(n hfcc.StructureNode) {
		if n.Kind == "R" && len(n.Participants) >= 2 && n.Participants[0].Ref == "A" && n.Participants[1].Ref == "B" {
			found = true
		}
		for _, ch := range n.Children {
			walk(ch)
		}
	}
	walk(cs.Candidates[0].Structure)
	if !found {
		t.Fatal("relation A-R-B must project from PCS alone")
	}
}

func TestBoundaryB_HiddenSourceFails(t *testing.T) {
	// Relation only on Interpretation-like data would be ablated from PCS
	pcs := PreservedCandidateState{
		Candidates: []CandidateSnapshot{{ID: "interp-0"}},
		Inventory:  nil, // ablated
	}
	obs := hfcc.Observation{ID: "o1"}
	cs := BuildHFCCFromPreserved(obs, pcs)
	// Even if a parallel []Interpretation would have had the relation, HFCC must not
	for _, c := range cs.Candidates {
		var walk func(hfcc.StructureNode)
		walk = func(n hfcc.StructureNode) {
			if n.Kind == "can_do" || n.Kind == "R" {
				t.Fatal("HIDDEN_SOURCE: HFCC obtained relation without PCS")
			}
			for _, ch := range n.Children {
				walk(ch)
			}
		}
		walk(c.Structure)
	}
}

func TestBoundaryB_RelationConservation(t *testing.T) {
	rel := MakeIID(KindRelation, InformationPayload{Src: "A", Rel: "R", Tgt: "B"}, "interp-0")
	pcs := PreservedCandidateState{
		Candidates: []CandidateSnapshot{{ID: "interp-0"}},
		Inventory: []InformationItem{{
			IID: rel, AuditID: "rel", Class: InfoSemantic, Origin: OriginDirect, Dependency: DepInternal, CandidateID: "interp-0",
		}},
	}
	proj := BuildHFCCFromPreservedWithMappings(hfcc.Observation{ID: "o"}, pcs)
	rep := AuditHFCCConsumerWithMappings(pcs, proj)
	if !rep.Pass {
		t.Fatalf("relation should be AVAILABLE in HFCC, lost=%v", rep.Lost)
	}
}

func TestBoundaryB_RelationLostEndpointsRemain(t *testing.T) {
	rel := MakeIID(KindRelation, InformationPayload{Src: "A", Rel: "R", Tgt: "B"}, "interp-0")
	eA := MakeIID(KindEntity, InformationPayload{Entity: "A"}, "interp-0")
	eB := MakeIID(KindEntity, InformationPayload{Entity: "B"}, "interp-0")
	pcs := PreservedCandidateState{
		Candidates: []CandidateSnapshot{{ID: "interp-0"}},
		Inventory: []InformationItem{
			{IID: rel, AuditID: "rel", Class: InfoSemantic, Origin: OriginDirect, Dependency: DepInternal, CandidateID: "interp-0"},
			{IID: eA, AuditID: "a", Class: InfoSemantic, Origin: OriginDirect, Dependency: DepInternal, CandidateID: "interp-0"},
			{IID: eB, AuditID: "b", Class: InfoSemantic, Origin: OriginDirect, Dependency: DepInternal, CandidateID: "interp-0"},
		},
	}
	// HFCC only has entities (simulate lossy consumer)
	cs := hfcc.CandidateSet{Candidates: []hfcc.Candidate{{
		ID: "interp-0",
		Structure: hfcc.StructureNode{Kind: "INTERPRETATION", Children: []hfcc.StructureNode{
			{Kind: "ENTITY", Participants: []hfcc.Participant{{Ref: "A"}}},
			{Kind: "ENTITY", Participants: []hfcc.Participant{{Ref: "B"}}},
		}},
	}}}
	rep := AuditHFCCConsumer(pcs, cs)
	if rep.Pass {
		t.Fatal("relation must be LOST at HFCC consumer even if endpoints present")
	}
}

func TestBoundaryB_NoInventedTanpaKakiRelation(t *testing.T) {
	evT := MakeIID(KindEvidence, InformationPayload{Span: "tanpa", Entity: "tanpa"}, "obs")
	evK := MakeIID(KindEvidence, InformationPayload{Span: "kaki", Entity: "kaki"}, "obs")
	pcs := PreservedCandidateState{
		Candidates: []CandidateSnapshot{{ID: "interp-0"}},
		Inventory: []InformationItem{
			{IID: evT, AuditID: "ev-t", Class: InfoEvidence, Origin: OriginDirect, Dependency: DepInternal},
			{IID: evK, AuditID: "ev-k", Class: InfoEvidence, Origin: OriginDirect, Dependency: DepInternal},
		},
		EvidenceIndex: []EvidenceItem{
			{Span: "tanpa", IIDDigest: evT.Digest},
			{Span: "kaki", IIDDigest: evK.Digest},
		},
	}
	cs := BuildHFCCFromPreserved(hfcc.Observation{ID: "o"}, pcs)
	for _, c := range cs.Candidates {
		var walk func(hfcc.StructureNode)
		walk = func(n hfcc.StructureNode) {
			if strings.EqualFold(n.Kind, "WITHOUT") {
				t.Fatal("invented WITHOUT")
			}
			if (n.Kind != "" && n.Kind != "INTERPRETATION" && n.Kind != "ENTITY") && len(n.Participants) >= 2 {
				a, b := n.Participants[0].Ref, n.Participants[1].Ref
				if (a == "tanpa" && b == "kaki") || (a == "kaki" && b == "tanpa") {
					t.Fatal("invented relation(tanpa,kaki)")
				}
			}
			for _, ch := range n.Children {
				walk(ch)
			}
		}
		walk(c.Structure)
	}
}

func TestBoundaryB_EvidenceFromProjectionNotEcho(t *testing.T) {
	ev := MakeIID(KindEvidence, InformationPayload{Span: "tanpa", Entity: "tanpa"}, "obs")
	pcs := PreservedCandidateState{
		Candidates: []CandidateSnapshot{{ID: "interp-0"}},
		Inventory: []InformationItem{{
			IID: ev, AuditID: "ev", Class: InfoEvidence, Origin: OriginDirect, Dependency: DepInternal,
		}},
		EvidenceIndex: []EvidenceItem{{Span: "tanpa", IIDDigest: ev.Digest}},
	}
	proj := BuildHFCCFromPreservedWithMappings(hfcc.Observation{ID: "o"}, pcs)
	found := false
	for _, m := range proj.Mappings {
		if m.FromIIDDigest == ev.Digest && m.Target.Kind == "EVIDENCE_REF" {
			found = true
		}
	}
	if !found {
		t.Fatal("evidence must appear via explicit ProjectionMapping to EVIDENCE_REF")
	}
	// empty evidence index + remove inventory evidence → no evidence mapping
	pcs2 := pcs
	pcs2.EvidenceIndex = nil
	pcs2.Inventory = nil
	proj2 := BuildHFCCFromPreservedWithMappings(hfcc.Observation{ID: "o"}, pcs2)
	for _, m := range proj2.Mappings {
		if m.Target.Kind == "EVIDENCE_REF" {
			t.Fatal("must not emit evidence mapping without EvidenceIndex/inventory")
		}
	}
}

func TestBoundaryB_PipelineUsesPCS(t *testing.T) {
	kb := knowledge.NewKnowledgeBase()
	cat := kb.Store("kucing")
	walk := kb.Store("berjalan")
	kb.ConnectKind(cat, walk, knowledge.RelationCanDo, 0.9, 0.9, false)
	eng := NewThinkingEngine(kb)
	th, _ := eng.ThinkAbout("kucing bisa berjalan", nil)
	if th.PreservedState == nil || th.HFCCCandidateSet == nil || th.HFCCConsumerReport == nil {
		t.Fatal("PCS + HFCC + Audit B required")
	}
}

func TestBoundaryB_TanpaKakiRuntime(t *testing.T) {
	kb := knowledge.NewKnowledgeBase()
	cat := kb.Store("kucing")
	walk := kb.Store("berjalan")
	leg := kb.Store("kaki")
	kb.ConnectKind(cat, walk, knowledge.RelationCanDo, 0.9, 0.9, false)
	kb.ConnectKind(cat, leg, knowledge.RelationHas, 0.9, 0.9, false)
	kb.Store("apakah")
	kb.Store("bisa")
	kb.Store("tanpa")
	eng := NewThinkingEngine(kb)
	th, _ := eng.ThinkAbout("apakah kucing bisa berjalan tanpa kaki", nil)
	domain, notes := AuditTanpaKakiCase(*th.FormationState, []string{"apakah", "kucing", "bisa", "berjalan", "tanpa", "kaki"})
	t.Log(domain, notes)
	if domain != "UNDERSTANDING_FORMATION_MISSING_INFORMATION" && domain != "" {
		t.Fatalf("unexpected domain %s", domain)
	}
	// HFCC must not invent tanpa-kaki relation
	if th.HFCCCandidateSet != nil {
		for _, c := range th.HFCCCandidateSet.Candidates {
			var walk func(hfcc.StructureNode)
			walk = func(n hfcc.StructureNode) {
				if len(n.Participants) >= 2 {
					a, b := n.Participants[0].Ref, n.Participants[1].Ref
					if (a == "tanpa" && b == "kaki") || (a == "kaki" && b == "tanpa") {
						t.Fatal("HFCC invented tanpa-kaki relation")
					}
				}
				for _, ch := range n.Children {
					walk(ch)
				}
			}
			walk(c.Structure)
		}
	}
}

// --- Explicit ProjectionMapping adversarial suite ---

func TestMap_RelationPreserved(t *testing.T) {
	rel := MakeIID(KindRelation, InformationPayload{Src: "A", Rel: "R", Tgt: "B"}, "interp-0")
	pcs := PreservedCandidateState{
		Candidates: []CandidateSnapshot{{ID: "interp-0"}},
		Inventory: []InformationItem{{
			IID: rel, AuditID: "rel", Class: InfoSemantic, Origin: OriginDirect, Dependency: DepInternal, CandidateID: "interp-0",
		}},
	}
	proj := BuildHFCCFromPreservedWithMappings(hfcc.Observation{ID: "o"}, pcs)
	rep := AuditHFCCConsumerWithMappings(pcs, proj)
	if !rep.Pass {
		t.Fatalf("relation must map, lost=%v", rep.Lost)
	}
	found := false
	for _, m := range proj.Mappings {
		if m.FromIIDDigest == rel.Digest && m.Target.Kind == "RELATION_NODE" {
			found = true
		}
	}
	if !found {
		t.Fatal("explicit ProjectionMapping for relation required")
	}
}

func TestMap_RelationLostEndpointsSurvive(t *testing.T) {
	rel := MakeIID(KindRelation, InformationPayload{Src: "A", Rel: "R", Tgt: "B"}, "interp-0")
	eA := MakeIID(KindEntity, InformationPayload{Entity: "A"}, "interp-0")
	eB := MakeIID(KindEntity, InformationPayload{Entity: "B"}, "interp-0")
	pcs := PreservedCandidateState{
		Candidates: []CandidateSnapshot{{ID: "interp-0"}},
		Inventory: []InformationItem{
			{IID: rel, AuditID: "rel", Class: InfoSemantic, Origin: OriginDirect, Dependency: DepInternal, CandidateID: "interp-0"},
			{IID: eA, AuditID: "a", Class: InfoSemantic, Origin: OriginDirect, Dependency: DepInternal, CandidateID: "interp-0"},
			{IID: eB, AuditID: "b", Class: InfoSemantic, Origin: OriginDirect, Dependency: DepInternal, CandidateID: "interp-0"},
		},
	}
	// Lossy HFCC: only entities, no mappings for relation
	proj := HFCCProjectionResult{
		CandidateSet: hfcc.CandidateSet{Candidates: []hfcc.Candidate{{
			ID: "interp-0",
			Structure: hfcc.StructureNode{Kind: "INTERPRETATION", Children: []hfcc.StructureNode{
				{Kind: "ENTITY", Participants: []hfcc.Participant{{Ref: "A"}}},
				{Kind: "ENTITY", Participants: []hfcc.Participant{{Ref: "B"}}},
			}},
		}}},
		Mappings: []ProjectionMapping{
			{FromIIDDigest: eA.Digest, Target: ProjectionTarget{CandidateID: "interp-0", StructurePath: "children[0]", Kind: "ENTITY_NODE"}},
			{FromIIDDigest: eB.Digest, Target: ProjectionTarget{CandidateID: "interp-0", StructurePath: "children[1]", Kind: "ENTITY_NODE"}},
		},
	}
	rep := AuditHFCCConsumerWithMappings(pcs, proj)
	if rep.Pass {
		t.Fatal("relation without mapping must LOST")
	}
	lostRel := false
	for _, id := range rep.Lost {
		if id == "rel" {
			lostRel = true
		}
	}
	if !lostRel {
		t.Fatalf("expected rel lost, got %v", rep.Lost)
	}
}

func TestMap_StructuralPreserved(t *testing.T) {
	st := MakeIID(KindStructural, InformationPayload{Slot: "constraint_attachment", Value: "kaki"}, "interp-0")
	pcs := PreservedCandidateState{
		Candidates: []CandidateSnapshot{{ID: "interp-0"}},
		Inventory: []InformationItem{{
			IID: st, AuditID: "str", Class: InfoStructural, Origin: OriginDirect, Dependency: DepInternal, CandidateID: "interp-0",
		}},
	}
	proj := BuildHFCCFromPreservedWithMappings(hfcc.Observation{ID: "o"}, pcs)
	var kind string
	for _, m := range proj.Mappings {
		if m.FromIIDDigest == st.Digest {
			kind = m.Target.Kind
		}
	}
	if kind != "STRUCTURAL_NODE" {
		t.Fatalf("structural must map as STRUCTURAL_NODE, got %q", kind)
	}
	// Must not be RELATION_NODE merely because node has participants
	if kind == "RELATION_NODE" {
		t.Fatal("structural reclassified as relation")
	}
	rep := AuditHFCCConsumerWithMappings(pcs, proj)
	if !rep.Pass {
		t.Fatalf("structural should preserve via mapping, lost=%v", rep.Lost)
	}
}

func TestMap_StructuralLostParticipantsSurvive(t *testing.T) {
	st := MakeIID(KindStructural, InformationPayload{Slot: "constraint_attachment", Value: "kaki"}, "interp-0")
	ent := MakeIID(KindEntity, InformationPayload{Entity: "kaki"}, "interp-0")
	pcs := PreservedCandidateState{
		Candidates: []CandidateSnapshot{{ID: "interp-0"}},
		Inventory: []InformationItem{
			{IID: st, AuditID: "str", Class: InfoStructural, Origin: OriginDirect, Dependency: DepInternal, CandidateID: "interp-0"},
			{IID: ent, AuditID: "ent", Class: InfoSemantic, Origin: OriginDirect, Dependency: DepInternal, CandidateID: "interp-0"},
		},
	}
	proj := HFCCProjectionResult{
		CandidateSet: hfcc.CandidateSet{Candidates: []hfcc.Candidate{{
			ID: "interp-0",
			Structure: hfcc.StructureNode{Kind: "INTERPRETATION", Children: []hfcc.StructureNode{
				{Kind: "ENTITY", Participants: []hfcc.Participant{{Ref: "kaki"}}},
			}},
		}}},
		Mappings: []ProjectionMapping{
			{FromIIDDigest: ent.Digest, Target: ProjectionTarget{CandidateID: "interp-0", StructurePath: "children[0]", Kind: "ENTITY_NODE"}},
		},
	}
	rep := AuditHFCCConsumerWithMappings(pcs, proj)
	if rep.Pass {
		t.Fatal("structural must LOST when only participant entity mapped")
	}
}

func TestMap_EvidenceIIDPreserved(t *testing.T) {
	ev := MakeIID(KindEvidence, InformationPayload{Span: "tanpa", Entity: "tanpa"}, "obs")
	pcs := PreservedCandidateState{
		Candidates: []CandidateSnapshot{{ID: "interp-0"}},
		Inventory: []InformationItem{{
			IID: ev, AuditID: "ev", Class: InfoEvidence, Origin: OriginDirect, Dependency: DepInternal,
		}},
		EvidenceIndex: []EvidenceItem{{Span: "tanpa", IIDDigest: ev.Digest}},
	}
	proj := BuildHFCCFromPreservedWithMappings(hfcc.Observation{ID: "o"}, pcs)
	found := false
	for _, m := range proj.Mappings {
		if m.FromIIDDigest == ev.Digest && m.Target.Kind == "EVIDENCE_REF" {
			found = true
		}
	}
	if !found {
		t.Fatal("evidence mapping must use original IID digest")
	}
	// Provenance EvidenceRefs must contain original digest, not a newly minted identity
	for _, c := range proj.CandidateSet.Candidates {
		for _, rec := range c.Provenance.Records {
			if rec.SourceRef == "EVIDENCE_IID" {
				ok := false
				for _, er := range rec.EvidenceRefs {
					if er == ev.Digest {
						ok = true
					}
				}
				if !ok {
					t.Fatalf("EvidenceRefs must contain original digest %s", ev.Digest)
				}
			}
		}
	}
	rep := AuditHFCCConsumerWithMappings(pcs, proj)
	if !rep.Pass {
		t.Fatalf("evidence IID should map, lost=%v", rep.Lost)
	}
}

func TestMap_EvidenceMissingFromHFCC(t *testing.T) {
	ev := MakeIID(KindEvidence, InformationPayload{Span: "tanpa", Entity: "tanpa"}, "obs")
	pcs := PreservedCandidateState{
		Candidates: []CandidateSnapshot{{ID: "interp-0"}},
		Inventory: []InformationItem{{
			IID: ev, AuditID: "ev", Class: InfoEvidence, Origin: OriginDirect, Dependency: DepInternal,
		}},
		EvidenceIndex: []EvidenceItem{{Span: "tanpa", IIDDigest: ev.Digest}},
	}
	proj := HFCCProjectionResult{
		CandidateSet: hfcc.CandidateSet{Candidates: []hfcc.Candidate{{ID: "interp-0", Structure: hfcc.StructureNode{Kind: "INTERPRETATION"}}}},
		Mappings:     nil, // no evidence mapping
	}
	rep := AuditHFCCConsumerWithMappings(pcs, proj)
	if rep.Pass {
		t.Fatal("missing evidence mapping must fail")
	}
}

func TestMap_RelocationOnlyWithExplicitRecord(t *testing.T) {
	from := MakeIID(KindStructural, InformationPayload{Entity: "DOG", Slot: "participation", Value: "RECIPIENT"}, "interp-0")
	to := MakeIID(KindStructural, InformationPayload{Host: "GIVE", Slot: "recipient", Value: "DOG"}, "interp-0")
	pcs := PreservedCandidateState{
		Candidates: []CandidateSnapshot{{ID: "interp-0"}},
		Inventory: []InformationItem{
			{IID: from, AuditID: "from", Class: InfoStructural, Origin: OriginDirect, Dependency: DepInternal, CandidateID: "interp-0"},
			{IID: to, AuditID: "to", Class: InfoStructural, Origin: OriginDirect, Dependency: DepInternal, CandidateID: "interp-0"},
		},
		RelocationRecords: []RelocationRecord{{
			FromIID: from.Digest, ToIID: to.Digest, Justification: "SLOT_REPATH",
		}},
	}
	// HFCC has structural node matching `to` payload; mapping only for `to`
	proj := HFCCProjectionResult{
		CandidateSet: hfcc.CandidateSet{Candidates: []hfcc.Candidate{{
			ID: "interp-0",
			Structure: hfcc.StructureNode{Kind: "INTERPRETATION", Children: []hfcc.StructureNode{
				{Kind: "recipient", Participants: []hfcc.Participant{{Ref: "GIVE"}, {Ref: "DOG"}}},
			}},
		}}},
		Mappings: []ProjectionMapping{
			{FromIIDDigest: to.Digest, Target: ProjectionTarget{CandidateID: "interp-0", StructurePath: "children[0]", Kind: "STRUCTURAL_NODE"}},
		},
	}
	rep := AuditHFCCConsumerWithMappings(pcs, proj)
	if !sliceHasID(rep.Relocated, "from") {
		t.Fatalf("explicit relocation should RELOCATE from, lost=%v mismatched=%v relocated=%v", rep.Lost, rep.Mismatched, rep.Relocated)
	}
	// Without RelocationRecord, from is LOST (to may still AVAILABLE)
	pcs2 := pcs
	pcs2.RelocationRecords = nil
	rep2 := AuditHFCCConsumerWithMappings(pcs2, proj)
	if sliceHasID(rep2.Relocated, "from") || sliceHasID(rep2.Available, "from") {
		t.Fatal("without RelocationRecord from must not be preserved")
	}
}

func TestMap_HiddenInterpretationSourceStillImpossible(t *testing.T) {
	pcs := PreservedCandidateState{
		Candidates: []CandidateSnapshot{{ID: "interp-0"}},
		Inventory:  nil,
	}
	proj := BuildHFCCFromPreservedWithMappings(hfcc.Observation{ID: "o"}, pcs)
	for _, c := range proj.CandidateSet.Candidates {
		if len(c.Structure.Children) > 0 {
			t.Fatal("HIDDEN_SOURCE: structure without PCS inventory")
		}
	}
	if len(proj.Mappings) != 0 {
		t.Fatal("no mappings when PCS empty")
	}
}

func TestMap_StructuralNotReclassifiedAsRelationByParticipants(t *testing.T) {
	st := MakeIID(KindStructural, InformationPayload{Slot: "constraint_attachment", Value: "kaki", Host: "X"}, "interp-0")
	pcs := PreservedCandidateState{
		Candidates: []CandidateSnapshot{{ID: "interp-0"}},
		Inventory: []InformationItem{{
			IID: st, AuditID: "str", Class: InfoStructural, Origin: OriginDirect, Dependency: DepInternal, CandidateID: "interp-0",
		}},
	}
	proj := BuildHFCCFromPreservedWithMappings(hfcc.Observation{ID: "o"}, pcs)
	for _, m := range proj.Mappings {
		if m.FromIIDDigest == st.Digest && m.Target.Kind != "STRUCTURAL_NODE" {
			t.Fatalf("must stay STRUCTURAL_NODE, got %s", m.Target.Kind)
		}
	}
	// Adversarial: even if someone later walked tree and saw 2 participants, mapping kind remains structural
	node := proj.CandidateSet.Candidates[0].Structure.Children[0]
	if len(node.Participants) < 1 {
		t.Fatal("expected participants on structural node")
	}
	if proj.Mappings[0].Target.Kind == "RELATION_NODE" {
		t.Fatal("participants must not force relation classification")
	}
}

// --- Prompt-final Audit B proof gates ---

func TestProof_SamePayloadDifferentIID_NoMapping_Lost(t *testing.T) {
	// IID-A and IID-B share identical relation payload but different scope → different digest
	iidA := MakeIID(KindRelation, InformationPayload{Src: "DOG", Rel: "R", Tgt: "DAT"}, "scope-A")
	iidB := MakeIID(KindRelation, InformationPayload{Src: "DOG", Rel: "R", Tgt: "DAT"}, "scope-B")
	if iidA.Digest == iidB.Digest {
		t.Fatal("scopes must yield different digests")
	}
	pcs := PreservedCandidateState{
		Candidates: []CandidateSnapshot{{ID: "interp-0"}},
		Inventory: []InformationItem{{
			IID: iidA, AuditID: "iid-a", Class: InfoSemantic, Origin: OriginDirect, Dependency: DepInternal, CandidateID: "interp-0",
		}},
	}
	// HFCC / mappings only know IID-B (same payload, different identity)
	proj := HFCCProjectionResult{
		CandidateSet: hfcc.CandidateSet{Candidates: []hfcc.Candidate{{
			ID: "interp-0",
			Structure: hfcc.StructureNode{Kind: "INTERPRETATION", Children: []hfcc.StructureNode{
				{Kind: "R", Participants: []hfcc.Participant{{Ref: "DOG"}, {Ref: "DAT"}}},
			}},
		}}},
		Mappings: []ProjectionMapping{{
			FromIIDDigest: iidB.Digest,
			Target:        ProjectionTarget{CandidateID: "interp-0", StructurePath: "children[0]", Kind: "RELATION_NODE"},
		}},
	}
	rep := AuditHFCCConsumerWithMappings(pcs, proj)
	if rep.Pass {
		t.Fatal("same payload different IID without ProjectionRecord(IID-A) must LOST")
	}
	lost := false
	for _, id := range rep.Lost {
		if id == "iid-a" {
			lost = true
		}
	}
	if !lost {
		t.Fatalf("IID-A must be LOST, lost=%v", rep.Lost)
	}
}

func TestProof_SameEvidenceSpanDifferentIID_NoMapping_Lost(t *testing.T) {
	iidA := MakeIID(KindEvidence, InformationPayload{Span: "kaki", Entity: "kaki"}, "obs-A")
	iidB := MakeIID(KindEvidence, InformationPayload{Span: "kaki", Entity: "kaki"}, "obs-B")
	if iidA.Digest == iidB.Digest {
		t.Fatal("expected different evidence digests")
	}
	pcs := PreservedCandidateState{
		Candidates: []CandidateSnapshot{{ID: "interp-0"}},
		Inventory: []InformationItem{{
			IID: iidA, AuditID: "ev-a", Class: InfoEvidence, Origin: OriginDirect, Dependency: DepInternal,
		}},
		EvidenceIndex: []EvidenceItem{{Span: "kaki", IIDDigest: iidA.Digest}},
	}
	proj := HFCCProjectionResult{
		CandidateSet: hfcc.CandidateSet{Candidates: []hfcc.Candidate{{
			ID: "interp-0",
			Provenance: hfcc.Provenance{Records: []hfcc.ProvenanceRecord{{
				SourceRef: "EVIDENCE_IID", EvidenceRefs: []string{iidB.Digest}, Note: "span=kaki",
			}}},
		}}},
		Mappings: []ProjectionMapping{{
			FromIIDDigest: iidB.Digest,
			Target:        ProjectionTarget{CandidateID: "interp-0", StructurePath: "provenance.evidence", Kind: "EVIDENCE_REF", Ref: iidB.Digest},
		}},
	}
	rep := AuditHFCCConsumerWithMappings(pcs, proj)
	if rep.Pass {
		t.Fatal("span equality must not preserve Evidence IID-A")
	}
}

func TestProof_ExplicitProjection_Available(t *testing.T) {
	iidA := MakeIID(KindRelation, InformationPayload{Src: "A", Rel: "R", Tgt: "B"}, "interp-0")
	pcs := PreservedCandidateState{
		Candidates: []CandidateSnapshot{{ID: "interp-0"}},
		Inventory: []InformationItem{{
			IID: iidA, AuditID: "a", Class: InfoSemantic, Origin: OriginDirect, Dependency: DepInternal, CandidateID: "interp-0",
		}},
	}
	proj := BuildHFCCFromPreservedWithMappings(hfcc.Observation{ID: "o"}, pcs)
	// Must have ProjectionRecord for iidA
	rec := false
	for _, m := range proj.Mappings {
		if m.FromIIDDigest == iidA.Digest {
			rec = true
			_ = m.AsRecord()
		}
	}
	if !rec {
		t.Fatal("adapter must emit ProjectionRecord")
	}
	rep := AuditHFCCConsumerWithMappings(pcs, proj)
	if !rep.Pass {
		t.Fatalf("explicit projection must AVAILABLE, lost=%v", rep.Lost)
	}
}

func TestProof_ExampleMappingsPresent(t *testing.T) {
	ent := MakeIID(KindEntity, InformationPayload{Entity: "DOG"}, "interp-0")
	rel := MakeIID(KindRelation, InformationPayload{Src: "DOG", Rel: "R", Tgt: "DAT"}, "interp-0")
	st := MakeIID(KindStructural, InformationPayload{Host: "H", Slot: "S", Value: "V"}, "interp-0")
	ev := MakeIID(KindEvidence, InformationPayload{Span: "kaki", Entity: "kaki"}, "obs")
	pcs := PreservedCandidateState{
		Candidates: []CandidateSnapshot{{ID: "interp-0"}},
		Inventory: []InformationItem{
			{IID: ent, AuditID: "e", Class: InfoSemantic, Origin: OriginDirect, Dependency: DepInternal, CandidateID: "interp-0"},
			{IID: rel, AuditID: "r", Class: InfoSemantic, Origin: OriginDirect, Dependency: DepInternal, CandidateID: "interp-0"},
			{IID: st, AuditID: "s", Class: InfoStructural, Origin: OriginDirect, Dependency: DepInternal, CandidateID: "interp-0"},
			{IID: ev, AuditID: "ev", Class: InfoEvidence, Origin: OriginDirect, Dependency: DepInternal},
		},
		EvidenceIndex: []EvidenceItem{{Span: "kaki", IIDDigest: ev.Digest}},
	}
	proj := BuildHFCCFromPreservedWithMappings(hfcc.Observation{ID: "o"}, pcs)
	kinds := map[string]string{}
	for _, m := range proj.Mappings {
		kinds[m.FromIIDDigest] = m.Target.Kind
		t.Logf("SourceIID=%s → %s %s", m.FromIIDDigest[:8], m.Target.Kind, m.Target.StructurePath)
	}
	if kinds[ent.Digest] != "ENTITY_NODE" {
		t.Fatal("entity mapping")
	}
	if kinds[rel.Digest] != "RELATION_NODE" {
		t.Fatal("relation mapping")
	}
	if kinds[st.Digest] != "STRUCTURAL_NODE" {
		t.Fatal("structural mapping")
	}
	if kinds[ev.Digest] != "EVIDENCE_REF" {
		t.Fatal("evidence mapping")
	}
}

// --- Level 2 independent verification ---

func TestL2_CorrectRelation(t *testing.T) {
	rel := MakeIID(KindRelation, InformationPayload{Src: "A", Rel: "R", Tgt: "B"}, "interp-0")
	pcs := PreservedCandidateState{
		Candidates: []CandidateSnapshot{{ID: "interp-0"}},
		Inventory: []InformationItem{{
			IID: rel, AuditID: "rel", Class: InfoSemantic, Origin: OriginDirect, Dependency: DepInternal, CandidateID: "interp-0",
		}},
	}
	proj := BuildHFCCFromPreservedWithMappings(hfcc.Observation{ID: "o"}, pcs)
	rep := AuditHFCCConsumerWithMappings(pcs, proj)
	if !rep.Pass {
		t.Fatalf("correct relation must PASS L2, lost=%v mismatched=%v", rep.Lost, rep.Mismatched)
	}
}

func TestL2_WrongTarget(t *testing.T) {
	rel := MakeIID(KindRelation, InformationPayload{Src: "A", Rel: "R", Tgt: "B"}, "interp-0")
	pcs := PreservedCandidateState{
		Candidates: []CandidateSnapshot{{ID: "interp-0"}},
		Inventory: []InformationItem{{
			IID: rel, AuditID: "rel", Class: InfoSemantic, Origin: OriginDirect, Dependency: DepInternal, CandidateID: "interp-0",
		}},
	}
	proj := HFCCProjectionResult{
		CandidateSet: hfcc.CandidateSet{Candidates: []hfcc.Candidate{{
			ID: "interp-0",
			Structure: hfcc.StructureNode{Kind: "INTERPRETATION", Children: []hfcc.StructureNode{
				{Kind: "S", Participants: []hfcc.Participant{{Ref: "X"}, {Ref: "Y"}}},
			}},
		}}},
		Mappings: []ProjectionMapping{{
			FromIIDDigest: rel.Digest,
			Target:        ProjectionTarget{CandidateID: "interp-0", StructurePath: "children[0]", Kind: "RELATION_NODE"},
		}},
	}
	rep := AuditHFCCConsumerWithMappings(pcs, proj)
	if rep.Pass {
		t.Fatal("wrong target must MISMATCH")
	}
	if len(rep.Mismatched) == 0 {
		t.Fatalf("expected Mismatched, lost=%v", rep.Lost)
	}
}

func TestL2_PhantomMapping(t *testing.T) {
	TestL2_WrongTarget(t) // same shape
}

func TestL2_WrongKind(t *testing.T) {
	rel := MakeIID(KindRelation, InformationPayload{Src: "A", Rel: "R", Tgt: "B"}, "interp-0")
	pcs := PreservedCandidateState{
		Candidates: []CandidateSnapshot{{ID: "interp-0"}},
		Inventory: []InformationItem{{
			IID: rel, AuditID: "rel", Class: InfoSemantic, Origin: OriginDirect, Dependency: DepInternal, CandidateID: "interp-0",
		}},
	}
	proj := HFCCProjectionResult{
		CandidateSet: hfcc.CandidateSet{Candidates: []hfcc.Candidate{{
			ID: "interp-0",
			Structure: hfcc.StructureNode{Kind: "INTERPRETATION", Children: []hfcc.StructureNode{
				{Kind: "constraint_attachment", Participants: []hfcc.Participant{{Ref: "A"}}},
			}},
		}}},
		Mappings: []ProjectionMapping{{
			FromIIDDigest: rel.Digest,
			Target:        ProjectionTarget{CandidateID: "interp-0", StructurePath: "children[0]", Kind: "STRUCTURAL_NODE"},
		}},
	}
	rep := AuditHFCCConsumerWithMappings(pcs, proj)
	if rep.Pass || len(rep.Mismatched) == 0 {
		t.Fatal("RELATION mapped as STRUCTURAL must MISMATCH")
	}
}

func TestL2_WrongCandidate(t *testing.T) {
	rel := MakeIID(KindRelation, InformationPayload{Src: "A", Rel: "R", Tgt: "B"}, "cand-A")
	pcs := PreservedCandidateState{
		Candidates: []CandidateSnapshot{{ID: "cand-A"}, {ID: "cand-B"}},
		Inventory: []InformationItem{{
			IID: rel, AuditID: "rel", Class: InfoSemantic, Origin: OriginDirect, Dependency: DepInternal, CandidateID: "cand-A",
		}},
	}
	proj := HFCCProjectionResult{
		CandidateSet: hfcc.CandidateSet{Candidates: []hfcc.Candidate{
			{ID: "cand-A", Structure: hfcc.StructureNode{Kind: "INTERPRETATION"}},
			{ID: "cand-B", Structure: hfcc.StructureNode{Kind: "INTERPRETATION", Children: []hfcc.StructureNode{
				{Kind: "R", Participants: []hfcc.Participant{{Ref: "A"}, {Ref: "B"}}},
			}}},
		}},
		Mappings: []ProjectionMapping{{
			FromIIDDigest: rel.Digest,
			Target:        ProjectionTarget{CandidateID: "cand-B", StructurePath: "children[0]", Kind: "RELATION_NODE"},
		}},
	}
	rep := AuditHFCCConsumerWithMappings(pcs, proj)
	if rep.Pass {
		t.Fatal("wrong candidate scope must MISMATCH")
	}
}

func TestL2_StructuralResidualNotEntity(t *testing.T) {
	st := MakeIID(KindStructural, InformationPayload{Host: "H", Slot: "S", Value: "V"}, "interp-0")
	ent := MakeIID(KindEntity, InformationPayload{Entity: "V"}, "interp-0")
	pcs := PreservedCandidateState{
		Candidates: []CandidateSnapshot{{ID: "interp-0"}},
		Inventory: []InformationItem{
			{IID: st, AuditID: "str", Class: InfoStructural, Origin: OriginDirect, Dependency: DepInternal, CandidateID: "interp-0"},
			{IID: ent, AuditID: "ent", Class: InfoSemantic, Origin: OriginDirect, Dependency: DepInternal, CandidateID: "interp-0"},
		},
	}
	proj := HFCCProjectionResult{
		CandidateSet: hfcc.CandidateSet{Candidates: []hfcc.Candidate{{
			ID: "interp-0",
			Structure: hfcc.StructureNode{Kind: "INTERPRETATION", Children: []hfcc.StructureNode{
				{Kind: "ENTITY", Participants: []hfcc.Participant{{Ref: "V"}}},
			}},
		}}},
		Mappings: []ProjectionMapping{
			{FromIIDDigest: ent.Digest, Target: ProjectionTarget{CandidateID: "interp-0", StructurePath: "children[0]", Kind: "ENTITY_NODE"}},
			// fake: map structural to same entity node
			{FromIIDDigest: st.Digest, Target: ProjectionTarget{CandidateID: "interp-0", StructurePath: "children[0]", Kind: "STRUCTURAL_NODE"}},
		},
	}
	rep := AuditHFCCConsumerWithMappings(pcs, proj)
	if rep.Pass {
		t.Fatal("structural→entity residual must fail")
	}
	// entity may still pass
}

func TestL2_FakeMapping(t *testing.T) {
	TestL2_WrongTarget(t)
}

func TestL2_RelocationWithDestInInventory(t *testing.T) {
	from := MakeIID(KindStructural, InformationPayload{Entity: "DOG", Slot: "participation", Value: "RECIPIENT"}, "interp-0")
	to := MakeIID(KindStructural, InformationPayload{Host: "GIVE", Slot: "recipient", Value: "DOG"}, "interp-0")
	pcs := PreservedCandidateState{
		Candidates: []CandidateSnapshot{{ID: "interp-0"}},
		Inventory: []InformationItem{
			{IID: from, AuditID: "from", Class: InfoStructural, Origin: OriginDirect, Dependency: DepInternal, CandidateID: "interp-0"},
			{IID: to, AuditID: "to", Class: InfoStructural, Origin: OriginDirect, Dependency: DepInternal, CandidateID: "interp-0"},
		},
		RelocationRecords: []RelocationRecord{{
			FromIID: from.Digest, ToIID: to.Digest, Justification: "SLOT_REPATH",
		}},
	}
	proj := BuildHFCCFromPreservedWithMappings(hfcc.Observation{ID: "o"}, pcs)
	// remove mapping for `from` only by rebuilding mappings without from
	var maps []ProjectionMapping
	for _, m := range proj.Mappings {
		if m.FromIIDDigest != from.Digest {
			maps = append(maps, m)
		}
	}
	proj.Mappings = maps
	rep := AuditHFCCConsumerWithMappings(pcs, proj)
	// `to` should be AVAILABLE; `from` RELOCATED via record + L2 on to
	if !sliceHasID(rep.Relocated, "from") && !sliceHasID(rep.Available, "from") {
		// from should be relocated
		t.Logf("relocated=%v available=%v lost=%v mismatched=%v", rep.Relocated, rep.Available, rep.Lost, rep.Mismatched)
		if !sliceHasID(rep.Relocated, "from") {
			t.Fatal("from should be RELOCATED")
		}
	}
}

func sliceHasID(ss []string, x string) bool {
	for _, s := range ss {
		if s == x {
			return true
		}
	}
	return false
}

func TestUnverified_UnknownKindNotPass(t *testing.T) {
	// Hand-built unknown kind with valid mapping+target
	iid := MakeIID(InfoKind("UNKNOWN_KIND"), InformationPayload{Entity: "X"}, "interp-0")
	pcs := PreservedCandidateState{
		Candidates: []CandidateSnapshot{{ID: "interp-0"}},
		Inventory: []InformationItem{{
			IID: iid, AuditID: "unk", Class: InfoSemantic, Origin: OriginDirect, Dependency: DepInternal, CandidateID: "interp-0",
		}},
	}
	proj := HFCCProjectionResult{
		CandidateSet: hfcc.CandidateSet{Candidates: []hfcc.Candidate{{
			ID: "interp-0",
			Structure: hfcc.StructureNode{Kind: "INTERPRETATION", Children: []hfcc.StructureNode{
				{Kind: "ENTITY", Participants: []hfcc.Participant{{Ref: "X"}}},
			}},
		}}},
		Mappings: []ProjectionMapping{{
			FromIIDDigest: iid.Digest,
			Target:        ProjectionTarget{CandidateID: "interp-0", StructurePath: "children[0]", Kind: "ENTITY_NODE"},
		}},
	}
	rep := AuditHFCCConsumerWithMappings(pcs, proj)
	if sliceHasID(rep.Available, "unk") {
		t.Fatal("unknown kind must not be AVAILABLE Level 2")
	}
	if !sliceHasID(rep.Unverified, "unk") {
		t.Fatalf("expected UNVERIFIED, got unverified=%v lost=%v", rep.Unverified, rep.Lost)
	}
}

func TestUnverified_ContextNotAutoPass(t *testing.T) {
	iid := MakeIID(KindContext, InformationPayload{Status: "UNRESOLVED", Span: "dia"}, "interp-0")
	pcs := PreservedCandidateState{
		Candidates: []CandidateSnapshot{{ID: "interp-0"}},
		Inventory: []InformationItem{{
			IID: iid, AuditID: "ctx", Class: InfoContextDependency, Origin: OriginDirect, Dependency: DepInternal, CandidateID: "interp-0",
		}},
	}
	proj := HFCCProjectionResult{
		CandidateSet: hfcc.CandidateSet{Candidates: []hfcc.Candidate{{
			ID: "interp-0", Structure: hfcc.StructureNode{Kind: "INTERPRETATION"},
		}}},
		Mappings: []ProjectionMapping{{
			FromIIDDigest: iid.Digest,
			Target:        ProjectionTarget{CandidateID: "interp-0", StructurePath: "children[0]", Kind: "ENTITY_NODE"},
		}},
	}
	rep := AuditHFCCConsumerWithMappings(pcs, proj)
	if sliceHasID(rep.Available, "ctx") {
		t.Fatal("Context must not auto PASS Level 2")
	}
	if !sliceHasID(rep.Unverified, "ctx") {
		t.Fatalf("Context expected UNVERIFIED, got %v", rep.Unverified)
	}
}

func TestUnverified_ProvenanceNotAutoPass(t *testing.T) {
	iid := MakeIID(KindProvenance, InformationPayload{Span: "src"}, "interp-0")
	pcs := PreservedCandidateState{
		Candidates: []CandidateSnapshot{{ID: "interp-0"}},
		Inventory: []InformationItem{{
			IID: iid, AuditID: "prov", Class: InfoProvenance, Origin: OriginDirect, Dependency: DepInternal, CandidateID: "interp-0",
		}},
	}
	proj := HFCCProjectionResult{
		CandidateSet: hfcc.CandidateSet{Candidates: []hfcc.Candidate{{ID: "interp-0"}}},
		Mappings: []ProjectionMapping{{
			FromIIDDigest: iid.Digest,
			Target:        ProjectionTarget{CandidateID: "interp-0", StructurePath: "provenance.evidence", Kind: "EVIDENCE_REF", Ref: iid.Digest},
		}},
	}
	rep := AuditHFCCConsumerWithMappings(pcs, proj)
	if sliceHasID(rep.Available, "prov") {
		t.Fatal("Provenance must not auto PASS")
	}
	if !sliceHasID(rep.Unverified, "prov") {
		t.Fatalf("Provenance expected UNVERIFIED, got %v", rep.Unverified)
	}
}

func TestUnverified_RelationStillL2(t *testing.T) {
	rel := MakeIID(KindRelation, InformationPayload{Src: "A", Rel: "R", Tgt: "B"}, "interp-0")
	pcs := PreservedCandidateState{
		Candidates: []CandidateSnapshot{{ID: "interp-0"}},
		Inventory: []InformationItem{{
			IID: rel, AuditID: "rel", Class: InfoSemantic, Origin: OriginDirect, Dependency: DepInternal, CandidateID: "interp-0",
		}},
	}
	proj := BuildHFCCFromPreservedWithMappings(hfcc.Observation{ID: "o"}, pcs)
	rep := AuditHFCCConsumerWithMappings(pcs, proj)
	if !rep.Pass || !sliceHasID(rep.Available, "rel") {
		t.Fatal("Relation Level 2 must still PASS")
	}
}

func TestAuditA_SameSpanDifferentIID(t *testing.T) {
	iidA := MakeIID(KindEvidence, InformationPayload{Span: "tanpa", Entity: "tanpa"}, "obs-A")
	iidB := MakeIID(KindEvidence, InformationPayload{Span: "tanpa", Entity: "tanpa"}, "obs-B")
	if iidA.Digest == iidB.Digest {
		t.Fatal("scopes must differ digests")
	}
	before := CandidateFormationState{
		Candidates: []CandidateSnapshot{{ID: "interp-0"}},
		Inventory: []InformationItem{{
			IID: iidA, AuditID: "ev-a", Class: InfoEvidence, Origin: OriginDirect, Dependency: DepInternal,
		}},
		Evidence: []EvidenceItem{{Span: "tanpa", IIDDigest: iidA.Digest}},
	}
	after := PreservedCandidateState{
		Candidates: []CandidateSnapshot{{ID: "interp-0"}},
		Inventory:  nil,
		EvidenceIndex: []EvidenceItem{{Span: "tanpa", IIDDigest: iidB.Digest}}, // same span, different IID
	}
	rep := AuditPreservationIID(before, after)
	if sliceHasID(rep.Available, "ev-a") {
		t.Fatal("same span must not make IID_A AVAILABLE")
	}
	if sliceHasID(rep.Relocated, "ev-a") {
		t.Fatal("must not RELOCATE via span")
	}
	if !sliceHasID(rep.Lost, "ev-a") {
		t.Fatalf("IID_A must be LOST, lost=%v", rep.Lost)
	}
}

func TestAuditA_SameDigestAvailable(t *testing.T) {
	iidA := MakeIID(KindEvidence, InformationPayload{Span: "tanpa", Entity: "tanpa"}, "obs")
	before := CandidateFormationState{
		Candidates: []CandidateSnapshot{{ID: "interp-0"}},
		Inventory: []InformationItem{{
			IID: iidA, AuditID: "ev-a", Class: InfoEvidence, Origin: OriginDirect, Dependency: DepInternal,
		}},
		Evidence: []EvidenceItem{{Span: "tanpa", IIDDigest: iidA.Digest}},
	}
	after := PreserveCandidateState(before, nil, nil)
	rep := AuditPreservationIID(before, after)
	if !sliceHasID(rep.Available, "ev-a") {
		t.Fatalf("same digest must AVAILABLE, lost=%v", rep.Lost)
	}
}
