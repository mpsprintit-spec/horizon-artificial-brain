package nativefoundation

// ReasoningLab is a deliberately narrow, language-free experiment on Horizon's
// one canonical Brain. It starts with zero neural units and zero synapses,
// acquires a small causal substrate from observed numeric transitions, then
// learns a recurrent operation policy from factual, paired outcomes.
// It is not an implementation of formal logic or general philosophical reasoning.
import (
    "encoding/json"
    "errors"
    "fmt"
    "math"
    "math/rand"
    "os"
    "path/filepath"

    k "github.com/project-horizon/horizon-core/services/ai/knowledge"
    r "github.com/project-horizon/horizon-core/services/ai/runtime"
)

const reasoningWidth = 3
const reasoningStages = 4

type reasoningTask struct {
    State map[k.NodeID]float64
    Stage int
    Truth int // evaluator-only: -1, 0 (conflicting evidence), or +1
}

type ReasoningLabReport struct {
    Seed int64 `json:"seed"`
    Blocks int `json:"training_pairs"`
    TrainingEpisodes int `json:"training_episodes"`
    InitialNodes int `json:"initial_nodes"`
    InitialSynapses int `json:"initial_synapses"`
    AcquiredTransitions int `json:"acquired_experience_transitions"`
    ChangedPolicyParameters int `json:"changed_policy_parameters"`
    Before Metrics `json:"before_training"`
    After Metrics `json:"after_training"`
    Baselines map[string]Metrics `json:"fixed_and_random_baselines"`
    ReversedEvidence Metrics `json:"reversed_evidence_holdout"`
    ConflictingEvidence Metrics `json:"conflicting_evidence_holdout"`
    Interpretation string `json:"interpretation"`
}

type reasoningScaffold struct {
    Chains [2][reasoningStages][reasoningWidth]k.NodeID
    Noise [reasoningWidth]k.NodeID
    Sources []k.NodeID
}

// This creates addresses only. No pretrained neuron, association, lexical
// token, goal node or policy parameter is imported from any stored brain.
func newReasoningLabBrain() (*k.Brain, reasoningScaffold, int, error) {
    brain := k.NewBrain()
    var layout reasoningScaffold
    if brain.Registry.Len() != 0 {
        return nil, layout, 0, errors.New("new brain is not empty")
    }
    index := 0
    alloc := func() (k.NodeID, error) {
        v := make([]float64, 2*reasoningStages*reasoningWidth+reasoningWidth)
        v[index] = 1
        index++
        node, created, err := brain.Registry.GetOrCreateRepresentation(k.NewNeuralVector(v), 1)
        if err != nil {
            return 0, err
        }
        if !created {
            return 0, errors.New("blank neural address unexpectedly reused")
        }
        layout.Sources = append(layout.Sources, node.ID)
        return node.ID, nil
    }
    for side := 0; side < 2; side++ {
        for stage := 0; stage < reasoningStages; stage++ {
            for lane := 0; lane < reasoningWidth; lane++ {
                id, err := alloc()
                if err != nil { return nil, layout, 0, err }
                layout.Chains[side][stage][lane] = id
            }
        }
    }
    for lane := 0; lane < reasoningWidth; lane++ {
        id, err := alloc()
        if err != nil { return nil, layout, 0, err }
        layout.Noise[lane] = id
    }

    // Numerical time-ordered experiences supply adjacency. This is experience
    // acquisition inside N, not a hardcoded inference engine. Nothing is named
    // true, false, premise, conclusion or contradiction in the learned synapses.
    transitions := 0
    for side := 0; side < 2; side++ {
        for stage := 0; stage < reasoningStages-1; stage++ {
            for lane := 0; lane < reasoningWidth; lane++ {
                from, to := layout.Chains[side][stage][lane], layout.Chains[side][stage+1][lane]
                brain.LearnNeuralTrace(k.NeuralTrace{
                    Sequence: []k.PatternStep{
                        {NodeID: from, Position: 0, Activation: .9},
                        {NodeID: to, Position: 1, Activation: .9},
                    },
                    Weight: .9, Confidence: 1, Now: epoch,
                })
                transitions++
            }
        }
    }
    return brain, layout, transitions, nil
}

