package runtime

import (
	"errors"
	"github.com/project-horizon/horizon-core/services/ai/activation"
	"github.com/project-horizon/horizon-core/services/ai/knowledge"
	"strings"
	"time"
)

const nativeJournalSchemaVersion = 14
const nativeCheckpointSchemaVersion = 20
const EventTypeNativeControl = "native_control_command"

type NativeObservation struct {
	Source   string                       `json:"source"`
	SampleID string                       `json:"sample_id"`
	State    map[knowledge.NodeID]float64 `json:"state"`
}
type NativeBegin struct {
	Intervention bool               `json:"intervention,omitempty"` // explicit controlled experiment, not an on-policy sample
	EpisodeID    string             `json:"episode_id"`
	Training     bool               `json:"training"`
	Observation  *NativeObservation `json:"observation,omitempty"`
}
type NativeStep struct {
	ExperimentAction *int    `json:"experiment_action,omitempty"`
	ActionOverride   *int    `json:"action_override,omitempty"` // evaluation ablation only; never eligible for learning
	EpisodeID        string  `json:"episode_id"`
	Sample           float64 `json:"sample"`
}
type NativeOutcome struct {
	ContrastID  string  `json:"contrast_id,omitempty"` // two observed, matched-action trajectories
	EpisodeID   string  `json:"episode_id"`
	TraceDigest string  `json:"trace_digest"`
	Source      string  `json:"source"`
	OutcomeID   string  `json:"outcome_id"`
	Provenance  string  `json:"provenance"` // observed-environment or observed-sensor
	Reward      float64 `json:"reward"`
	Learn       bool    `json:"learn"`
}
type NativeControlCommand struct {
	DiscardContrastID string                         `json:"discard_contrast_id,omitempty"` // abandon pending trial; retain its journal evidence
	ID                string                         `json:"id"`
	Initialize        *knowledge.NativeControlConfig `json:"initialize,omitempty"`
	Begin             *NativeBegin                   `json:"begin,omitempty"`
	Step              *NativeStep                    `json:"step,omitempty"`
	Outcome           *NativeOutcome                 `json:"outcome,omitempty"`
}
type NativeControlResult struct {
	Sequence    uint64                          `json:"sequence"`
	Duplicate   bool                            `json:"duplicate"`
	Readout     *knowledge.NativeReadout        `json:"readout,omitempty"`
	Work        activation.NativeOperationStats `json:"work"`
	Learning    *knowledge.NativeLearning       `json:"learning,omitempty"`
	Stopped     bool                            `json:"stopped"`
	Truncated   bool                            `json:"truncated"`
	TraceDigest string                          `json:"trace_digest,omitempty"`
	State       map[knowledge.NodeID]float64    `json:"state,omitempty"`
}

func (c NativeControlCommand) Validate() error {
	if c.ID == "" || len(c.ID) > 256 || strings.Contains(c.ID, "/") {
		return errors.New("native command needs bounded slash-free ID")
	}
	n := 0
	if c.DiscardContrastID != "" {
		n++
		if len(c.DiscardContrastID) > 256 || strings.Contains(c.DiscardContrastID, "/") {
			return errors.New("invalid discarded contrast")
		}
	}
	if c.Initialize != nil {
		n++
		if e := c.Initialize.Validate(); e != nil {
			return e
		}
	}
	if c.Begin != nil {
		n++
		if c.Begin.EpisodeID == "" {
			return errors.New("native episode ID required")
		}
		if o := c.Begin.Observation; o != nil {
			if o.Source == "" || o.SampleID == "" || len(o.State) == 0 {
				return errors.New("observed native input requires provenance and state")
			}
		}
	}
	if c.Step != nil {
		n++
		if c.Step.EpisodeID == "" || c.Step.Sample < 0 || c.Step.Sample >= 1 || c.Step.Sample != c.Step.Sample {
			return errors.New("invalid native step")
		}
	}
	if o := c.Outcome; o != nil {
		n++
		if len(o.ContrastID) > 256 || strings.Contains(o.ContrastID, "/") {
			return errors.New("invalid contrast ID")
		}
		if o.EpisodeID == "" || o.OutcomeID == "" || o.Source == "" || len(o.TraceDigest) != 64 || o.Reward < -1 || o.Reward > 1 || o.Reward != o.Reward || (o.Provenance != "observed-environment" && o.Provenance != "observed-sensor") {
			return errors.New("native feedback requires observed outcome provenance")
		}
	}
	if n != 1 {
		return errors.New("native command needs exactly one operation")
	}
	return nil
}

