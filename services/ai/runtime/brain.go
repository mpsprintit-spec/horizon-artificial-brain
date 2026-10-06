package runtime

import (
	"errors"
	"sort"
	"sync"
	"time"

	"github.com/project-horizon/horizon-core/services/ai/activation"
	"github.com/project-horizon/horizon-core/services/ai/dnf"
	"github.com/project-horizon/horizon-core/services/ai/knowledge"
	"github.com/project-horizon/horizon-core/services/ai/learning"
)

const BrainIdentity = "horizon-primary-brain"
const GroundingThreshold = 0.90

type Event struct {
	ID string
	Stimulus []string
	StimulusNodeIDs []knowledge.NodeID
	Context map[knowledge.NodeID]float64
	ContextTokens []string
	DataTokens []string
	Source string
	Modality string
	Cycles int
	Timestamp time.Time
	// PredictionOverride is used only at an explicit causal boundary where
	// the caller captured a prediction before an external action. It prevents
	// outcome processing from silently substituting a newer recurrent state.
	PredictionOverride *activation.Prediction
}

type ActionBinding struct {
	RequestID string
	BrainIdentity string
	Intent string
	TargetNodeIDs []knowledge.NodeID
	Synapses []SynapseBinding
}

type SynapseBinding struct {
	SourceNodeID knowledge.NodeID
	TargetNodeID knowledge.NodeID
	Inhibitory bool
}

type BrainRuntime struct {
	brain *knowledge.Brain
	dnf *dnf.Fabric
	activation *activation.Engine
	learning *learning.LearningUnit
	promotion *learning.PromotionEngine
	evidence *learning.EvidenceLedger
	actions map[string]ActionBinding
	eventLog *EventLog
	clock Clock
	mu sync.Mutex
	seq uint64
	lastCognitiveState CognitiveState
	lastObservationPopulation []knowledge.NodeID
	lastObservationAt time.Time
	inquiryValence map[InquiryAction]float64
}

func NewBrainRuntime(brain *knowledge.Brain) *BrainRuntime {
	if brain == nil { brain = knowledge.NewBootstrapBrain() }
	fabric, err := dnf.NewFabric(brain)
	if err != nil { return nil }
	return &BrainRuntime{brain: brain, dnf: fabric, activation: activation.NewEngine(brain), learning: learning.NewLearningUnit(brain), promotion: learning.NewPromotionEngine(brain, learning.DefaultLearningPolicy()), evidence: learning.NewEvidenceLedger(), actions: make(map[string]ActionBinding), inquiryValence: make(map[InquiryAction]float64), clock: WallClock{}}
}
func (r *BrainRuntime) DNF() *dnf.Fabric { if r == nil { return nil }; return r.dnf }
func (r *BrainRuntime) GroundObservation(token, source, modality string) (knowledge.GroundedRepresentation, error) { if r == nil || r.brain == nil { return knowledge.GroundedRepresentation{}, errors.New("brain runtime is not initialized") }; return r.brain.GroundObservation(token, source, modality, GroundingThreshold) }

// ActivateBootstrap rehydrates the persistent bootstrap populations into the
// runtime activation state after the canonical brain has been loaded. It does
// not create semantic definitions or additional memory; it only reactivates
// the numeric populations already persisted in BrainState.
func (r *BrainRuntime) ActivateBootstrap(now time.Time) (activation.Result, error) {
	if r == nil || r.brain == nil || r.activation == nil {
		return activation.Result{}, errors.New("brain runtime is not initialized")
	}
	if now.IsZero() {
		now = r.now()
	}

	context := make(map[knowledge.NodeID]float64)
	for _, experience := range r.brain.BrainState.BootstrapExperiences {
		level := experience.Activation
		if level <= 0 {
			level = 0.35
		}
		level = clamp01(level)
		for _, nodeID := range experience.Populations {
			if r.brain.Registry.GetByID(nodeID) == nil {
				continue
			}
			if level > context[nodeID] {
				context[nodeID] = level
			}
		}
	}
	if len(context) == 0 {
		return activation.Result{}, nil
	}

	return r.activation.ActivateWith(activation.Request{
		ContextBoosts: context,
		Cycles:        1,
		Now:           now,
	}), nil
}

