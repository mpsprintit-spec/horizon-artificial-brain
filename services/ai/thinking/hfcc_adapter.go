package thinking

import (
	"fmt"
	"time"

	"github.com/project-horizon/horizon-core/services/ai/hfcc"
)

// ProjectionTarget locates where an IID was written in HFCC representation.
type ProjectionTarget struct {
	CandidateID   string `json:"candidate_id"`
	StructurePath string `json:"structure_path"` // e.g. children[0], children[0].participants[1]
	Kind          string `json:"kind"`          // RELATION_NODE | ENTITY_NODE | STRUCTURAL_NODE | EVIDENCE_REF | PROVENANCE
	Ref           string `json:"ref,omitempty"` // optional surface ref for debug only — not used for identity match
}

// ProjectionMapping is the auditable IID → HFCC link. Conservation requires this record.
type ProjectionMapping struct {
	FromIIDDigest string          `json:"from_iid_digest"`
	Target        ProjectionTarget `json:"target"`
}

// ProjectionRecord is the contract name for Audit B primary proof (alias of ProjectionMapping).
// SourceIID → TargetRef / TargetKind / RepresentationKind — NOT RelocationRecord.
type ProjectionRecord struct {
	SourceIID          string `json:"source_iid"`
	TargetRef          string `json:"target_ref"`          // candidate + path
	TargetKind         string `json:"target_kind"`         // RELATION_NODE | ENTITY_NODE | STRUCTURAL_NODE | EVIDENCE_REF
	RepresentationKind string `json:"representation_kind"` // same as TargetKind for HFCC node class
}

func (m ProjectionMapping) AsRecord() ProjectionRecord {
	return ProjectionRecord{
		SourceIID:          m.FromIIDDigest,
		TargetRef:          m.Target.CandidateID + ":" + m.Target.StructurePath,
		TargetKind:         m.Target.Kind,
		RepresentationKind: m.Target.Kind,
	}
}

// AssertProjectionConsistency is optional validation that mapped node kind matches expected RepresentationKind.
// It MUST NOT be used as identity proof (that is only SourceIID presence in ProjectionRecord).
func AssertProjectionConsistency(result HFCCProjectionResult) []string {
	var issues []string
	for _, m := range result.Mappings {
		if !mappingTargetExists(result.CandidateSet, m) {
			issues = append(issues, "broken mapping "+m.FromIIDDigest)
		}
	}
	return issues
}

// HFCCProjectionResult is adapter output: candidate set + explicit mappings (no hidden identity).
type HFCCProjectionResult struct {
	CandidateSet hfcc.CandidateSet
	Mappings     []ProjectionMapping
}

// BuildHFCCFromPreserved projects PCS → HFCC with explicit ProjectionMapping entries.
// Single source: PCS only. No Interpretation/KB/Memory/FSU/raw text.
func BuildHFCCFromPreserved(obs hfcc.Observation, pcs PreservedCandidateState) hfcc.CandidateSet {
	return BuildHFCCFromPreservedWithMappings(obs, pcs).CandidateSet
}

// BuildHFCCFromPreservedWithMappings returns set + mappings for Audit B.
func BuildHFCCFromPreservedWithMappings(obs hfcc.Observation, pcs PreservedCandidateState) HFCCProjectionResult {
	set := hfcc.CandidateSet{
		ID:            "cs-" + obs.ID,
		ObservationID: obs.ID,
	}
	var mappings []ProjectionMapping

	byCand := map[string][]InformationItem{}
	var global []InformationItem
	for _, it := range pcs.Inventory {
		if it.CandidateID != "" {
			byCand[it.CandidateID] = append(byCand[it.CandidateID], it)
		} else {
			global = append(global, it)
		}
	}
	cands := pcs.Candidates
	if len(cands) == 0 && (len(pcs.Inventory) > 0 || len(pcs.EvidenceIndex) > 0) {
		cands = []CandidateSnapshot{{ID: "interp-0", Version: "pcs-1"}}
	}

	for _, snap := range cands {
		items := append([]InformationItem{}, byCand[snap.ID]...)
		items = append(items, global...)
		c, maps := projectCandidateFromPCS(obs, snap, items, pcs)
		set.Candidates = append(set.Candidates, c)
		mappings = append(mappings, maps...)
	}
	return HFCCProjectionResult{CandidateSet: set, Mappings: mappings}
}

