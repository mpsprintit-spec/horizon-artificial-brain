package runtime

import (
	"errors"
	"fmt"
	"sync"
)

// ErrRuntimeSupervisorClosed means the managed brain has been retired.
var ErrRuntimeSupervisorClosed = errors.New("runtime supervisor is closed")

// RuntimeSupervisor is the single publication boundary for a canonical
// BrainRuntime. All managed callers must access the runtime only inside With.
// The RWMutex is deliberately held throughout the callback: a handover waits
// for all active operations and prevents new ones from reaching the old brain.
//
// The callback must not retain or publish its *BrainRuntime argument*, its
// Brain pointer, or an asynchronous operation against it. Direct references
// acquired outside this supervisor are outside its safety guarantee.
type RuntimeSupervisor struct {
	mu sync.RWMutex
	current *BrainRuntime
	// epoch is closed when a managed runtime is replaced or retired. It
	// invalidates long-lived bridge subscriptions without holding With open.
	epoch chan struct{}
	closed bool
}

func NewRuntimeSupervisor(initial *BrainRuntime) (*RuntimeSupervisor, error) {
	if initial == nil || initial.brain == nil {
		return nil, errors.New("runtime supervisor requires a canonical brain runtime")
	}
	if initial.RecoveryRequired() {
		return nil, ErrRuntimeRecoveryRequired
	}
	return &RuntimeSupervisor{current: initial, epoch: make(chan struct{})}, nil
}

// With serializes the lifetime of a managed runtime operation against
// replacement. Do not call RecoverFromCheckpoint or Close from inside fn.
func (s *RuntimeSupervisor) With(fn func(*BrainRuntime) error) error {
	if fn == nil { return errors.New("runtime operation is nil") }
	return s.WithEpoch(func(r *BrainRuntime, _ <-chan struct{}) error { return fn(r) })
}

// WithEpoch also supplies an invalidation signal for long-lived read-only
// consumers, such as WebSocket telemetry streams. The signal closes on
// handover or supervisor shutdown. Do not retain the BrainRuntime argument or
// start background mutations outside the callback.
func (s *RuntimeSupervisor) WithEpoch(fn func(*BrainRuntime, <-chan struct{}) error) error {
	if s == nil { return errors.New("runtime supervisor is nil") }
	if fn == nil { return errors.New("runtime operation is nil") }
	s.mu.RLock()
	defer s.mu.RUnlock()
	if s.closed || s.current == nil { return ErrRuntimeSupervisorClosed }
	return fn(s.current, s.epoch)
}

// RecoverFromCheckpoint performs an exclusive, fail-closed handover.
//
// Once recovery starts, no new managed operation can enter and all earlier
// operations have returned. Rebuild validates the checkpoint and committed
// journal into a private runtime. If validation fails, the old runtime is
// left in place and no new instance is published.
//
// On success, the old runtime is retired (including any stray direct writer
// that calls a method honoring ensureWritableLocked), the journal is detached
// from the old instance, and the new runtime is published with the writer.
// The journal must not be shared with any other live runtime supervisor.
func (s *RuntimeSupervisor) RecoverFromCheckpoint(log *EventLog, checkpointPath string) error {
	if s == nil { return errors.New("runtime supervisor is nil") }
	if log == nil { return errors.New("event log is nil") }
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed || s.current == nil { return ErrRuntimeSupervisorClosed }

	old := s.current
	// Do not let a live runtime with a separate event writer advance the
	// canonical state while rebuilding from the supplied journal.
	old.mu.Lock()
	defer old.mu.Unlock()
	if old.eventLog != nil && old.eventLog != log {
		return errors.New("supervisor runtime is attached to a different event log")
	}
	// Hold old.mu for the entire rebuild. Even a direct runtime writer using
	// the same lock cannot advance its state while the journal is verified.
	// The lock order remains old runtime -> event log.
	recovered, err := RebuildRuntimeFromCheckpointAndJournal(log, checkpointPath)
	if err != nil { return fmt.Errorf("runtime handover aborted: %w", err) }
	if recovered == nil || recovered.RecoveryRequired() || recovered.eventLog != nil {
		return errors.New("recovery did not return a clean, detached runtime")
	}

	old.recoveryRequired = true
	old.eventLog = nil

	recovered.SetEventLog(log)
	s.current = recovered
	close(s.epoch)
	s.epoch = make(chan struct{})
	return nil
}

// Close retires the managed runtime. The supervisor cannot be reopened.
func (s *RuntimeSupervisor) Close() error {
	if s == nil { return errors.New("runtime supervisor is nil") }
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed { return nil }
	if s.current != nil {
		s.current.mu.Lock()
		s.current.recoveryRequired = true
		s.current.eventLog = nil
		s.current.mu.Unlock()
	}
	s.current = nil
	s.closed = true
	close(s.epoch)
	return nil
}