func (r *BrainRuntime) RegisterActionBinding(binding ActionBinding) error { if r == nil || r.brain == nil { return errors.New("brain runtime is not initialized") }; if binding.RequestID == "" { return errors.New("action binding request ID is required") }; if binding.BrainIdentity == "" { binding.BrainIdentity = BrainIdentity }; if binding.BrainIdentity != BrainIdentity { return errors.New("action binding belongs to a different brain") }; if len(binding.TargetNodeIDs) == 0 && len(binding.Synapses) == 0 { return errors.New("action binding requires at least one neural target") }; for _, syn := range binding.Synapses { if r.brain.Registry.GetByID(syn.SourceNodeID) == nil || r.brain.Registry.GetByID(syn.TargetNodeID) == nil { return errors.New("action binding references an unknown synapse node") } }; r.mu.Lock(); defer r.mu.Unlock(); binding.TargetNodeIDs = append([]knowledge.NodeID(nil), binding.TargetNodeIDs...); binding.Synapses = append([]SynapseBinding(nil), binding.Synapses...); r.actions[binding.RequestID] = binding; return nil }
func (r *BrainRuntime) actionBinding(requestID string) (ActionBinding, bool) { if r == nil { return ActionBinding{}, false }; r.mu.Lock(); defer r.mu.Unlock(); binding, ok := r.actions[requestID]; if !ok { return ActionBinding{}, false }; binding.TargetNodeIDs = append([]knowledge.NodeID(nil), binding.TargetNodeIDs...); binding.Synapses = append([]SynapseBinding(nil), binding.Synapses...); return binding, true }
func (r *BrainRuntime) SetClock(clock Clock) { if r == nil { return }; r.mu.Lock(); defer r.mu.Unlock(); if clock == nil { r.clock = WallClock{} } else { r.clock = clock } }
func (r *BrainRuntime) nowLocked() time.Time { if r.clock == nil { r.clock = WallClock{} }; now := r.clock.Now(); if now.IsZero() { now = time.Now().UTC() }; return now.UTC() }
func (r *BrainRuntime) now() time.Time { if r == nil { return time.Now().UTC() }; r.mu.Lock(); defer r.mu.Unlock(); return r.nowLocked() }
func (r *BrainRuntime) SetEventLog(log *EventLog) { if r == nil { return }; r.mu.Lock(); defer r.mu.Unlock(); r.eventLog = log }
func (r *BrainRuntime) Process(event Event) (activation.Result, uint64, error) {
	if r == nil || r.brain == nil || r.activation == nil { return activation.Result{}, 0, errors.New("brain runtime is not initialized") }
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.processLocked(event)
}