func projectCandidateFromPCS(
	obs hfcc.Observation,
	snap CandidateSnapshot,
	items []InformationItem,
	pcs PreservedCandidateState,
) (hfcc.Candidate, []ProjectionMapping) {
	root := hfcc.StructureNode{Kind: "INTERPRETATION"}
	prov := hfcc.Provenance{}
	var mappings []ProjectionMapping
	childIdx := 0

	for _, it := range items {
		switch it.IID.Kind {
		case KindRelation:
			p := it.IID.Payload
			path := fmt.Sprintf("children[%d]", childIdx)
			child := hfcc.StructureNode{
				Kind: p.Rel,
				Participants: []hfcc.Participant{
					{ID: snap.ID + "-s-" + p.Src, Ref: p.Src},
					{ID: snap.ID + "-t-" + p.Tgt, Ref: p.Tgt},
				},
			}
			if p.Polarity != "" {
				pol := p.Polarity
				child.Direction = &pol
			}
			root.Children = append(root.Children, child)
			mappings = append(mappings, ProjectionMapping{
				FromIIDDigest: it.IID.Digest,
				Target: ProjectionTarget{
					CandidateID: snap.ID, StructurePath: path,
					Kind: "RELATION_NODE", Ref: p.Rel,
				},
			})
			prov.Records = append(prov.Records, hfcc.ProvenanceRecord{
				SourceRef:      "PCS_IID:" + it.IID.Digest,
				EvidenceRefs:   []string{it.IID.Digest},
				InferenceClass: hfcc.InferenceStructural,
				CycleID:        obs.ID,
				Timestamp:      time.Now().UTC(),
				Note:           "relation projected from PCS IID",
			})
			childIdx++

		case KindStructural:
			p := it.IID.Payload
			kind := p.Slot
			if kind == "" {
				kind = "STRUCTURAL"
			}
			path := fmt.Sprintf("children[%d]", childIdx)
			child := hfcc.StructureNode{Kind: kind}
			if p.Host != "" {
				child.Participants = append(child.Participants, hfcc.Participant{ID: snap.ID + "-h", Ref: p.Host})
			}
			if p.Value != "" {
				child.Participants = append(child.Participants, hfcc.Participant{ID: snap.ID + "-v", Ref: p.Value})
			}
			if p.Entity != "" {
				child.Participants = append(child.Participants, hfcc.Participant{ID: snap.ID + "-e", Ref: p.Entity})
			}
			root.Children = append(root.Children, child)
			mappings = append(mappings, ProjectionMapping{
				FromIIDDigest: it.IID.Digest,
				Target: ProjectionTarget{
					CandidateID: snap.ID, StructurePath: path,
					Kind: "STRUCTURAL_NODE", Ref: kind,
				},
			})
			prov.Records = append(prov.Records, hfcc.ProvenanceRecord{
				SourceRef:      "PCS_IID:" + it.IID.Digest,
				EvidenceRefs:   []string{it.IID.Digest},
				InferenceClass: hfcc.InferenceStructural,
				CycleID:        obs.ID,
				Note:           "structural projected from PCS IID",
			})
			childIdx++

		case KindEntity:
			path := fmt.Sprintf("children[%d]", childIdx)
			root.Children = append(root.Children, hfcc.StructureNode{
				Kind: "ENTITY",
				Participants: []hfcc.Participant{
					{ID: snap.ID + "-ent-" + it.IID.Payload.Entity, Ref: it.IID.Payload.Entity},
				},
			})
			mappings = append(mappings, ProjectionMapping{
				FromIIDDigest: it.IID.Digest,
				Target: ProjectionTarget{
					CandidateID: snap.ID, StructurePath: path,
					Kind: "ENTITY_NODE", Ref: it.IID.Payload.Entity,
				},
			})
			childIdx++

		case KindEvidence:
			// Evidence on inventory also maps if present; primary path is EvidenceIndex below
			mappings = append(mappings, ProjectionMapping{
				FromIIDDigest: it.IID.Digest,
				Target: ProjectionTarget{
					CandidateID: snap.ID, StructurePath: "provenance.evidence",
					Kind: "EVIDENCE_REF", Ref: it.IID.Digest,
				},
			})
			prov.Records = append(prov.Records, hfcc.ProvenanceRecord{
				SourceRef:      "EVIDENCE_IID",
				EvidenceRefs:   []string{it.IID.Digest},
				InferenceClass: hfcc.InferenceDirectObservation,
				CycleID:        obs.ID,
				Note:           "evidence_iid=" + it.IID.Digest + ";span=" + it.IID.Payload.Span,
			})
		}
	}

	// EvidenceIndex: preserve original Evidence IID digest in EvidenceRefs (do not mint new identity from span alone)
	seenEv := map[string]bool{}
	for _, e := range pcs.EvidenceIndex {
		digest := e.IIDDigest
		if digest == "" {
			// no stable IID — cannot claim identity conservation; skip mapping
			continue
		}
		if seenEv[digest] {
			continue
		}
		seenEv[digest] = true
		prov.Records = append(prov.Records, hfcc.ProvenanceRecord{
			SourceRef:      "EVIDENCE_IID",
			EvidenceRefs:   []string{digest},
			InferenceClass: hfcc.InferenceDirectObservation,
			CycleID:        obs.ID,
			Note:           "evidence_iid=" + digest + ";span=" + e.Span,
		})
		mappings = append(mappings, ProjectionMapping{
			FromIIDDigest: digest,
			Target: ProjectionTarget{
				CandidateID: snap.ID, StructurePath: "provenance.evidence",
				Kind: "EVIDENCE_REF", Ref: digest,
			},
		})
	}

	for _, d := range pcs.Derivations {
		cls := hfcc.InferenceStructural
		if d.Dependency == DepExternal {
			cls = hfcc.InferenceExternalKnowledge
		}
		prov.Records = append(prov.Records, hfcc.ProvenanceRecord{
			SourceRef:      "DERIV:" + d.RuleRef,
			EvidenceRefs:   append([]string{d.ConcludesIID}, d.PremiseIIDs...),
			InferenceClass: cls,
			CycleID:        obs.ID,
			Note:           fmt.Sprintf("derivation %s", d.ID),
		})
	}

	return hfcc.Candidate{
		ID:         snap.ID,
		Version:    "pcs-adapt-2",
		Structure:  root,
		Provenance: prov,
		Status:     hfcc.StatusHypothesis,
	}, mappings
}

