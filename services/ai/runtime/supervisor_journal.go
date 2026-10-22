package runtime

import "errors"

// AttachEventLog connects the initial exclusive writer before the first
// revision. Continuing a nonempty history requires verified recovery instead.
// This method takes the owner lease itself; do not call it from a With callback.
func (s *RuntimeSupervisor) AttachEventLog(log *EventLog) error {
	if s == nil || log == nil {
		return errors.New("supervisor and event journal are required")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed || s.current == nil {
		return ErrRuntimeSupervisorClosed
	}
	r := s.current
	r.mu.Lock()
	defer r.mu.Unlock()
	if err := r.ensureWritableLocked(); err != nil {
		return err
	}
	if r.eventLog == log {
		return nil
	}
	if r.eventLog != nil || r.seq != 0 {
		return errors.New("journal replacement or nonzero sequence requires verified recovery")
	}
	log.mu.Lock()
	defer log.mu.Unlock()
	if log.file == nil || log.lockFile == nil {
		return errors.New("journal writer is not open and locked")
	}
	events, err := ReadEventLog(log.file.Name())
	if err != nil {
		return err
	}
	if len(events) != 0 {
		return errors.New("nonempty journal requires verified recovery")
	}
	r.eventLog = log
	return nil
}
