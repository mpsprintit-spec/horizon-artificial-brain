// Package hfcc implements HFCC v1.0 research/representation layer.
// It is isolated from FUG, Decision, Learning, Pulse, and Knowledge schema mutation.
package hfcc

import "time"

// InferenceClass distinguishes how a claim was obtained (provenance).
type InferenceClass string

const (
	InferenceDirectObservation     InferenceClass = "DIRECT_OBSERVATION"
	InferenceStructural            InferenceClass = "STRUCTURAL_INFERENCE"
	InferenceConstructional        InferenceClass = "CONSTRUCTIONAL_INFERENCE"
	InferenceContextual            InferenceClass = "CONTEXTUAL_INFERENCE"
	InferenceExternalKnowledge     InferenceClass = "EXTERNAL_KNOWLEDGE"
)

// ValidationStatus is research metadata, not evidence and not absolute truth.
type ValidationStatus string

const (
	StatusRaw                 ValidationStatus = "RAW"
	StatusAnnotated           ValidationStatus = "ANNOTATED"
	StatusHypothesis          ValidationStatus = "HYPOTHESIS"
	StatusCounterexampleTested ValidationStatus = "COUNTEREXAMPLE_TESTED"
	StatusCrossChecked        ValidationStatus = "CROSS_CHECKED"
	StatusValidated           ValidationStatus = "VALIDATED"
	StatusGold                ValidationStatus = "GOLD"
	StatusRejected            ValidationStatus = "REJECTED"
)

// ReconstructionResult classifies reconstruction outcomes (not collapsed).
type ReconstructionResult string

const (
	Recontructable             ReconstructionResult = "RECONSTRUCTABLE"
	NotReconstructable         ReconstructionResult = "NOT_RECONSTRUCTABLE"
	ReconAmbiguous             ReconstructionResult = "AMBIGUOUS"
	ReconIncomplete            ReconstructionResult = "INCOMPLETE"
	ExternalKnowledgeRequired  ReconstructionResult = "EXTERNAL_KNOWLEDGE_REQUIRED"
	InformationRelocated       ReconstructionResult = "INFORMATION_RELOCATED"
)

// StructuralCompare is structure-only comparison outcome.
type StructuralCompare string

const (
	StructIdentical  StructuralCompare = "IDENTICAL"
	StructEquivalent StructuralCompare = "EQUIVALENT"
	StructPartial    StructuralCompare = "PARTIAL"
	StructDifferent  StructuralCompare = "DIFFERENT"
)

// ResearchCondition is orthogonal to pure structural identity.
type ResearchCondition string

const (
	CondNone            ResearchCondition = ""
	CondAmbiguous       ResearchCondition = "AMBIGUOUS"
	CondContextRequired ResearchCondition = "CONTEXT_REQUIRED"
	CondUnderSpecified  ResearchCondition = "UNDER_SPECIFIED"
	CondUndetermined    ResearchCondition = "UNDETERMINED"
)

// Observation is raw input evidence, not a semantic candidate.
type Observation struct {
	ID        string            `json:"id"`
	RawText   string            `json:"raw_text"`
	Surface   []string          `json:"surface,omitempty"` // tokens/surface evidence only
	Meta      map[string]string `json:"meta,omitempty"`
	Timestamp time.Time         `json:"timestamp"`
}

// Participant preserves identity; participation is optional annotation (open vocabulary).
type Participant struct {
	ID            string  `json:"id"`
	Ref           string  `json:"ref"`                      // handle within experiment, not global ontology
	Participation *string `json:"participation,omitempty"` // optional; never inferred from word order
}

// StructureNode is a recursive semantic structure (nesting required).
// Kind is extensible (not Horizon RelationKind ontology).
// Direction is conditional (nil = not applicable).
// Function is optional communicative function when present.
type StructureNode struct {
	Kind         string          `json:"kind"`
	Participants []Participant   `json:"participants,omitempty"`
	Children     []StructureNode `json:"children,omitempty"`
	Direction    *string         `json:"direction,omitempty"` // e.g. "forward" when semantically required
	Function     *string         `json:"function,omitempty"`  // e.g. ASSERT, EVALUATE when present
}