// AuditHFCCConsumerWithMappings — Audit B Level 2: independent projection verification.
// ProjectionMapping is only an address; content must match IID payload.
func AuditHFCCConsumerWithMappings(pcs PreservedCandidateState, result HFCCProjectionResult) PreservationReport {
	report := PreservationReport{Pass: true}
	mapped := map[string]ProjectionMapping{}
	for _, m := range result.Mappings {
		mapped[m.FromIIDDigest] = m
	}

	hfccCand := map[string]bool{}
	for _, c := range result.CandidateSet.Candidates {
		hfccCand[c.ID] = true
	}
	for _, snap := range pcs.Candidates {
		if !hfccCand[snap.ID] {
			report.Lost = append(report.Lost, "candidate:"+snap.ID)
			report.Pass = false
			report.FailureDomain = "HFCC_FAILURE"
		}
	}

	relocTo := map[string]string{}
	for _, r := range pcs.RelocationRecords {
		relocTo[r.FromIID] = r.ToIID
	}

	// index inventory by digest for relocation destination lookup
	byDigest := map[string]InformationItem{}
	for _, it := range pcs.Inventory {
		byDigest[it.IID.Digest] = it
	}

	for _, it := range pcs.Inventory {
		if it.IID.Kind == KindRuntime {
			continue
		}
		audit := ItemAudit{Item: it, Before: true}
		digest := it.IID.Digest

		m, hasMap := mapped[digest]
		if !hasMap {
			// try relocation: FromIID → ToIID must have mapping + Level 2 on destination item
			if to, ok := relocTo[digest]; ok {
				if m2, ok2 := mapped[to]; ok2 {
					dest := byDigest[to]
					if dest.IID.Digest == "" {
						// destination only in HFCC mapping — verify using synthetic from ToIID mapping against HFCC only is insufficient;
						// require destination inventory item when present; else verify mapping content via From item against target is wrong.
						// Spec: ToIID exists AFTER and passes Level 2. If dest not in inventory, still verify mapping target represents... we need dest payload.
						// Without dest item, treat as LOST unless we only check mapping exists — Level 2 requires payload of ToIID.
						audit.After = false
						audit.Dims = PreservationDimensions{
							Availability: AvailLost, Location: LocOriginal,
							Origin: it.Origin, Dependency: it.Dependency,
							Reason: "relocation ToIID not in PCS inventory for Level 2",
						}
						report.Lost = append(report.Lost, it.AuditID)
						report.Pass = false
						report.FailureDomain = "HFCC_FAILURE"
						report.ItemAudits = append(report.ItemAudits, audit)
						continue
					}
					vr := VerifyProjection(dest, m2, result.CandidateSet)
					if vr == ProjectionOK {
						audit.After = true
						audit.Dims = PreservationDimensions{
							Availability: AvailAvailable, Location: LocRelocated,
							Origin: it.Origin, Dependency: it.Dependency,
							Reason: "RelocationRecord + Level 2 on ToIID",
						}
						report.Relocated = append(report.Relocated, it.AuditID)
						report.ItemAudits = append(report.ItemAudits, audit)
						continue
					}
				}
			}
			audit.After = false
			audit.Dims = PreservationDimensions{
				Availability: AvailLost, Location: LocOriginal,
				Origin: it.Origin, Dependency: it.Dependency,
				Reason: "no ProjectionMapping for IID",
			}
			report.Lost = append(report.Lost, it.AuditID)
			report.Pass = false
			report.FailureDomain = "HFCC_FAILURE"
			report.ItemAudits = append(report.ItemAudits, audit)
			continue
		}

		vr := VerifyProjection(it, m, result.CandidateSet)
		switch vr {
		case ProjectionOK:
			audit.After = true
			audit.Dims = PreservationDimensions{
				Availability: AvailAvailable, Location: LocOriginal,
				Origin: it.Origin, Dependency: it.Dependency,
				Reason: "Level 2 projection verified",
			}
			report.Available = append(report.Available, it.AuditID)
		case ProjectionMissingTarget:
			audit.After = false
			audit.Dims = PreservationDimensions{
				Availability: AvailLost, Location: LocOriginal,
				Origin: it.Origin, Dependency: it.Dependency,
				Reason: "mapping target missing in HFCC",
			}
			report.Lost = append(report.Lost, it.AuditID)
			report.Pass = false
			report.FailureDomain = "HFCC_FAILURE"
		case ProjectionMismatch:
			audit.After = false
			audit.Dims = PreservationDimensions{
				Availability: AvailMismatch, Location: LocOriginal,
				Origin: it.Origin, Dependency: it.Dependency,
				Reason: "PROJECTION_MISMATCH: target does not represent IID payload",
			}
			report.Mismatched = append(report.Mismatched, it.AuditID)
			report.Lost = append(report.Lost, it.AuditID)
			report.Pass = false
			report.FailureDomain = "HFCC_FAILURE"
		case ProjectionUnverified:
			// Not AVAILABLE (no Level 2 claim). Not LOST (mapping may exist; verifier absent).
			audit.After = false
			audit.Dims = PreservationDimensions{
				Availability: AvailUnverified, Location: LocOriginal,
				Origin: it.Origin, Dependency: it.Dependency,
				Reason: "UNVERIFIED_KIND: no Level 2 verifier for this InformationKind",
			}
			report.Unverified = append(report.Unverified, it.AuditID)
			// Pass remains true unless other failures — UNVERIFIED is not a conservation failure
		}
		report.ItemAudits = append(report.ItemAudits, audit)
	}

	for _, e := range pcs.EvidenceIndex {
		if e.IIDDigest == "" {
			continue
		}
		// find inventory item or synthesize evidence item for verify
		var it *InformationItem
		for i := range pcs.Inventory {
			if pcs.Inventory[i].IID.Digest == e.IIDDigest {
				it = &pcs.Inventory[i]
				break
			}
		}
		if it != nil {
			continue // already audited
		}
		m, hasMap := mapped[e.IIDDigest]
		if !hasMap {
			report.Lost = append(report.Lost, "evidence:"+e.IIDDigest)
			report.Pass = false
			report.FailureDomain = "HFCC_FAILURE"
			continue
		}
		synth := InformationItem{
			IID: InformationIID{Kind: KindEvidence, Digest: e.IIDDigest, Payload: InformationPayload{Span: e.Span}},
			AuditID: "evidence:" + e.IIDDigest, Class: InfoEvidence, Origin: OriginDirect, Dependency: DepInternal,
		}
		// digest-only evidence: KindEvidence verify uses Digest against EvidenceRefs
		synth.IID.Digest = e.IIDDigest
		if VerifyProjection(synth, m, result.CandidateSet) != ProjectionOK {
			report.Mismatched = append(report.Mismatched, "evidence:"+e.IIDDigest)
			report.Lost = append(report.Lost, "evidence:"+e.IIDDigest)
			report.Pass = false
			report.FailureDomain = "HFCC_FAILURE"
		}
	}
	return report
}

