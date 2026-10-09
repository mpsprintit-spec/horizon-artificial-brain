package runtime

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/project-horizon/horizon-core/services/ai/knowledge"
)

// KnowledgeProcessingResult reports the compilation and actual runtime
// processing of an imported document. It deliberately distinguishes storage
// from cognition: Compiled is the canonical representation; Outputs are the
// outputs returned by BrainRuntime for the grounded observations.
type KnowledgeProcessingResult struct {
	Compilation knowledge.KnowledgeCompilation
	Outputs []CognitiveOutput
}

// ProcessKnowledgeDocument compiles and imports a source-described document,
// grounds its exposures through the canonical observation path, and sends
// grounded node IDs through BrainRuntime. It does not assert that a single
// pass proves semantic understanding; evidence must be assessed separately.
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
	for i, experience := range compiled.Experiences {
		nodeIDs := make([]knowledge.NodeID, 0, len(experience.SymbolExposures))
		seen := make(map[knowledge.NodeID]bool)
		for _, exposure := range experience.SymbolExposures {
			surface := strings.TrimSpace(exposure.Symbol)
			if surface == "" {
				continue
			}
			grounded, err := r.brain.GroundObservationAt(surface, "knowledge-import:"+document.Source, exposure.Modality, GroundingThreshold, at)
			if err != nil {
				return result, fmt.Errorf("ground experience %q exposure %d: %w", experience.ID, exposure.SequencePosition, err)
			}
			for _, id := range grounded.Population {
				if id != 0 && !seen[id] {
					nodeIDs = append(nodeIDs, id)
					seen[id] = true
				}
			}
		}
		if len(nodeIDs) == 0 {
			continue
		}
		if err := r.LearnObservedTransition(nodeIDs, at); err != nil {
			return result, fmt.Errorf("learn transition for experience %q: %w", experience.ID, err)
		}
		eventID := fmt.Sprintf("knowledge-import:%s:%s", document.ID, experience.ID)
		output, err := r.CognitiveProcess(Event{
			ID: eventID,
			StimulusNodeIDs: nodeIDs,
			Source: "knowledge-import:" + document.Source,
			Modality: "knowledge",
			Cycles: 1,
			Timestamp: at.Add(time.Duration(i) * time.Nanosecond),
		})
		if err != nil {
			return result, fmt.Errorf("process experience %q: %w", experience.ID, err)
		}
		result.Outputs = append(result.Outputs, output)
	}
	return result, nil
}
