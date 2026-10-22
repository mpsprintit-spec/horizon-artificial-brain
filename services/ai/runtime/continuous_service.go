package runtime

import (
	"context"
	"errors"
	"time"
)

// ContinuousService drives the single BrainRuntime continuously. It owns no
// neural state of its own: every cycle is executed by BrainRuntime against the
// canonical Brain substrate.
//
// The service is deliberately transport-agnostic. HTTP, radio, Godot, or
// another adapter can feed observations into BrainRuntime independently while
// this loop supplies autonomous cognition when no external observation arrives.
type ContinuousService struct {
	Runtime *BrainRuntime
	// Supervisor, when set, resolves the active runtime afresh for every
	// cognitive cycle, so recovery cannot leave this loop on a stale brain.
	Supervisor *RuntimeSupervisor
	Interval time.Duration
	Cycles int
}

func NewContinuousService(runtime *BrainRuntime, interval time.Duration, cycles int) *ContinuousService {
	if interval <= 0 {
		interval = time.Second
	}
	if cycles < 1 {
		cycles = 1
	}
	return &ContinuousService{Runtime: runtime, Interval: interval, Cycles: cycles}
}

// NewManagedContinuousService performs each cognitive cycle through the
// supervisor's managed runtime lease. It never caches a BrainRuntime pointer.
func NewManagedContinuousService(supervisor *RuntimeSupervisor, interval time.Duration, cycles int) (*ContinuousService, error) {
	if supervisor == nil { return nil, errors.New("continuous brain supervisor is required") }
	if err := supervisor.With(func(*BrainRuntime) error { return nil }); err != nil { return nil, err }
	service := NewContinuousService(nil, interval, cycles)
	service.Supervisor = supervisor
	return service, nil
}

// Run blocks until ctx is cancelled. The first cognitive cycle runs
// immediately; subsequent cycles follow the configured interval.
func (s *ContinuousService) Run(ctx context.Context, onOutput func(CognitiveOutput, error)) error {
	if s == nil || (s.Runtime == nil && s.Supervisor == nil) {
		return errors.New("continuous brain service is not initialized")
	}
	if ctx == nil {
		return errors.New("continuous brain service requires a context")
	}
	if onOutput == nil {
		onOutput = func(CognitiveOutput, error) {}
	}

	run := func() error {
		var output CognitiveOutput
		var cycleErr error
		if s.Supervisor != nil {
			err := s.Supervisor.WithMutation(func(r *BrainRuntime) error {
				output, cycleErr = r.CognitiveThink(s.Cycles)
				return nil
			})
			if err != nil { cycleErr = err }
		} else {
			output, cycleErr = s.Runtime.CognitiveThink(s.Cycles)
		}
		// Do not invoke an external callback while the supervisor read lock
		// is held. It may legitimately request a recovery handover.
		onOutput(output, cycleErr)
		if errors.Is(cycleErr, ErrRuntimeSupervisorClosed) || errors.Is(cycleErr, ErrRuntimeRecoveryRequired) { return cycleErr }
		return nil
	}

	if err := run(); err != nil { return err }
	if err := ctx.Err(); err != nil {
		return err
	}

	ticker := time.NewTicker(s.Interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			if err := run(); err != nil { return err }
		}
	}
}
