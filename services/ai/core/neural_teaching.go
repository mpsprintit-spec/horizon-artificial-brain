package core

import (
	"bytes"
	"errors"
	"time"

	"github.com/project-horizon/horizon-core/services/ai/runtime"
	"github.com/project-horizon/horizon-core/services/ai/teaching"
)

// ExchangeHNL teaches or queries the active canonical Brain under its owner
// lease. It does not initialize a second Brain or authorize an external action.
func (h *HorizonEngine) ExchangeHNL(data []byte) ([]byte, error) {
	if h == nil || h.supervisor == nil {
		return nil, errors.New("horizon engine is unavailable")
	}
	if len(data) > teaching.MaxPacketBytes {
		return nil, errors.New("HNL packet exceeds resource bound")
	}
	frozen := bytes.Clone(data)
	packet, err := teaching.Unmarshal(frozen)
	if err != nil {
		return nil, err
	}
	if packet.Kind == teaching.Answer {
		return nil, errors.New("HNL answers cannot be submitted as lessons")
	}
	with := h.supervisor.With
	if packet.Kind == teaching.Teach {
		with = h.supervisor.WithMutation
	}
	var response []byte
	err = with(func(rt *runtime.BrainRuntime) error {
		var err error
		response, err = (teaching.Bridge{Runtime: rt}).Exchange(frozen)
		return err
	})
	return response, err
}

// ObserveCompositionInput publishes a typed numerical perception to the same
// managed continuation used by native thinking and checkpoint recovery.
func (h *HorizonEngine) ObserveCompositionInput(p runtime.CompositionPerception, at time.Time) (uint64, error) {
	if h == nil || h.supervisor == nil {
		return 0, errors.New("horizon engine is unavailable")
	}
	var sequence uint64
	err := h.supervisor.WithMutation(func(rt *runtime.BrainRuntime) error {
		var err error
		sequence, err = rt.ObserveCompositionInput(p, at)
		return err
	})
	return sequence, err
}