func (r *BrainRuntime) processLocked(event Event) (activation.Result, uint64, error) {
	if event.Timestamp.IsZero() { event.Timestamp = r.nowLocked() }
	nextSeq := r.seq + 1
	if r.eventLog != nil { if err := r.eventLog.Append(LoggedEvent{SchemaVersion: EventLogSchemaVersion, BrainIdentity: BrainIdentity, Sequence: nextSeq, Type: EventTypeProcess, Timestamp: event.Timestamp, Event: cloneEvent(event)}); err != nil { return activation.Result{}, r.seq, err } }
	result := r.activation.ActivateWith(activation.Request{
		StimulusTokens: append([]string(nil), event.Stimulus...),
		StimulusNodeIDs: append([]knowledge.NodeID(nil), event.StimulusNodeIDs...),
		ContextBoosts: cloneContext(event.Context),
		PredictionOverride: clonePrediction(event.PredictionOverride),
		Cycles: event.Cycles,
		Now: event.Timestamp,
	})
	r.seq = nextSeq
	return result, r.seq, nil
}
// LearnObservedTransition binds consecutive grounded populations as a
// substrate-native temporal transition. The actual reinforcement is delegated
// to the canonical brain state so prediction error, plasticity, and memory
// priority can alter future learning without introducing semantic labels.
func (r *BrainRuntime) LearnObservedTransition(current []knowledge.NodeID, now time.Time) error {
	if r == nil || r.brain == nil {
		return errors.New("brain runtime is not initialized")
	}
	if now.IsZero() {
		now = r.now()
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	if len(current) == 0 {
		r.lastObservationPopulation = nil
		r.lastObservationAt = time.Time{}
		return nil
	}

	previous := append([]knowledge.NodeID(nil), r.lastObservationPopulation...)
	previousAt := r.lastObservationAt
	currentCopy := append([]knowledge.NodeID(nil), current...)
	r.lastObservationPopulation = currentCopy
	r.lastObservationAt = now

	if len(previous) == 0 {
		return nil
	}

	// Keep the observation transition atomic with its runtime predecessor
	// snapshot. The canonical Brain mutation remains protected by the Brain
	// mutation boundary inside ReinforceTransitionPopulation.
	return r.brain.ReinforceTransitionPopulation(previous, currentCopy, previousAt, now, 0.25)
}

func (r *BrainRuntime) LearnExperience(experience learning.Experience, now time.Time) (uint64, error) { if r == nil || r.brain == nil || r.learning == nil || r.dnf == nil { return 0, errors.New("brain runtime is not initialized") }; r.mu.Lock(); defer r.mu.Unlock(); if now.IsZero() { now = r.nowLocked() }; nextSeq := r.seq + 1; copyExperience := experience; if r.eventLog != nil { if err := r.eventLog.Append(LoggedEvent{SchemaVersion: EventLogSchemaVersion, BrainIdentity: BrainIdentity, Sequence: nextSeq, Type: EventTypeLearn, Timestamp: now, Experience: &copyExperience}); err != nil { return r.seq, err } }; r.learning.LearnExperience(experience, now); r.seq = nextSeq; return r.seq, nil }
func (r *BrainRuntime) Think(cycles int) (activation.ThoughtResult, uint64, error) {
	return r.ThinkAt(cycles, time.Time{})
}

func (r *BrainRuntime) ThinkAt(cycles int, timestamp time.Time) (activation.ThoughtResult, uint64, error) {
	return r.thinkAtWithContext(cycles, nil, timestamp)
}

func (r *BrainRuntime) thinkAtWithContext(cycles int, context map[knowledge.NodeID]float64, timestamp time.Time) (activation.ThoughtResult, uint64, error) {
	if r == nil || r.brain == nil || r.activation == nil {
		return activation.ThoughtResult{}, 0, errors.New("brain runtime is not initialized")
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.thinkAtWithContextLocked(cycles, context, timestamp)
}

func (r *BrainRuntime) thinkAtWithContextLocked(cycles int, context map[knowledge.NodeID]float64, timestamp time.Time) (activation.ThoughtResult, uint64, error) {
	now := timestamp
	if now.IsZero() {
		now = r.nowLocked()
	}
	nextSeq := r.seq + 1
	if r.eventLog != nil {
		event := Event{Cycles: cycles, Timestamp: now, Context: cloneContext(context)}
		if err := r.eventLog.Append(LoggedEvent{SchemaVersion: EventLogSchemaVersion, BrainIdentity: BrainIdentity, Sequence: nextSeq, Type: EventTypeThink, Timestamp: now, Event: &event}); err != nil {
			return activation.ThoughtResult{}, r.seq, err
		}
	}
	thought := r.activation.ThinkWithContextAt(cycles, context, now)
	r.seq = nextSeq
	return thought, r.seq, nil
}
type CognitiveOutput struct { BrainIdentity string; Sequence uint64; Timestamp time.Time; RankedNodeIDs []knowledge.NodeID; Activations map[knowledge.NodeID]float64; Confidence map[knowledge.NodeID]float64; Resonance float64; PredictionError float64; Prediction activation.Prediction; StateDelta CognitiveStateDelta }
func (r *BrainRuntime) CognitiveProcess(event Event) (CognitiveOutput, error) {
	if r == nil {
		return CognitiveOutput{}, errors.New("brain runtime is not initialized")
	}
	if event.Timestamp.IsZero() {
		r.mu.Lock()
		event.Timestamp = r.nowLocked()
		r.mu.Unlock()
	}

	// Keep inference, prediction capture, learning, memory dynamics, and state
	// delta calculation in one runtime transaction. A second cognition cycle
	// must not interleave between activation and its prediction snapshot.
	r.mu.Lock()
	defer r.mu.Unlock()

	result, sequence, err := r.processLocked(event)
	if err != nil {
		return CognitiveOutput{}, err
	}

	output := cognitiveOutputFromResult(BrainIdentity, sequence, event.Timestamp, result)
	output.Prediction = r.activation.PredictionSnapshot()
	r.brain.RecordLearningSignal(output.PredictionError, event.Timestamp)
	r.brain.ApplyMemoryDynamics(event.Timestamp)
	output.StateDelta = output.State().Diff(r.lastCognitiveState)
	r.lastCognitiveState = output.State()
	return output, nil
}

func (r *BrainRuntime) CognitiveThink(cycles int) (CognitiveOutput, error) {
	if r == nil || r.brain == nil {
		return CognitiveOutput{}, errors.New("brain runtime is not initialized")
	}
	// Autonomous cognition is one serialized runtime transaction. This keeps
	// external observations, learning, and the continuous service from
	// interleaving at the runtime boundary while the canonical Brain lock
	// protects the underlying neural substrate.
	r.mu.Lock()
	defer r.mu.Unlock()

	now := r.nowLocked()
	// Autonomous cognition needs an endogenous drive when no external event is
	// present. The drive is deterministic: it prefers underused neural units
	// and increases with persistent uncertainty/curiosity. It is not random
	// stimulation and it does not assign semantic meaning to any node.
	curiosityContext := r.applyCuriosityDrive(now)

	thought, sequence, err := r.thinkAtWithContextLocked(cycles, curiosityContext, now)
	if err != nil {
		return CognitiveOutput{}, err
	}
	output := CognitiveOutput{
		BrainIdentity: BrainIdentity,
		Sequence: sequence,
		Timestamp: now,
		RankedNodeIDs: rankedNodeIDs(thought.RankedNodes),
		Activations: cloneNodeValues(thought.Activations),
		Confidence: cloneNodeValues(thought.Confidence),
		Resonance: thought.Resonance,
		PredictionError: thought.PredictionError,
		Prediction: thought.Prediction,
	}
	r.brain.RecordLearningSignal(output.PredictionError, now)
	r.brain.ApplyMemoryDynamics(now)
	output.StateDelta = output.State().Diff(r.lastCognitiveState)
	r.lastCognitiveState = output.State()
	return output, nil
}

// applyCuriosityDrive injects a bounded endogenous context signal before an
// autonomous thought cycle. Novelty is derived from usage history and
// uncertainty from the persistent prediction signal. The least-used
// under-activated populations therefore become candidates for exploration.
// No semantic label, random noise, or external input is introduced.
func (r *BrainRuntime) applyCuriosityDrive(now time.Time) map[knowledge.NodeID]float64 {
	if r == nil || r.brain == nil || now.IsZero() {
		return nil
	}

	// Curiosity reads canonical neural state as a snapshot. The write-back is
	// performed under the same Brain mutation boundary used by learning,
	// activation, and memory dynamics, so the continuous service can coexist
	// with external observations without racing on BrainState.
	r.brain.RLock()
	predictionError := clamp01(r.brain.BrainState.PredictionState["last_error"])
	policy := r.brain.BrainState.LearningPolicyState

	type candidate struct {
		nodeID knowledge.NodeID
		score  float64
	}
	candidates := make([]candidate, 0)
	for _, node := range r.brain.Registry.Nodes() {
		if node == nil {
			continue
		}
		usage := 1.0 / (1.0 + float64(maxInt64(node.Frequency, 0)))
		underActivation := clamp01(1.0 - node.Activation)
		novelty := clamp01(usage * clamp01(policy.NoveltySensitivity))
		uncertainty := clamp01((0.5*underActivation)+(0.5*predictionError)) * clamp01(policy.UncertaintySensitivity)
		score := clamp01(policy.CuriosityPressure) * clamp01(novelty+uncertainty)
		if score <= 0.05 {
			continue
		}
		candidates = append(candidates, candidate{nodeID: node.ID, score: score})
	}
	r.brain.RUnlock()

	if len(candidates) == 0 {
		return nil
	}
	sort.SliceStable(candidates, func(i, j int) bool {
		if candidates[i].score == candidates[j].score {
			return candidates[i].nodeID < candidates[j].nodeID
		}
		return candidates[i].score > candidates[j].score
	})

	boost := clamp01(candidates[0].score * 0.30)
	if boost <= 0 {
		return nil
	}

	r.brain.Lock()
	if r.brain.BrainState.CuriosityState == nil {
		r.brain.BrainState.CuriosityState = map[string]float64{}
	}
	r.brain.BrainState.CuriosityState["drive"] = boost
	r.brain.BrainState.CuriosityState["target_node"] = float64(candidates[0].nodeID)
	r.brain.BrainState.CuriosityState["updated_unix"] = float64(now.UnixNano())
	r.brain.Unlock()

	return map[knowledge.NodeID]float64{candidates[0].nodeID: boost}
}
func maxInt64(value int64, floor int64) int64 {
	if value < floor {
		return floor
	}
	return value
}
// PlanInquiry creates an internal information-seeking trajectory from the
// current uncertainty and persisted learning strategy. It records the selected
// policy result in the canonical brain state but never executes an external action.
func (r *BrainRuntime) PlanInquiry(uncertainty float64, at time.Time) (InquiryAgenda, error) {
	if r == nil || r.brain == nil {
		return InquiryAgenda{}, errors.New("brain runtime is not initialized")
	}
	if at.IsZero() {
		at = r.now()
	}
	uncertainty = clamp01(uncertainty)
	candidates := DefaultInquiryCandidates(uncertainty)

	// Learning policy is canonical Brain state. Snapshot it under the Brain
	// read lock so continuous cognition cannot race a concurrent policy update.
	r.brain.RLock()
	policy := r.brain.BrainState.LearningPolicyState
	r.brain.RUnlock()

	for i := range candidates {
		// Information value comes from the learned internal action/outcome
		// model. If the action has no learned model yet, leave its value at
		// zero: uncertainty alone does not imply that the action will provide
		// useful information. No action-specific curiosity constant is used.
		if value, modeled, err := r.PredictInquiryInformationValue(candidates[i].Action, uncertainty, at); err == nil && modeled {
			candidates[i].ExpectedInformationGain = value
		} else {
			candidates[i].ExpectedInformationGain = 0
		}

		candidates[i].PriorExperience = r.InquiryPriorExperience(candidates[i].Action, candidates[i].PriorExperience)
		switch candidates[i].Action {
		case InquiryReobserve, InquiryFocus:
			candidates[i].PriorExperience = clamp01(candidates[i].PriorExperience + 0.20*clamp01(policy.RepeatObservationBias))
		case InquiryChangeView, InquiryImitate, InquirySafeManipulation:
			candidates[i].PriorExperience = clamp01(candidates[i].PriorExperience + 0.20*clamp01(policy.ExplorationBias))
		case InquiryWait:
			candidates[i].PriorExperience = clamp01(candidates[i].PriorExperience + 0.20*clamp01(policy.DeferConclusionBias))
		}
	}

	agenda, err := BuildInquiryAgenda(BrainIdentity, r.LastSequence(), at, candidates, DefaultInquiryPolicy())
	if err != nil {
		return InquiryAgenda{}, err
	}
	agenda.Uncertainty = uncertainty
	if agenda.Selected != nil {
		r.brain.RecordInquirySelection(
			agenda.Sequence,
			string(agenda.Selected.Action),
			uncertainty,
			agenda.Selected.InformationValue,
			clamp01(agenda.Selected.Score/1.85),
			at,
		)
	}
	return agenda, nil
}

// CompleteInquiry closes the pending internal trajectory after an observation
// or other authorized adapter result. It updates the persisted learning
// strategy; it does not execute, authorize, or infer an external action.
func (r *BrainRuntime) CompleteInquiry(observedInformationGain, predictionError float64, at time.Time) error {
	if r == nil || r.brain == nil {
		return errors.New("brain runtime is not initialized")
	}
	if at.IsZero() {
		at = r.now()
	}
	r.brain.RecordInquiryOutcome(observedInformationGain, predictionError, at)
	return nil
}

func (r *BrainRuntime) LastSequence() uint64 { if r == nil { return 0 }; r.mu.Lock(); defer r.mu.Unlock(); return r.seq }
func cognitiveOutputFromResult(identity string, sequence uint64, now time.Time, result activation.Result) CognitiveOutput {
	return CognitiveOutput{
		BrainIdentity: identity, Sequence: sequence, Timestamp: now,
		RankedNodeIDs: rankedNodeIDs(result.RankedNodes),
		Activations: cloneNodeValues(result.Activations),
		Confidence: cloneNodeValues(result.Confidence),
		Resonance: result.Resonance, PredictionError: result.PredictionError,
	}
}
func rankedNodeIDs(nodes []*knowledge.ConceptNode) []knowledge.NodeID { out := make([]knowledge.NodeID, 0, len(nodes)); for _, node := range nodes { if node != nil { out = append(out, node.ID) } }; return out }
func cloneNodeValues(in map[knowledge.NodeID]float64) map[knowledge.NodeID]float64 { if in == nil { return map[knowledge.NodeID]float64{} }; out := make(map[knowledge.NodeID]float64, len(in)); for id, value := range in { out[id] = value }; return out }
func cloneContext(in map[knowledge.NodeID]float64) map[knowledge.NodeID]float64 { if in == nil { return nil }; out := make(map[knowledge.NodeID]float64, len(in)); for id, value := range in { out[id] = value }; return out }
func clonePrediction(in *activation.Prediction) *activation.Prediction { if in == nil { return nil }; return &activation.Prediction{State: cloneNodeValues(in.State), Confidence: cloneNodeValues(in.Confidence)} }
func cloneEvent(in Event) *Event { out := in; out.Stimulus = append([]string(nil), in.Stimulus...); out.StimulusNodeIDs = append([]knowledge.NodeID(nil), in.StimulusNodeIDs...); out.Context = cloneContext(in.Context); out.ContextTokens = append([]string(nil), in.ContextTokens...); out.DataTokens = append([]string(nil), in.DataTokens...); out.PredictionOverride = clonePrediction(in.PredictionOverride); return &out }
