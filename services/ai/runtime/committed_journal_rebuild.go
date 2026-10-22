package runtime

import (
	"errors"
	"fmt"

	"github.com/project-horizon/horizon-core/services/ai/knowledge"
)

// RebuildRuntimeFromCommittedJournal reconstructs a NEW, private runtime by
// replaying the verified, fully committed event stream from its beginning.
//
// A failed or poisoned runtime is deliberately not reused. The returned
// runtime is detached from its journal so the caller can atomically replace
// the serving runtime before calling SetEventLog. The old runtime must be
// retired and no longer allowed to accept writes before this handover.
//
// Recovery starts at sequence 1: the bootstrap factory must reproduce the
// exact original foundation. Checkpoint-offset replay needs separate,
// versioned temporal state and is not supported by this function.
//
// Call RecoverTrailingSensorIntent first only when an orphan intent has been
// verified; this function NEVER silently discards incomplete transactions.
func RebuildRuntimeFromCommittedJournal(log *EventLog, newFoundation func() *knowledge.Brain) (*BrainRuntime, error) {
	if log == nil {
		return nil, errors.New("event log is not initialized")
	}
	if newFoundation == nil {
		return nil, errors.New("recovery requires an explicit canonical brain foundation factory")
	}

	// Retain the exclusive writer lock, and prevent Append from changing the
	// journal between validation and completed replay.
	log.mu.Lock()
	defer log.mu.Unlock()
	if log.file == nil || log.lockFile == nil {
		return nil, errors.New("event log writer is not open and locked")
	}
	events, err := ReadEventLog(log.file.Name())
	if err != nil {
		return nil, fmt.Errorf("verify committed journal: %w", err)
	}

	brain := newFoundation()
	if brain == nil {
		return nil, errors.New("canonical recovery foundation is nil")
	}
	fresh := NewBrainRuntime(brain)
	if fresh == nil {
		return nil, errors.New("could not initialize recovered brain runtime")
	}
	if len(events) > 0 && events[0].DynamicsEncoding.HasCanonicalPropagation() {
		if err := validateCanonicalPropagationFoundation(brain); err != nil {
			return nil, err
		}
		fresh.dynamicsEncoding = knowledge.CanonicalPropagationDynamics
	}
	// fresh has not escaped to other goroutines and has no eventLog; replay
	// cannot interleave with its own live mutations or write new log entries.
	if err := ReplayEventLog(fresh, events); err != nil {
		return nil, fmt.Errorf("replay committed journal into isolated runtime: %w", err)
	}
	if fresh.LastSequence() != uint64(len(events)) {
		return nil, fmt.Errorf("recovery sequence mismatch: got %d want %d", fresh.LastSequence(), len(events))
	}
	if fresh.RecoveryRequired() {
		return nil, ErrRuntimeRecoveryRequired
	}
	return fresh, nil
}
