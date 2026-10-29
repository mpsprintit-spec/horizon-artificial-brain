package core

import (
	"errors"
	"github.com/project-horizon/horizon-core/services/ai/runtime"
)

// ReadGroundedDialogue is opt-in research I/O on the existing active owner.
// It creates no additional Brain and changes no ordinary Pulse behavior.
func (h *HorizonEngine) ReadGroundedDialogue(codec runtime.DialogueCodec, request runtime.GroundedDialogueRequest) (runtime.GroundedDialogueReadout, error) {
	var out runtime.GroundedDialogueReadout
	if h == nil || h.supervisor == nil {
		return out, errors.New("horizon engine unavailable")
	}
	err := h.supervisor.With(func(rt *runtime.BrainRuntime) error {
		var err error
		out, err = rt.ReadGroundedDialogue(codec, request)
		return err
	})
	return out, err
}
