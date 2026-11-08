package nativefoundation

import (
	"fmt"
	k "github.com/project-horizon/horizon-core/services/ai/knowledge"
	r "github.com/project-horizon/horizon-core/services/ai/runtime"
	"math"
	"math/rand"
	"path/filepath"
)

type Phase2Credit struct {
	Seed                                                int64  `json:"seed"`
	Arm                                                 string `json:"arm"`
	ExecutedEpisodes                                    int    `json:"executed_episodes"`
	Before, After                                       Metrics
	RelevantChanges, IrrelevantChanges, InactiveChanges int
	Precision                                           float64
	NuisanceSensitivity                                 float64
}

// Every reward below is evaluated from a state produced by executed primitives.
// The learner receives scalar outcomes, paired trial IDs and explicit action
// interventions; never goal IDs, population roles, or causal labels.
func creditTrial(rt *r.BrainRuntime, state map[k.NodeID]float64, actions []int, pair string) r.NativeControlResult {
	id := fmt.Sprintf("credit-%d", rt.LastSequence()+1)
	apply(rt, r.NativeControlCommand{ID: id, Begin: &r.NativeBegin{EpisodeID: id, Training: true, Intervention: true, Observation: &r.NativeObservation{Source: "randomized-credit-environment", SampleID: id, State: state}}})
	var out r.NativeControlResult
	for j, a := range actions {
		action := a
		out = apply(rt, r.NativeControlCommand{ID: fmt.Sprintf("%s-%d", id, j), Step: &r.NativeStep{EpisodeID: id, ExperimentAction: &action}})
	}
	reward := 0.0
	for j := 0; j < width; j++ {
		reward += out.State[unit(5, j)] / width
	}
	return apply(rt, r.NativeControlCommand{ID: id + "-out", Outcome: &r.NativeOutcome{EpisodeID: id, TraceDigest: out.TraceDigest, Source: "randomized-credit-environment", OutcomeID: id, Provenance: "observed-environment", Reward: reward, Learn: true, ContrastID: pair}})
}
func Phase2CreditExperiment(dir string, seed int64, paired bool, pairs, cases int, stream int64, centered bool) (Phase2Credit, error) {
	result := Phase2Credit{Seed: seed, Arm: "unpaired"}
	if paired {
		result.Arm = "paired"
		if centered {
			result.Arm = "paired-centered"
		}
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
	cfg.Version = k.NativeControlRelationalVersion
	cfg.ContrastMean = centered
	cfg.MaxSteps = 12
	apply(rt, r.NativeControlCommand{ID: "init", Initialize: &cfg})
	makeTasks := func(seed int64) []Phase2Task {
		rng := rand.New(rand.NewSource(seed))
		out := make([]Phase2Task, cases)
		for i := range out {
			x := .3 + .7*rng.Float64()
			z := 1.2 * rng.Float64()
			state := map[k.NodeID]float64{}
			var goal []k.NodeID
			for j := 0; j < width; j++ {
				state[unit(4, j)] = x
				state[unit(6, j)] = z
				goal = append(goal, unit(5, j))
			}
			out[i] = Phase2Task{State: state, Goal: goal, Relation: 1}
		}
		return out
	}
	data := makeTasks(stream + seed)
	result.Before = phase2Evaluate(rt, data, false, stream+seed, "trained")
	rng := rand.New(rand.NewSource(seed + 650000))
	sequences := [][]int{{3}, {0, 3}, {1, 0, 3}, {2, 0, 3}}
	for i := 0; i < pairs; i++ {
		// Choose population without consulting its reward role. Exogenous activity
		// is matched within each pair; correlated nuisance is deliberately included.
		population := []int{4, 6}[rng.Intn(2)]
		x := .3 + .7*rng.Float64()
		z := x * (.7 + .3*rng.Float64())
		first, second := map[k.NodeID]float64{}, map[k.NodeID]float64{}
		for j := 0; j < width; j++ {
			first[unit(4, j)] = x
			first[unit(6, j)] = z
			second[unit(4, j)] = x
			second[unit(6, j)] = z
			second[unit(population, j)] = 0
		}
		if rng.Intn(2) == 0 {
			first, second = second, first
		}
		actions := sequences[rng.Intn(len(sequences))]
		pair := ""
		if paired {
			pair = fmt.Sprintf("pair-%d", i)
		}
		creditTrial(rt, first, actions, pair)
		creditTrial(rt, second, actions, pair)
		result.ExecutedEpisodes += 2
		// Differential regression cannot identify action intercepts. Matched actual
		// zero-input trials anchor compute costs in both arms at identical budget.
		if i%8 == 0 {
			creditTrial(rt, map[k.NodeID]float64{unit(4, 0): 0}, actions, "")
			result.ExecutedEpisodes++
		}
	}
	result.After = phase2Evaluate(rt, data, false, stream+seed, "trained")
	// Frozen conditional readout sensitivity to a nuisance intervention. This is
	// not an exclusive-parameter precision measure for shared representations.
	for i := 0; i < cases; i++ {
		a := data[i].State
		b := map[k.NodeID]float64{}
		for id, v := range a {
			b[id] = v
		}
		for j := 0; j < width; j++ {
			b[unit(6, j)] = 0
		}
		qa, _ := brain.NativePolicy(a, true, 0)
		qb, _ := brain.NativePolicy(b, true, 0)
		result.NuisanceSensitivity += math.Abs((qa.Values[0]-qa.Values[3])-(qb.Values[0]-qb.Values[3])) / float64(cases)
	}
	if err = rt.Checkpoint(filepath.Join(dir, "trained-checkpoint.json")); err != nil {
		return result, err
	}
	before := parameters(brain)
	a, b := map[k.NodeID]float64{}, map[k.NodeID]float64{}
	for j := 0; j < width; j++ {
		a[unit(4, j)] = .8
		a[unit(6, j)] = .8
		b[unit(4, j)] = 0
		b[unit(6, j)] = .8
	}
	pair := ""
	if paired {
		pair = "precision-probe"
	}
	creditTrial(rt, a, []int{0, 3}, pair)
	if paired {
		creditTrial(rt, b, []int{0, 3}, pair)
	}
	after := parameters(brain)
	s := brain.NativeControlSnapshot()
	index := 0
	for _, id := range s.ParameterSources() {
		for _, group := range s.Outputs {
			for range group {
				if before[index] != after[index] && id != s.Bias {
					switch {
					case id >= unit(4, 0) && id <= unit(5, width-1):
						result.RelevantChanges++
					case id >= unit(6, 0) && id <= unit(6, width-1):
						result.IrrelevantChanges++
					default:
						result.InactiveChanges++
					}
				}
				index++
			}
		}
	}
	total := result.RelevantChanges + result.IrrelevantChanges + result.InactiveChanges
	if total > 0 {
		result.Precision = float64(result.RelevantChanges) / float64(total)
	}
	return result, nil
}

// CreditBoundary evaluates a frozen checkpoint near the known compute tariff.
// Unlike the initial credit task, some cases need stop and others propagate.
// The analytical optimum is evaluator-only; no answer is fed to the learner.
func CreditBoundary(checkpoint string, seed int64, cases int) (map[string]any, error) {
	if cases < 1 {
		return nil, fmt.Errorf("positive boundary case count required")
	}
	rt := r.NewBrainRuntime(k.NewBrain())
	if err := rt.RestoreCheckpoint(checkpoint); err != nil {
		return nil, err
	}
	rng := rand.New(rand.NewSource(seed))
	data := make([]Phase2Task, cases)
	optimal := 0.0
	for i := range data {
		x, z := .08*rng.Float64(), 1.8*rng.Float64()
		state := map[k.NodeID]float64{}
		var goal []k.NodeID
		for j := 0; j < width; j++ {
			state[unit(4, j)] = x
			state[unit(6, j)] = z
			goal = append(goal, unit(5, j))
		}
		data[i] = Phase2Task{State: state, Goal: goal, Relation: 1}
		optimal += math.Max(0, x-.04) / float64(cases)
	}
	metrics := phase2Evaluate(rt, data, false, seed+100000, "trained")
	return map[string]any{"metrics": metrics, "oracle_net_return": optimal, "regret": optimal - metrics.Return, "checkpoint": checkpoint, "stream": seed}, nil
}