func reasoningSample(layout reasoningScaffold, rng *rand.Rand, condition string) reasoningTask {
    stage := rng.Intn(reasoningStages)
    truth := 1
    if rng.Intn(2) == 0 { truth = -1 }
    if condition == "conflict" || (condition == "mixed" && rng.Intn(5) == 0) { truth = 0 }
    strong, weak := .83+.15*rng.Float64(), .03+.19*rng.Float64()
    if truth == 0 { strong, weak = .55+.03*rng.Float64(), .55+.03*rng.Float64() }
    if condition == "reversal" {
        // New, independently sampled evidence reverses the previous dominant
        // sign. No truth label is made available to the native policy.
        truth = -truth
    }
    a, b := strong, weak
    if truth < 0 { a, b = weak, strong }
    state := make(map[k.NodeID]float64)
    for lane := 0; lane < reasoningWidth; lane++ {
        state[layout.Chains[0][stage][lane]] = a
        state[layout.Chains[1][stage][lane]] = b
        state[layout.Noise[lane]] = .05+.25*rng.Float64()
    }
    return reasoningTask{State: state, Stage: stage, Truth: truth}
}

func reasoningFeedback(state map[k.NodeID]float64, task reasoningTask, layout reasoningScaffold) float64 {
    var signals [2]float64
    for side := 0; side < 2; side++ {
        for lane := 0; lane < reasoningWidth; lane++ {
            signals[side] += state[layout.Chains[side][reasoningStages-1][lane]] / reasoningWidth
        }
    }
    if task.Truth == 0 {
        return math.Max(-1, -.7*(signals[0]+signals[1]))
    }
    score := 2 * float64(task.Truth) * (signals[0]-signals[1])
    return math.Max(-1, math.Min(1, score))
}

func reasoningEpisode(rt *r.BrainRuntime, layout reasoningScaffold, task reasoningTask, rng *rand.Rand, mode string, first int, contrastID string) Metrics {
    id := fmt.Sprintf("reasoning-%d", rt.LastSequence()+1)
    training := contrastID != ""
    begin := &r.NativeBegin{EpisodeID:id, Training:training,
        Observation:&r.NativeObservation{Source:"reasoning-numeric-observation", SampleID:id, State:task.State}}
    if training {
        begin.Intervention = true
        begin.MatchKey = contrastID
    }
    apply(rt, r.NativeControlCommand{ID:id, Begin:begin})
    var out r.NativeControlResult
    operations, cost := 0, 0.0
    for step := 0; step < 7; step++ {
        action := &r.NativeStep{EpisodeID:id, Sample:rng.Float64()}
        if training {
            selected := first
            if step > 0 && step < reasoningStages-1-task.Stage { selected = 0 }
            if step >= reasoningStages-1-task.Stage && step > 0 { selected = 3 }
            action.ExperimentAction = &selected
        } else {
            selected := -1
            switch mode {
            case "stop": selected = 3
            case "static1": if step < 1 { selected = 0 } else { selected = 3 }
            case "static2": if step < 2 { selected = 0 } else { selected = 3 }
            case "static3": if step < 3 { selected = 0 } else { selected = 3 }
            case "random": selected = rng.Intn(4)
            }
            if selected >= 0 { action.ActionOverride = &selected }
        }
        out = apply(rt, r.NativeControlCommand{
            ID:fmt.Sprintf("%s-s%d",id,step), Step:action,
        })
        if out.Readout.Action != 3 {
            operations++
            cost += .025
        }
        if out.Stopped { break }
    }
    reward := reasoningFeedback(out.State,task,layout)
    outcome := &r.NativeOutcome{EpisodeID:id, TraceDigest:out.TraceDigest, Source:"reasoning-numeric-observation",
        OutcomeID:id, Provenance:"observed-environment", Reward:reward, Learn:training}
    if training {
        outcome.ContrastID = contrastID
        outcome.ContrastMode = "action"
    }
    apply(rt, r.NativeControlCommand{ID:id+"-out", Outcome:outcome})
    accuracy := 0.0
    if task.Truth == 0 {
        if reward >= -.1 { accuracy = 1 }
    } else if reward > .12 {
        accuracy = 1
    }
    return Metrics{Cases:1, Return:reward-cost, Reward:reward, Accuracy:accuracy,
        Operations:float64(operations), Stops:reasoningBoolFloat(out.Stopped && !out.Truncated),
        Truncations:reasoningBoolFloat(out.Truncated)}
}

func reasoningBoolFloat(b bool) float64 {
    if b { return 1 }
    return 0
}