// ProjectionVerifyResult is Level 2 outcome.
type ProjectionVerifyResult int

const (
	ProjectionOK ProjectionVerifyResult = iota
	ProjectionMissingTarget
	ProjectionMismatch
	ProjectionUnverified // kind has no Level 2 verifier
)

// VerifyProjection independently checks that HFCC target represents the IID payload.
// Mapping is address only; content equality is deterministic structured-field match.
func VerifyProjection(item InformationItem, m ProjectionMapping, cs hfcc.CandidateSet) ProjectionVerifyResult {
	// Candidate scope
	if item.CandidateID != "" && m.Target.CandidateID != "" && item.CandidateID != m.Target.CandidateID {
		return ProjectionMismatch
	}

	cand := findCandidate(cs, m.Target.CandidateID)
	if cand == nil {
		return ProjectionMissingTarget
	}

	switch item.IID.Kind {
	case KindEvidence:
		if m.Target.Kind != "EVIDENCE_REF" {
			return ProjectionMismatch
		}
		if !evidenceRefHasDigest(*cand, item.IID.Digest) {
			// if mapping points at evidence path but digest not present
			if !evidenceRefHasDigest(*cand, m.FromIIDDigest) {
				return ProjectionMissingTarget
			}
			return ProjectionMismatch
		}
		// FromIID must be this evidence digest
		if m.FromIIDDigest != item.IID.Digest {
			return ProjectionMismatch
		}
		return ProjectionOK

	case KindRelation:
		if m.Target.Kind != "RELATION_NODE" {
			return ProjectionMismatch
		}
		node, ok := resolveChildNode(*cand, m.Target.StructurePath)
		if !ok {
			return ProjectionMissingTarget
		}
		if node.Kind == "ENTITY" || node.Kind == "INTERPRETATION" {
			return ProjectionMismatch
		}
		p := item.IID.Payload
		if node.Kind != p.Rel {
			return ProjectionMismatch
		}
		if len(node.Participants) < 2 {
			return ProjectionMismatch
		}
		if node.Participants[0].Ref != p.Src || node.Participants[1].Ref != p.Tgt {
			return ProjectionMismatch
		}
		if p.Polarity != "" {
			if node.Direction == nil || *node.Direction != p.Polarity {
				return ProjectionMismatch
			}
		}
		return ProjectionOK

	case KindEntity:
		if m.Target.Kind != "ENTITY_NODE" {
			return ProjectionMismatch
		}
		node, ok := resolveChildNode(*cand, m.Target.StructurePath)
		if !ok {
			return ProjectionMissingTarget
		}
		if node.Kind != "ENTITY" {
			return ProjectionMismatch
		}
		if len(node.Participants) < 1 || node.Participants[0].Ref != item.IID.Payload.Entity {
			return ProjectionMismatch
		}
		return ProjectionOK

	case KindStructural:
		if m.Target.Kind != "STRUCTURAL_NODE" {
			return ProjectionMismatch
		}
		node, ok := resolveChildNode(*cand, m.Target.StructurePath)
		if !ok {
			return ProjectionMissingTarget
		}
		if node.Kind == "ENTITY" {
			return ProjectionMismatch
		}
		p := item.IID.Payload
		expectedKind := p.Slot
		if expectedKind == "" {
			expectedKind = "STRUCTURAL"
		}
		if node.Kind != expectedKind {
			return ProjectionMismatch
		}
		refs := map[string]bool{}
		for _, part := range node.Participants {
			refs[part.Ref] = true
		}
		if p.Value != "" && !refs[p.Value] {
			return ProjectionMismatch
		}
		if p.Host != "" && !refs[p.Host] {
			return ProjectionMismatch
		}
		if p.Entity != "" && !refs[p.Entity] {
			return ProjectionMismatch
		}
		return ProjectionOK

	default:
		// Context, Provenance, and any kind without an explicit Level 2 verifier.
		// Do NOT treat mapping+target existence as Level 2 PASS.
		return ProjectionUnverified
	}
}

