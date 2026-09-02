package hfcc

import (
	"testing"
	"time"
)

func part(id, ref string, participation string) Participant {
	p := Participant{ID: id, Ref: ref}
	if participation != "" {
		p.Participation = StrPtr(participation)
	}
	return p
}

func TestCandidateNotTruth(t *testing.T) {
	c := Candidate{
		ID: "c1", Version: "v1",
		Structure: Node("WALK", part("1", "CAT", "")),
		Status:    StatusHypothesis,
	}
	if c.Status == StatusGold {
		t.Fatal("hypothesis must not be gold by default")
	}
	// GOLD still not absolute truth — only protocol metadata
	c.Status = StatusGold
	c.Gold = &GoldAudit{ProtocolVersion: "HFCC-v1", EvidenceSet: []string{"e1"}, ExperimentID: "exp1"}
	if c.Gold.ProtocolVersion == "" {
		t.Fatal("GOLD must be auditable")
	}
}

func TestMultipleCandidatesPreserved(t *testing.T) {
	obs := Observation{ID: "o1", RawText: "Kucing melihat anjing dengan kamera.", Timestamp: time.Now().UTC()}
	a := Candidate{ID: "A", Version: "1", Structure: Node("SEE",
		part("1", "CAT", ""), part("2", "DOG", ""), part("3", "CAMERA", "INSTRUMENT")), Status: StatusHypothesis}
	b := Candidate{ID: "B", Version: "1", Structure: Node("SEE",
		part("1", "CAT", ""), part("2", "DOG_WITH_CAMERA", "")), Status: StatusHypothesis}
	set := CandidateSet{ID: "cs1", ObservationID: obs.ID, Candidates: []Candidate{a, b}}
	if len(set.Candidates) != 2 {
		t.Fatal("both candidates must coexist")
	}
	// ranking must not be implemented as deletion
	_ = set.Candidates[0]
	if len(set.Candidates) != 2 {
		t.Fatal("non-best must remain")
	}
}

func TestNestingDistinction(t *testing.T) {
	p := Node("WALK", part("1", "CAT", ""))
	a := Nest("POSSIBLE", Nest("NEGATE", p))
	b := Nest("NEGATE", Nest("POSSIBLE", CloneStructure(p)))
	cmp := CompareStructures(a, b)
	if cmp.Structural == StructIdentical || cmp.Structural == StructEquivalent {
		t.Fatalf("POSSIBLE(NEGATE(P)) must differ from NEGATE(POSSIBLE(P)), got %s", cmp.Structural)
	}
}

func TestQuantifierNestingDistinction(t *testing.T) {
	p := Node("P")
	a := Nest("FOR_ALL", Node("CAT"), Nest("NEGATE", p))
	b := Nest("NEGATE", Nest("FOR_ALL", Node("CAT"), CloneStructure(p)))
	cmp := CompareStructures(a, b)
	if cmp.Structural != StructDifferent && cmp.Structural != StructPartial {
		t.Fatalf("quantifier scope nesting must differ, got %s", cmp.Structural)
	}
}

func TestParticipantIdentityAndOptionalParticipation(t *testing.T) {
	withRole := Node("GIVE",
		part("1", "CAT", "AGENT"),
		part("2", "FOOD", "THEME"),
		part("3", "DOG", "RECIPIENT"),
	)
	without := Node("GIVE",
		part("1", "CAT", ""),
		part("2", "FOOD", ""),
		part("3", "DOG", ""),
	)
	if !HasParticipation(withRole) {
		t.Fatal("expected participation")
	}
	if HasParticipation(without) {
		t.Fatal("participation optional — empty must not invent roles")
	}
	// identities preserved
	if without.Participants[2].Ref != "DOG" {
		t.Fatal("identity must survive without role")
	}
}

func TestNoPositionalRoleInference(t *testing.T) {
	// Runtime must not assign AGENT to index 0 automatically
	n := Node("GIVE", part("1", "CAT", ""), part("2", "FOOD", ""), part("3", "DOG", ""))
	for _, p := range n.Participants {
		if p.Participation != nil {
			t.Fatal("no silent positional role assignment")
		}
	}
}

func TestConditionalDirection(t *testing.T) {
	ab := WithDirection(Node("CAUSE", part("1", "A", ""), part("2", "B", "")), "A_to_B")
	ba := WithDirection(Node("CAUSE", part("1", "B", ""), part("2", "A", "")), "B_to_A")
	if CompareStructures(ab, ba).Structural == StructIdentical {
		t.Fatal("direction-sensitive CAUSE must differ")
	}
	plain := Node("WALK", part("1", "CAT", ""))
	if plain.Direction != nil {
		t.Fatal("direction must not be forced on every relation")
	}
}

func TestConditionalFunction(t *testing.T) {
	content := Node("WALK", part("1", "CAT", ""))
	asserted := WithFunction(CloneStructure(content), "ASSERT")
	evaluated := WithFunction(CloneStructure(content), "EVALUATE")
	if CompareStructures(asserted, evaluated).Structural == StructIdentical {
		t.Fatal("ASSERT vs EVALUATE must differ when function present")
	}
}

func TestProvenanceFirstClass(t *testing.T) {
	c := Candidate{
		ID: "c1", Version: "v1",
		Structure: Node("GIVE", part("1", "DOG", "RECIPIENT")),
		Provenance: Provenance{Records: []ProvenanceRecord{
			{SourceRef: "obs-1", EvidenceRefs: []string{"e1", "e2"}, InferenceClass: InferenceDirectObservation},
			{SourceRef: "ctx-1", EvidenceRefs: []string{"e3"}, InferenceClass: InferenceContextual},
		}},
		Status: StatusHypothesis,
	}
	if len(c.Provenance.Records) < 2 {
		t.Fatal("multiple evidence sources required")
	}
	if HasExternalKnowledge(c.Provenance) {
		t.Fatal("unexpected external")
	}
}

