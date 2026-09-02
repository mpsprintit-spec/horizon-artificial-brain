package thinking

import (
	"fmt"
	"time"

	"github.com/project-horizon/horizon-core/services/ai/hfcc"
	"github.com/project-horizon/horizon-core/services/ai/knowledge"
)

// InterpretationCandidateID maps Interpretation index to HFCC candidate ID.
func InterpretationCandidateID(i int) string {
	return fmt.Sprintf("interp-%d", i)
}

// BuildHFCCCandidateSet adapts existing Horizon interpretations into HFCC candidates.
// Does NOT invent semantic roles from argument position or word order.
// Participation remains nil unless explicitly present in Horizon evidence (currently not forced).
// Nesting is only represented when Horizon actually has nested structure (flat relations stay flat).
func BuildHFCCCandidateSet(
	observation hfcc.Observation,
	kb *knowledge.KnowledgeBase,
	interps []Interpretation,
) hfcc.CandidateSet {
	set := hfcc.CandidateSet{
		ID:            "cs-" + observation.ID,
		ObservationID: observation.ID,
	}
	for i, interp := range interps {
		c := interpretationToCandidate(kb, observation, i, interp)
		set.Candidates = append(set.Candidates, c)
	}
	return set
}

func interpretationToCandidate(
	kb *knowledge.KnowledgeBase,
	obs hfcc.Observation,
	idx int,
	interp Interpretation,
) hfcc.Candidate {
	cid := InterpretationCandidateID(idx)
	root := hfcc.StructureNode{Kind: "INTERPRETATION"}
	// Flat honesty: each relation is a child structure node with participants (identity only).
	for _, r := range interp.Relations {
		srcTok, tgtTok := tokenRef(kb, r.SourceID), tokenRef(kb, r.TargetID)
		child := hfcc.StructureNode{
			Kind: string(r.Kind),
			Participants: []hfcc.Participant{
				{ID: fmt.Sprintf("%s-s-%d", cid, r.SourceID), Ref: srcTok},
				{ID: fmt.Sprintf("%s-t-%d", cid, r.TargetID), Ref: tgtTok},
			},
		}
		if r.Inhibitory {
			pol := "inhibitory"
			child.Direction = &pol // polarity marker only when present — not universal direction ontology
		}
		root.Children = append(root.Children, child)
	}
	// Propositions as additional structure (requested vs background left to provenance)
	for _, p := range interp.Propositions {
		child := hfcc.StructureNode{
			Kind: string(p.Relation),
			Participants: []hfcc.Participant{
				{ID: fmt.Sprintf("%s-pt-%s", cid, p.ID), Ref: p.TargetTok},
				{ID: fmt.Sprintf("%s-po-%s", cid, p.ID), Ref: p.ObjectTok},
			},
		}
		root.Children = append(root.Children, child)
	}
	// Optional function from Eval/FSU signals without token dictionary
	var fn *string
	for _, s := range interp.FunctionalSignals {
		if s.Kind == "focus_typed_structure" || s.Kind == "multi_path_support" {
			continue
		}
		// structural note only — leave function unset unless goal-like signal exists
		_ = s
	}
	if interp.EvalStatus != "" {
		// communicative stance not equated to token "apakah"
	}
	_ = fn

	prov := hfcc.Provenance{}
	// Relations from interpretation structure → structural inference unless marked observation
	for _, r := range interp.Relations {
		src := "MEMORY"
		cls := hfcc.InferenceStructural
		if r.Provenance == "direct" || r.Provenance == "activation" {
			cls = hfcc.InferenceDirectObservation
			src = "OBSERVATION_OR_ACTIVATION"
		}
		prov.Records = append(prov.Records, hfcc.ProvenanceRecord{
			SourceRef:      src,
			EvidenceRefs:   []string{fmt.Sprintf("%d-%s-%d", r.SourceID, r.Kind, r.TargetID)},
			InferenceClass: cls,
			CycleID:        obs.ID,
			Timestamp:      time.Now().UTC(),
			Note:           "adapted from Horizon Interpretation; not automatic truth",
		})
	}
	for _, p := range interp.Propositions {
		cls := hfcc.InferenceStructural
		if p.Requested {
			cls = hfcc.InferenceConstructional
		}
		prov.Records = append(prov.Records, hfcc.ProvenanceRecord{
			SourceRef:      "PROPOSITION",
			EvidenceRefs:   []string{p.ID},
			InferenceClass: cls,
			CycleID:        obs.ID,
			Note:           "requested=" + fmt.Sprintf("%v", p.Requested),
		})
	}

	return hfcc.Candidate{
		ID:         cid,
		Version:    "h2-adapt-1",
		Structure:  root,
		Provenance: prov,
		Status:     hfcc.StatusHypothesis, // never auto-GOLD
	}
}

func tokenRef(kb *knowledge.KnowledgeBase, id knowledge.NodeID) string {
	if kb == nil {
		return fmt.Sprintf("#%d", id)
	}
	if n := kb.Registry.GetByID(id); n != nil {
		return n.Token
	}
	return fmt.Sprintf("#%d", id)
}
