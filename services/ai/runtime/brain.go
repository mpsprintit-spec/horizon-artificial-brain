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
	Brain      *knowledge.Brain
	Activation *activation.Engine
	Learning   *learning.LearningUnit
	mu         sync.Mutex
	seq        uint64
}

func NewBrainRuntime(brain *knowledge.Brain) *BrainRuntime {
	if brain == nil {
		brain = knowledge.NewBrain()
	}
	return &BrainRuntime{Brain: brain, Activation: activation.NewEngine(brain), Learning: learning.NewLearningUnit(brain)}
}

func (r *BrainRuntime) Process(event Event) (activation.Result, uint64, error) {
	if r == nil || r.Brain == nil || r.Activation == nil {
		return activation.Result{}, 0, errors.New("brain runtime is not initialized")
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if event.Timestamp.IsZero() { event.Timestamp = time.Now().UTC() }
	r.seq++
	result := r.Activation.ActivateWith(activation.Request{StimulusTokens: append([]string(nil), event.Stimulus...), ContextBoosts: cloneContext(event.Context), Cycles: event.Cycles, Now: event.Timestamp})
	return result, r.seq, nil
}

func (r *BrainRuntime) LearnExperience(experience learning.Experience, now time.Time) (uint64, error) {
	if r == nil || r.Brain == nil || r.Learning == nil { return 0, errors.New("brain runtime is not initialized") }
	r.mu.Lock()
	defer r.mu.Unlock()
	if now.IsZero() { now = time.Now().UTC() }
	r.Learning.LearnExperience(experience, now)
	r.seq++
	return r.seq, nil
}

func (r *BrainRuntime) Think(cycles int) (activation.ThoughtResult, uint64, error) {
	if r == nil || r.Brain == nil || r.Activation == nil { return activation.ThoughtResult{}, 0, errors.New("brain runtime is not initialized") }
	r.mu.Lock()
	defer r.mu.Unlock()
	r.seq++
	return r.Activation.ThinkWithPrediction(cycles), r.seq, nil
}

// CognitiveOutput is the single neutral boundary from neural cognition to
// interpretation/policy. It contains neural state only.
type CognitiveOutput struct {
	BrainIdentity   string
	Sequence        uint64
	Timestamp       time.Time
	RankedNodeIDs   []knowledge.NodeID
	Activations     map[knowledge.NodeID]float64
	Confidence      map[knowledge.NodeID]float64
	Resonance       float64
	PredictionError float64
	Prediction      activation.Prediction
}

func (r *BrainRuntime) CognitiveProcess(event Event) (CognitiveOutput, error) {
	result, sequence, err := r.Process(event)
	if err != nil { return CognitiveOutput{}, err }
	now := event.Timestamp
	if now.IsZero() { now = time.Now().UTC() }
	return cognitiveOutputFromResult(BrainIdentity, sequence, now, result), nil
}

func (r *BrainRuntime) CognitiveThink(cycles int) (CognitiveOutput, error) {
	thought, sequence, err := r.Think(cycles)
	if err != nil { return CognitiveOutput{}, err }
	return CognitiveOutput{
		BrainIdentity: BrainIdentity, Sequence: sequence, Timestamp: time.Now().UTC(),
		RankedNodeIDs: rankedNodeIDs(thought.RankedNodes), Activations: cloneNodeValues(thought.Activations),
		Confidence: cloneNodeValues(thought.Confidence), Resonance: thought.Resonance,
		PredictionError: thought.PredictionError, Prediction: thought.Prediction,
	}, nil
}

func (r *BrainRuntime) LastSequence() uint64 {
	if r == nil { return 0 }
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.seq
}

func cognitiveOutputFromResult(identity string, sequence uint64, now time.Time, result activation.Result) CognitiveOutput {
	return CognitiveOutput{BrainIdentity: identity, Sequence: sequence, Timestamp: now, RankedNodeIDs: rankedNodeIDs(result.RankedNodes), Activations: cloneNodeValues(result.Activations), Confidence: cloneNodeValues(result.Confidence), Resonance: result.Resonance}
}

func rankedNodeIDs(nodes []*knowledge.ConceptNode) []knowledge.NodeID {
	out := make([]knowledge.NodeID, 0, len(nodes))
	for _, node := range nodes { if node != nil { out = append(out, node.ID) } }
	return out
}

func cloneNodeValues(in map[knowledge.NodeID]float64) map[knowledge.NodeID]float64 {
	if in == nil { return map[knowledge.NodeID]float64{} }
	out := make(map[knowledge.NodeID]float64, len(in))
	for id, value := range in { out[id] = value }
	return out
}

func cloneContext(in map[knowledge.NodeID]float64) map[knowledge.NodeID]float64 {
	if in == nil { return nil }
	out := make(map[knowledge.NodeID]float64, len(in))
	for id, value := range in { out[id] = value }
	return out
}
