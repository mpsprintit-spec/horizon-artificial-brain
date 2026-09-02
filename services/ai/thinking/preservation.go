package thinking

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/project-horizon/horizon-core/services/ai/knowledge"
)

// --- Observation reference ---

type ObservationRef struct {
	ID        string `json:"id"`
	Version   string `json:"version"`
	SourceRef string `json:"source_ref"`
}

// --- Audit information kinds (not semantic ontology) ---

type InfoKind string

const (
	KindEntity     InfoKind = "ENTITY_INFORMATION"
	KindRelation   InfoKind = "RELATION_INFORMATION"
	KindStructural InfoKind = "STRUCTURAL_INFORMATION"
	KindEvidence   InfoKind = "EVIDENCE_INFORMATION"
	KindContext    InfoKind = "CONTEXT_INFORMATION"
	KindProvenance InfoKind = "PROVENANCE_INFORMATION"
	KindRuntime    InfoKind = "RUNTIME_METADATA"
)

// Reporting class (functional reporting label; still not semantic truth).
type InformationClass string

const (
	InfoSemantic          InformationClass = "SEMANTIC"
	InfoStructural        InformationClass = "STRUCTURAL"
	InfoEvidence          InformationClass = "EVIDENCE"
	InfoProvenance        InformationClass = "PROVENANCE"
	InfoContextDependency InformationClass = "CONTEXT_DEPENDENCY"
	InfoRuntimeMetadata   InformationClass = "RUNTIME_METADATA"
)

type InfoOrigin string

const (
	OriginDirect  InfoOrigin = "DIRECT"
	OriginDerived InfoOrigin = "DERIVED"
)

type InfoDependency string

const (
	DepInternal InfoDependency = "INTERNAL"
	DepExternal InfoDependency = "EXTERNAL_KNOWLEDGE"
)

type Availability string

const (
	AvailAvailable           Availability = "AVAILABLE"
	AvailExplicitlyDiscarded Availability = "EXPLICITLY_DISCARDED"
	AvailLost                Availability = "LOST"
	AvailMismatch            Availability = "MISMATCH" // target exists but does not represent IID
	AvailUnverified          Availability = "UNVERIFIED" // no Level 2 verifier for this kind
	AvailNotFormed           Availability = "NOT_FORMED" // diagnostic only; not a preservation fate of an IID
)

type Location string

const (
	LocOriginal  Location = "ORIGINAL"
	LocRelocated Location = "RELOCATED"
)

// InformationPayload is structured identity material (not free-text bag matching).
type InformationPayload struct {
	Entity   string `json:"entity,omitempty"`
	Src      string `json:"src,omitempty"`
	Rel      string `json:"rel,omitempty"`
	Tgt      string `json:"tgt,omitempty"`
	Polarity string `json:"polarity,omitempty"` // e.g. inhibitory
	Host     string `json:"host,omitempty"`
	Slot     string `json:"slot,omitempty"`
	Value    string `json:"value,omitempty"`
	Span     string `json:"span,omitempty"`
	Metric   string `json:"metric,omitempty"`
	Status   string `json:"status,omitempty"` // e.g. UNRESOLVED
}

// InformationIID is a stable audit identity (not semantic ontology identity).
type InformationIID struct {
	Kind     InfoKind           `json:"kind"`
	Payload  InformationPayload `json:"payload"`
	Scope    string             `json:"scope"` // observation + candidate scope
	Digest   string             `json:"digest"`
}

func MakeIID(kind InfoKind, payload InformationPayload, scope string) InformationIID {
	id := InformationIID{Kind: kind, Payload: payload, Scope: scope}
	id.Digest = fingerprintIID(id)
	return id
}

func fingerprintIID(id InformationIID) string {
	b, _ := json.Marshal(struct {
		Kind    InfoKind           `json:"k"`
		Payload InformationPayload `json:"p"`
		Scope   string             `json:"s"`
	}{id.Kind, id.Payload, id.Scope})
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:8])
}

// InformationItem carries one auditable unit with typed IID.
type InformationItem struct {
	IID           InformationIID   `json:"iid"`
	AuditID       string           `json:"audit_id"` // human/debug handle
	Class         InformationClass `json:"class"`
	Origin        InfoOrigin       `json:"origin"`
	Dependency    InfoDependency   `json:"dependency"`
	ProvenanceRef string           `json:"provenance_ref,omitempty"`
	CandidateID   string           `json:"candidate_id,omitempty"`
}