// ProvenanceRecord is one evidence channel supporting a claim.
type ProvenanceRecord struct {
	SourceRef      string         `json:"source_ref"`
	EvidenceRefs   []string       `json:"evidence_refs,omitempty"`
	InferenceClass InferenceClass `json:"inference_class"`
	DependentOn    []string       `json:"dependent_on,omitempty"` // claim IDs this evidence depends on
	CycleID        string         `json:"cycle_id,omitempty"`
	Timestamp      time.Time      `json:"timestamp,omitempty"`
	Note           string         `json:"note,omitempty"`
}

// Provenance aggregates multiple evidence sources (first-class).
type Provenance struct {
	Records []ProvenanceRecord `json:"records"`
}

// GoldAudit makes GOLD auditable (protocol + evidence + experiment).
type GoldAudit struct {
	ProtocolVersion string   `json:"protocol_version"`
	EvidenceSet     []string `json:"evidence_set"`
	ExperimentID    string   `json:"experiment_id"`
	ResultSummary   string   `json:"result_summary,omitempty"`
}

// Candidate is a hypothesis/interpretation snapshot (immutable by convention).
type Candidate struct {
	ID        string           `json:"id"`
	Version   string           `json:"version"`
	Structure StructureNode    `json:"structure"`
	Provenance Provenance      `json:"provenance"`
	Status    ValidationStatus `json:"status"`
	Gold      *GoldAudit       `json:"gold,omitempty"`
}

// CandidateSet retains multiple interpretations; ranking must not delete members.
type CandidateSet struct {
	ID            string      `json:"id"`
	ObservationID string      `json:"observation_id"`
	Candidates    []Candidate `json:"candidates"`
}

// ComparisonResult separates structural outcome from research conditions.
type ComparisonResult struct {
	Structural StructuralCompare  `json:"structural"`
	Condition  ResearchCondition  `json:"condition,omitempty"`
	Notes      []string           `json:"notes,omitempty"`
}

// AblationSpec selects which information channels to remove in a derived snapshot.
type AblationSpec struct {
	RemoveParticipation bool `json:"remove_participation"`
	RemoveDirection     bool `json:"remove_direction"`
	RemoveFunction      bool `json:"remove_function"`
	RemoveNesting       bool `json:"remove_nesting"` // flatten children into bag (lossy)
	RemoveProvenance    bool `json:"remove_provenance"`
	RemoveParticipantID bool `json:"remove_participant_id"` // strip identity refs
}

// AblationReport describes what happened; does not auto-conclude dimension redundancy.
type AblationReport struct {
	Retained              []string             `json:"retained,omitempty"`
	Lost                  []string             `json:"lost,omitempty"`
	Relocated             []string             `json:"relocated,omitempty"`
	AmbiguityIntroduced   bool                 `json:"ambiguity_introduced"`
	ExternalDependency    bool                 `json:"external_dependency"`
	Reconstruction        ReconstructionResult `json:"reconstruction,omitempty"`
	Notes                 []string             `json:"notes,omitempty"`
}

// ExperimentRecord supports audit/reproducibility of research runs.
type ExperimentRecord struct {
	ExperimentID         string                 `json:"experiment_id"`
	Input                string                 `json:"input,omitempty"`
	InputVersion         string                 `json:"input_version,omitempty"`
	CandidateVersion     string                 `json:"candidate_version,omitempty"`
	DimensionsRemoved    []string               `json:"dimensions_removed,omitempty"`
	Evidence             []string               `json:"evidence,omitempty"`
	ExternalDependencies []string               `json:"external_dependencies,omitempty"`
	ReconstructionResult ReconstructionResult   `json:"reconstruction_result,omitempty"`
	ComparisonResult     *ComparisonResult      `json:"comparison_result,omitempty"`
	ValidationResult     string                 `json:"validation_result,omitempty"`
	Timestamp            time.Time              `json:"timestamp"`
	Status               string                 `json:"status,omitempty"`
}
