package runtime

// Continuation only: no initialization/training of A-D and no policy reset.
// The numerical environment reuses the C outcome meter and existing APIs.
import (
	"bytes"
	"encoding/json"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	k "github.com/project-horizon/horizon-core/services/ai/knowledge"
)

type e1Probe struct {
	Signal, Noise       float64
	Condition, Nuisance int
	Values, Returns     []float64
	Action, Optimal     int
}
type e1Metrics struct {
	Accuracy, OptimalProbability, Regret, NuisanceSensitivity, MaxNuisanceConsequence float64
	Probes                                                                            []e1Probe
}

func e1Parent(t *testing.T) (string, [][]k.NodeID) {
	t.Helper()
	root := os.Getenv("HORIZON_CONTINUAL_PARENT")
	if root == "" {
		t.Skip("requires manifest-verified PR68 parent; run scripts/prepare_continual_parent.py")
	}
	var pop [][]k.NodeID
	raw, err := os.ReadFile(filepath.Join(root, "populations.json"))
	if err != nil {
		t.Fatal(err)
	}
	if err = json.Unmarshal(raw, &pop); err != nil || len(pop) != 9 {
		t.Fatal("invalid PR68 populations", err)
	}
	return root, pop
}
func e1Load(t *testing.T, parent, directory string) (*BrainRuntime, *EventLog) {
	t.Helper()
	if err := os.MkdirAll(directory, 0755); err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(filepath.Join(parent, "events.jsonl"))
	if err != nil {
		t.Fatal(err)
	}
	file, err := os.OpenFile(filepath.Join(directory, "events.jsonl"), os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
	if err != nil {
		t.Fatal("refuse to overwrite continuation journal", err)
	}
	if _, err = file.Write(raw); err != nil {
		t.Fatal(err)
	}
	if err = file.Close(); err != nil {
		t.Fatal(err)
	}
	log, err := OpenEventLog(filepath.Join(directory, "events.jsonl"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { log.Close() })
	r, err := RebuildRuntimeFromCheckpointAndJournal(log, filepath.Join(parent, "05-restart-continuation.json"))
	if err != nil {
		t.Fatal(err)
	}
	if r.LastSequence() != 10817 || r.brain.NativeControlSnapshot().Updates != 1536 {
		t.Fatal("wrong parent; no replacement Brain allowed")
	}
	canonical, err := r.brain.CanonicalJSON()
	if err != nil {
		t.Fatal(err)
	}
	if k.NativeDigest(json.RawMessage(canonical)) != "cfbe7b6f625b2832962acc777e095f7cc9d3303a41deccdeaf82d40e15287acb" || k.NativeDigest(json.RawMessage(continuationBytes(t, r))) != "d2d9a3a0d648993dcff17b4a8e34bedb061c701373091d31e5c64351fa66af91" {
		t.Fatal("PR68 complete state hash mismatch")
	}
	r.SetEventLog(log)
	return r, log
}
func e1State(pop [][]k.NodeID, condition, nuisance int, signal, noise float64) map[k.NodeID]float64 {
	x := flexCState(pop, condition, signal)
	for _, id := range pop[nuisance] {
		x[id] = noise
	}
	return x
}
func e1Measure(t *testing.T, checkpoint string, pop [][]k.NodeID, options ...bool) e1Metrics {
	t.Helper()
	r := orderedFixtureClone(t, checkpoint)
	if len(options) > 1 && options[1] {
		s := r.brain.NativeControlSnapshot()
		for _, from := range s.ParameterSources() {
			for _, group := range s.Outputs {
				for _, to := range group {
					e := r.brain.Registry.GetByID(from).Synapses[to][0]
					e.Weight = 0
					e.Dynamic.Weight = 0
					e.Inhibitory = false
				}
			}
		}
	}
	signals, noises := []float64{.4, .6, .8}, []float64{.1, .5, .9}
	if len(options) > 0 && options[0] {
		signals, noises = []float64{.35, .55, .75}, []float64{.2, .4, .8}
	}
	m := e1Metrics{}
	for _, signal := range signals {
		for condition := 0; condition < 2; condition++ {
			clean := flexCState(pop, condition, signal)
			q, err := r.brain.NativePolicy(clean, true, 0)
			if err != nil {
				t.Fatal(err)
			}
			cleanReturns := []float64{}
			for action := 0; action < 2; action++ {
				cleanReturns = append(cleanReturns, flexTrial(t, r, pop, clean, fmt.Sprintf("clean-%g-%d-%d", signal, condition, action), "", action, false))
			}
			for _, nuisance := range []int{0, 8} {
				for _, noise := range noises {
					state := e1State(pop, condition, nuisance, signal, noise)
					p, err := r.brain.NativePolicy(state, true, 0)
					if err != nil {
						t.Fatal(err)
					}
					returns := []float64{}
					for action := 0; action < 2; action++ {
						value := flexTrial(t, r, pop, state, fmt.Sprintf("held-%g-%d-%d-%g-%d", signal, condition, nuisance, noise, action), "", action, false)
						returns = append(returns, value)
						m.MaxNuisanceConsequence = math.Max(m.MaxNuisanceConsequence, math.Abs(value-cleanReturns[action]))
					}
					best := 0
					if returns[1] > returns[0] {
						best = 1
					}
					if p.Action == best {
						m.Accuracy++
					}
					m.OptimalProbability += p.Probabilities[best]
					m.Regret += returns[best] - returns[p.Action]
					m.NuisanceSensitivity += math.Abs((p.Values[0] - p.Values[1]) - (q.Values[0] - q.Values[1]))
					m.Probes = append(m.Probes, e1Probe{signal, noise, condition, nuisance, p.Values, returns, p.Action, best})
				}
			}
		}
	}
	n := float64(len(m.Probes))
	m.Accuracy /= n
	m.OptimalProbability /= n
	m.Regret /= n
	m.NuisanceSensitivity /= n
	return m
}

// Mechanistically grounded combination probe: B's learned context remains
// eligible across intervening observed nuisance frames. It does not invent an
// unobserved intermediate or claim a new internally planned composition.
func e1ContextThroughNoise(t *testing.T, checkpoint string, pop [][]k.NodeID) []float64 {
	t.Helper()
	margins := []float64{}
	for side := 0; side < 2; side++ {
		rr := orderedFixtureClone(t, checkpoint)
		flexQuery(t, rr, pop, []int{3 + side}, nil, true, false)
		flexQuery(t, rr, pop, []int{8, 0}, nil, false, false)
		flexQuery(t, rr, pop, []int{5}, nil, false, false)
		p := rr.activation.PredictionSnapshot().State
		margins = append(margins, flexMean(p, pop[6+side])-flexMean(p, pop[7-side]))
	}
	return margins
}

// Measure the encoder's complete linear input basis through canonical edges
// on an evaluation copy. This is diagnosis, never a trained parameter store.
func e1Basis(t *testing.T, checkpoint string, states []map[k.NodeID]float64) [][]float64 {
	t.Helper()
	r := orderedFixtureClone(t, checkpoint)
	s := r.brain.NativeControlSnapshot()
	set := func(from k.NodeID, weight float64) {
		for _, to := range s.Outputs[0] {
			e := r.brain.Registry.GetByID(from).Synapses[to][0]
			e.Weight = weight
			e.Dynamic.Weight = weight
			e.Inhibitory = false
		}
	}
	for _, from := range s.ParameterSources() {
		for _, group := range s.Outputs {
			for _, to := range group {
				e := r.brain.Registry.GetByID(from).Synapses[to][0]
				e.Weight = 0
				e.Dynamic.Weight = 0
				e.Inhibitory = false
			}
		}
	}
	result := [][]float64{}
	for _, state := range states {
		values := []float64{}
		for _, from := range s.ParameterSources() {
			set(from, .125)
			p, err := r.brain.NativePolicy(state, true, 0)
			if err != nil {
				t.Fatal(err)
			}
			values = append(values, p.Values[0])
			set(from, 0)
		}
		result = append(result, values)
	}
	return result
}
func e1Retention(t *testing.T, checkpoint string, pop [][]k.NodeID) map[string]any {
	t.Helper()
	data, err := os.ReadFile("testdata/continual_flexibility_v1.json")
	if err != nil {
		t.Fatal(err)
	}
	var cfg flexCurriculum
	if err = json.Unmarshal(data, &cfg); err != nil {
		t.Fatal(err)
	}
	return map[string]any{"A": flexMargins(t, checkpoint, pop, cfg.A), "B": flexMargins(t, checkpoint, pop, cfg.B), "C": flexC(t, checkpoint, pop, cfg.EvalAmplitudes), "D": flexReuse(t, checkpoint, pop, true)}
}
func TestContinualEPrerequisites(t *testing.T) {
	parent, pop := e1Parent(t)
	r, _ := e1Load(t, parent, t.TempDir())
	cp := orderedSave(t, r)
	// B learned p->r / q->s. With the SAME r+gate activity, these contexts
	// demand different actions if the observed goal switches between r and s.
	states := []map[k.NodeID]float64{}
	for side := 0; side < 2; side++ {
		x := flexCState(pop, 0, .4)
		for _, id := range pop[6] {
			x[id] = .8
		}
		for _, id := range pop[3+side] {
			x[id] = .6
		}
		states = append(states, x)
	}
	basis := e1Basis(t, cp, states)
	if !reflect.DeepEqual(basis[0], basis[1]) {
		t.Fatal("re-audit prerequisite: context basis no longer aliases")
	}
	returns := [][]float64{}
	for side, state := range states {
		rr := orderedFixtureClone(t, cp)
		row := []float64{}
		for action := 0; action < 2; action++ {
			flexTrial(t, rr, pop, state, fmt.Sprintf("context-%d-%d", side, action), "", action, false)
			value := flexMean(rr.activation.CurrentState(), pop[6+side])
			if action == 0 {
				value -= .05
			}
			row = append(row, value)
		}
		returns = append(returns, row)
	}
	if !(returns[0][1] > returns[0][0] && returns[1][0] > returns[1][1]) {
		t.Fatal("counterexample did not require opposite operations", returns)
	}
	result := map[string]any{"parent_sequence": r.LastSequence(), "E1_baseline": e1Measure(t, cp, pop), "retention": e1Retention(t, cp, pop), "context_goal_basis": basis, "context_goal_factual_returns": returns, "E2_status": "blocked for direct current-state policy: exact feature collision; not a failed training"}
	if root := os.Getenv("HORIZON_CONTINUAL_EVIDENCE_DIR"); root != "" {
		if err := os.MkdirAll(root, 0755); err != nil {
			t.Fatal(err)
		}
		flexJSON(t, filepath.Join(root, "prerequisites.json"), result)
	}
	t.Logf("parent=%d updates=1536 E1=%+v E2 alias=%v returns=%v", r.LastSequence(), result["E1_baseline"], basis, returns)
}
func TestContinualE1(t *testing.T) {
	parent, pop := e1Parent(t)
	directory := os.Getenv("HORIZON_CONTINUAL_TRAINING_DIR")
	if directory == "" {
		directory = t.TempDir()
	}
	r, wal := e1Load(t, parent, directory)
	owner := r.brain
	r.SetClock(NewFixedClock(flexAt(r)))
	start := filepath.Join(directory, "05-parent-verified.json")
	if err := r.Checkpoint(start); err != nil {
		t.Fatal(err)
	}
	before := e1Measure(t, start, pop)
	validationBefore := e1Measure(t, start, pop, true)
	derivedBefore := e1ContextThroughNoise(t, start, pop)
	retainedBefore := e1Retention(t, start, pop)
	flexJSON(t, filepath.Join(directory, "before.json"), map[string]any{"E1": before, "retention": retainedBefore})
	// Source freeze and local targets are locked before this main continuation.
	// A pre-registered bounded continuation: up to four blocks, same Brain and
	// code. Only the separate validation set controls stopping, never holdout.
	const blockRounds, maxBlocks = 48, 4
	pairs, blocks := 0, 0
	validation := []e1Metrics{}
	for block := 0; block < maxBlocks; block++ {
		for round := block * blockRounds; round < (block+1)*blockRounds; round++ {
			for _, signal := range []float64{.3, .5, .7, .9} {
				for _, noise := range []float64{0, .25, .6, 1} {
					for _, nuisance := range []int{0, 8} {
						for condition := 0; condition < 2; condition++ {
							pair := fmt.Sprintf("E1-%d-%g-%g-%d-%d", round, signal, noise, nuisance, condition)
							state := e1State(pop, condition, nuisance, signal, noise)
							for _, action := range []int{round % 2, 1 - round%2} {
								flexTrial(t, r, pop, state, fmt.Sprintf("%s-%d", pair, action), pair, action, true)
							}
							pairs++
						}
					}
				}
			}
		}
		blocks++
		r.SetClock(NewFixedClock(flexAt(r)))
		blockCP := filepath.Join(directory, fmt.Sprintf("06-E1-block-%d.json", blocks))
		if err := r.Checkpoint(blockCP); err != nil {
			t.Fatal(err)
		}
		v := e1Measure(t, blockCP, pop, true)
		validation = append(validation, v)
		flexJSON(t, filepath.Join(directory, "validation.json"), validation)
		t.Logf("block %d pairs=%d validation accuracy=%g regret=%g sensitivity=%g", blocks, pairs, v.Accuracy, v.Regret, v.NuisanceSensitivity)
		if v.Accuracy >= .9 && v.Regret <= validationBefore.Regret*.5 && v.NuisanceSensitivity <= validationBefore.NuisanceSensitivity*.7 {
			break
		}
	}
	if r.brain != owner || r.activation.Memory != owner || r.learning.Kb != owner {
		t.Fatal("continuation replaced Brain owner")
	}
	r.SetClock(NewFixedClock(flexAt(r)))
	final := filepath.Join(directory, "06-E1.json")
	if err := r.Checkpoint(final); err != nil {
		t.Fatal(err)
	}
	after := e1Measure(t, final, pop)
	retainedAfter := e1Retention(t, final, pop)
	genesis, err := RebuildRuntimeFromCommittedJournal(wal, k.NewBrain)
	if err != nil {
		t.Fatal(err)
	}
	suffix, err := RebuildRuntimeFromCheckpointAndJournal(wal, start)
	if err != nil {
		t.Fatal(err)
	}
	for _, rr := range []*BrainRuntime{genesis, suffix} {
		if !bytes.Equal(continuationBytes(t, r), continuationBytes(t, rr)) {
			t.Fatal("E1 complete replay mismatch")
		}
	}
	restarted, err := RebuildRuntimeFromCheckpointAndJournal(wal, final)
	if err != nil {
		t.Fatal(err)
	}
	restarted.SetEventLog(nil)
	for _, rr := range []*BrainRuntime{r, restarted} {
		flexQuery(t, rr, pop, []int{0}, nil, true, false)
		if _, _, err := rr.ThinkAt(1, flexAt(rr)); err != nil {
			t.Fatal(err)
		}
	}
	if !bytes.Equal(continuationBytes(t, r), continuationBytes(t, restarted)) {
		t.Fatal("E1 next recurrent transition changed on restart")
	}
	r.SetClock(NewFixedClock(flexAt(r)))
	next := filepath.Join(directory, "07-E1-restarted.json")
	if err = r.Checkpoint(next); err != nil {
		t.Fatal(err)
	}
	updates := r.brain.NativeControlSnapshot().Updates
	retainedRestart := e1Retention(t, next, pop)
	ablation := e1Measure(t, next, pop, false, true)
	derivedAfter := e1ContextThroughNoise(t, next, pop)
	for _, build := range []func() (*BrainRuntime, error){func() (*BrainRuntime, error) { return RebuildRuntimeFromCommittedJournal(wal, k.NewBrain) }, func() (*BrainRuntime, error) { return RebuildRuntimeFromCheckpointAndJournal(wal, start) }} {
		rr, err := build()
		if err != nil {
			t.Fatal(err)
		}
		if !bytes.Equal(continuationBytes(t, r), continuationBytes(t, rr)) {
			t.Fatal("final E1 checkpoint replay mismatch")
		}
	}
	flexJSON(t, filepath.Join(directory, "results.json"), map[string]any{"parent_sequence": 10817, "final_sequence": r.LastSequence(), "native_updates": updates, "pairs": pairs, "blocks": blocks, "before": before, "after": after, "validation_before": validationBefore, "validation": validation, "policy_ablation": ablation, "retention_before": retainedBefore, "retention_after_training": retainedAfter, "retention_after_restart": retainedRestart, "derived_context_noise_before": derivedBefore, "derived_context_noise_after": derivedAfter, "complete_genesis_replay": true, "complete_suffix_replay": true, "next_transition_equal": true, "final_checkpoint": "07-E1-restarted.json", "source": "PR68 continuation; no A-D training, no model initialization"})
	checked, err := RebuildRuntimeFromCheckpointAndJournal(wal, next)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(continuationBytes(t, r), continuationBytes(t, checked)) {
		t.Fatal("final checkpoint failed verified restore")
	}
	events, err := ReadEventLog(filepath.Join(directory, "events.jsonl"))
	if err != nil {
		t.Fatal(err)
	}
	for i := len(events) - 1; i >= 0; i-- {
		event := events[i]
		if event.NativeControl == nil || event.NativeControl.Outcome == nil {
			continue
		}
		beforeRetry := continuationBytes(t, checked)
		retry, err := checked.ApplyNativeControl(*event.NativeControl, event.Timestamp)
		if err != nil || !retry.Duplicate || !bytes.Equal(beforeRetry, continuationBytes(t, checked)) {
			t.Fatal("E1 outcome retried learning", err)
		}
		changed := *event.NativeControl
		outcome := *changed.Outcome
		outcome.Reward += .01
		changed.Outcome = &outcome
		if _, err = checked.ApplyNativeControl(changed, event.Timestamp); err == nil || !bytes.Equal(beforeRetry, continuationBytes(t, checked)) {
			t.Fatal("E1 changed outcome accepted")
		}
		break
	}
	if updates != uint64(1536+pairs) {
		t.Error("native learning update lost or duplicated")
	}
	if before.MaxNuisanceConsequence > 1e-12 || after.MaxNuisanceConsequence > 1e-12 {
		t.Error("proposed nuisance actually changed factual return")
	}
	if after.Accuracy < .9 || after.Accuracy < before.Accuracy+.1 || after.Regret > before.Regret*.5 || after.NuisanceSensitivity > before.NuisanceSensitivity*.7 {
		t.Errorf("E1 local target not met: before=%+v after=%+v", before, after)
	}
	if ablation.Accuracy >= after.Accuracy {
		t.Error("E1 improvement survives policy-parameter ablation")
	}
	for name, values := range retainedRestart {
		if name == "C" {
			if values.(*flexPolicyMetrics).Accuracy < 5.0/6 {
				t.Error("C choice retention failed")
			}
			continue
		}
		for _, v := range values.([]float64) {
			if v < .1 {
				t.Error("retention", name, v)
			}
		}
	}
	t.Logf("E1 seq=%d updates=%d accuracy=%g -> %g regret=%g -> %g nuisance sensitivity=%g -> %g", r.LastSequence(), updates, before.Accuracy, after.Accuracy, before.Regret, after.Regret, before.NuisanceSensitivity, after.NuisanceSensitivity)
}