// RelocationRecord is the only valid basis for LOCATION=RELOCATED.
type RelocationRecord struct {
	FromIID       string `json:"from_iid"` // digest
	ToIID         string `json:"to_iid"`
	Justification string `json:"justification"` // ENCODING_CHANGE | SLOT_REPATH
}

// DerivationRecord minimum graph-capable derivation dependency.
type DerivationRecord struct {
	ID           string         `json:"id"`
	ConcludesIID string         `json:"concludes_iid"` // digest of C
	PremiseIIDs  []string       `json:"premise_iids"`
	RuleRef      string         `json:"rule_ref"`
	Dependency   InfoDependency `json:"dependency"`
}

type EvidenceItem struct {
	ID         string `json:"id"`
	SourceRef  string `json:"source_ref"`
	Span       string `json:"span,omitempty"`
	ContentRef string `json:"content_ref"`
	IIDDigest  string `json:"iid_digest,omitempty"`
}

type CandidateSnapshot struct {
	ID                  string `json:"id"`
	Version             string `json:"version"`
	StructureRef        string `json:"structure_ref"`
	InterpretationIndex int    `json:"interpretation_index"`
}

// CandidateFormationState is BEFORE projection at the boundary.
type CandidateFormationState struct {
	Observation  ObservationRef       `json:"observation"`
	Candidates   []CandidateSnapshot  `json:"candidates"`
	Inventory    []InformationItem    `json:"inventory"`
	Evidence     []EvidenceItem       `json:"evidence"`
	Derivations  []DerivationRecord   `json:"derivations,omitempty"`
	RankingOrder []string             `json:"ranking_order,omitempty"`
	FormedAt     time.Time            `json:"formed_at"`
}

// PreservedCandidateState is the authoritative AFTER source for preservation audit.
// HFCC remains a consumer; evidence survival is judged from this state, not echo.
type PreservedCandidateState struct {
	Observation      ObservationRef      `json:"observation"`
	Candidates       []CandidateSnapshot  `json:"candidates"`
	Inventory        []InformationItem    `json:"inventory"`
	EvidenceIndex    []EvidenceItem       `json:"evidence_index"` // what THIS state retains
	RelocationRecords []RelocationRecord  `json:"relocation_records,omitempty"`
	Derivations      []DerivationRecord   `json:"derivations,omitempty"`
	PreservedAt      time.Time            `json:"preserved_at"`
}

type PreservationDimensions struct {
	Availability Availability   `json:"availability"`
	Location     Location       `json:"location"`
	Origin       InfoOrigin     `json:"origin"`
	Dependency   InfoDependency `json:"dependency"`
	Reason       string         `json:"reason,omitempty"`
}

type ItemAudit struct {
	Item   InformationItem        `json:"item"`
	Before bool                   `json:"before"`
	After  bool                   `json:"after"`
	Dims   PreservationDimensions `json:"dims"`
}

type PreservationReport struct {
	Pass              bool        `json:"pass"`
	Available         []string    `json:"available,omitempty"`
	Relocated         []string    `json:"relocated,omitempty"`
	Derived           []string    `json:"derived,omitempty"`
	ExplicitlyDiscard []string    `json:"explicitly_discarded,omitempty"`
	Lost              []string    `json:"lost,omitempty"`
	Mismatched         []string    `json:"mismatched,omitempty"`
	Unverified         []string    `json:"unverified,omitempty"`
	NotFormed         []string    `json:"not_formed,omitempty"`
	ExternalDeps      []string    `json:"external_dependencies,omitempty"`
	ItemAudits        []ItemAudit `json:"item_audits,omitempty"`
	FailureDomain     string      `json:"failure_domain,omitempty"`
	Notes             []string    `json:"notes,omitempty"`
}

