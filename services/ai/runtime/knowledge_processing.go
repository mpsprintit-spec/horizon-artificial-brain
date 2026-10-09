package runtime

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/project-horizon/horizon-core/services/ai/knowledge"
)

// KnowledgeProcessingResult reports compilation and actual runtime processing.
// Compilation is the canonical representation; Outputs are per-exposure
// runtime outputs, preserving the source experience's temporal ordering.
type KnowledgeProcessingResult struct {
	Compilation knowledge.KnowledgeCompilation
	Outputs []CognitiveOutput
}

// ProcessKnowledgeDocument compiles and imports a source-described document,
// grounds each exposure through the canonical observation path, then processes
// exposures in sequence through BrainRuntime. This creates observable runtime
// events and adjacent-observation transitions; it does not prove understanding.
func (r *BrainRuntime) ProcessKnowledgeDocument(document knowledge.KnowledgeDocument, at time.Time) (KnowledgeProcessingResult, error) {
	if r == nil || r.brain == nil {
		return KnowledgeProcessingResult{}, errors.New("brain runtime is not initialized")
	}
	if at.IsZero() {
		r.mu.Lock()
		at = r.nowLocked()
		r.mu.Unlock()
	}
	at = at.UTC()
	compiled, err := knowledge.CompileKnowledgeDocument(document, at)
	if err != nil {
		return KnowledgeProcessingResult{}, err
	}
	if _, err := r.brain.ImportKnowledgeDocument(document, at); err != nil {
		return KnowledgeProcessingResult{}, err
	}

	result := KnowledgeProcessingResult{Compilation: compiled}
	step := 0
	for _, experience := range compiled.Experiences {
		// Learn the state-level before -> after transition through the existing
		// numeric population mechanism as well as processing individual surface
		// exposures below. These vectors are source-derived representations;
		// this step does not certify the transition as factually true.
		if len(experience.BeforeState.Representation) > 0 && len(experience.AfterState.Representation) > 0 {
			beforeState := knowledge.NewNeuralVector(experience.BeforeState.Representation)
			afterState := knowledge.NewNeuralVector(experience.AfterState.Representation)
			if err := r.brain.LearnVectorTransition(beforeState, afterState, 0.90, 4, at); err != nil {
				return result, fmt.Errorf("learn state transition for experience %q: %w", experience.ID, err)
			}
		}
		exposures := append([]knowledge.SymbolExposure(nil), experience.SymbolExposures...)
		// Compilation already emits before -> changes -> after. Stable sorting
		// protects this ordering if the canonical compiler later changes.
		for i := 1; i < len(exposures); i++ {
			for j := i; j > 0 && exposures[j].SequencePosition < exposures[j-1].SequencePosition; j-- {
				exposures[j], exposures[j-1] = exposures[j-1], exposures[j]
			}
		}
		for _, exposure := range exposures {
			surface := strings.TrimSpace(exposure.Symbol)
			if surface == "" {
				continue
			}
			eventAt := at.Add(time.Duration(step) * time.Nanosecond)
			step++
			grounded, err := r.brain.GroundObservationAt(surface, "knowledge-import:"+document.Source, exposure.Modality, GroundingThreshold, eventAt)
			if err != nil {
				return result, fmt.Errorf("ground experience %q exposure %d: %w", experience.ID, exposure.SequencePosition, err)
			}
			nodeIDs := make([]knowledge.NodeID, 0, len(grounded.Population))
			seen := make(map[knowledge.NodeID]bool)
			for _, id := range grounded.Population {
				if id != 0 && !seen[id] {
					nodeIDs = append(nodeIDs, id)
					seen[id] = true
				}
			}
			if len(nodeIDs) == 0 {
				continue
			}
			if err := r.LearnObservedTransition(nodeIDs, eventAt); err != nil {
				return result, fmt.Errorf("learn transition for experience %q exposure %d: %w", experience.ID, exposure.SequencePosition, err)
			}
			eventID := fmt.Sprintf("knowledge-import:%s:%s:%03d", document.ID, experience.ID, step-1)
			output, err := r.CognitiveProcess(Event{
				ID: eventID,
				StimulusNodeIDs: nodeIDs,
				Source: "knowledge-import:" + document.Source,
				Modality: exposure.Modality,
				Cycles: 1,
				Timestamp: eventAt,
			})
			if err != nil {
				return result, fmt.Errorf("process experience %q exposure %d: %w", experience.ID, exposure.SequencePosition, err)
			}
			result.Outputs = append(result.Outputs, output)
		}
	}
	return result, nil
}
