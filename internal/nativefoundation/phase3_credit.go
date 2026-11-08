package nativefoundation

import (
	"fmt"
	k "github.com/project-horizon/horizon-core/services/ai/knowledge"
	r "github.com/project-horizon/horizon-core/services/ai/runtime"
	"math"
	"math/rand"
	"path/filepath"
	"time"
)

type Phase3CreditResult struct {
	TaskGroups        map[string]Metrics `json:"task_groups,omitempty"`
	Depth             int                `json:"max_training_depth"`
	Seed              int64              `json:"seed"`
	Arm               string             `json:"arm"`
	Blocks            int                `json:"blocks"`
	Episodes          int                `json:"executed_training_episodes"`
	Noise             bool               `json:"shared_reward_offset"`
	Before, After     Metrics
	Baselines         map[string]Metrics `json:"baselines"`
	Training          Metrics            `json:"randomized_intervention_training"`
	Encoder           string             `json:"encoder"`
	OracleReturn      float64            `json:"oracle_return"`
	Regret            float64            `json:"regret"`
	AdvantageRMSE     float64            `json:"advantage_rmse"`
	NuisanceEffect    float64            `json:"nuisance_effect"`
	SignalEffectError float64            `json:"signal_effect_error"`
	EffectPrecision   float64            `json:"effect_precision"`
	EffectRecall      float64            `json:"effect_recall"`
	ParameterChanges  int                `json:"parameter_changes"`
	Seconds           float64            `json:"seconds"`
}

// Each block is a randomized 2x2 factorial experiment: two actual initial
// states and two assigned first operations, with stop as common continuation.
// A factual exogenous reward offset is held fixed across the four executions.
// It is not observed by the learner and is never included in policy features.
// The environment computes every reward AFTER executing the native operation.
// Goal IDs and analytical effects remain evaluator-only.
func phase3CreditTrial(rt *r.BrainRuntime, state map[k.NodeID]float64, action int, pair, mode, block string, offset float64, depth int) Metrics {
	id := fmt.Sprintf("p3-trial-%d", rt.LastSequence()+1)
	apply(rt, r.NativeControlCommand{ID: id, Begin: &r.NativeBegin{EpisodeID: id, Training: true, Intervention: true, MatchKey: block, Observation: &r.NativeObservation{Source: "phase3-factorial-environment", SampleID: id, State: state}}})
	out := apply(rt, r.NativeControlCommand{ID: id + "-action", Step: &r.NativeStep{EpisodeID: id, ExperimentAction: &action}})
	cost, operations := 0.0, 0.0
	if !out.Stopped {
		operations, cost = 1, .04 // Config fixes the diagnostic tariff at .04.
		for step := 1; step < depth; step++ {
			propagate := 0
			out = apply(rt, r.NativeControlCommand{ID: fmt.Sprintf("%s-continue-%d", id, step), Step: &r.NativeStep{EpisodeID: id, ExperimentAction: &propagate}})
			operations++
			cost += .04
		}
		stop := 3
		out = apply(rt, r.NativeControlCommand{ID: id + "-stop", Step: &r.NativeStep{EpisodeID: id, ExperimentAction: &stop}})
	}
	reward := offset
	for j := 0; j < width; j++ {
		reward += .25 * out.State[unit(5, j)] / width
	}
	apply(rt, r.NativeControlCommand{ID: id + "-out", Outcome: &r.NativeOutcome{EpisodeID: id, TraceDigest: out.TraceDigest, Source: "phase3-factorial-environment", OutcomeID: id, Provenance: "observed-environment", Reward: reward, Learn: true, ContrastID: pair, ContrastMode: mode}})
	return Metrics{Return: reward - cost, Reward: reward, Operations: operations, Cases: 1, Stops: 1}
}

func phase3CreditTasks(seed int64, cases, depth int) []Phase2Task {
	rng := rand.New(rand.NewSource(seed))
	tasks := make([]Phase2Task, cases)
	for i := range tasks {
		stage := 4
		if depth > 1 {
			stage = 5 - depth + rng.Intn(depth)
		}
		if rng.Intn(4) == 0 {
			stage = 5
		}
		x, z := rng.Float64(), 1.5*rng.Float64()
		state := map[k.NodeID]float64{}
		goal := []k.NodeID{}
		for j := 0; j < width; j++ {
			state[unit(stage, j)] = x
			state[unit(6, j)] = z
			goal = append(goal, unit(5, j))
		}
		tasks[i] = Phase2Task{State: state, Goal: goal, Relation: .25}
	}
	return tasks
}