// BuildCandidateFormationState projects typed IIDs from interpretations (no invented semantics).
func BuildCandidateFormationState(
	obsRef ObservationRef,
	kb *knowledge.KnowledgeBase,
	stimulus []string,
	interps []Interpretation,
) CandidateFormationState {
	state := CandidateFormationState{
		Observation: obsRef,
		FormedAt:    time.Now().UTC(),
	}
	scopeObs := obsRef.ID + "|obs"

	for i, tok := range stimulus {
		payload := InformationPayload{Span: tok, Entity: tok}
		iid := MakeIID(KindEvidence, payload, scopeObs)
		state.Evidence = append(state.Evidence, EvidenceItem{
			ID: fmt.Sprintf("e-%03d", i+1), SourceRef: obsRef.ID, Span: tok, ContentRef: tok, IIDDigest: iid.Digest,
		})
		state.Inventory = append(state.Inventory, InformationItem{
			IID: iid, AuditID: fmt.Sprintf("ev-%s-%d", obsRef.ID, i),
			Class: InfoEvidence, Origin: OriginDirect, Dependency: DepInternal,
		})
		// Entity handle for known tokens (same span) — distinct kind from pure evidence if needed
		ent := MakeIID(KindEntity, InformationPayload{Entity: tok}, scopeObs)
		state.Inventory = append(state.Inventory, InformationItem{
			IID: ent, AuditID: fmt.Sprintf("ent-%s-%d", obsRef.ID, i),
			Class: InfoSemantic, Origin: OriginDirect, Dependency: DepInternal,
		})
	}

	for i, interp := range interps {
		cid := InterpretationCandidateID(i)
		scope := obsRef.ID + "|" + cid
		snap := CandidateSnapshot{
			ID: cid, Version: "formation-1",
			StructureRef: structureRefFromInterp(kb, interp), InterpretationIndex: i,
		}
		state.Candidates = append(state.Candidates, snap)
		state.RankingOrder = append(state.RankingOrder, cid)

		for _, p := range interp.Propositions {
			iid := MakeIID(KindRelation, InformationPayload{
				Src: p.TargetTok, Rel: string(p.Relation), Tgt: p.ObjectTok,
			}, scope)
			state.Inventory = append(state.Inventory, InformationItem{
				IID: iid, AuditID: fmt.Sprintf("sem-%s-%s", cid, p.ID),
				Class: InfoSemantic, Origin: OriginDirect, Dependency: DepInternal,
				CandidateID: cid, ProvenanceRef: p.ID,
			})
		}
		for _, r := range interp.Relations {
			src, tgt := tokenRef(kb, r.SourceID), tokenRef(kb, r.TargetID)
			pol := ""
			if r.Inhibitory {
				pol = "inhibitory"
			}
			origin := OriginDirect
			if r.Provenance == "reverse" || r.Provenance == "collateral" {
				origin = OriginDerived
			}
			iid := MakeIID(KindRelation, InformationPayload{
				Src: src, Rel: string(r.Kind), Tgt: tgt, Polarity: pol,
			}, scope)
			state.Inventory = append(state.Inventory, InformationItem{
				IID: iid, AuditID: fmt.Sprintf("rel-%s-%d-%s-%d", cid, r.SourceID, r.Kind, r.TargetID),
				Class: InfoSemantic, Origin: origin, Dependency: DepInternal,
				CandidateID: cid, ProvenanceRef: string(r.Provenance),
			})
		}
		for _, c := range interp.Constraints {
			iid := MakeIID(KindStructural, InformationPayload{
				Slot: "constraint_attachment", Value: c.Token,
			}, scope)
			state.Inventory = append(state.Inventory, InformationItem{
				IID: iid, AuditID: fmt.Sprintf("str-%s-%s", cid, c.Token),
				Class: InfoStructural, Origin: OriginDirect, Dependency: DepInternal,
				CandidateID: cid,
			})
		}
		rt := MakeIID(KindRuntime, InformationPayload{Metric: fmt.Sprintf("total_score=%.4f", interp.TotalScore)}, scope)
		state.Inventory = append(state.Inventory, InformationItem{
			IID: rt, AuditID: fmt.Sprintf("rt-%s-score", cid),
			Class: InfoRuntimeMetadata, Origin: OriginDerived, Dependency: DepInternal,
			CandidateID: cid,
		})
	}
	return state
}

func structureRefFromInterp(kb *knowledge.KnowledgeBase, interp Interpretation) string {
	var parts []string
	for _, r := range interp.Relations {
		parts = append(parts, fmt.Sprintf("%s>%s>%s", tokenRef(kb, r.SourceID), r.Kind, tokenRef(kb, r.TargetID)))
	}
	for _, p := range interp.Propositions {
		parts = append(parts, fmt.Sprintf("P:%s(%s,%s)", p.Relation, p.TargetTok, p.ObjectTok))
	}
	for _, c := range interp.Constraints {
		parts = append(parts, "C:"+c.Token)
	}
	sort.Strings(parts)
	return strings.Join(parts, "|")
}