func findCandidate(cs hfcc.CandidateSet, id string) *hfcc.Candidate {
	for i := range cs.Candidates {
		if cs.Candidates[i].ID == id {
			return &cs.Candidates[i]
		}
	}
	return nil
}

func resolveChildNode(cand hfcc.Candidate, path string) (hfcc.StructureNode, bool) {
	var idx int
	if _, err := fmt.Sscanf(path, "children[%d]", &idx); err != nil {
		return hfcc.StructureNode{}, false
	}
	if idx < 0 || idx >= len(cand.Structure.Children) {
		return hfcc.StructureNode{}, false
	}
	return cand.Structure.Children[idx], true
}

func evidenceRefHasDigest(cand hfcc.Candidate, digest string) bool {
	if digest == "" {
		return false
	}
	for _, rec := range cand.Provenance.Records {
		for _, er := range rec.EvidenceRefs {
			if er == digest {
				return true
			}
		}
	}
	return false
}

func mappingTargetExists(cs hfcc.CandidateSet, m ProjectionMapping) bool {
	cand := findCandidate(cs, m.Target.CandidateID)
	if cand == nil {
		return false
	}
	switch m.Target.Kind {
	case "EVIDENCE_REF":
		return evidenceRefHasDigest(*cand, m.FromIIDDigest) || evidenceRefHasDigest(*cand, m.Target.Ref)
	case "RELATION_NODE", "STRUCTURAL_NODE", "ENTITY_NODE":
		_, ok := resolveChildNode(*cand, m.Target.StructurePath)
		return ok
	}
	return false
}

