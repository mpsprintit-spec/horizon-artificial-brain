package main

// Isolated finite environment and protocol runner. No simulator or reference
// probability is compiled into the production inference command.
import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	k "github.com/project-horizon/horizon-core/services/ai/knowledge"
	r "github.com/project-horizon/horizon-core/services/ai/runtime"
	"io"
	"math"
	"math/rand"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"syscall"
	"testing"
	"time"
)

type example struct {
	ID              string
	Context, Target int
	Outcomes        [3]int
	Costs           [3]float64
	Order           [3]int
	Unseen          bool
}
type protocol struct {
	Source, ParentSHA, PrefixSHA        string
	Train, Confirmatory, Batches, Batch int
	Seeds                               map[string]int64
	DataSHA                             map[string]string
	Definitions                         map[string]string
}

func must(t *testing.T, e error) {
	t.Helper()
	if e != nil {
		t.Fatal(e)
	}
}
func path(root, n string) string { return filepath.Join(root, n) }
func save(t *testing.T, p string, v any) {
	b, e := json.MarshalIndent(v, "", "  ")
	must(t, e)
	must(t, os.MkdirAll(filepath.Dir(p), 0700))
	must(t, os.WriteFile(p, b, 0600))
}
func read(t *testing.T, p string, v any) {
	b, e := os.ReadFile(p)
	must(t, e)
	must(t, json.Unmarshal(b, v))
}
func hash(t *testing.T, p string) string {
	f, e := os.Open(p)
	must(t, e)
	defer f.Close()
	h := sha256.New()
	_, e = io.Copy(h, f)
	must(t, e)
	return fmt.Sprintf("%x", h.Sum(nil))
}
func copyFile(t *testing.T, a, b string) {
	f, e := os.Open(a)
	must(t, e)
	defer f.Close()
	o, e := os.OpenFile(b, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
	must(t, e)
	_, e = io.Copy(o, f)
	must(t, e)
	must(t, o.Close())
}
func reference(context int) []float64 {
	z := make([]float64, 108)
	sum := 0.
	for state := range z {
		y := state % 4
		v := state / 4
		o := []int{v % 3, (v / 3) % 3, (v / 9) % 3}
		yf := math.Cos(math.Pi * float64(y) / 2)
		v0 := .35 * math.Sin(float64((context+1)*(y+1))*.7)
		for i := 0; i < 3; i++ {
			f := float64(o[i] - 1)
			v0 += .3*math.Sin(float64(context+i+2))*f + .65*math.Cos(float64(context+i+1)*.6)*yf*f
			for j := i + 1; j < 3; j++ {
				v0 += .45 * math.Sin(float64(context+i+j+1)*.8) * f * float64(o[j]-1)
			}
		}
		z[state] = math.Exp(v0)
		sum += z[state]
	}
	for i := range z {
		z[i] /= sum
	}
	return z
}
func draw(p []float64, rng *rand.Rand) int {
	u := rng.Float64()
	for i, v := range p {
		u -= v
		if u < 0 {
			return i
		}
	}
	return len(p) - 1
}
func makeCase(i int, seed int64, train bool) example {
	rng := rand.New(rand.NewSource(seed))
	c := example{Context: i % 5}
	p := reference(c.Context)
	s := draw(p, rng)
	for train && (s/4)%7 == 6 {
		s = draw(p, rng)
	}
	c.Target = s % 4
	s /= 4
	for op := 0; op < 3; op++ {
		c.Outcomes[op] = s % 3
		s /= 3
		c.Costs[op] = .015 + .025*rng.Float64()
	}
	copy(c.Order[:], rng.Perm(3))
	c.Unseen = (c.Outcomes[0]+3*c.Outcomes[1]+9*c.Outcomes[2])%7 == 6
	c.ID = k.CognitiveDigest(struct {
		Seed int64
		Case example
	}{seed, c})
	return c
}
func input(c example, history []int, greedy bool, u float64) k.CognitiveInput {
	x := make([]float64, 28)
	x[c.Context] = 1
	x[24] = float64(len(history)) / 4
	mask := []bool{true, true, true, true, true, true, true, true}
	for _, op := range history {
		mask[op] = false
	}
	for op, cost := range c.Costs {
		x[25+op] = cost / .1
	}
	if len(history) > 0 {
		op := history[len(history)-1]
		x[5+c.Outcomes[op]] = 1
		x[8+op] = 1
		x[11+op] = 1
	}
	return k.CognitiveInput{Features: x, Mask: mask, Reset: len(history) == 0, Draw: u, NeuralMeta: &k.NeuralMetaInput{Mode: "recurrent", Costs: append([]float64(nil), c.Costs[:]...), Remaining: 3 - len(history), Greedy: greedy}}
}
func trace(c example, h []int, greedy bool, draws []float64) []k.CognitiveInput {
	out := []k.CognitiveInput{}
	for i := 0; i <= len(h); i++ {
		u := .5
		if i < len(draws) {
			u = draws[i]
		}
		out = append(out, input(c, h[:i], greedy, u))
	}
	return out
}
func terminalMask(tr []k.CognitiveInput) {
	for a := 0; a < 3; a++ {
		tr[len(tr)-1].Mask[a] = false
	}
}
func predict(t *testing.T, rt *r.BrainRuntime, tr []k.CognitiveInput, candidate bool, drop k.Phase4DropV5) k.CoordinatedPredictionV5 {
	t.Helper()
	if candidate {
		p, e := rt.PreviewCoordinatedV5(tr, drop)
		must(t, e)
		return p
	}
	p, e := rt.PreviewPhase4V5(k.Phase2ProbeV5{Trace: tr}, drop)
	must(t, e)
	out := k.CoordinatedPredictionV5{Phase4PredictionV5: p, Competence: make([]float64, 5), TerminalUtility: make([]float64, 5)}
	readout := p.Frozen.Readout
	for y, py := range readout.ClassBelief {
		for a := 0; a < 5; a++ {
			out.Competence[a] += py * readout.TerminalCompetence[y][a]
			out.TerminalUtility[a] += py * readout.TerminalReward[y][a]
		}
	}
	return out
}
func payoff(c example, a int) float64 {
	if a < 3 {
		return -c.Costs[a]
	}
	if a == 7 {
		return .35
	}
	if a-3 == c.Target {
		return 1
	}
	return 0
}
func trueClass(c example, h []int) []float64 {
	p := reference(c.Context)
	q := make([]float64, 4)
	sum := 0.
	for state, v := range p {
		match := true
		for _, op := range h {
			if (state/4/int(math.Pow(3, float64(op))))%3 != c.Outcomes[op] {
				match = false
			}
		}
		if match {
			q[state%4] += v
			sum += v
		}
	}
	for i := range q {
		q[i] /= sum
	}
	return q
}

type row struct {
	ID                                                                             string
	Prefix, Action, Operations                                                     int
	Unseen                                                                         bool
	Return, Terminal, Expected, Cost, PAbstain, Competence, PredictedReward, Value float64
	Policy                                                                         []float64
}

func rollout(t *testing.T, rt *r.BrainRuntime, c example, candidate, greedy bool, prefix int, seed int64, drop k.Phase4DropV5) (k.CoordinatedEpisodeV5, row) {
	ep := k.CoordinatedEpisodeV5{}
	h := []int{}
	cost := 0.
	if prefix >= 0 {
		for _, op := range c.Order[:prefix] {
			h = append(h, op)
			cost += c.Costs[op]
		}
	}
	rng := rand.New(rand.NewSource(seed))
	us := []float64{}
	for i := 0; i < 4; i++ {
		us = append(us, rng.Float64())
	}
	result := row{ID: c.ID, Prefix: prefix, Unseen: c.Unseen}
	for step := len(h); step <= 3; step++ {
		tr := trace(c, h, greedy, us)
		if prefix >= 0 {
			terminalMask(tr)
		}
		p := predict(t, rt, tr, candidate, drop)
		last := tr[len(tr)-1]
		reward := payoff(c, p.Action)
		f := k.Phase4FrameV5{Features: p.Features, Mask: last.Mask, Action: p.Action, Reward: reward, Draw: last.Draw, BehaviorProbability: p.Policy[p.Action], BehaviorValue: p.Value}
		ep.Traces = append(ep.Traces, tr)
		ep.Frames = append(ep.Frames, f)
		if p.Action >= 3 {
			q := trueClass(c, h)
			expected := .35
			if p.Action != 7 {
				expected = q[p.Action-3]
			}
			result.Action = p.Action
			result.Operations = len(h)
			result.Cost = cost
			result.Terminal = reward
			result.Return = reward - cost
			result.Expected = expected - cost
			result.PAbstain = p.Policy[7]
			result.Value = p.Value
			result.Policy = p.Policy
			result.Competence = p.Competence[p.Action-3]
			result.PredictedReward = p.TerminalUtility[p.Action-3]
			return ep, result
		}
		cost += c.Costs[p.Action]
		h = append(h, p.Action)
	}
	t.Fatal("unclosed episode")
	return ep, result
}
func loadProtocol(t *testing.T, root string) protocol {
	var p protocol
	read(t, path(root, "protocol.json"), &p)
	b, e := exec.Command("git", "rev-parse", "HEAD").Output()
	must(t, e)
	if string(b) != p.Source+"\n" {
		t.Fatal("source differs from seal")
	}
	status, e := exec.Command("git", "status", "--porcelain").Output()
	must(t, e)
	if len(status) > 0 {
		t.Fatal("sealed worktree dirty")
	}
	for n, h := range p.DataSHA {
		if hash(t, path(root, "data/"+n+".json")) != h {
			t.Fatal("sealed split modified")
		}
	}
	return p
}
func seal(t *testing.T, root string) {
	parent := os.Getenv("HORIZON_THREE_PARENT")
	if parent == "" {
		t.Fatal("verified authentic parent required")
	}
	p := protocol{ParentSHA: "abfac2da545539eadf6be36269e61c5c58ba3c206b66abdb9e285e56f98b4d7a", PrefixSHA: "99e66015ade8070224520e8b508ae9a46deac342684ecf5298fc5b25c95c2b85", Train: 2048, Confirmatory: 1024, Batches: 128, Batch: 16, Seeds: map[string]int64{"train": 988210000000, "development": 988220000000, "confirmatory": 988230000000, "draw": 988240000000, "bootstrap": 988250000000}, DataSHA: map[string]string{}, Definitions: map[string]string{"arms": "one coordinated candidate vs authentic historical head-only control, same initialized N and Adam state, same data/cost/budget/draw streams; MUSE update and coordinated contract are intentional treatment", "warmup": "first64 batches externally acquired prefix depth i%4; all answers+abstain legal; acquisition sunk cost recorded; next64 batches autonomous full-task MC return incl operation costs", "gradient": "world+observation GRU+legacy actor frozen; selected outcome MUSE BCE/reward gradient with posterior mixture, no verified class label; head actor-critic updates, separate existing clocks/moments, stop-gradient between feature estimator and policy loss", "inference": "one final actor, historical actor channels removed, observed-only Bayesian conditioning, known abstain payoff.35, VOC one-step policy-aware BMPS continuation, no oracle", "primary": "greedy whole-task realized net return; paired1024-case bootstrap2000, +.02 vs abstain and historical control with lowerCI>0; one seed pilot", "secondary": "terminal prefixes0..3, seen/unseen, selected MUSE Brier/rewardMSE, action histogram, first128 fixed-parameter lesions/probe correlation; no postconfirm fitting"}}
	b, e := exec.Command("git", "rev-parse", "HEAD").Output()
	must(t, e)
	p.Source = string(b[:len(b)-1])
	if hash(t, parent) != p.ParentSHA {
		t.Fatal("parent SHA mismatch")
	}
	must(t, os.MkdirAll(path(root, "data"), 0700))
	copyFile(t, parent, path(root, "parent.json"))
	rt := r.NewBrainRuntime(k.NewBrain())
	must(t, rt.RestoreCheckpoint(path(root, "parent.json")))
	save(t, path(root, "RECOVERY.json"), map[string]any{"parent_sha": p.ParentSHA, "native_checkpoint_restored": true, "phase4_parameters": 1097, "source": "authentic initialized D_aligned_full, no guessed/replaced weights"})
	rt = nil
	runtime.GC()
	for name, n := range map[string]int{"train": p.Train, "development": 128, "confirmatory": p.Confirmatory} {
		cases := make([]example, n)
		for i := range cases {
			cases[i] = makeCase(i, p.Seeds[name]+int64(i), name == "train")
		}
		save(t, path(root, "data/"+name+".json"), cases)
		p.DataSHA[name] = hash(t, path(root, "data/"+name+".json"))
	}
	save(t, path(root, "protocol.json"), p)
}
func train(t *testing.T, root, arm string) {
	p := loadProtocol(t, root)
	var receipt struct {
		Passed bool `json:"passed"`
	}
	read(t, path(root, "PREFIT_PERSISTENT.json"), &receipt)
	if !receipt.Passed {
		t.Fatal("persistent prefit seal required")
	}
	dir := path(root, "state/"+arm)
	must(t, os.MkdirAll(dir, 0700))
	copyFile(t, path(root, "parent.json"), filepath.Join(dir, "parent.json"))
	copyFile(t, path(root, "parent-events.jsonl"), filepath.Join(dir, "events.jsonl"))
	if hash(t, filepath.Join(dir, "events.jsonl")) != p.PrefixSHA {
		t.Fatal("parent complete prefix SHA mismatch")
	}
	wal, e := r.OpenEventLog(filepath.Join(dir, "events.jsonl"))
	must(t, e)
	defer wal.Close()
	rt, e := r.RebuildRuntimeFromCheckpointAndJournal(wal, filepath.Join(dir, "parent.json"))
	must(t, e)
	rt.SetEventLog(wal)
	initial, e := rt.CognitiveModelParameters()
	must(t, e)
	var cases []example
	read(t, path(root, "data/train.json"), &cases)
	logs := []any{}
	start := time.Now()
	candidate := arm == "candidate"
	for batch := 0; batch < p.Batches; batch++ {
		coord := k.CoordinatedBatchV5{}
		old := k.Phase4BatchV5{}
		sum, ops := 0., 0
		for j := 0; j < p.Batch; j++ {
			i := batch*p.Batch + j
			prefix := -1
			if batch < 64 {
				prefix = i % 4
			}
			ep, result := rollout(t, rt, cases[i], candidate, false, prefix, p.Seeds["draw"]+int64(i), k.Phase4DropV5{})
			coord.Episodes = append(coord.Episodes, ep)
			oldEP := k.Phase4EpisodeV5{CaseID: cases[i].ID, Frames: ep.Frames}
			if prefix >= 0 {
				old.Protocol = k.Phase4TerminalProtocolV5
				acq := &k.Phase4TerminalAcquisitionV5{TraceDigest: k.CognitiveDigest(ep.Traces[0]), Cost: result.Cost}
				for _, op := range cases[i].Order[:prefix] {
					acq.Observed = append(acq.Observed, k.VerifiedMentalResult{Operation: op, Outcome: cases[i].Outcomes[op]})
				}
				oldEP.Acquisition = acq
			}
			old.Episodes = append(old.Episodes, oldEP)
			sum += result.Return
			ops += result.Operations
		}
		w, e := rt.CognitiveModelParameters()
		must(t, e)
		id := fmt.Sprintf("repair-%s-%04d", arm, batch)
		command := r.CognitiveModelCommand{ID: id}
		if candidate {
			coord.ParametersHash = k.CognitiveDigest(w)
			coord.Evidence = k.CognitiveEvidence{ID: id, Source: "actual-executed-environment", Verifier: "selected reward only, no oracle target", Digest: k.CognitiveDigest(coord.Episodes)}
			command.Coordinated = &coord
		} else {
			old.ParametersHash = k.CognitiveDigest(w)
			old.Evidence = k.CognitiveEvidence{ID: id, Source: "actual-executed-environment", Verifier: "original reward/cost", Digest: k.CognitiveDigest(old.Episodes)}
			command.Phase4Training = &old
		}
		res, e := rt.ApplyCognitiveModel(command, time.Unix(2400000000+int64(batch), 0))
		must(t, e)
		logs = append(logs, map[string]any{"update": batch + 1, "net_return": sum / float64(p.Batch), "operations": float64(ops) / float64(p.Batch), "learning": res})
		if (batch+1)%32 == 0 {
			fmt.Println("REPAIR_UPDATE", arm, batch+1, sum/float64(p.Batch))
		}
	}
	final, e := rt.CognitiveModelParameters()
	must(t, e)
	changed := 0
	for i := range initial {
		if initial[i] != final[i] {
			changed++
		}
	}
	must(t, rt.Checkpoint(filepath.Join(dir, "trained.json")))
	rt = nil
	runtime.GC()
	replayed, e := r.RebuildRuntimeFromCheckpointAndJournal(wal, filepath.Join(dir, "parent.json"))
	must(t, e)
	w, e := replayed.CognitiveModelParameters()
	must(t, e)
	if k.CognitiveDigest(w) != k.CognitiveDigest(final) {
		t.Fatal("actual128 updates replay differs")
	}
	var mem syscall.Rusage
	syscall.Getrusage(syscall.RUSAGE_SELF, &mem)
	save(t, path(root, "reports/TRAIN_"+arm+".json"), map[string]any{"passed": true, "source": p.Source, "changed_N_parameters": changed, "initial_hash": k.CognitiveDigest(initial), "final_hash": k.CognitiveDigest(final), "native_full128update_replay_identical": true, "wall_including_native_replay_seconds": time.Since(start).Seconds(), "peak_RSS_KiB": mem.Maxrss, "updates": logs})
}

type metrics struct {
	N, Answered, Correct                                                                                             int
	Return, Expected, Terminal, Coverage, SelectiveAccuracy, Operations, Cost, PAbstain, Brier, RewardMSE, CriticMSE float64
	Actions                                                                                                          [8]int
}

func summary(rows []row) metrics {
	m := metrics{N: len(rows)}
	for _, x := range rows {
		m.Return += x.Return
		m.Terminal += x.Terminal
		m.Expected += x.Expected
		m.Operations += float64(x.Operations)
		m.Cost += x.Cost
		m.PAbstain += x.PAbstain
		m.Actions[x.Action]++
		m.CriticMSE += (x.Value - x.Terminal) * (x.Value - x.Terminal)
		if x.Action != 7 {
			m.Answered++
			m.Correct += int(x.Terminal)
			m.Brier += (x.Competence - x.Terminal) * (x.Competence - x.Terminal)
			m.RewardMSE += (x.PredictedReward - x.Terminal) * (x.PredictedReward - x.Terminal)
		}
	}
	if len(rows) > 0 {
		d := float64(len(rows))
		m.Return /= d
		m.Terminal /= d
		m.Expected /= d
		m.Operations /= d
		m.Cost /= d
		m.PAbstain /= d
		m.Coverage = float64(m.Answered) / d
		m.CriticMSE /= d
	}
	if m.Answered > 0 {
		m.SelectiveAccuracy = float64(m.Correct) / float64(m.Answered)
		m.Brier /= float64(m.Answered)
		m.RewardMSE /= float64(m.Answered)
	}
	return m
}
func bootstrap(delta []float64, seed int64) map[string]any {
	rng := rand.New(rand.NewSource(seed))
	b := make([]float64, 2000)
	avg := 0.
	for _, x := range delta {
		avg += x
	}
	avg /= float64(len(delta))
	for j := range b {
		for range delta {
			b[j] += delta[rng.Intn(len(delta))]
		}
		b[j] /= float64(len(delta))
	}
	sort.Float64s(b)
	return map[string]any{"point": avg, "CI95": [2]float64{b[49], b[1949]}, "cases": len(delta), "lower_above_zero": b[49] > 0}
}
func evaluate(t *testing.T, root string) {
	p := loadProtocol(t, root)
	var cases []example
	read(t, path(root, "data/confirmatory.json"), &cases)
	results := map[string][]row{}
	for _, arm := range []string{"control", "candidate"} {
		rt := r.NewBrainRuntime(k.NewBrain())
		must(t, rt.RestoreCheckpoint(path(root, "state/"+arm+"/trained.json")))
		before, _ := rt.CognitiveModelParameters()
		rows := []row{}
		prefixes := map[int][]row{}
		for i, c := range cases {
			_, x := rollout(t, rt, c, arm == "candidate", true, -1, p.Seeds["draw"]+int64(i), k.Phase4DropV5{})
			rows = append(rows, x)
			for d := 0; d <= 3; d++ {
				_, x = rollout(t, rt, c, arm == "candidate", true, d, p.Seeds["draw"]+int64(i), k.Phase4DropV5{})
				prefixes[d] = append(prefixes[d], x)
			}
		}
		results[arm] = rows
		ms := map[string]metrics{"whole": summary(rows)}
		for d, x := range prefixes {
			ms[fmt.Sprintf("prefix%d", d)] = summary(x)
		}
		for _, unseen := range []bool{false, true} {
			sub := []row{}
			for _, x := range rows {
				if x.Unseen == unseen {
					sub = append(sub, x)
				}
			}
			ms[fmt.Sprintf("unseen_%t", unseen)] = summary(sub)
		}
		save(t, path(root, "reports/EVAL_"+arm+".json"), map[string]any{"metrics": ms, "whole_rows": rows, "prefix_rows": prefixes})
		if arm == "candidate" {
			lesions := map[string]any{}
			for _, name := range []string{"full", "noMUSE", "noVOC", "fullTerminalReference", "noRecurrence"} {
				rr := []row{}
				for i, c := range cases[:128] {
					drop := k.Phase4DropV5{MUSE: name == "noMUSE", VOC: name == "noVOC"}
					if name == "noRecurrence" {
						tr := trace(c, nil, true, nil)
						tr[0].NoRecurrence = true
						_ = tr
					} // full trajectory recurrence lesion below uses replayed prefix probes
					if name == "noRecurrence" || name == "fullTerminalReference" {
						for d := 0; d < 4; d++ {
							tr := trace(c, c.Order[:d], true, nil)
							for j := range tr {
								tr[j].NoRecurrence = name == "noRecurrence"
							}
							terminalMask(tr)
							pr, e := rt.PreviewCoordinatedV5(tr, drop)
							must(t, e)
							x := row{ID: c.ID, Prefix: d, Action: pr.Action, Terminal: payoff(c, pr.Action), Return: payoff(c, pr.Action), Policy: pr.Policy, PAbstain: pr.Policy[7]}
							rr = append(rr, x)
						}
					} else {
						_, x := rollout(t, rt, c, true, true, -1, p.Seeds["draw"]+int64(i), drop)
						rr = append(rr, x)
					}
				}
				lesions[name] = map[string]any{"metrics": summary(rr), "rows": rr}
			}
			save(t, path(root, "reports/LESIONS.json"), lesions)
			times := []float64{}
			tr := trace(cases[0], nil, true, nil)
			for i := 0; i < 35; i++ {
				start := time.Now()
				_, e := rt.PreviewCoordinatedV5(tr, k.Phase4DropV5{})
				must(t, e)
				if i >= 5 {
					times = append(times, float64(time.Since(start).Nanoseconds())/1e6)
				}
			}
			sort.Float64s(times)
			var mem runtime.MemStats
			runtime.ReadMemStats(&mem)
			save(t, path(root, "reports/BENCHMARK.json"), map[string]any{"p50_ms": times[15], "p95_ms": times[28], "heap_alloc_bytes": mem.HeapAlloc, "scope": "full native three-operation lookahead preview, cloud CPU, serial after5warmup,30samples; not mobile"})
		}
		after, _ := rt.CognitiveModelParameters()
		if k.CognitiveDigest(before) != k.CognitiveDigest(after) {
			t.Fatal("evaluation mutated canonical N")
		}
		rt = nil
		runtime.GC()
	}
	da, dc := make([]float64, len(cases)), make([]float64, len(cases))
	for i := range cases {
		da[i] = results["candidate"][i].Return - .35
		dc[i] = results["candidate"][i].Return - results["control"][i].Return
	}
	a, c := bootstrap(da, p.Seeds["bootstrap"]), bootstrap(dc, p.Seeds["bootstrap"]+1)
	pass := a["point"].(float64) >= .02 && c["point"].(float64) >= .02 && a["lower_above_zero"].(bool) && c["lower_above_zero"].(bool)
	save(t, path(root, "reports/PRIMARY.json"), map[string]any{"candidate": summary(results["candidate"]), "control": summary(results["control"]), "vs_abstain": a, "vs_control": c, "scientific_gate_pass": pass, "one_seed_pilot": true})
	fmt.Println("PRIMARY", summary(results["candidate"]).Return, summary(results["control"]).Return, "gate", pass)
}
func TestCoordinatedV5ControlledExperiment(t *testing.T) {
	root := os.Getenv("HORIZON_THREE_ROOT")
	if root == "" {
		t.Skip("explicit isolated opt-in research only")
	}
	switch os.Getenv("HORIZON_THREE_MODE") {
	case "seal":
		seal(t, root)
	case "train-control":
		train(t, root, "control")
	case "train-candidate":
		train(t, root, "candidate")
	case "evaluate":
		evaluate(t, root)
	default:
		t.Fatal("explicit protocol stage required")
	}
}
