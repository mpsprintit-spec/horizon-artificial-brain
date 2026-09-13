package runtime

import (
	"errors"
	"sync"
	"time"

	"github.com/project-horizon/horizon-core/services/ai/activation"
	"github.com/project-horizon/horizon-core/services/ai/knowledge"
	"github.com/project-horizon/horizon-core/services/ai/learning"
)

const BrainIdentity = "horizon-primary-brain"

type Event struct {
	ID        string
	Stimulus  []string
	Context   map[knowledge.NodeID]float64
	Cycles    int
	Timestamp time.Time
}

type BrainRuntime struct {
	brain      *knowledge.Brain
	activation *activation.Engine
	learning   *learning.LearningUnit
	eventLog   *EventLog
	mu         sync.Mutex
	seq        uint64
}

func NewBrainRuntime(brain *knowledge.Brain) *BrainRuntime {
	if brain == nil { brain = knowledge.NewBrain() }
	return &BrainRuntime{brain: brain, activation: activation.NewEngine(brain), learning: learning.NewLearningUnit(brain)}
}

// SetEventLog attaches the durable append-only transition log.
func (r *BrainRuntime) SetEventLog(log *EventLog) {
	if r == nil { return }
	r.mu.Lock(); defer r.mu.Unlock()
	r.eventLog = log
}

func (r *BrainRuntime) Process(event Event) (activation.Result, uint64, error) {
	if r == nil || r.brain == nil || r.activation == nil { return activation.Result{}, 0, errors.New("brain runtime is not initialized") }
	r.mu.Lock(); defer r.mu.Unlock()
	if event.Timestamp.IsZero() { event.Timestamp = time.Now().UTC() }
	r.seq++
	result := r.activation.ActivateWith(activation.Request{StimulusTokens: append([]string(nil), event.Stimulus...), ContextBoosts: cloneContext(event.Context), Cycles: event.Cycles, Now: event.Timestamp})
	if r.eventLog != nil {
		if err := r.eventLog.Append(LoggedEvent{SchemaVersion: EventLogSchemaVersion, BrainIdentity: BrainIdentity, Sequence: r.seq, Type: EventTypeProcess, Timestamp: event.Timestamp, Event: cloneEvent(event)}); err != nil { return result, r.seq, err }
	}
	return result, r.seq, nil
}

func (r *BrainRuntime) LearnExperience(experience learning.Experience, now time.Time) (uint64, error) {
	if r == nil || r.brain == nil || r.learning == nil { return 0, errors.New("brain runtime is not initialized") }
	r.mu.Lock(); defer r.mu.Unlock()
	if now.IsZero() { now = time.Now().UTC() }
	r.learning.LearnExperience(experience, now)
	r.seq++
	if r.eventLog != nil {
		copyExperience := experience
		if err := r.eventLog.Append(LoggedEvent{SchemaVersion: EventLogSchemaVersion, BrainIdentity: BrainIdentity, Sequence: r.seq, Type: EventTypeLearn, Timestamp: now, Experience: &copyExperience}); err != nil { return r.seq, err }
	}
	return r.seq, nil
}

func (r *BrainRuntime) Think(cycles int) (activation.ThoughtResult, uint64, error) {
	if r == nil || r.brain == nil || r.activation == nil { return activation.ThoughtResult{}, 0, errors.New("brain runtime is not initialized") }
	r.mu.Lock(); defer r.mu.Unlock()
	r.seq++
	thought := r.activation.ThinkWithPrediction(cycles)
	if r.eventLog != nil {
		event := Event{Cycles: cycles, Timestamp: time.Now().UTC()}
		if err := r.eventLog.Append(LoggedEvent{SchemaVersion: EventLogSchemaVersion, BrainIdentity: BrainIdentity, Sequence: r.seq, Type: EventTypeThink, Timestamp: event.Timestamp, Event: &event}); err != nil { return thought, r.seq, err }
	}
	return thought, r.seq, nil
}

// CognitiveOutput is the neutral boundary from neural cognition to interpretation/policy.
type CognitiveOutput struct {
	BrainIdentity string
	Sequence uint64
	Timestamp time.Time
	RankedNodeIDs []knowledge.NodeID
	Activations map[knowledge.NodeID]float64
	Confidence map[knowledge.NodeID]float64
	Resonance float64
	PredictionError float64
	Prediction activation.Prediction
}

func (r *BrainRuntime) CognitiveProcess(event Event) (CognitiveOutput, error) {
	result, sequence, err := r.Process(event); if err != nil { return CognitiveOutput{}, err }
	now := event.Timestamp; if now.IsZero() { now = time.Now().UTC() }
	return cognitiveOutputFromResult(BrainIdentity, sequence, now, result), nil
}

func (r *BrainRuntime) CognitiveThink(cycles int) (CognitiveOutput, error) {
	thought, sequence, err := r.Think(cycles); if err != nil { return CognitiveOutput{}, err }
	return CognitiveOutput{BrainIdentity: BrainIdentity, Sequence: sequence, Timestamp: time.Now().UTC(), RankedNodeIDs: rankedNodeIDs(thought.RankedNodes), Activations: cloneNodeValues(thought.Activations), Confidence: cloneNodeValues(thought.Confidence), Resonance: thought.Resonance, PredictionError: thought.PredictionError, Prediction: thought.Prediction}, nil
}

func (r *BrainRuntime) LastSequence() uint64 {
	if r == nil { return 0 }
	r.mu.Lock(); defer r.mu.Unlock(); return r.seq
}

func cognitiveOutputFromResult(identity string, sequence uint64, now time.Time, result activation.Result) CognitiveOutput {
	return CognitiveOutput{BrainIdentity: identity, Sequence: sequence, Timestamp: now, RankedNodeIDs: rankedNodeIDs(result.RankedNodes), Activations: cloneNodeValues(result.Activations), Confidence: cloneNodeValues(result.Confidence), Resonance: result.Resonance}
}

func rankedNodeIDs(nodes []*knowledge.ConceptNode) []knowledge.NodeID { out := make([]knowledge.NodeID, 0, len(nodes)); for _, node := range nodes { if node != nil { out = append(out, node.ID) } }; return out }
func cloneNodeValues(in map[knowledge.NodeID]float64) map[knowledge.NodeID]float64 { if in == nil { return map[knowledge.NodeID]float64{} }; out := make(map[knowledge.NodeID]float64, len(in)); for id, value := range in { out[id] = value }; return out }
func cloneContext(in map[knowledge.NodeID]float64) map[knowledge.NodeID]float64 { if in == nil { return nil }; out := make(map[knowledge.NodeID]float64, len(in)); for id, value := range in { out[id] = value }; return out }
func cloneEvent(in Event) *Event { out := in; out.Stimulus = append([]string(nil), in.Stimulus...); out.Context = cloneContext(in.Context); return &out }
