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
	ContextTokens []string
	DataTokens    []string
	Source    string
	Modality  string
	Cycles    int
	Timestamp time.Time
}

type BrainRuntime struct {
	brain      *knowledge.Brain
	activation *activation.Engine
	learning   *learning.LearningUnit
	eventLog   *EventLog
	clock      Clock
	mu         sync.Mutex
	seq        uint64
}

func NewBrainRuntime(brain *knowledge.Brain) *BrainRuntime {
	if brain == nil {
		brain = knowledge.NewBrain()
	}
	return &BrainRuntime{brain: brain, activation: activation.NewEngine(brain), learning: learning.NewLearningUnit(brain), clock: WallClock{}}
}

func (r *BrainRuntime) SetClock(clock Clock) {
	if r == nil { return }
	r.mu.Lock(); defer r.mu.Unlock()
	if clock == nil { r.clock = WallClock{} } else { r.clock = clock }
}

func (r *BrainRuntime) now() time.Time {
	if r.clock == nil { r.clock = WallClock{} }
	now := r.clock.Now()
	if now.IsZero() { now = time.Now().UTC() }
	return now.UTC()
}

func (r *BrainRuntime) SetEventLog(log *EventLog) {
	if r == nil { return }
	r.mu.Lock(); defer r.mu.Unlock()
	r.eventLog = log
}

// Process records the accepted event before mutating neural state. This gives
// the event log a write-ahead role: a failed log append cannot leave an
// unrecorded neural mutation behind.
func (r *BrainRuntime) Process(event Event) (activation.Result, uint64, error) {
	if r == nil || r.brain == nil || r.activation == nil { return activation.Result{}, 0, errors.New("brain runtime is not initialized") }
	r.mu.Lock(); defer r.mu.Unlock()
	if event.Timestamp.IsZero() { event.Timestamp = r.now() }
	nextSeq := r.seq + 1
	if r.eventLog != nil {
		if err := r.eventLog.Append(LoggedEvent{SchemaVersion: EventLogSchemaVersion, BrainIdentity: BrainIdentity, Sequence: nextSeq, Type: EventTypeProcess, Timestamp: event.Timestamp, Event: cloneEvent(event)}); err != nil { return activation.Result{}, r.seq, err }
	}
	result := r.activation.ActivateWith(activation.Request{StimulusTokens: append([]string(nil), event.Stimulus...), ContextBoosts: cloneContext(event.Context), Cycles: event.Cycles, Now: event.Timestamp})
	r.seq = nextSeq
	return result, r.seq, nil
}

// LearnExperience uses the same write-ahead boundary as Process. The durable
// experience record exists before the neural substrate is modified.
func (r *BrainRuntime) LearnExperience(experience learning.Experience, now time.Time) (uint64, error) {
	if r == nil || r.brain == nil || r.learning == nil { return 0, errors.New("brain runtime is not initialized") }
	r.mu.Lock(); defer r.mu.Unlock()
	if now.IsZero() { now = r.now() }
	nextSeq := r.seq + 1
	copyExperience := experience
	if r.eventLog != nil {
		if err := r.eventLog.Append(LoggedEvent{SchemaVersion: EventLogSchemaVersion, BrainIdentity: BrainIdentity, Sequence: nextSeq, Type: EventTypeLearn, Timestamp: now, Experience: &copyExperience}); err != nil { return r.seq, err }
	}
	r.learning.LearnExperience(experience, now)
	r.seq = nextSeq
	return r.seq, nil
}

func (r *BrainRuntime) Think(cycles int) (activation.ThoughtResult, uint64, error) { return r.ThinkAt(cycles, time.Time{}) }

func (r *BrainRuntime) ThinkAt(cycles int, timestamp time.Time) (activation.ThoughtResult, uint64, error) {
	if r == nil || r.brain == nil || r.activation == nil { return activation.ThoughtResult{}, 0, errors.New("brain runtime is not initialized") }
	r.mu.Lock(); defer r.mu.Unlock()
	now := timestamp
	if now.IsZero() { now = r.now() }
	nextSeq := r.seq + 1
	if r.eventLog != nil {
		event := Event{Cycles: cycles, Timestamp: now}
		if err := r.eventLog.Append(LoggedEvent{SchemaVersion: EventLogSchemaVersion, BrainIdentity: BrainIdentity, Sequence: nextSeq, Type: EventTypeThink, Timestamp: now, Event: &event}); err != nil { return activation.ThoughtResult{}, r.seq, err }
	}
	thought := r.activation.ThinkWithPredictionAt(cycles, now)
	r.seq = nextSeq
	return thought, r.seq, nil
}

type CognitiveOutput struct {
	BrainIdentity  string
	Sequence       uint64
	Timestamp      time.Time
	RankedNodeIDs  []knowledge.NodeID
	Activations    map[knowledge.NodeID]float64
	Confidence     map[knowledge.NodeID]float64
	Resonance      float64
	PredictionError float64
	Prediction     activation.Prediction
}

func (r *BrainRuntime) CognitiveProcess(event Event) (CognitiveOutput, error) {
	result, sequence, err := r.Process(event)
	if err != nil { return CognitiveOutput{}, err }
	now := event.Timestamp
	if now.IsZero() { now = r.clock.Now() }
	return cognitiveOutputFromResult(BrainIdentity, sequence, now, result), nil
}

func (r *BrainRuntime) CognitiveThink(cycles int) (CognitiveOutput, error) {
	thought, sequence, err := r.Think(cycles)
	if err != nil { return CognitiveOutput{}, err }
	return CognitiveOutput{BrainIdentity: BrainIdentity, Sequence: sequence, Timestamp: r.now(), RankedNodeIDs: rankedNodeIDs(thought.RankedNodes), Activations: cloneNodeValues(thought.Activations), Confidence: cloneNodeValues(thought.Confidence), Resonance: thought.Resonance, PredictionError: thought.PredictionError, Prediction: thought.Prediction}, nil
}

func (r *BrainRuntime) LastSequence() uint64 {
	if r == nil { return 0 }
	r.mu.Lock(); defer r.mu.Unlock(); return r.seq
}

func cognitiveOutputFromResult(identity string, sequence uint64, now time.Time, result activation.Result) CognitiveOutput {
	return CognitiveOutput{BrainIdentity: identity, Sequence: sequence, Timestamp: now, RankedNodeIDs: rankedNodeIDs(result.RankedNodes), Activations: cloneNodeValues(result.Activations), Confidence: cloneNodeValues(result.Confidence), Resonance: result.Resonance}
}

func rankedNodeIDs(nodes []*knowledge.ConceptNode) []knowledge.NodeID {
	out := make([]knowledge.NodeID, 0, len(nodes)); for _, node := range nodes { if node != nil { out = append(out, node.ID) } }; return out
}
func cloneNodeValues(in map[knowledge.NodeID]float64) map[knowledge.NodeID]float64 {
	if in == nil { return map[knowledge.NodeID]float64{} }
	out := make(map[knowledge.NodeID]float64, len(in)); for id, value := range in { out[id] = value }; return out
}
func cloneContext(in map[knowledge.NodeID]float64) map[knowledge.NodeID]float64 {
	if in == nil { return nil }
	out := make(map[knowledge.NodeID]float64, len(in)); for id, value := range in { out[id] = value }; return out
}
func cloneEvent(in Event) *Event {
	out := in
	out.Stimulus = append([]string(nil), in.Stimulus...)
	out.Context = cloneContext(in.Context)
	out.ContextTokens = append([]string(nil), in.ContextTokens...)
	out.DataTokens = append([]string(nil), in.DataTokens...)
	return &out
}