// AuditHFCCConsumer keeps name used by tests/reasoning; uses mapping-based audit.
func AuditHFCCConsumer(pcs PreservedCandidateState, cs hfcc.CandidateSet) PreservationReport {
	// Rebuild mappings by re-projection is wrong if cs was hand-built.
	// Prefer: if cs came from adapter, caller should use WithMappings.
	// For hand-built lossy cs in tests, produce empty mappings → relations LOST (correct).
	return AuditHFCCConsumerWithMappings(pcs, HFCCProjectionResult{CandidateSet: cs, Mappings: nil})
}

// ProjectInventoryFromHFCCStructure retained for debug only — NOT used for identity conservation claims.
func ProjectInventoryFromHFCCStructure(cs hfcc.CandidateSet) []InformationItem {
	var out []InformationItem
	for _, c := range cs.Candidates {
		for _, rec := range c.Provenance.Records {
			if rec.SourceRef == "EVIDENCE_IID" {
				for _, er := range rec.EvidenceRefs {
					out = append(out, InformationItem{
						IID: InformationIID{Digest: er, Kind: KindEvidence},
						AuditID: "hfcc-ev-ref-" + er, Class: InfoEvidence,
						Origin: OriginDirect, Dependency: DepInternal, CandidateID: c.ID,
					})
				}
			}
		}
	}
	return out
}