// PreserveCandidateState builds the authoritative preserved snapshot at the boundary.
// Copies formation inventory/evidence by value; does not consult HFCC for evidence.
// relocs/derivs may be empty unless explicitly supplied by adapter/tests.
func PreserveCandidateState(
	formation CandidateFormationState,
	relocs []RelocationRecord,
	derivs []DerivationRecord,
) PreservedCandidateState {
	inv := append([]InformationItem{}, formation.Inventory...)
	// Runtime metadata is not retained in preserved research inventory (explicit discard policy).
	var kept []InformationItem
	for _, it := range inv {
		if it.IID.Kind == KindRuntime || it.Class == InfoRuntimeMetadata {
			continue
		}
		kept = append(kept, it)
	}
	ev := append([]EvidenceItem{}, formation.Evidence...)
	cands := append([]CandidateSnapshot{}, formation.Candidates...)
	d := append([]DerivationRecord{}, formation.Derivations...)
	d = append(d, derivs...)
	return PreservedCandidateState{
		Observation:       formation.Observation,
		Candidates:        cands,
		Inventory:         kept,
		EvidenceIndex:     ev,
		RelocationRecords: append([]RelocationRecord{}, relocs...),
		Derivations:       d,
		PreservedAt:       time.Now().UTC(),
	}
}

// AuditPreservationIID compares BEFORE formation inventory to AFTER PreservedCandidateState by IID.
// Token/string overlap is not used. RELOCATED requires RelocationRecord.
func AuditPreservationIID(before CandidateFormationState, after PreservedCandidateState) PreservationReport {
	report := PreservationReport{Pass: true}
	afterByDigest := map[string]InformationItem{}
	for _, it := range after.Inventory {
		afterByDigest[it.IID.Digest] = it
	}
	// Evidence after: only what EvidenceIndex holds
	afterEv := map[string]bool{} // digest-only identity; span is metadata, not identity
	for _, e := range after.EvidenceIndex {
		if e.IIDDigest != "" {
			afterEv[e.IIDDigest] = true
		}
		// empty IIDDigest: do not index by Span — cannot claim identity conservation
	}
	relocTo := map[string]string{} // from digest -> to digest
	for _, r := range after.RelocationRecords {
		relocTo[r.FromIID] = r.ToIID
	}
	afterCand := map[string]bool{}
	for _, c := range after.Candidates {
		afterCand[c.ID] = true
	}

	for _, it := range before.Inventory {
		audit := ItemAudit{Item: it, Before: true}
		if it.IID.Kind == KindRuntime || it.Class == InfoRuntimeMetadata {
			audit.Dims = PreservationDimensions{
				Availability: AvailExplicitlyDiscarded, Location: LocOriginal,
				Origin: it.Origin, Dependency: it.Dependency, Reason: "RUNTIME_ONLY",
			}
			report.ExplicitlyDiscard = append(report.ExplicitlyDiscard, it.AuditID)
			report.ItemAudits = append(report.ItemAudits, audit)
			continue
		}

		if it.IID.Kind == KindEvidence {
			if it.IID.Digest == "" {
				audit.After = false
				audit.Dims = PreservationDimensions{
					Availability: AvailUnverified, Location: LocOriginal,
					Origin: it.Origin, Dependency: it.Dependency,
					Reason: "evidence IID digest empty; span is not identity",
				}
				report.Unverified = append(report.Unverified, it.AuditID)
				report.ItemAudits = append(report.ItemAudits, audit)
				continue
			}
			ok := afterEv[it.IID.Digest]
			audit.After = ok
			if ok {
				audit.Dims = PreservationDimensions{
					Availability: AvailAvailable, Location: LocOriginal,
					Origin: it.Origin, Dependency: it.Dependency,
				}
				report.Available = append(report.Available, it.AuditID)
			} else {
				audit.Dims = PreservationDimensions{
					Availability: AvailLost, Location: LocOriginal,
					Origin: it.Origin, Dependency: it.Dependency,
					Reason: "evidence IID digest not in EvidenceIndex (span ignored)",
				}
				report.Lost = append(report.Lost, it.AuditID)
				report.Pass = false
				report.FailureDomain = "PRESERVATION_FAILURE"
			}
			report.ItemAudits = append(report.ItemAudits, audit)
			continue
		}

		if _, ok := afterByDigest[it.IID.Digest]; ok {
			audit.After = true
			audit.Dims = PreservationDimensions{
				Availability: AvailAvailable, Location: LocOriginal,
				Origin: it.Origin, Dependency: it.Dependency,
			}
			report.Available = append(report.Available, it.AuditID)
			if it.Origin == OriginDerived {
				report.Derived = append(report.Derived, it.AuditID)
			}
			if it.Dependency == DepExternal {
				report.ExternalDeps = append(report.ExternalDeps, it.AuditID)
			}
			report.ItemAudits = append(report.ItemAudits, audit)
			continue
		}

		if to, ok := relocTo[it.IID.Digest]; ok {
			if _, exists := afterByDigest[to]; exists {
				audit.After = true
				audit.Dims = PreservationDimensions{
					Availability: AvailAvailable, Location: LocRelocated,
					Origin: it.Origin, Dependency: it.Dependency,
					Reason: "explicit RelocationRecord",
				}
				report.Relocated = append(report.Relocated, it.AuditID)
				report.ItemAudits = append(report.ItemAudits, audit)
				continue
			}
		}

		audit.After = false
		audit.Dims = PreservationDimensions{
			Availability: AvailLost, Location: LocOriginal,
			Origin: it.Origin, Dependency: it.Dependency,
			Reason: "IID absent after boundary; no valid relocation",
		}
		report.Lost = append(report.Lost, it.AuditID)
		report.Pass = false
		report.FailureDomain = "PRESERVATION_FAILURE"
		report.ItemAudits = append(report.ItemAudits, audit)
	}

	for _, c := range before.Candidates {
		if !afterCand[c.ID] {
			report.Lost = append(report.Lost, "candidate:"+c.ID)
			report.Pass = false
			report.FailureDomain = "PRESERVATION_FAILURE"
			report.Notes = append(report.Notes, "candidate alternative lost: "+c.ID)
		}
	}
	return report
}

