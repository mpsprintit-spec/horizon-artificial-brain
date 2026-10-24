package runtime

import (
	"errors"
	"math"
	"slices"
	"strings"
	"time"

	"github.com/project-horizon/horizon-core/services/ai/knowledge"
)

// NumericMotorCommand is an external actuator coordinate contract. ActuatorID
// identifies a configured I/O adapter, never a word or internal semantic node.
type NumericMotorCommand struct {
	ActuatorID string    `json:"actuator_id"`
	Values     []float64 `json:"values"`
}

// This is frozen operational causal evidence, not a learned answer store.
// It is captured before dispatch and authenticated with the action binding.
type NumericActionPrediction struct {
	ReferenceRevision uint64    `json:"reference_revision"`
	SceneAt           time.Time `json:"scene_at"`
	Motor             []float64 `json:"motor"`
	LocalState        float64   `json:"local_state"`
	GoalState         float64   `json:"goal_state"`
}

func (p *NumericActionPrediction) validate() error {
	if p == nil {
		return nil
	}
	if p.SceneAt.IsZero() || p.ReferenceRevision == 0 || math.IsNaN(p.LocalState) || math.IsInf(p.LocalState, 0) || math.Abs(p.LocalState) > 16 || math.Abs(p.GoalState) != 1 {
		return errors.New("invalid frozen numerical prediction")
	}
	return (&NumericMotorCommand{ActuatorID: "numeric-coordinate-contract", Values: p.Motor}).Validate()
}

func cloneNumericPrediction(p *NumericActionPrediction) *NumericActionPrediction {
	if p == nil {
		return nil
	}
	out := *p
	out.Motor = slices.Clone(p.Motor)
	return &out
}
func sameNumericPrediction(a, b *NumericActionPrediction) bool {
	if a == nil || b == nil {
		return a == b
	}
	return a.ReferenceRevision == b.ReferenceRevision && a.SceneAt.Equal(b.SceneAt) && a.LocalState == b.LocalState && a.GoalState == b.GoalState && slices.Equal(a.Motor, b.Motor)
}

func (m *NumericMotorCommand) Validate() error {
	if m == nil || strings.TrimSpace(m.ActuatorID) == "" || len(m.ActuatorID) > 256 || len(m.Values) != 4 {
		return errors.New("invalid numerical motor command")
	}
	for _, v := range m.Values {
		if math.IsNaN(v) || math.IsInf(v, 0) || v < -1 || v > 1 {
			return errors.New("invalid motor coordinates")
		}
	}
	if m.Values[3] != 1 || math.Abs(m.Values[2]) != 1 {
		return errors.New("numeric action requires an explicit nonzero-force proposal")
	}
	return nil
}

func cloneNumericMotor(m *NumericMotorCommand) *NumericMotorCommand {
	if m == nil {
		return nil
	}
	return &NumericMotorCommand{ActuatorID: m.ActuatorID, Values: slices.Clone(m.Values)}
}

// ValidateCompositionAction checks the committed current neural/physical
// revision. It is only a preflight; authorization remains in SafetyBoundary.
func (r *BrainRuntime) ValidateCompositionAction(proposal knowledge.CompositionReadout, at time.Time) error {
	if r == nil || r.brain == nil {
		return errors.New("runtime is unavailable")
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.validateCompositionActionLocked(proposal, at)
}

func (r *BrainRuntime) validateCompositionActionLocked(p knowledge.CompositionReadout, at time.Time) error {
	if err := r.ensureWritableLocked(); err != nil {
		return err
	}
	if r.composition == nil || r.composition.Scene == nil || !p.Accepted || p.Status != "proposal" || p.BrainIdentity != BrainIdentity || p.Sequence != r.seq || !p.SceneAt.Equal(r.composition.Scene.ObservedAt) {
		return errors.New("composition action has no current committed reference")
	}
	if at.IsZero() {
		at = r.nowLocked()
	}
	current, err := r.compositionReadoutLocked(at)
	if err != nil {
		return err
	}
	if current == nil || !current.Accepted || current.Status != "proposal" || !slices.Equal(p.Motor, current.Motor) || !slices.Equal(p.Goal, current.Goal) {
		return errors.New("composition action disagrees with current observed/neural state")
	}
	return nil
}

// BindCompositionAction is called after authorization and before external
// dispatch, inside the same owner mutation lease. The journal links subsequent
// actual outcome evidence to the neural readout; replay never moves a device.
func (r *BrainRuntime) BindCompositionAction(p knowledge.CompositionReadout, requestID string, at time.Time) error {
	if r == nil || r.brain == nil {
		return errors.New("runtime is unavailable")
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if err := r.validateCompositionActionLocked(p, at); err != nil {
		return err
	}
	activation, err := r.brain.CompositionActivation(p)
	if err != nil {
		return err
	}
	ids := make([]knowledge.NodeID, 0, len(activation))
	for id := range activation {
		ids = append(ids, id)
	}
	slices.Sort(ids)
	prediction := &NumericActionPrediction{ReferenceRevision: p.Sequence, SceneAt: p.SceneAt, Motor: slices.Clone(p.Motor), LocalState: p.LocalEffect, GoalState: p.Goal[3]}
	return r.registerActionBindingLocked(ActionBinding{RequestID: requestID, BrainIdentity: BrainIdentity, Intent: "numeric-motor", TargetNodeIDs: ids, NumericPrediction: prediction})
}
