package runtime

import (
	"errors"
	"fmt"
	"github.com/project-horizon/horizon-core/services/ai/knowledge"
	"math"
	"time"
)

const MaxNumericalRolloutSteps = 16

// Domain is an explicit sensor/action codec, not a fitted transition law.
type NumericalDomain struct {
	ID                uint16 `json:"id"`
	StateDimensions   int    `json:"state_dimensions"`
	ActionDimensions  int    `json:"action_dimensions"`
	ContextDimensions int    `json:"context_dimensions"`
	StepNS            int64  `json:"step_ns"`
}

func (d NumericalDomain) Validate() error {
	if d.ID == 0 || d.StateDimensions < 1 || d.StateDimensions > 6 || d.ActionDimensions < 0 || d.ActionDimensions > 6 || d.ContextDimensions < 0 || d.ContextDimensions > 6 || d.StepNS <= 0 || d.StepNS > int64(time.Hour) {
		return errors.New("invalid bounded numerical domain")
	}
	return nil
}

type NumericalFrame struct {
	Hypothetical     bool      `json:"hypothetical,omitempty"`
	Values           []float64 `json:"values"`
	MeasurementError []float64 `json:"measurement_error,omitempty"`
	At               time.Time `json:"at"`
}
type NumericalTransition struct {
	Before    NumericalFrame `json:"before"`
	After     NumericalFrame `json:"after"`
	Action    []float64      `json:"action"`
	Context   []float64      `json:"context,omitempty"`
	EpisodeID string         `json:"episode_id"`
}
type NumericalTransitionLesson struct {
	ID                string                `json:"id"`
	Domain            NumericalDomain       `json:"domain"`
	Output            int                   `json:"output"`
	RequiredPrecision *float64              `json:"required_precision,omitempty"`
	Fit               []NumericalTransition `json:"fit"`
	Check             []NumericalTransition `json:"check"`
}

func boundedNumerical(v []float64, n int) bool {
	if len(v) != n {
		return false
	}
	for _, x := range v {
		if math.IsNaN(x) || math.IsInf(x, 0) || math.Abs(x) > 1 {
			return false
		}
	}
	return true
}
func validateNumericalFrame(f NumericalFrame, n int) error {
	if f.At.IsZero() || !boundedNumerical(f.Values, n) {
		return errors.New("invalid numerical frame")
	}
	if len(f.MeasurementError) > 0 {
		if len(f.MeasurementError) != n {
			return errors.New("frame error dimensions disagree")
		}
		for _, x := range f.MeasurementError {
			if math.IsNaN(x) || math.IsInf(x, 0) || x < 0 || x > 2 {
				return errors.New("invalid frame measurement error")
			}
		}
	}
	return nil
}
func numericalScene(at time.Time) knowledge.CompositionScene {
	return knowledge.CompositionScene{ObservedAt: at, Point: []float64{0, 0, 1}, Patches: []knowledge.CompositionPatch{{Values: []float64{0, 0, 0, -1, 0}, XY: []float64{0, 0}}}}
}
func numericalTransitionQuery(d NumericalDomain, output int) []float64 {
	v := make([]float64, 16)
	v[0] = -.875
	v[1] = float64(d.ID) / 65536
	v[2] = float64(output) / 8
	v[3] = float64(d.StateDimensions) / 8
	v[4] = float64(d.ActionDimensions) / 8
	v[5] = float64(d.ContextDimensions) / 8
	v[6] = float64(d.StepNS) / float64(time.Hour)
	v[7] = -.625
	return v
}
func numericalTransitionInput(d NumericalDomain, output int, before NumericalFrame, action, context []float64, tolerance *float64) knowledge.CompositionInput {
	// Temporal order is timestep, context, commanded action, observed state.
	seq := make([][]float64, 4)
	for i := range seq {
		seq[i] = make([]float64, 7)
	}
	seq[0][0] = float64(d.StepNS) / float64(time.Hour)
	seq[0][6] = 1
	if len(context) > 0 {
		copy(seq[1], context)
		seq[1][6] = 1
	}
	if d.ActionDimensions > 0 {
		copy(seq[2], action)
		seq[2][6] = 1
	}
	copy(seq[3], before.Values)
	seq[3][6] = 1
	in := knowledge.CompositionInput{Words: [][]float64{numericalTransitionQuery(d, output)}, Sequence: seq, Scene: numericalScene(before.At)}
	if tolerance != nil {
		v := *tolerance
		in.RequiredPrecision = &v
	}
	if len(before.MeasurementError) > 0 {
		in.MeasurementError = make([][]float64, 4)
		for i := range in.MeasurementError {
			in.MeasurementError[i] = make([]float64, 6)
		}
		copy(in.MeasurementError[3], before.MeasurementError)
	}
	return in
}

