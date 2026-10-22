package core

import (
	"errors"

	"github.com/project-horizon/horizon-core/services/ai/runtime"
)

// InspectActiveBrain is the canonical read-only inspection entry point.
// Snapshot data is detached: it has no live neural pointers, and is valid
// after a checkpoint handover. Managed snapshots capture exactly one
// runtime revision under the supervisor read lease.
func (h *HorizonEngine) InspectActiveBrain() (runtime.BrainInspection,error) {
	if h==nil {return runtime.BrainInspection{},errors.New("horizon engine is unavailable")}
	var snapshot runtime.BrainInspection
	var err error
	if h.Supervisor!=nil {
		err=h.Supervisor.With(func(active *runtime.BrainRuntime)error {
			snapshot,err=active.InspectBrain()
			return err
		})
	} else {
		if h.Runtime==nil {return runtime.BrainInspection{},errors.New("brain runtime is unavailable")}
		snapshot,err=h.Runtime.InspectBrain()
	}
	return snapshot,err
}
