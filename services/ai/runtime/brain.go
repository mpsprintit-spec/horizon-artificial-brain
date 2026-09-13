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

type CognitiveOutput struct {
	BrainIdentity   string
	Sequence        uint64
	Activations     map[knowledge.NodeID]float64
	Confidence      map[knowledge.NodeID]float64
	RankedNodes     []knowledge.NodeID
	Resonance       float64
	Prediction      map[knowledge.NodeID]float64
	PredictionConf  map[knowledge.NodeID]float64
	PredictionError float64
}

type BrainRuntime struct {
	Brain      *knowledge.Brain
	Activation *activation.Engine
	Learning   *learning.LearningUnit

	mu  sync.Mutex
	seq uint64
}

func NewBrainRuntime(brain *knowledge.Brain) *BrainRuntime {
	if brain == nil {
		brain = knowledge.NewBrain()
	}
	return &BrainRuntime{
		Brain:      brain,
		Activation: activation.NewEngine(brain),
		Learning:   learning.NewLearningUnit(brain),
	}
}

func (r *BrainRuntime) Process(event Event) (activation.Result, uint64, error) {
	if r == nil || r.Brain == nil || r.Activation == nil {
		return activation.Result{}, 0, errors.New("brain runtime is not initialized")
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if event.Timestamp.IsZero() {
		event.Timestamp = time.Now().UTC()
	}
	r.seq++
	sequence := r.seq
	result := r.Activation.ActivateWith(activation.Request{
		StimulusTokens: append([]string(nil), event.Stimulus...),
		ContextBoosts:  cloneContext(event.Context),
		Cycles:         event.Cycles,
		Now:            event.Timestamp,
	})
	return result, sequence, nil
}

// LearnExperience is the only runtime-owned entry point for new experience
// mutation. It serializes structural learning with cognition so the single
// persistent Brain cannot be concurrently mutated by an external adapter.
// LearningUnit's neural LearnExperience performs no semantic RelationKind
// assignment.
func (r *BrainRuntime) LearnExperience(experience learning.Experience, now time.Time) (uint64, error) {
	if r == nil || r.Brain == nil || r.Learning == nil {
		return 0, errors.New("brain runtime is not initialized")
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if now.IsZero() {
		now = time.Now().UTC()
	}
	r.Learning.LearnExperience(experience, now)
	r.seq++
	return r.seq, nil
}

func (r *BrainRuntime) CognitiveProcess(event Event) (CognitiveOutput, error) {
	result, sequence, err := r.Process(event)
	if err != nil {
		return CognitiveOutput{}, err
	}
	return cognitiveOutputFromResult(result, sequence), nil
}

func (r *BrainRuntime) Think(cycles int) (activation.ThoughtResult, uint64, error) {
	if r == nil || r.Brain == nil || r.Activation == nil {
		return activation.ThoughtResult{}, 0, errors.New("brain runtime is not initialized")
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	r.seq++
	sequence := r.seq
	return r.Activation.ThinkWithPrediction(cycles), sequence, nil
}

func (r *BrainRuntime) CognitiveThink(cycles int) (CognitiveOutput, error) {
	thought, sequence, err := r.Think(cycles)
	if err != nil {
		return CognitiveOutput{}, err
	}
	return CognitiveOutput{
		BrainIdentity:   BrainIdentity,
		Sequence:        sequence,
		Activations:     cloneFloatMap(thought.Activations),
		Confidence:      cloneFloatMap(thought.Confidence),
		RankedNodes:     rankedNodeIDs(thought.RankedNodes),
		Resonance:       thought.Resonance,
		Prediction:      cloneFloatMap(thought.Prediction.State),
		PredictionConf:  cloneFloatMap(thought.Prediction.Confidence),
		PredictionError: thought.PredictionError,
	}, nil
}

func cognitiveOutputFromResult(result activation.Result, sequence uint64) CognitiveOutput {
	return CognitiveOutput{
		BrainIdentity: BrainIdentity,
		Sequence:      sequence,
		Activations:   cloneFloatMap(result.Activations),
		Confidence:    cloneFloatMap(result.Confidence),
		RankedNodes:   rankedNodeIDs(result.RankedNodes),
		Resonance:     result.Resonance,
	}
}

func rankedNodeIDs(nodes []*knowledge.ConceptNode) []knowledge.NodeID {
	if len(nodes) == 0 {
		return nil
	}
	out := make([]knowledge.NodeID, 0, len(nodes))
	for _, node := range nodes {
		if node != nil {
			out = append(out, node.ID)
		}
	}
	return out
}

func cloneFloatMap(in map[knowledge.NodeID]float64) map[knowledge.NodeID]float64 {
	if len(in) == 0 {
		return nil
	}
	out := make(map[knowledge.NodeID]float64, len(in))
	for id, value := range in {
		out[id] = value
	}
	return out
}

func cloneContext(in map[knowledge.NodeID]float64) map[knowledge.NodeID]float64 {
	if len(in) == 0 {
		return nil
	}
	out := make(map[knowledge.NodeID]float64, len(in))
	for id, value := range in {
		out[id] = value
	}
	return out
}