// ApplyNativeControl is explicit opt-in. It cannot execute an external action,
// grant authorization, or use a hypothetical forecast as an observed reward.
// The environment adapter supplies an observed reward only AFTER execution.
func (r *BrainRuntime) ApplyNativeControl(command NativeControlCommand, at time.Time) (NativeControlResult, error) {
	out := NativeControlResult{}
	if r == nil || r.brain == nil {
		return out, errors.New("runtime unavailable")
	}
	c, err := freezeComposition(command)
	if err != nil {
		return out, err
	}
	if err = c.Validate(); err != nil {
		return out, err
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if err = r.ensureWritableLocked(); err != nil {
		return out, err
	}
	if at.IsZero() {
		at = r.nowLocked()
	}
	at = at.UTC()
	prefix := nativeCommandPrefix(c)
	key := prefix + knowledge.NativeDigest(c)
	if r.nativeCommandIndex == nil {
		r.nativeCommandIndex = map[string]string{}
		for id := range r.processedEvents {
			if strings.HasPrefix(id, "native/") {
				i := strings.LastIndex(id, "/")
				r.nativeCommandIndex[id[:i+1]] = id
			}
		}
	}
	if old, ok := r.nativeCommandIndex[prefix]; ok {
		if old != key {
			return out, errors.New("native ID reused with changed command")
		}
		return NativeControlResult{Sequence: r.processedEvents[old], Duplicate: true}, nil
	}
	var transition *activation.NativeTransition
	s := r.brain.NativeControlSnapshot()
	if c.Initialize != nil {
		if err = r.brain.ValidateNativeInit(*c.Initialize); err != nil {
			return out, err
		}
	} else if s == nil {
		return out, errors.New("native control is not initialized")
	}
	if c.DiscardContrastID != "" {
		if s == nil || s.Episode != nil || s.PendingContrast == nil || s.PendingContrast.ID != c.DiscardContrastID {
			return out, errors.New("discard requires matching idle pending contrast")
		}
	}
	if c.Begin != nil {
		if p := s.PendingContrast; p != nil && at.Before(p.Episode.Decisions[len(p.Episode.Decisions)-1].At) {
			return out, errors.New("native clock rollback across contrast trials")
		}
		if c.Begin.Intervention && (!c.Begin.Training || s.Config.Version != knowledge.NativeControlRelationalVersion || s.Config.Lambda != 1) {
			return out, errors.New("invalid intervention experiment mode")
		}
		if s.Episode != nil {
			return out, errors.New("native episode already open")
		}
		state := r.activation.CurrentState()
		if c.Begin.Observation != nil {
			state = c.Begin.Observation.State
		}
		if len(state) > s.Config.MaxActiveNodes {
			return out, errors.New("native input exceeds active-state budget")
		}
		if len(state) == 0 {
			return out, errors.New("native episode needs existing activity or an observation")
		}
		if _, err = r.brain.NativePolicy(state, true, 0); err != nil {
			return out, err
		}
		for _, group := range s.Outputs {
			for _, id := range group {
				if _, ok := state[id]; ok {
					return out, errors.New("native observation cannot inject effector activity")
				}
			}
		}
		for _, id := range s.Features {
			if _, ok := state[id]; ok {
				return out, errors.New("native observation cannot inject feature activity")
			}
		}
		if _, ok := state[s.Bias]; ok {
			return out, errors.New("native observation cannot inject bias")
		}
	}
	if c.Step != nil {
		ep := s.Episode
		if ep == nil || ep.ID != c.Step.EpisodeID || ep.Stopped || len(ep.Decisions) >= s.Config.MaxSteps || at.Before(ep.Started) {
			return out, errors.New("native step outside active episode")
		}
		if len(ep.Decisions) > 0 && at.Before(ep.Decisions[len(ep.Decisions)-1].At) {
			return out, errors.New("native clock rollback")
		}
		p, e := r.brain.NativePolicy(r.activation.CurrentState(), !ep.Training, c.Step.Sample)
		if e != nil {
			return out, e
		}
		if ep.Intervention != (c.Step.ExperimentAction != nil) {
			return out, errors.New("intervention action missing or unauthorized")
		}
		if a := c.Step.ExperimentAction; a != nil {
			if *a < 0 || *a >= len(p.Values) || c.Step.ActionOverride != nil {
				return out, errors.New("invalid experiment action")
			}
			p.Action = *a
		}
		if c.Step.ActionOverride != nil {
			if ep.Training || *c.Step.ActionOverride < 0 || *c.Step.ActionOverride >= len(p.Values) {
				return out, errors.New("action override is evaluation-only")
			}
			p.Action = *c.Step.ActionOverride
		}
		transition, err = r.activation.PrepareNativeOperation(s.Config.Operations[p.Action], at, s.Config.MaxActiveNodes, s.Config.MaxSynapseReads)
		if err != nil {
			return out, err
		}
		out.Readout = &p
	}
	if o := c.Outcome; o != nil {
		ep := s.Episode
		if ep == nil || ep.ID != o.EpisodeID || !ep.Stopped || len(ep.Decisions) == 0 || o.TraceDigest != knowledge.NativeDigest(ep) || at.Before(ep.Decisions[len(ep.Decisions)-1].At) {
			return out, errors.New("native outcome does not bind completed trajectory")
		}
		if err = r.brain.ValidateNativeContrast(o.ContrastID, o.Source, o.OutcomeID, o.Learn); err != nil {
			return out, err
		}
		if o.ContrastID != "" {
			if _, used := r.processedEvents["native-contrast/"+knowledge.NativeDigest([]string{o.Source, o.ContrastID})]; used {
				return out, errors.New("contrast already consumed")
			}
		}
		if o.Learn && !ep.Training {
			return out, errors.New("greedy evaluation cannot train")
		}
	}
	err = r.runJournalCommandLocked(LoggedEvent{Type: EventTypeNativeControl, Timestamp: at, NativeControl: &c}, func() error {
		if c.Initialize != nil {
			if e := r.brain.InitializeNativeControl(*c.Initialize, at); e != nil {
				return e
			}
		}
		if c.DiscardContrastID != "" {
			if e := r.brain.DiscardNativeContrast(c.DiscardContrastID); e != nil {
				return e
			}
		}
		if c.Begin != nil {
			if o := c.Begin.Observation; o != nil {
				snapshot := r.activation.SnapshotState()
				snapshot.InternalState = o.State
				snapshot.InternalConfidence = map[knowledge.NodeID]float64{}
				for id := range o.State {
					snapshot.InternalConfidence[id] = .5
				}
				snapshot.LastPrediction = nil
				snapshot.LastPredictionConf = nil
				snapshot.ObservationPrediction = nil
				snapshot.LastTrace = nil
				snapshot.ContextState = nil
				r.activation.RestoreState(snapshot)
			}
			if e := r.brain.BeginNativeEpisode(c.Begin.EpisodeID, c.Begin.Training, at, c.Begin.Intervention); e != nil {
				return e
			}
		}
		if c.Step != nil {
			before := r.activation.CurrentState()
			a := out.Readout.Action
			op := s.Config.Operations[a]
			work, e := r.activation.CommitNativeOperation(transition)
			if e != nil {
				return e
			}
			out.Work = work
			after := r.activation.CurrentState()
			d := knowledge.NativeDecision{State: before, Values: out.Readout.Values, Probabilities: out.Readout.Probabilities, Action: a, Cost: op.Cost, Before: knowledge.NativeStateDigest(before), After: knowledge.NativeStateDigest(after), At: at}
			if e = r.brain.AppendNativeDecision(d); e != nil {
				return e
			}
			out.State = after
			ep := r.brain.NativeControlSnapshot().Episode
			out.Stopped, out.Truncated = ep.Stopped, ep.Truncated
			out.TraceDigest = knowledge.NativeDigest(ep)
		}
		if o := c.Outcome; o != nil {
			learning, e := r.brain.LearnNativeObservedOutcome(o.Reward, o.Learn, o.ContrastID, o.Source, o.OutcomeID, at)
			if e != nil {
				return e
			}
			out.Learning = &learning
			if learning.Contrast && !learning.ContrastPending {
				r.processedEvents["native-contrast/"+knowledge.NativeDigest([]string{o.Source, o.ContrastID})] = r.seq + 1
			}
		}
		r.seq++
		r.processedEvents[key] = r.seq
		r.nativeCommandIndex[prefix] = key
		return nil
	})
	out.Sequence = r.seq
	return out, err
}

// A feedback identifier is single-use across command retries AND episodes.
// The prefix is reconstructible from committed journal inputs; only the exact
// command can retry. Changed IDs cannot make one observed outcome count twice.
func nativeCommandPrefix(c NativeControlCommand) string {
	if c.Outcome != nil {
		return "native/outcome-" + knowledge.NativeDigest([]string{c.Outcome.Source, c.Outcome.OutcomeID}) + "/"
	}
	return "native/" + c.ID + "/"
}
