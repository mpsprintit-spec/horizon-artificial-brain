package nativefoundation

import (
	"encoding/json"
	"fmt"
	k "github.com/project-horizon/horizon-core/services/ai/knowledge"
	r "github.com/project-horizon/horizon-core/services/ai/runtime"
	"math"
	"math/rand"
	"os"
	"path/filepath"
	"time"
)

type Phase2Graph struct {
	Groups [][]k.NodeID `json:"groups"`
}
type Phase2Task struct {
	State       map[k.NodeID]float64
	Goal, Noise []k.NodeID
	Relation    float64
}
type Phase2Result struct {
	Seed        int64                         `json:"seed"`
	Encoder     string                        `json:"encoder"`
	Train       Metrics                       `json:"stochastic_training"`
	Evaluations map[string]map[string]Metrics `json:"evaluation"`
	Adaptation  map[string]Metrics            `json:"adaptation"`
	Changed     int                           `json:"changed_parameters"`
	MaxDelta    float64                       `json:"max_parameter_delta"`
	Parameters  int                           `json:"parameters"`
	Duration    float64                       `json:"seconds"`
}

// Equivalent graphs have disjoint, randomly permuted addresses. The third
// graph doubles population width; the fourth has a different chain length.
// Fixture topology is supplied, not learned. No goal labels reach the policy.
func phase2Fixture(dir string, seed int64) (*k.Brain, []Phase2Graph, []k.NodeID, error) {
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
	var nodes []node
	var edges []edge
	var ids []k.NodeID
	var graphs []Phase2Graph
	rng := rand.New(rand.NewSource(seed + 110003))
	address := rng.Perm(2000)
	cursor := 0
	for _, spec := range [][2]int{{6, 4}, {6, 4}, {6, 8}, {9, 4}} {
		depth, width := spec[0], spec[1]
		graph := Phase2Graph{Groups: make([][]k.NodeID, depth+1)}
		for g := range graph.Groups {
			for j := 0; j < width; j++ {
				id := k.NodeID(address[cursor] + 1)
				cursor++
				graph.Groups[g] = append(graph.Groups[g], id)
				ids = append(ids, id)
				nodes = append(nodes, node{id, []float64{rng.Float64(), rng.Float64(), rng.Float64()}, .25})
			}
		}
		for g, group := range graph.Groups {
			next := g + 1
			if g >= depth-1 {
				next = g
			}
			for _, from := range group {
				for _, to := range graph.Groups[next] {
					w := 1 / float64(width)
					edges = append(edges, edge{from, to, w, 1, epoch, k.DynamicState{Weight: w, Confidence: 1, LastActivation: epoch}})
				}
			}
		}
		graphs = append(graphs, graph)
	}
	data, err := json.Marshal(struct {
		Schema int    `json:"schema_version"`
		Units  []node `json:"neural_units"`
		Edges  []edge `json:"synapses"`
	}{2, nodes, edges})
	if err != nil {
		return nil, nil, nil, err
	}
	path := filepath.Join(dir, "fixture.json")
	if err = os.WriteFile(path, data, 0600); err != nil {
		return nil, nil, nil, err
	}
	brain := k.NewBrain()
	err = brain.Load(path)
	return brain, graphs, ids, err
}
func phase2Tasks(graph Phase2Graph, seed int64, n int, split string) []Phase2Task {
	rng := rand.New(rand.NewSource(seed))
	out := make([]Phase2Task, n)
	terminal := len(graph.Groups) - 2
	for i := range out {
		depth := rng.Intn(4)
		amp := .65 + .35*rng.Float64()
		noise := .25 * rng.Float64()
		relation := 1.0
		if split == "longer" {
			depth = 4 + rng.Intn(2)
		}
		if split == "longest" {
			depth = 6 + rng.Intn(2)
		}
		if split == "noise" {
			noise = .3 + .3*rng.Float64()
		}
		if split == "amplitude" {
			amp = .3 + .3*rng.Float64()
		}
		if split == "reversal" {
			relation = -1
		}
		state := map[k.NodeID]float64{}
		for _, id := range graph.Groups[terminal-depth] {
			state[id] = amp * (.8 + .2*rng.Float64())
		}
		for _, id := range graph.Groups[terminal+1] {
			state[id] = noise * (.5 + .5*rng.Float64())
		}
		if split == "combination" {
			for _, id := range graph.Groups[terminal-2] {
				state[id] += .2 * rng.Float64()
			}
		}
		out[i] = Phase2Task{state, graph.Groups[terminal], graph.Groups[terminal+1], relation}
	}
	return out
}
func phase2Reward(state map[k.NodeID]float64, t Phase2Task) float64 {
	signal, noise := 0.0, 0.0
	for _, id := range t.Goal {
		signal += state[id] / float64(len(t.Goal))
	}
	for _, id := range t.Noise {
		noise += state[id] / float64(len(t.Noise))
	}
	return math.Max(-1, math.Min(1, t.Relation*signal*math.Exp(-3*noise)))
}
func phase2Evaluate(rt *r.BrainRuntime, data []Phase2Task, training bool, seed int64, arm string) Metrics {
	m := Metrics{Cases: len(data)}
	rng := rand.New(rand.NewSource(seed))
	for _, task := range data {
		id := fmt.Sprintf("p2-%d", rt.LastSequence()+1)
		apply(rt, r.NativeControlCommand{ID: id, Begin: &r.NativeBegin{EpisodeID: id, Training: training, Observation: &r.NativeObservation{Source: "phase2-executed-environment", SampleID: id, State: task.State}}})
		var out r.NativeControlResult
		cost := 0.0
		ops := 0
		for step := 0; step < 12; step++ {
			cmd := &r.NativeStep{EpisodeID: id, Sample: rng.Float64()}
			forced := -1
			if arm == "random" {
				forced = rng.Intn(4)
			}
			for count := 0; count <= 8; count++ {
				if arm == fmt.Sprintf("static%d", count) {
					forced = 3
					if step < count {
						forced = 0
					}
				}
			}
			if forced >= 0 {
				cmd.ActionOverride = &forced
			}
			out = apply(rt, r.NativeControlCommand{ID: fmt.Sprintf("%s-%d", id, step), Step: cmd})
			if out.Readout.Action != 3 {
				ops++
				cost += .04
			}
			if out.Stopped {
				break
			}
		}
		reward := phase2Reward(out.State, task)
		apply(rt, r.NativeControlCommand{ID: id + "-out", Outcome: &r.NativeOutcome{EpisodeID: id, TraceDigest: out.TraceDigest, Source: "phase2-executed-environment", OutcomeID: id, Provenance: "observed-environment", Reward: reward, Learn: training}})
		m.Return += reward - cost
		m.Reward += reward
		m.Operations += float64(ops)
		if reward >= .45 {
			m.Accuracy++
		}
		if out.Truncated {
			m.Truncations++
		} else if out.Stopped {
			m.Stops++
		}
	}
	n := float64(len(data))
	m.Return /= n
	m.Reward /= n
	m.Operations /= n
	m.Accuracy /= n
	m.Stops /= n
	m.Truncations /= n
	return m
}
func Phase2Experiment(dir string, seed int64, encoder string, episodes, cases int, stream int64, lambda, epsilon float64) (Phase2Result, error) {
	start := time.Now()
	result := Phase2Result{Seed: seed, Encoder: encoder, Evaluations: map[string]map[string]Metrics{}, Adaptation: map[string]Metrics{}}
	if err := preparePhase2Directory(dir); err != nil {
		return result, err
	}
	brain, graphs, ids, err := phase2Fixture(dir, seed)
	if err != nil {
		return result, err
	}
	rt := r.NewBrainRuntime(brain)
	cfg := Config(seed, lambda)
	cfg.Sources = ids
	cfg.MaxSteps = 12
	cfg.Epsilon = epsilon
	if epsilon > 0 {
		cfg.Version = k.NativeControlRelationalVersion
	}
	if encoder != "identity" {
		cfg.Version = k.NativeControlRelationalVersion
		cfg.Encoder = "local-relational-v1"
	}
	apply(rt, r.NativeControlCommand{ID: "init", Initialize: &cfg})
	params := func() []float64 {
		out := []float64{}
		s := brain.NativeControlSnapshot()
		for _, id := range s.ParameterSources() {
			for _, group := range s.Outputs {
				for _, to := range group {
					v := brain.Registry.GetByID(id).Synapses[to][0]
					w := v.Weight
					if v.Inhibitory {
						w = -w
					}
					out = append(out, w)
				}
			}
		}
		return out
	}
	before := params()
	result.Parameters = len(before)
	splits := []struct {
		name, condition string
		graph           int
	}{{"test", "test", 0}, {"longer", "longer", 0}, {"permutation", "test", 1}, {"new-populations", "test", 2}, {"longest", "longest", 3}, {"combination", "combination", 1}, {"noise", "noise", 1}, {"amplitude", "amplitude", 1}}
	for _, split := range splits {
		data := phase2Tasks(graphs[split.graph], stream+seed, cases, split.condition)
		result.Evaluations[split.name] = map[string]Metrics{"untrained": phase2Evaluate(rt, data, false, stream+seed, "greedy")}
	}
	result.Train = phase2Evaluate(rt, phase2Tasks(graphs[0], seed+210000, episodes, "train"), true, seed+310000, "greedy")
	after := params()
	for i, v := range after {
		if v != before[i] {
			result.Changed++
		}
		result.MaxDelta = math.Max(result.MaxDelta, math.Abs(v-before[i]))
	}
	cp := filepath.Join(dir, "trained-checkpoint.json")
	if err = rt.Checkpoint(cp); err != nil {
		return result, err
	}
	for _, split := range splits {
		data := phase2Tasks(graphs[split.graph], stream+seed, cases, split.condition)
		for _, arm := range []string{"trained", "random", "static0", "static3", "static5", "static7"} {
			result.Evaluations[split.name][arm] = phase2Evaluate(rt, data, false, stream+seed, arm)
		}
	}
	for _, shift := range []string{"longer", "reversal"} {
		adapted := r.NewBrainRuntime(k.NewBrain())
		if err = adapted.RestoreCheckpoint(cp); err != nil {
			return result, err
		}
		result.Adaptation[shift+"-training"] = phase2Evaluate(adapted, phase2Tasks(graphs[0], seed+410000, 512, shift), true, seed+510000, "greedy")
		result.Adaptation[shift+"-new"] = phase2Evaluate(adapted, phase2Tasks(graphs[0], stream+seed+200000, cases, shift), false, stream+seed+300000, "greedy")
		result.Adaptation[shift+"-retention"] = phase2Evaluate(adapted, phase2Tasks(graphs[0], stream+seed+400000, cases, "test"), false, stream+seed+500000, "greedy")
	}
	result.Duration = time.Since(start).Seconds()
	return result, nil
}

