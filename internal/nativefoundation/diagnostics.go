package nativefoundation

import (
	"fmt"
	k "github.com/project-horizon/horizon-core/services/ai/knowledge"
	r "github.com/project-horizon/horizon-core/services/ai/runtime"
	"math"
	"os"
	"path/filepath"
)

// CausalDiagnostic reports interventions on disposable replicas, never on the
// saved training substrate. Temporal credit is not labeled causal precision.
type CausalDiagnostic struct {
	RelevantAblation                 Metrics `json:"relevant_world_edge_ablation"`
	UnrelatedAblation                Metrics `json:"unrelated_world_edge_ablation"`
	RelevantParameterUpdates         int     `json:"relevant_parameter_updates"`
	ActiveDistractorParameterUpdates int     `json:"active_distractor_parameter_updates"`
	InactiveParameterUpdates         int     `json:"inactive_parameter_updates"`
	CreditPrecision                  float64 `json:"credit_precision_excluding_bias"`
	FixedActionDistractorEffect      float64 `json:"fixed_action_distractor_reward_effect"`
}

func causalDiagnostics(dir string, seed, stream int64, cases int) (CausalDiagnostic, error) {
	out := CausalDiagnostic{}
	for _, stage := range []int{3, 0} {
		brain := k.NewBrain()
		rt := r.NewBrainRuntime(brain)
		if err := rt.RestoreCheckpoint(filepath.Join(dir, "trained-checkpoint.json")); err != nil {
			return out, err
		}
		// Test depth 0..3 never activates stage 0. Stage 3 is on the path for
		// depth 2 and 3. No policy parameter is removed or retrained.
		brain.Lock()
		for j := 0; j < width; j++ {
			for z := 0; z < width; z++ {
				delete(brain.Registry.GetByID(unit(stage, j)).Synapses, unit(stage+1, z))
			}
		}
		brain.Unlock()
		m := evaluate(rt, tasks(stream+seed, cases, "test"), false, stream+seed, "trained")
		if stage == 3 {
			out.RelevantAblation = m
		} else {
			out.UnrelatedAblation = m
		}
	}
	probeDir := filepath.Join(dir, "credit-probe")
	if err := os.MkdirAll(probeDir, 0700); err != nil {
		return out, err
	}
	brain, err := NewFixture(probeDir, false)
	if err != nil {
		return out, err
	}
	rt := r.NewBrainRuntime(brain)
	cfg := Config(seed, 1)
	apply(rt, r.NativeControlCommand{ID: "init", Initialize: &cfg})
	before := parameters(brain)
	state := map[k.NodeID]float64{}
	for j := 0; j < width; j++ {
		state[unit(4, j)] = .8
		state[unit(6, j)] = .8
	}
	apply(rt, r.NativeControlCommand{ID: "probe", Begin: &r.NativeBegin{EpisodeID: "probe", Training: true, Observation: &r.NativeObservation{Source: "credit-probe", SampleID: "probe", State: state}}})
	// Both actions have nonzero on-policy probability; choose their inverse-CDF
	// samples for a controlled trajectory without permitting training overrides.
	apply(rt, r.NativeControlCommand{ID: "propagate", Step: &r.NativeStep{EpisodeID: "probe", Sample: 0}})
	stop := apply(rt, r.NativeControlCommand{ID: "stop", Step: &r.NativeStep{EpisodeID: "probe", Sample: math.Nextafter(1, 0)}})
	if stop.Readout.Action != 3 {
		return out, fmt.Errorf("credit probe stop not selected")
	}
	signal := func(x map[k.NodeID]float64) float64 {
		v := 0.0
		for j := 0; j < width; j++ {
			v += x[unit(5, j)] / width
		}
		return v
	}
	reward := signal(stop.State)
	without := map[k.NodeID]float64{}
	for id, v := range stop.State {
		if id < unit(6, 0) {
			without[id] = v
		}
	}
	out.FixedActionDistractorEffect = reward - signal(without)
	apply(rt, r.NativeControlCommand{ID: "outcome", Outcome: &r.NativeOutcome{EpisodeID: "probe", TraceDigest: stop.TraceDigest, Source: "credit-probe", OutcomeID: "observed-terminal-signal", Provenance: "observed-environment", Reward: reward, Learn: true}})
	after := parameters(brain)
	s := brain.NativeControlSnapshot()
	perSource := len(s.Outputs) * width
	for i, w := range after {
		if w == before[i] {
			continue
		}
		sourceIndex := i / perSource
		if sourceIndex >= len(s.Config.Sources) {
			continue
		}
		stage := (int(s.Config.Sources[sourceIndex]) - 1) / width
		switch stage {
		case 4, 5:
			out.RelevantParameterUpdates++
		case 6:
			out.ActiveDistractorParameterUpdates++
		default:
			out.InactiveParameterUpdates++
		}
	}
	total := out.RelevantParameterUpdates + out.ActiveDistractorParameterUpdates + out.InactiveParameterUpdates
	if total > 0 {
		out.CreditPrecision = float64(out.RelevantParameterUpdates) / float64(total)
	}
	return out, nil
}
