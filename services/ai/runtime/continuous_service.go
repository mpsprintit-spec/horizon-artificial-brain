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

// Run blocks until ctx is cancelled. The first cognitive cycle runs
// immediately; subsequent cycles follow the configured interval.
func (s *ContinuousService) Run(ctx context.Context, onOutput func(CognitiveOutput, error)) error {
	if s == nil || s.Runtime == nil {
		return errors.New("continuous brain service is not initialized")
	}
	if ctx == nil {
		return errors.New("continuous brain service requires a context")
	}
	if onOutput == nil {
		onOutput = func(CognitiveOutput, error) {}
	}

	run := func() {
		output, err := s.Runtime.CognitiveThink(s.Cycles)
		onOutput(output, err)
	}

	run()
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
			run()
		}
	}
}
