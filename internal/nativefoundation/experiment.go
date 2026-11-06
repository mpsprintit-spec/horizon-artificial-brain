// Package nativefoundation is a controlled numeric environment, not Horizon's
// thinking path. It supplies fixtures, real executed outcomes and comparisons.
package nativefoundation

import (
	"encoding/json"
	"fmt"
	a "github.com/project-horizon/horizon-core/services/ai/activation"
	k "github.com/project-horizon/horizon-core/services/ai/knowledge"
	r "github.com/project-horizon/horizon-core/services/ai/runtime"
	"math"
	"math/rand"
	"os"
	"path/filepath"
	"time"
)

const stages = 6
const width = 4

var epoch = time.Date(2026, 11, 6, 0, 0, 0, 0, time.UTC)

type Task struct {
	State          map[k.NodeID]float64
	CostMultiplier float64
	Relation       float64
}
type Metrics struct {
	Return      float64 `json:"net_return"`
	Reward      float64 `json:"reward"`
	Accuracy    float64 `json:"accuracy"`
	Operations  float64 `json:"operations"`
	Stops       float64 `json:"stops"`
	Truncations float64 `json:"truncations"`
	Cases       int     `json:"cases"`
}
type Result struct {
	Causal                 CausalDiagnostic              `json:"causal_diagnostic"`
	Adaptation             map[string]Metrics            `json:"adaptation"`
	WorldSynapsesUnchanged bool                          `json:"world_synapses_unchanged"`
	Seed                   int64                         `json:"seed"`
	Lambda                 float64                       `json:"lambda"`
	Train                  Metrics                       `json:"stochastic_training"`
	Evaluations            map[string]map[string]Metrics `json:"greedy_or_baseline_evaluation"`
	ParameterChanges       int                           `json:"parameter_changes"`
	ParameterMaxDelta      float64                       `json:"parameter_max_delta"`
	DurationSeconds        float64                       `json:"duration_seconds"`
}

