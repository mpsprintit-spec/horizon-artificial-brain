package runtime

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
)

const MonitorSchemaVersion = 1

type MonitorHandshake struct {
	Type string `json:"type"`
	BrainIdentity string `json:"brain_identity"`
	SchemaVersion int `json:"schema_version"`
	RuntimeCommit string `json:"runtime_commit"`
	StateRevision uint64 `json:"state_revision"`
	CanonicalStateHash string `json:"canonical_state_hash"`
	Capabilities []string `json:"capabilities"`
}

type MonitorSnapshot struct {
	Type string `json:"type"`
	BrainIdentity string `json:"brain_identity"`
	StateRevision uint64 `json:"state_revision"`
	CapturedAt string `json:"captured_at"`
	CanonicalStateHash string `json:"canonical_state_hash"`
	Counts MonitorCounts `json:"counts"`
	NeuralUnits []json.RawMessage `json:"neural_units"`
	Populations []json.RawMessage `json:"populations"`
	Synapses []json.RawMessage `json:"synapses"`
	TemporalPatterns []json.RawMessage `json:"temporal_patterns"`
	ExperienceTraces []json.RawMessage `json:"experience_traces"`
	BrainState MonitorBrainState `json:"brain_state"`
}

type MonitorCounts struct {
	NeuralUnits int `json:"neural_units"`
	Populations int `json:"populations"`
	Synapses int `json:"synapses"`
	TemporalPatterns int `json:"temporal_patterns"`
	ExperienceTraces int `json:"experience_traces"`
}

type MonitorBrainState struct {
	Prediction map[string]float64 `json:"prediction"`
	Error map[string]float64 `json:"error"`
	Memory map[string]float64 `json:"memory"`
	Plasticity map[string]float64 `json:"plasticity"`
	Curiosity map[string]float64 `json:"curiosity"`
	SelfModel map[string]float64 `json:"self_model"`
}

type monitorCanonicalBrain struct {
	NeuralUnits []json.RawMessage `json:"neural_units"`
	Populations []json.RawMessage `json:"populations"`
	Synapses []json.RawMessage `json:"synapses"`
	TemporalPatterns []json.RawMessage `json:"temporal_patterns"`
	ExperienceTraces []json.RawMessage `json:"experience_traces"`
	PredictionState map[string]float64 `json:"prediction_state"`
	ErrorState map[string]float64 `json:"error_state"`
	MemoryState map[string]float64 `json:"memory_state"`
	PlasticityState map[string]float64 `json:"plasticity_state"`
	CuriosityState map[string]float64 `json:"curiosity_state"`
	SelfModelState map[string]float64 `json:"self_model_state"`
}

func (r *BrainRuntime) MonitorSnapshot() (MonitorSnapshot, error) {
	if r == nil || r.brain == nil {
		return MonitorSnapshot{}, errors.New("brain runtime is not initialized")
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	data, err := r.brain.CanonicalJSON()
	if err != nil {
		return MonitorSnapshot{}, err
	}
	var brain monitorCanonicalBrain
	if err := json.Unmarshal(data, &brain); err != nil {
		return MonitorSnapshot{}, err
	}
	hash := sha256.Sum256(data)

	sequence := r.seq
	now := r.nowLocked()

	return MonitorSnapshot{
		Type: "brain.snapshot",
		BrainIdentity: BrainIdentity,
		StateRevision: sequence,
		CapturedAt: now.Format("2006-01-02T15:04:05.000Z07:00"),
		CanonicalStateHash: "sha256:" + hex.EncodeToString(hash[:]),
		Counts: MonitorCounts{
			NeuralUnits: len(brain.NeuralUnits),
			Populations: len(brain.Populations),
			Synapses: len(brain.Synapses),
			TemporalPatterns: len(brain.TemporalPatterns),
			ExperienceTraces: len(brain.ExperienceTraces),
		},
		NeuralUnits: brain.NeuralUnits,
		Populations: brain.Populations,
		Synapses: brain.Synapses,
		TemporalPatterns: brain.TemporalPatterns,
		ExperienceTraces: brain.ExperienceTraces,
		BrainState: MonitorBrainState{
			Prediction: cloneMonitorFloatMap(brain.PredictionState),
			Error: cloneMonitorFloatMap(brain.ErrorState),
			Memory: cloneMonitorFloatMap(brain.MemoryState),
			Plasticity: cloneMonitorFloatMap(brain.PlasticityState),
			Curiosity: cloneMonitorFloatMap(brain.CuriosityState),
			SelfModel: cloneMonitorFloatMap(brain.SelfModelState),
		},
	}, nil
}

func (r *BrainRuntime) MonitorHandshake(runtimeCommit string) (MonitorHandshake, error) {
	snapshot, err := r.MonitorSnapshot()
	if err != nil {
		return MonitorHandshake{}, err
	}
	return MonitorHandshake{
		Type: "brain.handshake",
		BrainIdentity: snapshot.BrainIdentity,
		SchemaVersion: MonitorSchemaVersion,
		RuntimeCommit: runtimeCommit,
		StateRevision: snapshot.StateRevision,
		CanonicalStateHash: snapshot.CanonicalStateHash,
		Capabilities: []string{"snapshot", "recurrent_state", "plasticity", "checkpoint", "inquiry", "observation"},
	}, nil
}

func cloneMonitorFloatMap(in map[string]float64) map[string]float64 {
	out := make(map[string]float64, len(in))
	for key, value := range in {
		out[key] = value
	}
	return out
}
