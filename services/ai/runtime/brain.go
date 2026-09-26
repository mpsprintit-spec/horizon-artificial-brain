package runtime

import (
	"errors"
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
	inquiryValence map[InquiryAction]float64
}

func NewBrainRuntime(brain *knowledge.Brain) *BrainRuntime {
	if brain == nil { brain = knowledge.NewBrain() }
	fabric, err := dnf.NewFabric(brain)
	if err != nil { return nil }
	return &BrainRuntime{brain: brain, dnf: fabric, activation: activation.NewEngine(brain), learning: learning.NewLearningUnit(brain), promotion: learning.NewPromotionEngine(brain, learning.DefaultLearningPolicy()), evidence: learning.NewEvidenceLedger(), actions: make(map[string]ActionBinding), inquiryValence: make(map[InquiryAction]float64), clock: WallClock{}}
}
func (r *BrainRuntime) DNF() *dnf.Fabric { if r == nil { return nil }; return r.dnf }
func (r *BrainRuntime) GroundObservation(token, source, modality string) (knowledge.GroundedRepresentation, error) { if r == nil || r.brain == nil { return knowledge.GroundedRepresentation{}, errors.New("brain runtime is not initialized") }; return r.brain.GroundObservation(token, source, modality, GroundingThreshold) }
func (r *BrainRuntime) RegisterActionBinding(binding ActionBinding) error { if r == nil || r.brain == nil { return errors.New("brain runtime is not initialized") }; if binding.RequestID == "" { return errors.New("action binding request ID is required") }; if binding.BrainIdentity == "" { binding.BrainIdentity = BrainIdentity }; if binding.BrainIdentity != BrainIdentity { return errors.New("action binding belongs to a different brain") }; if len(binding.TargetNodeIDs) == 0 && len(binding.Synapses) == 0 { return errors.New("action binding requires at least one neural target") }; for _, syn := range binding.Synapses { if r.brain.Registry.GetByID(syn.SourceNodeID) == nil || r.brain.Registry.GetByID(syn.TargetNodeID) == nil { return errors.New("action binding references an unknown synapse node") } }; r.mu.Lock(); defer r.mu.Unlock(); binding.TargetNodeIDs = append([]knowledge.NodeID(nil), binding.TargetNodeIDs...); binding.Synapses = append([]SynapseBinding(nil), binding.Synapses...); r.actions[binding.RequestID] = binding; return nil }
func (r *BrainRuntime) actionBinding(requestID string) (ActionBinding, bool) { if r == nil { return ActionBinding{}, false }; r.mu.Lock(); defer r.mu.Unlock(); binding, ok := r.actions[requestID]; if !ok { return ActionBinding{}, false }; binding.TargetNodeIDs = append([]knowledge.NodeID(nil), binding.TargetNodeIDs...); binding.Synapses = append([]SynapseBinding(nil), binding.Synapses...); return binding, true }
func (r *BrainRuntime) SetClock(clock Clock) { if r == nil { return }; r.mu.Lock(); defer r.mu.Unlock(); if clock == nil { r.clock = WallClock{} } else { r.clock = clock } }
func (r *BrainRuntime) nowLocked() time.Time { if r.clock == nil { r.clock = WallClock{} }; now := r.clock.Now(); if now.IsZero() { now = time.Now().UTC() }; return now.UTC() }
func (r *BrainRuntime) now() time.Time { if r == nil { return time.Now().UTC() }; r.mu.Lock(); defer r.mu.Unlock(); return r.nowLocked() }
func (r *BrainRuntime) SetEventLog(log *EventLog) { if r == nil { return }; r.mu.Lock(); defer r.mu.Unlock(); r.eventLog = log }
func (r *BrainRuntime) Process(event Event) (activation.Result, uint64, error) {
	if r == nil || r.brain == nil || r.activation == nil { return activation.Result{}, 0, errors.New("brain runtime is not initialized") }
	r.mu.Lock(); defer r.mu.Unlock()
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
func (r *BrainRuntime) LearnExperience(experience learning.Experience, now time.Time) (uint64, error) { if r == nil || r.brain == nil || r.learning == nil || r.dnf == nil { return 0, errors.New("brain runtime is not initialized") }; r.mu.Lock(); defer r.mu.Unlock(); if now.IsZero() { now = r.nowLocked() }; nextSeq := r.seq + 1; copyExperience := experience; if r.eventLog != nil { if err := r.eventLog.Append(LoggedEvent{SchemaVersion: EventLogSchemaVersion, BrainIdentity: BrainIdentity, Sequence: nextSeq, Type: EventTypeLearn, Timestamp: now, Experience: &copyExperience}); err != nil { return r.seq, err } }; r.learning.LearnExperience(experience, now); r.seq = nextSeq; return r.seq, nil }
func (r *BrainRuntime) Think(cycles int) (activation.ThoughtResult, uint64, error) { return r.ThinkAt(cycles, time.Time{}) }
func (r *BrainRuntime) ThinkAt(cycles int, timestamp time.Time) (activation.ThoughtResult, uint64, error) { if r == nil || r.brain == nil || r.activation == nil { return activation.ThoughtResult{}, 0, errors.New("brain runtime is not initialized") }; r.mu.Lock(); defer r.mu.Unlock(); now := timestamp; if now.IsZero() { now = r.nowLocked() }; nextSeq := r.seq + 1; if r.eventLog != nil { event := Event{Cycles: cycles, Timestamp: now}; if err := r.eventLog.Append(LoggedEvent{SchemaVersion: EventLogSchemaVersion, BrainIdentity: BrainIdentity, Sequence: nextSeq, Type: EventTypeThink, Timestamp: now, Event: &event}); err != nil { return activation.ThoughtResult{}, r.seq, err } }; thought := r.activation.ThinkWithPredictionAt(cycles, now); r.seq = nextSeq; return thought, r.seq, nil }
type CognitiveOutput struct { BrainIdentity string; Sequence uint64; Timestamp time.Time; RankedNodeIDs []knowledge.NodeID; Activations map[knowledge.NodeID]float64; Confidence map[knowledge.NodeID]float64; Resonance float64; PredictionError float64; Prediction activation.Prediction; StateDelta CognitiveStateDelta }
func (r *BrainRuntime) CognitiveProcess(event Event) (CognitiveOutput, error) { if r == nil { return CognitiveOutput{}, errors.New("brain runtime is not initialized") }; if event.Timestamp.IsZero() { r.mu.Lock(); event.Timestamp = r.nowLocked(); r.mu.Unlock() }; result, sequence, err := r.Process(event); if err != nil { return CognitiveOutput{}, err }; output := cognitiveOutputFromResult(BrainIdentity, sequence, event.Timestamp, result); output.Prediction = r.activation.PredictionSnapshot(); r.mu.Lock(); output.StateDelta = output.State().Diff(r.lastCognitiveState); r.lastCognitiveState = output.State(); r.mu.Unlock(); return output, nil }
func (r *BrainRuntime) CognitiveThink(cycles int) (CognitiveOutput, error) { thought, sequence, err := r.Think(cycles); if err != nil { return CognitiveOutput{}, err }; now := r.now(); output := CognitiveOutput{BrainIdentity: BrainIdentity, Sequence: sequence, Timestamp: now, RankedNodeIDs: rankedNodeIDs(thought.RankedNodes), Activations: cloneNodeValues(thought.Activations), Confidence: cloneNodeValues(thought.Confidence), Resonance: thought.Resonance, PredictionError: thought.PredictionError, Prediction: thought.Prediction}; r.mu.Lock(); output.StateDelta = output.State().Diff(r.lastCognitiveState); r.lastCognitiveState = output.State(); r.mu.Unlock(); return output, nil }
func (r *BrainRuntime) LastSequence() uint64 { if r == nil { return 0 }; r.mu.Lock(); defer r.mu.Unlock(); return r.seq }
func cognitiveOutputFromResult(identity string, sequence uint64, now time.Time, result activation.Result) CognitiveOutput { return CognitiveOutput{BrainIdentity: identity, Sequence: sequence, Timestamp: now, RankedNodeIDs: rankedNodeIDs(result.RankedNodes), Activations: cloneNodeValues(result.Activations), Resonance: result.Resonance, PredictionError: result.PredictionError} }
func rankedNodeIDs(nodes []*knowledge.ConceptNode) []knowledge.NodeID { out := make([]knowledge.NodeID, 0, len(nodes)); for _, node := range nodes { if node != nil { out = append(out, node.ID) } }; return out }
func cloneNodeValues(in map[knowledge.NodeID]float64) map[knowledge.NodeID]float64 { if in == nil { return map[knowledge.NodeID]float64{} }; out := make(map[knowledge.NodeID]float64, len(in)); for id, value := range in { out[id] = value }; return out }
func cloneContext(in map[knowledge.NodeID]float64) map[knowledge.NodeID]float64 { if in == nil { return nil }; out := make(map[knowledge.NodeID]float64, len(in)); for id, value := range in { out[id] = value }; return out }
func clonePrediction(in *activation.Prediction) *activation.Prediction { if in == nil { return nil }; return &activation.Prediction{State: cloneNodeValues(in.State), Confidence: cloneNodeValues(in.Confidence)} }
func cloneEvent(in Event) *Event { out := in; out.Stimulus = append([]string(nil), in.Stimulus...); out.StimulusNodeIDs = append([]knowledge.NodeID(nil), in.StimulusNodeIDs...); out.Context = cloneContext(in.Context); out.ContextTokens = append([]string(nil), in.ContextTokens...); out.DataTokens = append([]string(nil), in.DataTokens...); out.PredictionOverride = clonePrediction(in.PredictionOverride); return &out }
