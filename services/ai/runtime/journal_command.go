package runtime

import (
	"encoding/json"
	"fmt"
)

// runJournalCommandLocked records a complete deterministic input BEFORE its
// first mutation, then applies exactly one revision under r.mu. A durable
// command is a redo commitment: recovery reapplies the entire transition even
// if the live caller lost its acknowledgment. It is not a post-state snapshot.
func (r *BrainRuntime) runJournalCommandLocked(command LoggedEvent, apply func() error) error {
	if err := r.ensureWritableLocked(); err != nil {
		return err
	}
	command.SchemaVersion, command.BrainIdentity, command.Sequence = EventLogSchemaVersion, BrainIdentity, r.seq+1
	command.SurfaceEncoding = r.surfaceEncoding
	command.DynamicsEncoding = r.dynamicsEncoding
	r.decorateCompositionJournalLocked(&command)
	if command.Timestamp.IsZero() {
		command.Timestamp = r.nowLocked()
	}
	command.Timestamp = command.Timestamp.UTC()
	if err := validateJournalPayload(command); err != nil {
		return err
	}
	// Encoding also rejects NaN, invalid RawMessage and other unreplayable
	// caller data before any append or mutation.
	encoded, err := json.Marshal(command)
	if err != nil {
		return fmt.Errorf("encode runtime command: %w", err)
	}
	if len(encoded) >= 16<<20 {
		return fmt.Errorf("runtime command exceeds journal record limit")
	}
	if r.eventLog != nil {
		if err := r.eventLog.Append(command); err != nil {
			r.markRecoveryRequiredLocked()
			return fmt.Errorf("%w: append %s: %v", ErrRuntimeRecoveryRequired, command.Type, err)
		}
	}
	completed := false
	defer func() {
		if !completed {
			r.markRecoveryRequiredLocked()
		}
	}()
	switch command.Type {
	case EventTypeActionBinding, EventTypeSurfaceRebinding, EventTypeWrittenGroundingMigration, EventTypeOutcome:
		// Metadata-only changes do not advance neural time. Candidate outcomes
		// settle retention only if their evidence actually promotes a synapse.
	default:
		r.settleMemoryTimeLocked(command.Timestamp)
	}
	if err := apply(); err != nil {
		return fmt.Errorf("%w: apply %s: %v", ErrRuntimeRecoveryRequired, command.Type, err)
	}
	if r.seq != command.Sequence {
		return fmt.Errorf("%w: %s revision mismatch", ErrRuntimeRecoveryRequired, command.Type)
	}
	completed = true
	return nil
}

func processedJournalEvent(logged LoggedEvent) *Event {
	if logged.Type == EventTypeCompositionPerception && logged.CompositionPerception != nil {
		return &Event{ID: logged.CompositionPerception.ID}
	}
	if logged.Type == EventTypeCompositionExposure && logged.CompositionExposure != nil {
		return &Event{ID: logged.CompositionExposure.ID}
	}
	if logged.Type == EventTypeProcess || logged.Type == EventTypeCognitiveProcess || logged.Type == EventTypeObservation || logged.Type == EventTypeInquiryOutcome {
		return logged.Event
	}
	return nil
}
