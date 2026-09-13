package runtime

import (
	"errors"
	"sync"
	"time"

	"github.com/project-horizon/horizon-core/services/ai/activation"
	"github.com/project-horizon/horizon-core/services/ai/knowledge"
	"github.com/project-horizon/horizon-core/services/ai/learning"
)

// Event is the runtime boundary for an experience entering Horizon.
// Runtime deliberately carries substrate-neutral stimulus and context; it does
// not classify semantic intent or select an answer.
type Event struct {
	ID        string
	Stimulus  []string
	Context   map[knowledge.NodeID]float64
	Cycles    int
	Timestamp time.Time
}

// BrainRuntime is the single runtime facade for the persistent Horizon brain.
// It owns ordering and access to the existing neural substrate. It does not
// create a second memory or a second intelligence.
type BrainRuntime struct {
	Brain      *knowledge.Brain
	Activation *activation.Engine
	Learning   *learning.LearningUnit

	mu      sync.Mutex
	seq     uint64
	lastSeq uint64
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

// Process advances the same persistent neural substrate for one ordered event.
// No semantic relation vocabulary, interpretation engine, or execution path is
// involved here.
func (r *BrainRuntime) Process(event Event) (activation.Result, uint64, error) {
	if r == nil || r.Brain == nil || r.Activation == nil {
		return activation.Result{}, 0, errors.New("brain runtime is not initialized")
	}

	r.mu.Lock()
	if event.Timestamp.IsZero() {
		event.Timestamp = time.Now().UTC()
	}
	r.seq++
	sequence := r.seq
	r.lastSeq = sequence
	r.mu.Unlock()

	result := r.Activation.ActivateWith(activation.Request{
		StimulusTokens: append([]string(nil), event.Stimulus...),
		ContextBoosts:  cloneContext(event.Context),
		Cycles:         event.Cycles,
		Now:            event.Timestamp,
	})
	return result, sequence, nil
}

// Think advances the existing internal state without a new external event.
func (r *BrainRuntime) Think(cycles int) (activation.ThoughtResult, uint64, error) {
	if r == nil || r.Brain == nil || r.Activation == nil {
		return activation.ThoughtResult{}, 0, errors.New("brain runtime is not initialized")
	}

	r.mu.Lock()
	r.seq++
	sequence := r.seq
	r.lastSeq = sequence
	r.mu.Unlock()

	return r.Activation.ThinkWithPrediction(cycles), sequence, nil
}

func (r *BrainRuntime) LastSequence() uint64 {
	if r == nil {
		return 0
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.lastSeq
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
