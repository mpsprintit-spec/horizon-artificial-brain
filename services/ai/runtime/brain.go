package runtime

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/project-horizon/horizon-core/services/ai/activation"
	"github.com/project-horizon/horizon-core/services/ai/dnf"
	"github.com/project-horizon/horizon-core/services/ai/knowledge"
	"github.com/project-horizon/horizon-core/services/ai/learning"
)

const BrainIdentity = "horizon-primary-brain"
const GroundingThreshold = 0.90

var ErrRuntimeRecoveryRequired = errors.New("runtime recovery required")

// Call only while r.mu is held. A poisoned runtime must be discarded and
// rebuilt from a verified checkpoint and committed event log.
func (r *BrainRuntime) ensureWritableLocked() error {
	if r.recoveryRequired { return ErrRuntimeRecoveryRequired }
	return nil
}

// markRecoveryRequiredLocked invalidates both cognition and live monitoring.
// Call with r.mu held, including uncertain commits and successful retirement.
// Failed recovery validation must not call this on the still-active runtime.
func (r *BrainRuntime) markRecoveryRequiredLocked() {
	r.recoveryRequired = true
	r.telemetry.Close()
}

func (r *BrainRuntime) RecoveryRequired() bool {
	if r == nil { return false }
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.recoveryRequired
}

type Event struct {
	NeuralTrace *knowledge.ObservationTrace `json:"neural_trace,omitempty"`
	ID string `json:"id"`
	Stimulus []string `json:"stimulus,omitempty"`
	StimulusNodeIDs []knowledge.NodeID `json:"stimulus_node_ids,omitempty"`
	Context map[knowledge.NodeID]float64 `json:"context,omitempty"`
	ContextTokens []string `json:"context_tokens,omitempty"`
	DataTokens []string `json:"data_tokens,omitempty"`
	Source string `json:"source,omitempty"`
	Modality string `json:"modality,omitempty"`
	Cycles int `json:"cycles"`
	Timestamp time.Time `json:"timestamp"`
	// PredictionOverride is used only at an explicit causal boundary where
	// the caller captured a prediction before an external action. It prevents
	// outcome processing from silently substituting a newer recurrent state.
	PredictionOverride *activation.Prediction
	Observation *ObservationEnvelope `json:"observation,omitempty"`
	SensorObservation *SensorObservation `json:"sensor_observation,omitempty"`
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
	sensorMu sync.Mutex
	// Set when a sensor journal operation may have left an incomplete write.
	recoveryRequired bool
	// Monotonic ownership claim. Managed runtime lifetime and journal binding
	// can only be changed by its supervisor, not by cached runtime references.
	supervised bool
	// Replay selects the recorded adapter contract on a private runtime.
	// Serving runtimes use WrittenTextEncoding; this is not persistent Brain state.
	surfaceEncoding knowledge.SurfaceEncoding
	dynamicsEncoding knowledge.DynamicsEncoding
	seq uint64
	lastCognitiveState CognitiveState
	lastObservationPopulation []knowledge.NodeID
	lastObservationAt time.Time
	inquiryValence map[InquiryAction]float64
	telemetry *TelemetryHub
	processedEvents map[string]uint64
}

