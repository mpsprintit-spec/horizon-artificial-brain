package runtime

import (
	"errors"
	"slices"
	"strings"
)

func cloneActionBinding(binding ActionBinding) ActionBinding {
	binding.NumericPrediction=cloneNumericPrediction(binding.NumericPrediction)
	binding.TargetNodeIDs = append(binding.TargetNodeIDs[:0:0], binding.TargetNodeIDs...)
	binding.Synapses = append(binding.Synapses[:0:0], binding.Synapses...)
	return binding
}

func cloneActionBindings(bindings map[string]ActionBinding) map[string]ActionBinding {
	out := make(map[string]ActionBinding, len(bindings))
	for id, binding := range bindings {
		out[id] = cloneActionBinding(binding)
	}
	return out
}

func (r *BrainRuntime) validateActionBindingLocked(binding ActionBinding) (ActionBinding, error) {
	if err:=binding.NumericPrediction.validate();err!=nil{return ActionBinding{},err}
	if strings.TrimSpace(binding.RequestID) == "" {
		return ActionBinding{}, errors.New("action binding request ID is required")
	}
	if binding.BrainIdentity == "" {
		binding.BrainIdentity = BrainIdentity
	}
	if binding.BrainIdentity != BrainIdentity {
		return ActionBinding{}, errors.New("action binding belongs to a different brain")
	}
	if len(binding.TargetNodeIDs) == 0 && len(binding.Synapses) == 0 {
		return ActionBinding{}, errors.New("action binding requires at least one neural target")
	}
	for _, id := range binding.TargetNodeIDs {
		if r.brain.Registry.GetByID(id) == nil {
			return ActionBinding{}, errors.New("action binding references an unknown target node")
		}
	}
	for _, syn := range binding.Synapses {
		if r.brain.Registry.GetByID(syn.SourceNodeID) == nil || r.brain.Registry.GetByID(syn.TargetNodeID) == nil {
			return ActionBinding{}, errors.New("action binding references an unknown synapse node")
		}
	}
	return cloneActionBinding(binding), nil
}

func (r *BrainRuntime) registerActionBindingLocked(binding ActionBinding) error {
	if err := r.ensureWritableLocked(); err != nil {
		return err
	}
	binding, err := r.validateActionBindingLocked(binding)
	if err != nil {
		return err
	}
	if previous, exists := r.actions[binding.RequestID]; exists {
		if previous.RequestID == binding.RequestID && previous.BrainIdentity == binding.BrainIdentity && previous.Intent == binding.Intent && sameNumericPrediction(previous.NumericPrediction,binding.NumericPrediction) && slices.Equal(previous.TargetNodeIDs, binding.TargetNodeIDs) && slices.Equal(previous.Synapses, binding.Synapses) {
			return nil
		}
		return errors.New("action binding request ID already has different targets")
	}
	err = r.runJournalCommandLocked(LoggedEvent{Type: EventTypeActionBinding, Timestamp: r.nowLocked(), ActionBinding: &binding}, func() error { r.actions[binding.RequestID] = binding; r.seq++; return nil })
	if err == nil && r.telemetry != nil {
		r.publishTelemetryLocked(EventTypeActionBinding, r.nowLocked(), nil, nil, nil, nil, nil)
	}
	return err
}