func reasoningMeasure(rt *r.BrainRuntime, layout reasoningScaffold, tasks []reasoningTask, seed int64, mode string) Metrics {
    m := Metrics{Cases:len(tasks)}
    rng := rand.New(rand.NewSource(seed))
    for _, task := range tasks {
        e := reasoningEpisode(rt,layout,task,rng,mode,0,"")
        m.Return += e.Return
        m.Reward += e.Reward
        m.Accuracy += e.Accuracy
        m.Operations += e.Operations
        m.Stops += e.Stops
        m.Truncations += e.Truncations
    }
    if len(tasks)>0 {
        n := float64(len(tasks))
        m.Return /= n; m.Reward /= n; m.Accuracy /= n
        m.Operations /= n; m.Stops /= n; m.Truncations /= n
    }
    return m
}

func reasoningSet(layout reasoningScaffold, seed int64, cases int, condition string) []reasoningTask {
    rng := rand.New(rand.NewSource(seed))
    out := make([]reasoningTask,cases)
    for i:=range out { out[i]=reasoningSample(layout,rng,condition) }
    return out
}

// RunReasoningLab does not touch main, production checkpoints or existing
// brain files. Its output directory must not exist before invocation.
func RunReasoningLab(dir string, seed int64, blocks, cases int) (ReasoningLabReport,error) {
    report:=ReasoningLabReport{Seed:seed,Blocks:blocks,TrainingEpisodes:2*blocks,
        Baselines:map[string]Metrics{},
        Interpretation:"Narrow learned neural operation control only; abstract reasoning remains unproven."}
    if blocks<1 || cases<1 || dir=="" {return report, errors.New("fresh output directory and positive budgets required")}
    if err:=os.Mkdir(dir,0700);err!=nil{return report,err}
    // newReasoningLabBrain asserts that its sole Brain starts at zero units.
    report.InitialNodes=0
    report.InitialSynapses=0
    actual,layout,observed,err:=newReasoningLabBrain()
    if err!=nil{return report,err}
    report.AcquiredTransitions=observed
    rt:=r.NewBrainRuntime(actual)
    if rt==nil {return report,errors.New("native runtime refused blank scaffold")}
    cfg:=Config(seed,1)
    cfg.Version=k.NativeControlCausalVersion
    cfg.Sources=layout.Sources
    cfg.MaxSteps=7
    cfg.Operations=[]k.NativeOperation{
        {Kind:"propagate",Gain:1,Cost:.025},
        {Kind:"focus",Gain:.5,Cost:.025},
        {Kind:"relax",Gain:.5,Cost:.025},
        {Kind:"stop"},
    }
    apply(rt,r.NativeControlCommand{ID:"reasoning-init",Initialize:&cfg})
    before:=parameters(actual)
    tasks:=reasoningSet(layout,seed+920001,cases,"mixed")
    report.Before=reasoningMeasure(rt,layout,tasks,seed+1200,"learned")
    for _,arm:=range []string{"stop","static1","static2","static3","random"} {
        report.Baselines[arm]=reasoningMeasure(rt,layout,tasks,seed+1200,arm)
    }

    train:=rand.New(rand.NewSource(seed+550033))
    for i:=0;i<blocks;i++ {
        task:=reasoningSample(layout,train,"mixed")
        pair:=fmt.Sprintf("reasoning-pair-%d",i)
        second:=3
        if i%4==1 {second=1}
        if i%4==2 {second=2}
        if i%4==3 {second=3}
        // Each arm executes its actual actions and receives feedback only
        // after its trajectory. The second factual outcome updates N.
        reasoningEpisode(rt,layout,task,train,"",0,pair)
        reasoningEpisode(rt,layout,task,train,"",second,pair)
    }
    after:=parameters(actual)
    for i,w:=range after {if w!=before[i] {report.ChangedPolicyParameters++}}
    report.After=reasoningMeasure(rt,layout,tasks,seed+1200,"learned")
    report.ReversedEvidence=reasoningMeasure(rt,layout,reasoningSet(layout,seed+980001,cases,"reversal"),seed+2300,"learned")
    report.ConflictingEvidence=reasoningMeasure(rt,layout,reasoningSet(layout,seed+990001,cases,"conflict"),seed+3400,"learned")
    if err:=rt.Checkpoint(filepath.Join(dir,"brain-checkpoint.json"));err!=nil {return report,err}
    data,err:=json.MarshalIndent(report,"","  ")
    if err!=nil{return report,err}
    if err=os.WriteFile(filepath.Join(dir,"metrics.json"),data,0600);err!=nil{return report,err}
    return report,nil
}
