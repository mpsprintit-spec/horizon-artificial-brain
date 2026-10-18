package runtime

import (
	"errors"
	"fmt"

	"github.com/project-horizon/horizon-core/services/ai/knowledge"
)

// RecoverFromCommittedJournal is the genesis-replay alternative when no
// authenticated checkpoint exists. It provides the same exclusive handover
// and stale-runtime retirement guarantees as checkpoint-offset recovery.
// The explicit factory must reproduce the original canonical bootstrap.
func (s *RuntimeSupervisor) RecoverFromCommittedJournal(log *EventLog, foundation func() *knowledge.Brain) error {
	if s == nil { return errors.New("runtime supervisor is nil") }
	if log == nil || foundation == nil { return errors.New("event journal and brain foundation are required") }
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed || s.current == nil { return ErrRuntimeSupervisorClosed }
	old := s.current
	old.mu.Lock()
	defer old.mu.Unlock()
	if old.eventLog != nil && old.eventLog != log {
		return errors.New("supervisor runtime is attached to a different event log")
	}
	// No direct writer honoring old.mu can alter old state or its journal
	// while verification and replay construct a private runtime.
	recovered, err := RebuildRuntimeFromCommittedJournal(log, foundation)
	if err != nil { return fmt.Errorf("runtime genesis handover aborted: %w",err) }
	if recovered == nil || recovered.RecoveryRequired() || recovered.eventLog != nil {
		return errors.New("genesis recovery did not return a clean detached runtime")
	}
	old.markRecoveryRequiredLocked()
	old.eventLog = nil
	recovered.SetEventLog(log)
	markSupervisorOwnedBrain(recovered.brain)
	s.current = recovered
	close(s.epoch)
	s.epoch = make(chan struct{})
	return nil
}