func Phase3CreditExperiment(dir string, seed int64, arm string, blocks, cases int, stream int64, noisy bool, encoder string, depth int) (Phase3CreditResult, error) {
	start := time.Now()
	result := Phase3CreditResult{Seed: seed, Arm: arm, Blocks: blocks, Noise: noisy, Encoder: encoder, Depth: depth}
	if depth < 1 || depth > 2 {
		return result, fmt.Errorf("Phase III diagnostic supports depths one or two")
	}
	if encoder != "identity" && encoder != "relational" {
		return result, fmt.Errorf("invalid Phase III encoder")
	}
	if arm != "action" && arm != "feature" && arm != "unpaired" && arm != "unmatched" {
		return result, fmt.Errorf("unsupported Phase III credit arm")
	}
	if err := preparePhase2Directory(dir); err != nil {
		return result, err
	}
	brain, err := NewFixture(dir, false)
	if err != nil {
		return result, err
	}
	rt := r.NewBrainRuntime(brain)
	cfg := Config(seed, 1)
	cfg.Version = k.NativeControlCausalVersion
	cfg.MaxSteps = 12
	if encoder == "relational" {
		cfg.Encoder = "local-relational-v1"
	}
	cfg.ContrastMean = arm == "feature"
	apply(rt, r.NativeControlCommand{ID: "init", Initialize: &cfg})
	before := parameters(brain)
	data := phase3CreditTasks(stream+seed, cases, depth)
	result.Before = phase3CreditEvaluate(rt, data, stream+seed+100000)
	result.Baselines = map[string]Metrics{}
	for _, arm := range []string{"static0", "static1", "static2", "random"} {
		result.Baselines[arm] = phase3CreditEvaluate(rt, data, stream+seed+100000, arm)
	}
	trial := func(state map[k.NodeID]float64, action int, pair, mode, block string, offset float64, steps int) {
		m := phase3CreditTrial(rt, state, action, pair, mode, block, offset, steps)
		result.Training.Cases++
		result.Training.Return += m.Return
		result.Training.Reward += m.Reward
		result.Training.Operations += m.Operations
	}
	rng := rand.New(rand.NewSource(seed + 13100000))
	unmatchedRNG := rand.New(rand.NewSource(seed + 15100000))
	for i := 0; i < blocks; i++ {
		stage := 4
		if depth > 1 {
			stage = 5 - depth + rng.Intn(depth)
		}
		if rng.Intn(4) == 0 {
			stage = 5
		}
		x, z := rng.Float64(), rng.Float64()
		// Half the blocks have correlated distractors; randomized population
		// removal breaks that correlation. Selection never consults causal labels.
		if i%2 == 0 {
			z = .8*x + .2*z
		}
		a, b := rng.Intn(4), rng.Intn(3)
		if b >= a {
			b++
		}
		population := []int{stage, 6}[rng.Intn(2)]
		s0, s1 := map[k.NodeID]float64{}, map[k.NodeID]float64{}
		for j := 0; j < width; j++ {
			s0[unit(stage, j)] = x
			s1[unit(stage, j)] = x
			s0[unit(6, j)] = z
			s1[unit(6, j)] = z
			s1[unit(population, j)] = 0
		}
		offset := .6*(2*rng.Float64()-1) + .1*math.Sin(6*z)
		if !noisy {
			offset = 0
		}
		block := fmt.Sprintf("exogenous-%d", i)
		if arm == "feature" {
			for j, action := range []int{a, b} {
				pair := fmt.Sprintf("feature-%d-%d", i, j)
				trial(s0, action, pair, "", block, offset, max(1, 5-stage))
				trial(s1, action, pair, "", block, offset, max(1, 5-stage))
			}
		} else {
			for j, state := range []map[k.NodeID]float64{s0, s1} {
				pair, mode := "", ""
				if arm != "unpaired" {
					pair = fmt.Sprintf("action-%d-%d", i, j)
					mode = "action"
				}
				trial(state, a, pair, mode, block, offset, max(1, 5-stage))
				secondOffset := offset
				// Negative control deliberately violates the adapter assumption while
				// keeping its claimed key: a key is not a certificate of hidden matching.
				if arm == "unmatched" && noisy {
					secondOffset = .6*(2*unmatchedRNG.Float64()-1) + .1*math.Sin(6*z)
				}
				trial(state, b, pair, mode, block, secondOffset, max(1, 5-stage))
			}
		}
		result.Episodes += 4
	}
	result.Training.Return /= float64(result.Training.Cases)
	result.Training.Reward /= float64(result.Training.Cases)
	result.Training.Operations /= float64(result.Training.Cases)
	result.Training.Stops = 1
	result.After = phase3CreditEvaluate(rt, data, stream+seed+100000)
	result.TaskGroups = map[string]Metrics{}
	for _, stage := range []int{3, 4, 5} {
		group := []Phase2Task{}
		for _, task := range data {
			if task.State[unit(stage, 0)] > 0 {
				group = append(group, task)
			}
		}
		if len(group) > 0 {
			result.TaskGroups[fmt.Sprintf("depth%d", 5-stage)] = phase3CreditEvaluate(rt, group, stream+seed+100000)
		}
	}
	for _, task := range data {
		result.OracleReturn += phase3CreditOptimum(task) / float64(cases)
	}
	result.Regret = result.OracleReturn - result.After.Return
	after := parameters(brain)
	for i, w := range after {
		if w != before[i] {
			result.ParameterChanges++
		}
	}
	// Causal EFFECT precision/recall, not exclusive-synapse change precision:
	// predictions of a controlled input intervention on the propagate-stop
	// advantage are compared to actual analytical environment effects.
	tp, fp, fn := 0, 0, 0
	for i := 0; i < cases; i++ {
		x, z := .1+.8*float64(i+1)/float64(cases), 1.4*float64((i*37)%cases)/float64(cases)
		state := map[k.NodeID]float64{}
		for j := 0; j < width; j++ {
			state[unit(4, j)] = x
			state[unit(6, j)] = z
		}
		q, _ := brain.NativePolicy(state, true, 0)
		adv := q.Values[0] - q.Values[3]
		result.AdvantageRMSE += (adv - (.25*x - .04)) * (adv - (.25*x - .04)) / float64(cases)
		for _, pop := range []int{4, 6} {
			changed := map[k.NodeID]float64{}
			for id, v := range state {
				changed[id] = v
			}
			for j := 0; j < width; j++ {
				changed[unit(pop, j)] = 0
			}
			q2, _ := brain.NativePolicy(changed, true, 0)
			effect := adv - (q2.Values[0] - q2.Values[3])
			truth := 0.0
			if pop == 4 {
				truth = .25 * x
			}
			if pop == 4 {
				result.SignalEffectError += math.Abs(effect-truth) / float64(cases)
			} else {
				result.NuisanceEffect += math.Abs(effect) / float64(cases)
			}
			predicted, actual := math.Abs(effect) > .02, math.Abs(truth) > .02
			if predicted && actual {
				tp++
			}
			if predicted && !actual {
				fp++
			}
			if !predicted && actual {
				fn++
			}
		}
	}
	result.AdvantageRMSE = math.Sqrt(result.AdvantageRMSE)
	if tp+fp > 0 {
		result.EffectPrecision = float64(tp) / float64(tp+fp)
	}
	if tp+fn > 0 {
		result.EffectRecall = float64(tp) / float64(tp+fn)
	}
	if err = rt.Checkpoint(filepath.Join(dir, "trained-checkpoint.json")); err != nil {
		return result, err
	}
	result.Seconds = time.Since(start).Seconds()
	return result, nil
}