// AuditTanpaKakiCase: independent BEFORE inspection only; no WITHOUT invention.
func AuditTanpaKakiCase(before CandidateFormationState, stimulus []string) (domain string, notes []string) {
	hasCanDo, hasTanpaEv, hasKakiEv, hasStructuralTanpa := false, false, false, false
	for _, tok := range stimulus {
		if tok == "tanpa" {
			hasTanpaEv = true
		}
		if tok == "kaki" {
			hasKakiEv = true
		}
	}
	for _, it := range before.Inventory {
		switch it.IID.Kind {
		case KindRelation:
			if strings.EqualFold(it.IID.Payload.Rel, string(knowledge.RelationCanDo)) ||
				strings.Contains(strings.ToLower(it.IID.Payload.Rel), "can_do") {
				hasCanDo = true
			}
			// structural relation tanpa–kaki would need Rel linking those endpoints
			if (it.IID.Payload.Src == "tanpa" && it.IID.Payload.Tgt == "kaki") ||
				(it.IID.Payload.Src == "kaki" && it.IID.Payload.Tgt == "tanpa") {
				hasStructuralTanpa = true
			}
		case KindStructural:
			if it.IID.Payload.Value == "tanpa" || it.IID.Payload.Slot == "tanpa" {
				hasStructuralTanpa = true
			}
		case KindEvidence:
			if it.IID.Payload.Span == "tanpa" {
				hasTanpaEv = true
			}
			if it.IID.Payload.Span == "kaki" {
				hasKakiEv = true
			}
		}
	}
	notes = append(notes, fmt.Sprintf("before_can_do=%v tanpa_evidence=%v kaki_evidence=%v structural_tanpa_kaki=%v",
		hasCanDo, hasTanpaEv, hasKakiEv, hasStructuralTanpa))
	if (hasTanpaEv || hasKakiEv) && !hasStructuralTanpa {
		notes = append(notes, "NOT_FORMED: structural/relation information linking tanpa and kaki")
		return "UNDERSTANDING_FORMATION_MISSING_INFORMATION", notes
	}
	return "", notes
}

// Legacy helpers kept for older tests that constructed bare inventories — prefer AuditPreservationIID.
func AuditPreservation(before CandidateFormationState, afterInventory []InformationItem, afterCandidates []string) PreservationReport {
	// Build a synthetic PreservedCandidateState from after inventory (fixture path).
	after := PreservedCandidateState{
		Inventory: afterInventory,
		PreservedAt: time.Now().UTC(),
	}
	for _, id := range afterCandidates {
		after.Candidates = append(after.Candidates, CandidateSnapshot{ID: id})
	}
	for _, it := range afterInventory {
		if it.IID.Kind == KindEvidence {
			after.EvidenceIndex = append(after.EvidenceIndex, EvidenceItem{
				Span: it.IID.Payload.Span, IIDDigest: it.IID.Digest, ContentRef: it.IID.Payload.Span,
			})
		}
	}
	return AuditPreservationIID(before, after)
}