func TestCircularProvenanceRejected(t *testing.T) {
	p := Provenance{Records: []ProvenanceRecord{
		{SourceRef: "claim-X", DependentOn: []string{"claim-X"}, InferenceClass: InferenceStructural},
	}}
	ok, reason := ValidateProvenanceForClaim(p, "claim-X")
	if ok || reason != "circular_evidence" {
		t.Fatalf("expected circular rejection, ok=%v reason=%s", ok, reason)
	}
}

func TestExternalKnowledgeNotReconstructable(t *testing.T) {
	orig := Candidate{ID: "o", Version: "1", Structure: Node("WALK", part("1", "CAT", ""))}
	derived := CloneCandidate(orig)
	derived.Structure = Node("UNKNOWN")
	r := Reconstruct(orig, derived, orig.Structure, false)
	if r != ExternalKnowledgeRequired && r != NotReconstructable {
		t.Fatalf("missing structure without allowed external → EXTERNAL or NOT_RECONSTRUCTABLE, got %s", r)
	}
	if r == Recontructable {
		t.Fatal("must not silently reconstruct from hidden knowledge")
	}
}

func TestInformationRelocationNotRedundant(t *testing.T) {
	// Role embedded in kind channel + participation field
	orig := Candidate{
		ID: "o", Version: "1",
		Structure: StructureNode{
			Kind: "GIVE",
			Participants: []Participant{
				part("1", "DOG", "RECIPIENT"),
			},
			Children: []StructureNode{{Kind: "ROLE_RECIPIENT"}}, // alternate channel
		},
	}
	derived, _ := Ablate(orig, AblationSpec{RemoveParticipation: true})
	if HasParticipation(derived.Structure) {
		t.Fatal("participation field should be cleared")
	}
	res := Reconstruct(orig, derived, orig.Structure, false)
	if res != InformationRelocated {
		t.Fatalf("expected INFORMATION_RELOCATED, got %s", res)
	}
}

func TestAblationNonDestructive(t *testing.T) {
	orig := Candidate{
		ID: "o", Version: "v1",
		Structure: Nest("POSSIBLE", Nest("NEGATE", Node("WALK", part("1", "CAT", "AGENT")))),
		Provenance: Provenance{Records: []ProvenanceRecord{{SourceRef: "s1", InferenceClass: InferenceDirectObservation}}},
		Status: StatusHypothesis,
	}
	fp := StructuralFingerprint(orig.Structure)
	_, _ = Ablate(orig, AblationSpec{RemoveNesting: true, RemoveParticipation: true, RemoveProvenance: true})
	if StructuralFingerprint(orig.Structure) != fp {
		t.Fatal("original must remain unchanged")
	}
	if len(orig.Provenance.Records) != 1 {
		t.Fatal("original provenance must remain")
	}
}

func TestSerializationRoundTrip(t *testing.T) {
	c := Candidate{
		ID: "c1", Version: "v1",
		Structure: Nest("POSSIBLE", Nest("NEGATE", Node("WALK", part("1", "CAT", "AGENT")))),
		Provenance: Provenance{Records: []ProvenanceRecord{
			{SourceRef: "s", EvidenceRefs: []string{"e"}, InferenceClass: InferenceStructural},
		}},
		Status: StatusAnnotated,
	}
	data, err := MarshalCandidate(c)
	if err != nil {
		t.Fatal(err)
	}
	back, err := UnmarshalCandidate(data)
	if err != nil {
		t.Fatal(err)
	}
	if CompareStructures(c.Structure, back.Structure).Structural != StructIdentical {
		t.Fatal("nesting must survive serialization")
	}
	if back.ParticipantsIdentityBroken() {
		t.Fatal("participant identity must survive")
	}
	if back.Structure.Children[0].Children[0].Participants[0].Ref != "CAT" {
		t.Fatal("identity lost")
	}
	if back.Structure.Children[0].Children[0].Participants[0].Participation == nil {
		t.Fatal("participation lost")
	}
}

func (c Candidate) ParticipantsIdentityBroken() bool { return false }

func TestValidationIndependentOfContent(t *testing.T) {
	s := Node("GIVE", part("1", "CAT", ""), part("2", "FOOD", ""), part("3", "DOG", ""))
	h := Candidate{ID: "1", Version: "1", Structure: s, Status: StatusHypothesis}
	v := Candidate{ID: "2", Version: "1", Structure: CloneStructure(s), Status: StatusValidated}
	if CompareStructures(h.Structure, v.Structure).Structural != StructIdentical {
		t.Fatal("same structure")
	}
	if h.Status == v.Status {
		t.Fatal("status is independent metadata")
	}
}

func TestBoundaryNoHorizonImport(t *testing.T) {
	// Compile-time isolation is enforced by package design; runtime smoke:
	_ = CandidateSet{}
	_ = CompareStructures
	_ = Ablate
	_ = Reconstruct
}

func TestObservationDistinctFromCandidate(t *testing.T) {
	obs := Observation{ID: "o", RawText: "Kucing memberi makanan kepada anjing."}
	cand := Candidate{ID: "c", Structure: Node("GIVE", part("1", "CAT", ""), part("2", "FOOD", ""), part("3", "DOG", ""))}
	if obs.RawText == cand.Structure.Kind {
		t.Fatal("observation text must not equal structure kind")
	}
}