func unit(stage, j int) k.NodeID { return k.NodeID(stage*width + j + 1) }
func NewFixture(dir string, ablate bool) (*k.Brain, error) {
	type node struct {
		ID             k.NodeID  `json:"id"`
		Representation []float64 `json:"representation"`
		Threshold      float64   `json:"threshold"`
	}
	type edge struct {
		Source     k.NodeID       `json:"source_id"`
		Target     k.NodeID       `json:"target_id"`
		Weight     float64        `json:"weight"`
		Confidence float64        `json:"confidence"`
		Last       time.Time      `json:"last_activation"`
		Dynamic    k.DynamicState `json:"dynamic"`
	}
	nodes := []node{}
	edges := []edge{}
	for stage := 0; stage < stages+1; stage++ {
		for j := 0; j < width; j++ {
			id := unit(stage, j)
			nodes = append(nodes, node{id, []float64{math.Sin(float64(id)), math.Cos(float64(id))}, .25})
			target := stage + 1
			if stage >= stages-1 {
				target = stage
			}
			if ablate && stage == 3 {
				continue
			}
			for z := 0; z < width; z++ {
				edges = append(edges, edge{id, unit(target, z), .25, 1, epoch, k.DynamicState{Weight: .25, Confidence: 1, LastActivation: epoch}})
			}
		}
	}
	data, e := json.Marshal(struct {
		Schema int    `json:"schema_version"`
		Units  []node `json:"neural_units"`
		Edges  []edge `json:"synapses"`
	}{2, nodes, edges})
	if e != nil {
		return nil, e
	}
	path := filepath.Join(dir, "fixture.json")
	if e = os.WriteFile(path, data, 0600); e != nil {
		return nil, e
	}
	brain := k.NewBrain()
	if e = brain.Load(path); e != nil {
		return nil, e
	}
	return brain, nil
}
func Config(seed int64, lambda float64) k.NativeControlConfig {
	ids := []k.NodeID{}
	for stage := 0; stage < stages+1; stage++ {
		for j := 0; j < width; j++ {
			ids = append(ids, unit(stage, j))
		}
	}
	return k.NativeControlConfig{MaxActiveNodes: 4096, MaxSynapseReads: 100000, Version: k.NativeControlVersion, Sources: ids, Operations: []k.NativeOperation{{Kind: "propagate", Gain: 1, Cost: .04}, {Kind: "focus", Gain: .5, Cost: .04}, {Kind: "relax", Gain: .5, Cost: .04}, {Kind: "stop"}}, Seed: seed, LearningRate: .15, Gamma: 1, Lambda: lambda, Temperature: .12, MaxSteps: 8}
}
func tasks(seed int64, n int, split string) []Task {
	rng := rand.New(rand.NewSource(seed))
	out := make([]Task, n)
	for i := range out {
		depth := rng.Intn(4)
		amp := .65 + .35*rng.Float64()
		noise := .25 * rng.Float64()
		cost, relation := 1.0, 1.0
		if split == "longer" {
			depth = 4 + rng.Intn(2)
		}
		if split == "noise" {
			noise = .3 + .3*rng.Float64()
		}
		if split == "cost" {
			cost = 4
		}
		if split == "relation" {
			relation = -1
		}
		if split == "amplitude" {
			amp = .3 + .3*rng.Float64()
		}
		state := map[k.NodeID]float64{}
		for j := 0; j < width; j++ {
			state[unit(stages-1-depth, j)] = amp * (.8 + .2*rng.Float64())
			state[unit(stages, j)] = noise * (.5 + .5*rng.Float64())
		}
		if split == "combination" {
			for j := 0; j < width; j++ {
				state[unit(2, j)] += .2 * rng.Float64()
			}
		}
		out[i] = Task{state, cost, relation}
	}
	return out
}
func score(state map[k.NodeID]float64, task Task) float64 {
	signal, noise := 0.0, 0.0
	for j := 0; j < width; j++ {
		signal += state[unit(stages-1, j)] / width
		noise += state[unit(stages, j)] / width
	}
	return math.Max(-1, math.Min(1, task.Relation*signal*math.Exp(-3*noise)))
}
func apply(rt *r.BrainRuntime, c r.NativeControlCommand) r.NativeControlResult {
	out, e := rt.ApplyNativeControl(c, epoch.Add(time.Duration(rt.LastSequence()+1)*time.Microsecond))
	if e != nil {
		panic(fmt.Sprintf("%s: %v", c.ID, e))
	}
	return out
}
func run(rt *r.BrainRuntime, task Task, training bool, rng *rand.Rand, arm string) (reward, cost float64, operations int, stopped, truncated bool) {
	id := fmt.Sprintf("ep-%d", rt.LastSequence()+1)
	apply(rt, r.NativeControlCommand{ID: id, Begin: &r.NativeBegin{EpisodeID: id, Training: training, Observation: &r.NativeObservation{Source: "controlled-numeric-environment", SampleID: id, State: task.State}}})
	var out r.NativeControlResult
	for step := 0; step < 8; step++ {
		input := &r.NativeStep{EpisodeID: id, Sample: rng.Float64()}
		forced := -1
		switch arm {
		case "random":
			forced = rng.Intn(4)
		case "static0":
			forced = 3
		case "static1":
			if step < 1 {
				forced = 0
			} else {
				forced = 3
			}
		case "static2":
			if step < 2 {
				forced = 0
			} else {
				forced = 3
			}
		case "static4":
			if step < 4 {
				forced = 0
			} else {
				forced = 3
			}
		case "static5":
			if step < 5 {
				forced = 0
			} else {
				forced = 3
			}
		case "static3":
			if step < 3 {
				forced = 0
			} else {
				forced = 3
			}
		case "ablated":
			forced = 2
		}
		if forced >= 0 {
			input.ActionOverride = &forced
		}
		out = apply(rt, r.NativeControlCommand{ID: fmt.Sprintf("%s-step%d", id, step), Step: input})
		if out.Readout.Action != 3 {
			operations++
			cost += .04
		}
		if out.Stopped {
			break
		}
	}
	reward = score(out.State, task)
	observed := math.Max(-1, reward-cost*(task.CostMultiplier-1))
	apply(rt, r.NativeControlCommand{ID: id + "-out", Outcome: &r.NativeOutcome{EpisodeID: id, TraceDigest: out.TraceDigest, Source: "controlled-numeric-environment", OutcomeID: id, Provenance: "observed-environment", Reward: observed, Learn: training}})
	return observed, cost, operations, out.Stopped && !out.Truncated, out.Truncated
}
func evaluate(rt *r.BrainRuntime, data []Task, train bool, seed int64, arm string) Metrics {
	m := Metrics{Cases: len(data)}
	rng := rand.New(rand.NewSource(seed))
	for _, task := range data {
		reward, cost, ops, stop, trunc := run(rt, task, train, rng, arm)
		m.Reward += reward
		m.Return += reward - cost
		m.Operations += float64(ops)
		if reward >= .45 {
			m.Accuracy++
		}
		if stop {
			m.Stops++
		}
		if trunc {
			m.Truncations++
		}
	}
	n := float64(len(data))
	m.Reward /= n
	m.Return /= n
	m.Operations /= n
	m.Accuracy /= n
	m.Stops /= n
	m.Truncations /= n
	return m
}
func parameters(brain *k.Brain) []float64 {
	s := brain.NativeControlSnapshot()
	out := []float64{}
	for _, id := range append(s.Config.Sources, s.Bias) {
		for _, group := range s.Outputs {
			for _, target := range group {
				e := brain.Registry.GetByID(id).Synapses[target][0]
				w := e.Weight
				if e.Inhibitory {
					w = -w
				}
				out = append(out, w)
			}
		}
	}
	return out
}
func existing(dir string, data []Task) Metrics {
	m := Metrics{Cases: len(data)}
	brain, e := NewFixture(dir, false)
	if e != nil {
		panic(e)
	}
	engine := a.NewEngine(brain)
	for i, task := range data {
		conf := map[k.NodeID]float64{}
		for id := range task.State {
			conf[id] = .5
		}
		engine.RestoreState(a.StateSnapshot{InternalState: task.State, InternalConfidence: conf})
		out := engine.ThinkWithContextDynamicsAt(2, nil, epoch.Add(time.Duration(i)*time.Millisecond), k.CurrentDynamics)
		reward := score(out.Activations, task)
		m.Reward += reward
		m.Return += reward - .08*task.CostMultiplier
		m.Operations += 2
		if reward >= .45 {
			m.Accuracy++
		}
		m.Stops++
	}
	n := float64(len(data))
	m.Return /= n
	m.Reward /= n
	m.Operations /= n
	m.Accuracy /= n
	m.Stops /= n
	return m
}
func Experiment(dir string, seed int64, lambda float64, episodes, cases int, stream int64) (Result, error) {
	start := time.Now()
	if e := os.MkdirAll(dir, 0700); e != nil {
		return Result{}, e
	}
	brain, e := NewFixture(dir, false)
	if e != nil {
		return Result{}, e
	}
	rt := r.NewBrainRuntime(brain)
	cfg := Config(seed, lambda)
	apply(rt, r.NativeControlCommand{ID: "initialize", Initialize: &cfg})
	before := parameters(brain)
	worldBefore := worldParameters(brain)
	untrained := map[string]Metrics{}
	for _, split := range []string{"test", "combination", "longer", "noise", "amplitude", "cost", "relation"} {
		untrained[split] = evaluate(rt, tasks(stream+seed, cases, split), false, stream+seed, "untrained")
	}
	result := Result{Seed: seed, Lambda: lambda, Evaluations: map[string]map[string]Metrics{}}
	result.Train = evaluate(rt, tasks(seed+10000, episodes, "train"), true, seed+50000, "trained")
	after := parameters(brain)
	result.WorldSynapsesUnchanged = worldBefore == worldParameters(brain)
	for i, w := range after {
		if w != before[i] {
			result.ParameterChanges++
		}
		result.ParameterMaxDelta = math.Max(result.ParameterMaxDelta, math.Abs(w-before[i]))
	}
	if e = rt.Checkpoint(filepath.Join(dir, "trained-checkpoint.json")); e != nil {
		return result, e
	}
	for _, split := range []string{"test", "combination", "longer", "noise", "amplitude", "cost", "relation"} {
		data := tasks(stream+seed, cases, split)
		arms := map[string]Metrics{"untrained": untrained[split]}
		for _, arm := range []string{"trained", "random", "static0", "static1", "static2", "static3", "static4", "static5", "ablated"} {
			arms[arm] = evaluate(rt, data, false, stream+seed, arm)
		}
		if split == "test" {
			arms["existing-think2"] = existing(dir, data)
		}
		result.Evaluations[split] = arms
	}
	result.Adaptation = map[string]Metrics{}
	result.Causal, e = causalDiagnostics(dir, seed, stream, cases)
	if e != nil {
		return result, e
	}
	for _, shift := range []string{"longer", "relation"} {
		adapted := r.NewBrainRuntime(k.NewBrain())
		if e := adapted.RestoreCheckpoint(filepath.Join(dir, "trained-checkpoint.json")); e != nil {
			return result, e
		}
		result.Adaptation[shift+"-training"] = evaluate(adapted, tasks(seed+300000, 512, shift), true, seed+400000, "trained")
		result.Adaptation[shift+"-new-holdout"] = evaluate(adapted, tasks(seed+500000, cases, shift), false, seed+600000, "trained")
		result.Adaptation[shift+"-previous-task-retention"] = evaluate(adapted, tasks(seed+700000, cases, "test"), false, seed+800000, "trained")
	}
	result.DurationSeconds = time.Since(start).Seconds()
	return result, nil
}

func worldParameters(brain *k.Brain) string {
	out := []k.DynamicState{}
	for stage := 0; stage < stages+1; stage++ {
		for j := 0; j < width; j++ {
			from := unit(stage, j)
			target := stage + 1
			if stage >= stages-1 {
				target = stage
			}
			for z := 0; z < width; z++ {
				if list := brain.Registry.GetByID(from).Synapses[unit(target, z)]; len(list) > 0 {
					out = append(out, list[0].Dynamic)
				}
			}
		}
	}
	return k.NativeDigest(out)
}