// CompileNumericalTransitionTarget creates one atomic native learning command.
// Coordinates are taught independently; a caller receives a durable receipt for
// each target rather than an ambiguous partially committed vector update.
func CompileNumericalTransitionTarget(l NumericalTransitionLesson, at time.Time) (knowledge.CompositionExposure, error) {
	var e knowledge.CompositionExposure
	if err := l.Domain.Validate(); err != nil {
		return e, err
	}
	if l.ID == "" || len(l.ID) > 128 || l.Output < 0 || l.Output >= l.Domain.StateDimensions || len(l.Fit) < 4 || len(l.Check) < 4 || len(l.Fit) > 64 || len(l.Check) > 64 || at.IsZero() {
		return e, errors.New("invalid transition lesson target or budget")
	}
	batch := &knowledge.NResolutionLesson{}
	for set, rows := range [][]NumericalTransition{l.Fit, l.Check} {
		for _, r := range rows {
			if r.Before.Hypothetical || r.After.Hypothetical {
				return e, errors.New("imagined states cannot be taught as observed outcomes")
			}
			if err := validateNumericalFrame(r.Before, l.Domain.StateDimensions); err != nil {
				return e, err
			}
			if err := validateNumericalFrame(r.After, l.Domain.StateDimensions); err != nil {
				return e, err
			}
			if !boundedNumerical(r.Action, l.Domain.ActionDimensions) || (len(r.Context) > 0 && !boundedNumerical(r.Context, l.Domain.ContextDimensions)) || r.EpisodeID == "" || len(r.EpisodeID) > 128 || r.After.At.Sub(r.Before.At) != time.Duration(l.Domain.StepNS) || r.After.At.After(at) {
				return e, errors.New("invalid, future or unaligned transition evidence")
			}
			in := numericalTransitionInput(l.Domain, l.Output, r.Before, r.Action, r.Context, l.RequiredPrecision)
			outcomeError := 0.
			if len(r.After.MeasurementError) > 0 {
				outcomeError = r.After.MeasurementError[l.Output]
			}
			o := knowledge.NObservation{OutcomeError: outcomeError, Input: in, Outcome: r.After.Values[l.Output], ObservedAt: r.After.At, EpisodeID: r.EpisodeID}
			if set == 0 {
				batch.Fit = append(batch.Fit, o)
			} else {
				batch.Check = append(batch.Check, o)
			}
		}
	}
	e = knowledge.CompositionExposure{ID: fmt.Sprintf("transition:%s:%d", l.ID, l.Output), Source: "observed-numerical-transition", EpisodeID: l.ID, IndependenceGroup: l.ID, CausalGroup: "observed-before-action-after", Reliability: 1, Input: batch.Fit[0].Input, Resolution: batch, Motor: []float64{0, 0, 0, 0}, After: numericalScene(at)}
	return e, e.Validate()
}
func (r *BrainRuntime) LearnNumericalTransition(l NumericalTransitionLesson, at time.Time) (uint64, *CompositionLearningReceipt, error) {
	if r == nil {
		return 0, nil, errors.New("runtime unavailable")
	}
	c, e := r.CompositionStatus()
	if e != nil {
		return 0, nil, e
	}
	if c == nil || c.Config.Version != knowledge.NeuralCompositionMetricV6 {
		return 0, nil, errors.New("numeric transitions require explicit v6")
	}
	if at.IsZero() {
		return 0, nil, errors.New("transition evidence requires an explicit trusted observation time")
	}
	exp, e := CompileNumericalTransitionTarget(l, at)
	if e != nil {
		return 0, nil, e
	}
	return r.LearnCompositionWithDiagnosis(exp, at)
}

