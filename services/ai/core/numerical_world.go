package core

import (
	"errors"
	"github.com/project-horizon/horizon-core/services/ai/knowledge"
	"github.com/project-horizon/horizon-core/services/ai/runtime"
	"time"
)

func (h *HorizonEngine) LearnNumericalTransition(l runtime.NumericalTransitionLesson, at time.Time) (uint64, *runtime.CompositionLearningReceipt, error) {
	if h == nil || h.supervisor == nil {
		return 0, nil, errors.New("engine unavailable")
	}
	var seq uint64
	var receipt *runtime.CompositionLearningReceipt
	e := h.supervisor.WithMutation(func(r *runtime.BrainRuntime) error {
		var e error
		seq, receipt, e = r.LearnNumericalTransition(l, at)
		return e
	})
	return seq, receipt, e
}
func (h *HorizonEngine) ImagineNumericalTrajectory(req runtime.NumericalImaginationRequest) (runtime.NumericalTrajectory, error) {
	if h == nil || h.supervisor == nil {
		return runtime.NumericalTrajectory{}, errors.New("engine unavailable")
	}
	var out runtime.NumericalTrajectory
	e := h.supervisor.With(func(r *runtime.BrainRuntime) error { var e error; out, e = r.ImagineNumericalTrajectory(req); return e })
	return out, e
}
func (h *HorizonEngine) CompositionDiagnosis() (*knowledge.NDiagnosis, error) {
	if h == nil || h.supervisor == nil {
		return nil, errors.New("engine unavailable")
	}
	var out *knowledge.NDiagnosis
	e := h.supervisor.With(func(r *runtime.BrainRuntime) error { var e error; out, e = r.CompositionDiagnosis(); return e })
	return out, e
}