// Research output is append-only across runs. A new run needs a fresh directory;
// this prevents accidental replacement of an earlier neural checkpoint.
func preparePhase2Directory(dir string) error {
	entries, err := os.ReadDir(dir)
	if err == nil && len(entries) > 0 {
		return fmt.Errorf("experiment directory already contains evidence: %s", dir)
	}
	if err != nil && !os.IsNotExist(err) {
		return err
	}
	return os.MkdirAll(dir, 0700)
}

// Phase2RetentionBaseline evaluates the frozen pre-adaptation checkpoint on
// exactly the old-task cases used for the post-adaptation retention metric.
func Phase2RetentionBaseline(checkpoint string, seed, stream int64, cases int) (Metrics, error) {
	if cases < 1 {
		return Metrics{}, fmt.Errorf("positive retention case count required")
	}
	var fixture struct {
		Units []struct {
			ID k.NodeID `json:"id"`
		} `json:"neural_units"`
	}
	data, err := os.ReadFile(filepath.Join(filepath.Dir(checkpoint), "fixture.json"))
	if err != nil {
		return Metrics{}, err
	}
	if err = json.Unmarshal(data, &fixture); err != nil {
		return Metrics{}, err
	}
	if len(fixture.Units) < 28 {
		return Metrics{}, fmt.Errorf("not a Phase II fixture")
	}
	graph := Phase2Graph{Groups: make([][]k.NodeID, 7)}
	for group := 0; group < 7; group++ {
		for j := 0; j < 4; j++ {
			graph.Groups[group] = append(graph.Groups[group], fixture.Units[group*4+j].ID)
		}
	}
	rt := r.NewBrainRuntime(k.NewBrain())
	if err = rt.RestoreCheckpoint(checkpoint); err != nil {
		return Metrics{}, err
	}
	return phase2Evaluate(rt, phase2Tasks(graph, stream+seed+400000, cases, "test"), false, stream+seed+500000, "trained"), nil
}