// Accuracy means attaining the evaluator optimum, not the Phase II reward>=.45
// threshold (which is inapplicable to this differently scaled environment).
func phase3CreditEvaluate(rt *r.BrainRuntime, tasks []Phase2Task, seed int64, arms ...string) Metrics {
	arm := "trained"
	if len(arms) > 0 {
		arm = arms[0]
	}
	out := Metrics{Cases: len(tasks)}
	for i, task := range tasks {
		m := phase2Evaluate(rt, []Phase2Task{task}, false, seed+int64(i), arm)
		optimum := phase3CreditOptimum(task)
		out.Return += m.Return
		out.Reward += m.Reward
		out.Operations += m.Operations
		out.Stops += m.Stops
		out.Truncations += m.Truncations
		if math.Abs(m.Return-optimum) < 1e-6 {
			out.Accuracy++
		}
	}
	n := float64(len(tasks))
	out.Return /= n
	out.Reward /= n
	out.Operations /= n
	out.Stops /= n
	out.Truncations /= n
	out.Accuracy /= n
	return out
}

// Only used by the evaluator. The policy never receives a stage or this value.
func phase3CreditOptimum(task Phase2Task) float64 {
	best := .25 * task.State[unit(5, 0)]
	for stage := 0; stage < 5; stage++ {
		best = math.Max(best, .25*task.State[unit(stage, 0)]-.04*float64(5-stage))
	}
	return best
}
