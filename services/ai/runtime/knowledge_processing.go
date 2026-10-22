package runtime

import (
	"errors"
	"fmt"
	"encoding/json"
	"io"
	"strings"
	"time"

	"github.com/project-horizon/horizon-core/services/ai/knowledge"
	"github.com/project-horizon/horizon-core/services/ai/activation"
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
// ProcessKnowledgeDocument is one durable write-ahead revision when a journal
// is attached. The complete source document and event-time are fsynced before
// any neural import, projection or learning can begin. Ordered exposure steps
// all belong to that revision, rather than creating unlogged sequence gaps.
// On any append/processing error, the runtime is poisoned; recovery must
// reconstruct a new instance from a committed journal/checkpoint.
func (r *BrainRuntime) ProcessKnowledgeDocument(document knowledge.KnowledgeDocument, at time.Time) (KnowledgeProcessingResult, error) {
	if r == nil || r.brain == nil { return KnowledgeProcessingResult{}, errors.New("brain runtime is not initialized") }
	r.mu.Lock()
	defer r.mu.Unlock()
	if err := r.ensureWritableLocked(); err != nil { return KnowledgeProcessingResult{}, err }
	if at.IsZero() { at = r.nowLocked() }
	at = at.UTC()

	// Freeze caller-owned slices/maps before both validation and journal
	// encoding. Only the frozen version may affect the canonical neural state.
	raw, err := json.Marshal(document)
	if err != nil { return KnowledgeProcessingResult{}, fmt.Errorf("encode knowledge document: %w", err) }
	var frozen knowledge.KnowledgeDocument
	if err := json.Unmarshal(raw, &frozen); err != nil { return KnowledgeProcessingResult{}, fmt.Errorf("freeze knowledge document: %w", err) }
	compiled, err := knowledge.CompileKnowledgeDocument(frozen, at)
	if err != nil { return KnowledgeProcessingResult{}, err }
	if r.eventLog == nil {
		// Standalone legacy imports preserve per-exposure sequence semantics.
		return r.processKnowledgeDocumentLocked(frozen, compiled, at, false)
	}
	nextSeq := r.seq + 1
	if err := r.eventLog.Append(LoggedEvent{
		SchemaVersion: EventLogSchemaVersion,
		BrainIdentity: BrainIdentity,
		Sequence: nextSeq,
		Type: EventTypeKnowledgeImport,
		Timestamp: at,
		KnowledgeDocument: &frozen,
	}); err != nil {
		// Append might have partially persisted a record. Never assume the
		// journal is clean or let this in-memory runtime accept more writes.
		r.markRecoveryRequiredLocked()
		return KnowledgeProcessingResult{}, fmt.Errorf("%w: append knowledge import: %v", ErrRuntimeRecoveryRequired, err)
	}
	r.seq = nextSeq
	result, err := r.processKnowledgeDocumentLocked(frozen, compiled, at, true)
	if err != nil {
		r.markRecoveryRequiredLocked()
		return result, fmt.Errorf("%w: apply committed knowledge import: %v", ErrRuntimeRecoveryRequired, err)
	}
	return result, nil
}

// replayKnowledgeDocument applies one logged import revision to an isolated
// runtime, without appending to the original log. It is invoked only after
// ReadEventLog has verified the committed record and strict sequence order.
func (r *BrainRuntime) replayKnowledgeDocument(document knowledge.KnowledgeDocument, at time.Time) error {
	if r == nil || r.brain == nil { return errors.New("brain runtime is not initialized") }
	r.mu.Lock()
	defer r.mu.Unlock()
	if err := r.ensureWritableLocked(); err != nil { return err }
	if r.eventLog != nil { return errors.New("knowledge import replay requires detached journal") }
	if at.IsZero() { return errors.New("knowledge import replay requires a timestamp") }
	compiled, err := knowledge.CompileKnowledgeDocument(document, at)
	if err != nil { return err }
	r.seq++
	if _, err := r.processKnowledgeDocumentLocked(document, compiled, at.UTC(), true); err != nil {
		r.markRecoveryRequiredLocked()
		return fmt.Errorf("%w: replay knowledge import: %v", ErrRuntimeRecoveryRequired, err)
	}
	return nil
}

// processKnowledgeDocumentLocked assumes r.mu. A journaled import has exactly
// one sequence/revision, even when there are several cognition microsteps.
func (r *BrainRuntime) processKnowledgeDocumentLocked(document knowledge.KnowledgeDocument, compiled knowledge.KnowledgeCompilation, at time.Time, singleRevision bool) (KnowledgeProcessingResult, error) {
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
			if err := r.learnObservedTransitionLocked(nodeIDs, eventAt); err != nil {
				return result, fmt.Errorf("learn transition for experience %q exposure %d: %w", experience.ID, exposure.SequencePosition, err)
			}
			eventID := fmt.Sprintf("knowledge-import:%s:%s:%03d", document.ID, experience.ID, step-1)
			event := Event{
				ID: eventID,
				StimulusNodeIDs: nodeIDs,
				Source: "knowledge-import:" + document.Source,
				Modality: exposure.Modality,
				Cycles: 1,
				Timestamp: eventAt,
			}
			var output CognitiveOutput
			if singleRevision {
				// In an atomic document revision, all microsteps operate on the
				// same sequence without writing separate process records.
				output, err = r.cognitiveKnowledgeExposureLocked(event)
			} else {
				output, err = r.cognitiveProcessLocked(event)
			}
			if err != nil {
				return result, fmt.Errorf("process experience %q exposure %d: %w", experience.ID, exposure.SequencePosition, err)
			}
			result.Outputs = append(result.Outputs, output)
		}
	}
	return result, nil
}

 
// cognitiveKnowledgeExposureLocked runs the same activation, prediction,
// learning-signal and memory-dynamics stages as normal CognitiveProcess,
// without creating a second journal revision or unlogged process sequence.
// Only call this with r.mu held by a committed knowledge-import transaction.
func (r *BrainRuntime) cognitiveKnowledgeExposureLocked(event Event) (CognitiveOutput, error) {
	if err := r.ensureWritableLocked(); err != nil { return CognitiveOutput{}, err }
	result := r.activation.ActivateWith(activation.Request{
		StimulusNodeIDs: append([]knowledge.NodeID(nil), event.StimulusNodeIDs...),
		Cycles: event.Cycles,
		Now: event.Timestamp,
	})
	output := cognitiveOutputFromResult(BrainIdentity, r.seq, event.Timestamp, result)
	output.Prediction = r.activation.PredictionSnapshot()
	r.brain.RecordLearningSignal(output.PredictionError, event.Timestamp)
	r.brain.ApplyMemoryDynamics(event.Timestamp)
	output.StateDelta = output.State().Diff(r.lastCognitiveState)
	r.lastCognitiveState = output.State()
	return output, nil
}


// ProcessKnowledgeFrameCorpus decodes a structured knowledge corpus and sends
// each record through the canonical runtime pipeline. It returns partial
// results alongside an error if processing stops partway through the corpus.
func (r *BrainRuntime) ProcessKnowledgeFrameCorpus(reader io.Reader, at time.Time) ([]KnowledgeProcessingResult, error) {
	corpus, err := knowledge.DecodeKnowledgeFrameCorpus(reader)
	if err != nil {
		return nil, err
	}
	documents, err := corpus.KnowledgeDocuments()
	if err != nil {
		return nil, err
	}
	results := make([]KnowledgeProcessingResult, 0, len(documents))
	for _, document := range documents {
		result, err := r.ProcessKnowledgeDocument(document, at)
		if err != nil {
			return results, fmt.Errorf("process knowledge frame %q: %w", document.ID, err)
		}
		results = append(results, result)
	}
	return results, nil
}