func NewBrainRuntime(brain *knowledge.Brain) *BrainRuntime {
	if brain == nil { brain = knowledge.NewBootstrapBrain() }
	if !brain.BindRuntime() { return nil }
	fabric, err := dnf.NewFabric(brain)
	if err != nil { return nil }
	return &BrainRuntime{brain: brain, dnf: fabric, activation: activation.NewEngine(brain), learning: learning.NewLearningUnit(brain), promotion: learning.NewPromotionEngine(brain, learning.DefaultLearningPolicy()), evidence: learning.NewEvidenceLedger(), actions: make(map[string]ActionBinding), inquiryValence: make(map[InquiryAction]float64), clock: WallClock{}, telemetry: NewTelemetryHub(256), processedEvents: make(map[string]uint64), surfaceEncoding: knowledge.WrittenTextEncoding, dynamicsEncoding: knowledge.OrderedDynamics}
}
func (r *BrainRuntime) telemetryEventLocked(eventType string, at time.Time, event *Event, observation *ObservationEnvelope, outcome *OutcomeEvent) TelemetryEvent {
	payload := struct { Type string `json:"type"`; Sequence uint64 `json:"sequence"`; Event *Event `json:"event,omitempty"`; Observation *ObservationEnvelope `json:"observation,omitempty"`; Outcome *OutcomeEvent `json:"outcome,omitempty"` }{eventType, r.seq, event, observation, outcome}
	data, _ := json.Marshal(payload); eh := sha256.Sum256(data)
	brain, _ := r.brain.CanonicalJSON(); bh := sha256.Sum256(brain)
	r.brain.RLock()
	plasticity := cloneMonitorFloatMap(r.brain.BrainState.PlasticityState)
	r.brain.RUnlock()
	return TelemetryEvent{Type:eventType, BrainIdentity:BrainIdentity, StateRevision:r.seq, Timestamp:at.UTC(), EventHash:"sha256:"+hex.EncodeToString(eh[:]), CanonicalStateHash:"sha256:"+hex.EncodeToString(bh[:]), Event:event, Observation:observation, Outcome:outcome, Plasticity:plasticity}
}
func (r *BrainRuntime) Telemetry() *TelemetryHub {
	if r == nil {
		return nil
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.recoveryRequired {
		return nil
	}
	if r.telemetry == nil {
		r.telemetry = NewTelemetryHub(256)
	}
	return r.telemetry
}
func (r *BrainRuntime) PublishObservationTelemetry(envelope ObservationEnvelope, output CognitiveOutput) {
	if r == nil {
		return
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.recoveryRequired {
		return
	}
	if r.telemetry == nil {
		r.telemetry = NewTelemetryHub(256)
	}
	envelope.Sequence = output.Sequence
	e := r.telemetryEventLocked("observation", output.Timestamp, nil, &envelope, nil)
	e.StateDelta = &output.StateDelta
	e.PredictionError = &output.PredictionError
	r.telemetry.Publish(e)
}
func (r *BrainRuntime) SaveBrain(path string) error {
	if r == nil || r.brain == nil { return errors.New("brain runtime is not initialized") }
	if strings.TrimSpace(path) == "" { return errors.New("brain path is empty") }
	r.mu.Lock()
	defer r.mu.Unlock()
	if err := r.ensureWritableLocked(); err != nil { return err }
	return r.brain.Save(path)
}
func (r *BrainRuntime) LoadBrain(path string) error {
	if r == nil || r.brain == nil { return errors.New("brain runtime is not initialized") }
	if strings.TrimSpace(path) == "" { return errors.New("brain path is empty") }
	r.mu.Lock()
	defer r.mu.Unlock()
	if err := r.ensureWritableLocked(); err != nil { return err }
	// Manual brain import resets the local event sequence. It cannot be
	// reconciled with an existing attached journal; use checkpoint recovery.
	if r.eventLog != nil { return errors.New("manual brain load is forbidden while an event journal is attached; use verified checkpoint recovery") }
	if err := r.brain.Load(path); err != nil { return err }
	r.seq = 0
	r.lastCognitiveState = CognitiveState{}
	r.lastObservationPopulation = nil
	r.lastObservationAt = time.Time{}
	r.processedEvents = make(map[string]uint64)
	return nil
}
// RestoreEventIndex is only a standalone recovery/bootstrap operation. A
// writer-attached runtime must obtain its sequence from committed replay, not
// from an externally supplied index that could skip journal transactions.
// Validate the entire index before changing any runtime state.
func (r *BrainRuntime) RestoreEventIndex(events []LoggedEvent) error {
	if r == nil || r.brain == nil { return errors.New("brain runtime is not initialized") }
	r.mu.Lock()
	defer r.mu.Unlock()
	if err := r.ensureWritableLocked(); err != nil { return err }
	if r.supervised { return ErrManagedRuntimeAccess }
	if r.eventLog != nil { return errors.New("cannot restore event index while journal writer is attached") }
	if r.seq != 0 { return errors.New("cannot restore event index into a nonempty runtime") }
	index := make(map[string]uint64)
	expected := uint64(1)
	for _, event := range events {
		if !supportedEventLogSchema(event.SchemaVersion) || event.BrainIdentity != BrainIdentity {
			return errors.New("event index contains incompatible brain or schema")
		}
		if event.Sequence != expected {
			return fmt.Errorf("event index discontinuity: got %d want %d", event.Sequence, expected)
		}
		if input := processedJournalEvent(event); input != nil && input.ID != "" {
			if _, exists := index[input.ID]; exists { return fmt.Errorf("duplicate event ID %q in restored event index", input.ID) }
			index[input.ID] = event.Sequence
		}
		expected++
	}
	r.processedEvents = index
	r.seq = uint64(len(events))
	return nil
}

// DNF is a legacy standalone escape hatch. Managed and journaled runtimes
// expose detached inspection instead of a Fabric with a mutable Brain pointer.
func (r *BrainRuntime) DNF() *dnf.Fabric {
	if r == nil {
		return nil
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.ensureWritableLocked() != nil || r.supervised || r.eventLog != nil {
		return nil
	}
	return r.dnf
}

func (r *BrainRuntime) GroundObservation(token, source, modality string) (knowledge.GroundedRepresentation, error) {
	if r == nil || r.brain == nil { return knowledge.GroundedRepresentation{}, errors.New("brain runtime is not initialized") }
	r.mu.Lock()
	defer r.mu.Unlock()
	if err := r.ensureWritableLocked(); err != nil { return knowledge.GroundedRepresentation{}, err }
	if r.eventLog != nil { return knowledge.GroundedRepresentation{}, errors.New("standalone grounding is forbidden with an attached journal; use ProcessObservation") }
	return r.brain.GroundObservationWithEncodingAt(token, source, modality, GroundingThreshold, r.nowLocked(), r.surfaceEncoding)
}

// ActivateBootstrap rehydrates the persistent bootstrap populations into the
// runtime activation state after the canonical brain has been loaded. It does
// not create semantic definitions or additional memory; it only reactivates
// the numeric populations already persisted in BrainState.
func (r *BrainRuntime) ActivateBootstrap(now time.Time) (activation.Result, error) {
	if r == nil || r.brain == nil || r.activation == nil {
		return activation.Result{}, errors.New("brain runtime is not initialized")
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if err := r.ensureWritableLocked(); err != nil { return activation.Result{}, err }
	if now.IsZero() { now = r.nowLocked() }

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

	result, _, err := r.processWithPersistenceLocked(Event{Context: context, Cycles: 1, Timestamp: now}, true)
	return result, err
}

// RegisterActionBinding must validate its targets and commit the binding
// under the same runtime mutation lock. Otherwise a concurrent reload or a
// supervisor retirement could invalidate the validation before the write.
func (r *BrainRuntime) RegisterActionBinding(binding ActionBinding) error {
	if r == nil || r.brain == nil { return errors.New("brain runtime is not initialized") }
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.registerActionBindingLocked(binding)
}

func (r *BrainRuntime) actionBinding(requestID string) (ActionBinding, bool) { if r == nil { return ActionBinding{}, false }; r.mu.Lock(); defer r.mu.Unlock(); binding, ok := r.actions[requestID]; if !ok { return ActionBinding{}, false }; binding.TargetNodeIDs = append([]knowledge.NodeID(nil), binding.TargetNodeIDs...); binding.Synapses = append([]SynapseBinding(nil), binding.Synapses...); return binding, true }
func (r *BrainRuntime) SetClock(clock Clock) { if r == nil { return }; r.mu.Lock(); defer r.mu.Unlock(); if r.recoveryRequired { return }; if clock == nil { r.clock = WallClock{} } else { r.clock = clock } }
func (r *BrainRuntime) nowLocked() time.Time { if r.clock == nil { r.clock = WallClock{} }; now := r.clock.Now(); if now.IsZero() { now = time.Now().UTC() }; return now.UTC() }
func (r *BrainRuntime) now() time.Time { if r == nil { return time.Now().UTC() }; r.mu.Lock(); defer r.mu.Unlock(); return r.nowLocked() }
// SetEventLog is retained for standalone callers. It never changes the writer
// of a managed, retired or poisoned runtime. Managed integrations use the
// owner's AttachEventLog or verified recovery.
func (r *BrainRuntime) SetEventLog(log *EventLog) { if r == nil { return }; r.mu.Lock(); defer r.mu.Unlock(); if r.recoveryRequired || r.supervised { return }; r.eventLog = log }
// HasProcessedEvent reports whether an event ID is already in the durable runtime index.
// It is used by adapters to reject duplicate samples before they mutate the neural substrate.
func (r *BrainRuntime) HasProcessedEvent(id string) bool {
	if r == nil || strings.TrimSpace(id) == "" { return false }
	r.mu.Lock()
	defer r.mu.Unlock()
	_, exists := r.processedEvents[id]
	return exists
}

func (r *BrainRuntime) Process(event Event) (activation.Result, uint64, error) {
	if r == nil || r.brain == nil || r.activation == nil { return activation.Result{}, 0, errors.New("brain runtime is not initialized") }
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.processLocked(event)
}

func (r *BrainRuntime) processLocked(event Event) (activation.Result, uint64, error) {
	return r.processWithPersistenceLocked(event, true)
}

func (r *BrainRuntime) processWithPersistenceLocked(event Event, persist bool) (activation.Result, uint64, error) {
	if err := r.ensureWritableLocked(); err != nil {
		return activation.Result{}, r.seq, err
	}
	if event.ID != "" {
		if _, exists := r.processedEvents[event.ID]; exists {
			return activation.Result{}, r.seq, fmt.Errorf("duplicate event ID %q", event.ID)
		}
	}
	if event.Timestamp.IsZero() {
		event.Timestamp = r.nowLocked()
	}
	if err := r.validateNeuralEventLocked(event); err != nil { return activation.Result{}, r.seq, err }
	if persist {
		if event.SensorObservation != nil {
			return activation.Result{}, r.seq, errors.New("use the sensor observation boundary for sensor events")
		}
		event = *cloneEvent(event)
		var result activation.Result
		err := r.runJournalCommandLocked(LoggedEvent{Type: EventTypeProcess, Timestamp: event.Timestamp, Event: cloneEvent(event)}, func() error {
			var applyErr error
			result, _, applyErr = r.processWithPersistenceLocked(event, false)
			return applyErr
		})
		if err == nil && r.telemetry != nil && event.Observation == nil {
			r.telemetry.Publish(r.telemetryEventLocked(EventTypeProcess, event.Timestamp, &event, nil, nil))
		}
		return result, r.seq, err
	}
	nextSeq := r.seq + 1
	result := r.activation.ActivateWith(activation.Request{
		SurfaceEncoding: &r.surfaceEncoding,
		DynamicsEncoding: &r.dynamicsEncoding,
		CarryPreviousContext: knowledge.CanonicalSurfaceModality(event.Modality) == "language",
		NeuralTrace: event.NeuralTrace.Clone(),
		StimulusTokens:     append([]string(nil), event.Stimulus...),
		StimulusNodeIDs:    append([]knowledge.NodeID(nil), event.StimulusNodeIDs...),
		ContextBoosts:      cloneContext(event.Context),
		PredictionOverride: clonePrediction(event.PredictionOverride),
		Cycles:             event.Cycles,
		Now:                event.Timestamp,
	})
	r.seq = nextSeq
	if event.ID != "" {
		r.processedEvents[event.ID] = r.seq
	}
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
	r.mu.Lock()
	defer r.mu.Unlock()
	if err := r.ensureWritableLocked(); err != nil { return err }
	if r.eventLog != nil { return errors.New("standalone temporal reinforcement is forbidden with an attached journal; use ProcessObservation") }
	return r.learnObservedTransitionLocked(current, now)
}

// learnObservedTransitionLocked is part of the serialized runtime mutation
// boundary. Callers must hold r.mu throughout the preceding sensor projection
// and cognitive processing stages.
func (r *BrainRuntime) learnObservedTransitionLocked(current []knowledge.NodeID, now time.Time) error {
	if err := r.ensureWritableLocked(); err != nil { return err }
	if now.IsZero() {
		now = r.nowLocked()
	}
	if len(current) == 0 {
		r.lastObservationPopulation = nil
		r.lastObservationAt = time.Time{}
		return nil
	}

	previous := append([]knowledge.NodeID(nil), r.lastObservationPopulation...)
	previousAt := r.lastObservationAt
	currentCopy := append([]knowledge.NodeID(nil), current...)

	if len(previous) > 0 {
		// Commit the new temporal predecessor only after reinforcement succeeds.
		if err := r.brain.ReinforceTransitionPopulation(previous, currentCopy, previousAt, now, 0.25); err != nil {
			return err
		}
	}
	r.lastObservationPopulation = currentCopy
	r.lastObservationAt = now
	return nil
}

func (r *BrainRuntime) LearnExperience(experience learning.Experience, now time.Time) (uint64, error) {
	if r == nil || r.brain == nil || r.learning == nil || r.dnf == nil {
		return 0, errors.New("brain runtime is not initialized")
	}
	var err error
	experience, err = freezeExperience(experience)
	if err != nil { return 0, err }
	r.mu.Lock()
	defer r.mu.Unlock()
	if now.IsZero() {
		now = r.nowLocked()
	}
	now = now.UTC()
	if err = r.ensureWritableLocked(); err != nil { return r.seq, err }
	if err = r.validateExperienceLocked(experience); err != nil { return r.seq, err }
	err = r.runJournalCommandLocked(LoggedEvent{Type: EventTypeLearn, Timestamp: now, Experience: &experience}, func() error {
		r.learning.LearnExperienceWithDynamics(experience, now, r.surfaceEncoding, r.dynamicsEncoding)
		r.seq++
		return nil
	})
	return r.seq, err
}

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
	return r.thinkAtWithContextPersistenceLocked(cycles, context, timestamp, true)
}

func (r *BrainRuntime) thinkAtWithContextPersistenceLocked(cycles int, context map[knowledge.NodeID]float64, timestamp time.Time, persist bool) (activation.ThoughtResult, uint64, error) {
	if err := r.ensureWritableLocked(); err != nil {
		return activation.ThoughtResult{}, r.seq, err
	}
	now := timestamp
	if now.IsZero() {
		now = r.nowLocked()
	}
	if persist {
		context = cloneContext(context)
		event := Event{Cycles: cycles, Timestamp: now, Context: context}
		var thought activation.ThoughtResult
		err := r.runJournalCommandLocked(LoggedEvent{Type: EventTypeThink, Timestamp: now, Event: &event}, func() error {
			var applyErr error
			thought, _, applyErr = r.thinkAtWithContextPersistenceLocked(cycles, context, now, false)
			return applyErr
		})
		return thought, r.seq, err
	}
	nextSeq := r.seq + 1
	thought := r.activation.ThinkWithContextDynamicsAt(cycles, context, now, r.dynamicsEncoding)
	r.seq = nextSeq
	return thought, r.seq, nil
}

type CognitiveOutput struct { BrainIdentity string; Sequence uint64; Timestamp time.Time; RankedNodeIDs []knowledge.NodeID; Activations map[knowledge.NodeID]float64; Confidence map[knowledge.NodeID]float64; Resonance float64; PredictionError float64; Prediction activation.Prediction; StateDelta CognitiveStateDelta }
func (r *BrainRuntime) CognitiveProcess(event Event) (CognitiveOutput, error) {
	if r == nil || r.brain == nil || r.activation == nil {
		return CognitiveOutput{}, errors.New("brain runtime is not initialized")
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.cognitiveProcessLocked(event)
}

// cognitiveProcessLocked keeps event persistence, activation, prediction,
// learning signals and state deltas in one runtime mutation boundary.
// Callers must already hold r.mu.
func (r *BrainRuntime) cognitiveProcessLocked(event Event) (CognitiveOutput, error) {
	return r.cognitiveProcessWithPersistenceLocked(event, true)
}

func (r *BrainRuntime) cognitiveProcessWithPersistenceLocked(event Event, persist bool) (CognitiveOutput, error) {
	if err := r.ensureWritableLocked(); err != nil {
		return CognitiveOutput{}, err
	}
	if event.Timestamp.IsZero() {
		event.Timestamp = r.nowLocked()
	}
	if err := r.validateNeuralEventLocked(event); err != nil { return CognitiveOutput{}, err }
	if persist {
		if event.SensorObservation != nil {
			return CognitiveOutput{}, errors.New("use the sensor observation boundary for sensor events")
		}
		if event.ID != "" {
			if _, exists := r.processedEvents[event.ID]; exists {
				return CognitiveOutput{}, fmt.Errorf("duplicate event ID %q", event.ID)
			}
		}
		event = *cloneEvent(event)
		var output CognitiveOutput
		err := r.runJournalCommandLocked(LoggedEvent{Type: EventTypeCognitiveProcess, Timestamp: event.Timestamp, Event: cloneEvent(event)}, func() error {
			var err error
			output, err = r.cognitiveProcessWithPersistenceLocked(event, false)
			return err
		})
		if err != nil {
			return CognitiveOutput{}, err
		}
		if r.telemetry != nil && event.Observation == nil {
			e := r.telemetryEventLocked(EventTypeProcess, event.Timestamp, &event, nil, nil)
			e.StateDelta = &output.StateDelta
			e.PredictionError = &output.PredictionError
			r.telemetry.Publish(e)
		}
		return output, nil
	}

	result, sequence, err := r.processWithPersistenceLocked(event, false)
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
	if r == nil || r.brain == nil || r.activation == nil { return CognitiveOutput{}, errors.New("brain runtime is not initialized") }
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.cognitiveThinkAtLocked(cycles, r.nowLocked(), true)
}

func (r *BrainRuntime) cognitiveThinkAtLocked(cycles int, now time.Time, persist bool) (CognitiveOutput, error) {
	if err := r.ensureWritableLocked(); err != nil {
		return CognitiveOutput{}, err
	}
	if persist {
		var output CognitiveOutput
		event := Event{Cycles: cycles, Timestamp: now}
		err := r.runJournalCommandLocked(LoggedEvent{Type: EventTypeCognitiveThink, Timestamp: now, Event: &event}, func() error {
			var err error
			output, err = r.cognitiveThinkAtLocked(cycles, now, false)
			return err
		})
		if err != nil {
			return CognitiveOutput{}, err
		}
		if r.telemetry != nil {
			e := r.telemetryEventLocked(EventTypeThink, now, nil, nil, nil)
			e.StateDelta = &output.StateDelta
			e.PredictionError = &output.PredictionError
			r.telemetry.Publish(e)
		}
		return output, nil
	}
	curiosityContext := r.applyCuriosityDrive(now)
	thought, sequence, err := r.thinkAtWithContextPersistenceLocked(cycles, curiosityContext, now, false)
	if err != nil {
		return CognitiveOutput{}, err
	}
	output := CognitiveOutput{BrainIdentity: BrainIdentity, Sequence: sequence, Timestamp: now, RankedNodeIDs: rankedNodeIDs(thought.RankedNodes), Activations: cloneNodeValues(thought.Activations), Confidence: cloneNodeValues(thought.Confidence), Resonance: thought.Resonance, PredictionError: thought.PredictionError, Prediction: thought.Prediction}
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

	// Every autonomous cycle records a curiosity heartbeat even when the
	// current policy produces no actionable candidate. This separates the
	// existence of an autonomous cognition cycle from the optional drive/target.
	r.brain.Lock()
	if r.brain.BrainState.CuriosityState == nil {
		r.brain.BrainState.CuriosityState = map[string]float64{}
	}
	r.brain.BrainState.CuriosityState["last_update_unix"] = float64(now.UnixNano())
	r.brain.Unlock()

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
		if score <= 0 {
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

	// Planning is one runtime transaction. The selected sequence, action
	// bindings, learned inquiry model, and persisted selection must come from
	// the same runtime snapshot; an autonomous cognition cycle must not
	// interleave between those steps.
	r.mu.Lock()
	defer r.mu.Unlock()
	if err := r.ensureWritableLocked(); err != nil { return InquiryAgenda{}, err }
	if r.eventLog != nil { return InquiryAgenda{}, errors.New("standalone inquiry planning is forbidden with an attached journal; use ProcessObservation") }
	return r.planInquiryLocked(uncertainty, at)
}

func (r *BrainRuntime) planInquiryLocked(uncertainty float64, at time.Time) (InquiryAgenda, error) {
	if err := r.ensureWritableLocked(); err != nil { return InquiryAgenda{}, err }

	if at.IsZero() {
		at = r.nowLocked()
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
		targetSet := r.inquiryTargetSetLocked(candidates[i].Action)
		if value, modeled, err := r.predictInquiryInformationValueWithTargets(candidates[i].Action, targetSet, uncertainty, at); err == nil && modeled {
			candidates[i].ExpectedInformationGain = value
		} else {
			candidates[i].ExpectedInformationGain = 0
		}

		candidates[i].PriorExperience = r.inquiryPriorExperienceLocked(candidates[i].Action, candidates[i].PriorExperience)
		switch candidates[i].Action {
		case InquiryReobserve, InquiryFocus:
			candidates[i].PriorExperience = clamp01(candidates[i].PriorExperience + 0.20*clamp01(policy.RepeatObservationBias))
		case InquiryChangeView, InquiryImitate, InquirySafeManipulation:
			candidates[i].PriorExperience = clamp01(candidates[i].PriorExperience + 0.20*clamp01(policy.ExplorationBias))
		case InquiryWait:
			candidates[i].PriorExperience = clamp01(candidates[i].PriorExperience + 0.20*clamp01(policy.DeferConclusionBias))
		}
	}

	agenda, err := BuildInquiryAgenda(BrainIdentity, r.seq, at, candidates, DefaultInquiryPolicy())
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

	// Completion is serialized with inquiry planning and autonomous cognition.
	// The observation outcome therefore updates the same runtime transaction
	// boundary that owns the pending inquiry trajectory.
	r.mu.Lock()
	defer r.mu.Unlock()
	if err := r.ensureWritableLocked(); err != nil { return err }
	if r.eventLog != nil { return errors.New("standalone inquiry completion is forbidden with an attached journal; use ProcessInquiryObservation") }

	if at.IsZero() {
		at = r.nowLocked()
	}
	r.completeInquiryLocked(observedInformationGain, predictionError, at)
	return nil
}

func (r *BrainRuntime) completeInquiryLocked(observedInformationGain, predictionError float64, at time.Time) {
	r.brain.RecordInquiryOutcome(observedInformationGain, predictionError, at)
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
func cloneEvent(in Event) *Event {
	out := in
	out.NeuralTrace = in.NeuralTrace.Canonical()
	out.Stimulus = append([]string(nil), in.Stimulus...)
	out.StimulusNodeIDs = append([]knowledge.NodeID(nil), in.StimulusNodeIDs...)
	out.Context = cloneContext(in.Context)
	out.ContextTokens = append([]string(nil), in.ContextTokens...)
	out.DataTokens = append([]string(nil), in.DataTokens...)
	out.PredictionOverride = clonePrediction(in.PredictionOverride)
	if in.Observation != nil {
		envelope := *in.Observation
		envelope.Payload = append(json.RawMessage(nil), in.Observation.Payload...)
		envelope.Tokens = append([]string(nil), in.Observation.Tokens...)
		envelope.ContextTokens = append([]string(nil), in.Observation.ContextTokens...)
		envelope.DataTokens = append([]string(nil), in.Observation.DataTokens...)
		out.Observation = &envelope
	}
	if in.SensorObservation != nil {
		sensor := in.SensorObservation.Clone()
		out.SensorObservation = &sensor
	}
	return &out
}