type NumericalImaginationRequest struct {
	Domain     NumericalDomain `json:"domain"`
	Start      NumericalFrame  `json:"start"`
	Actions    [][]float64     `json:"actions"`
	Context    []float64       `json:"context,omitempty"`
	Tolerances []float64       `json:"tolerances"`
}
type NumericalImaginedStep struct {
	State    NumericalFrame                 `json:"state"`
	Readouts []knowledge.CompositionReadout `json:"readouts"`
}
type NumericalTrajectory struct {
	BrainIdentity string                  `json:"brain_identity"`
	Sequence      uint64                  `json:"sequence"`
	Hypothetical  bool                    `json:"hypothetical"`
	Steps         []NumericalImaginedStep `json:"steps"`
	StopReason    string                  `json:"stop_reason"`
}

func (req NumericalImaginationRequest) validate() error {
	if err := req.Domain.Validate(); err != nil {
		return err
	}
	if err := validateNumericalFrame(req.Start, req.Domain.StateDimensions); err != nil {
		return err
	}
	if len(req.Actions) < 1 || len(req.Actions) > MaxNumericalRolloutSteps || len(req.Tolerances) != req.Domain.StateDimensions || (len(req.Context) > 0 && !boundedNumerical(req.Context, req.Domain.ContextDimensions)) {
		return errors.New("invalid imagination budget, context or tolerance")
	}
	for _, row := range req.Actions {
		if !boundedNumerical(row, req.Domain.ActionDimensions) {
			return errors.New("invalid imagined action")
		}
	}
	for _, x := range req.Tolerances {
		if math.IsNaN(x) || math.IsInf(x, 0) || x <= 0 || x > 2 {
			return errors.New("invalid requested state precision")
		}
	}
	if req.Start.At.After(req.Start.At.Add(time.Duration(len(req.Actions)) * time.Duration(req.Domain.StepNS))) {
		return errors.New("imagination timestamp overflow")
	}
	return nil
}

// ImagineNumericalTrajectory is a detached native rollout. No teacher after-state,
// checkpoint mutation, observation, learning, motor command or hardcoded plant is
// used. Predicted values and their error envelopes become the next input.
func (r *BrainRuntime) ImagineNumericalTrajectory(req NumericalImaginationRequest) (NumericalTrajectory, error) {
	out := NumericalTrajectory{BrainIdentity: BrainIdentity, Hypothetical: true, StopReason: "untrained-transition"}
	if r == nil || r.brain == nil {
		return out, errors.New("runtime unavailable")
	}
	if e := req.validate(); e != nil {
		return out, e
	}
	f, e := freezeComposition(req)
	if e != nil {
		return out, e
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if e = r.ensureWritableLocked(); e != nil {
		return out, e
	}
	if r.composition == nil || r.composition.Config.Version != knowledge.NeuralCompositionMetricV6 {
		return out, errors.New("imagination requires explicit v6")
	}
	out.Sequence = r.seq
	current := f.Start
	for _, action := range f.Actions {
		next := NumericalFrame{Hypothetical: true, Values: make([]float64, f.Domain.StateDimensions), MeasurementError: make([]float64, f.Domain.StateDimensions), At: current.At.Add(time.Duration(f.Domain.StepNS))}
		step := NumericalImaginedStep{}
		for j := 0; j < f.Domain.StateDimensions; j++ {
			in := numericalTransitionInput(f.Domain, j, current, action, f.Context, &f.Tolerances[j])
			read, e := r.brain.ReadComposition(r.composition.Config, in, in.Scene.ObservedAt)
			if e != nil {
				return out, e
			}
			read.Sequence = r.seq
			read.BrainIdentity = BrainIdentity
			step.Readouts = append(step.Readouts, read)
			if read.NResolution == nil || read.Status == "untrained-n-resolution" {
				out.StopReason = "untrained-transition"
				return out, nil
			}
			if !read.Accepted || read.NResolution.EstimatedAbsError == nil {
				out.StopReason = "precision-not-met"
				return out, nil
			}
			value, err := read.NResolution.Value, *read.NResolution.EstimatedAbsError
			if math.Abs(value)+err > 1 {
				out.StopReason = "outside-encoded-domain"
				return out, nil
			}
			next.Values[j] = value
			next.MeasurementError[j] = err
		}
		step.State = next
		out.Steps = append(out.Steps, step)
		current = next
	}
	out.StopReason = "complete"
	return out, nil
}
func (r *BrainRuntime) CompositionDiagnosis() (*knowledge.NDiagnosis, error) {
	if r == nil || r.brain == nil {
		return nil, errors.New("runtime unavailable")
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if e := r.ensureWritableLocked(); e != nil {
		return nil, e
	}
	return r.brain.LastNDiagnosis(), nil
}
