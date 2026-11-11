package nativefoundation

import (
    "encoding/json"
    "math"
    "os"
    "path/filepath"
    "testing"

    k "github.com/project-horizon/horizon-core/services/ai/knowledge"
)

func TestReasoningLabUsesOneInitiallyEmptyBrainAndObservedFeedback(t *testing.T) {
    dir:=filepath.Join(t.TempDir(),"new-reasoning-lab")
    got,err:=RunReasoningLab(dir,73,32,24)
    if err!=nil { t.Fatal(err) }
    if got.InitialNodes!=0 || got.InitialSynapses!=0 {
        t.Fatalf("not initially blank: %d nodes %d synapses",got.InitialNodes,got.InitialSynapses)
    }
    if got.AcquiredTransitions!=18 || got.TrainingEpisodes!=64 {
        t.Fatalf("unexpected genuine experience/trial counts: %+v",got)
    }
    if got.ChangedPolicyParameters==0 {
        t.Fatal("factual outcomes did not change any canonical neural policy weights")
    }
    for _,path:=range []string{"brain-checkpoint.json","metrics.json"} {
        if _,err:=os.Stat(filepath.Join(dir,path));err!=nil { t.Fatalf("missing isolated artifact %s: %v",path,err) }
    }
    data,err:=os.ReadFile(filepath.Join(dir,"metrics.json"))
    if err!=nil {t.Fatal(err)}
    var decoded ReasoningLabReport
    if err=json.Unmarshal(data,&decoded);err!=nil {t.Fatal(err)}
    if math.IsNaN(got.After.Return)||math.IsInf(got.After.Return,0) {t.Fatal("nonfinite post-training return")}
    t.Logf("blank-brain lab: before=%+.5f after=%+.5f static3=%+.5f random=%+.5f changed=%d conflict=%+.5f reverse=%+.5f",
        got.Before.Return,got.After.Return,got.Baselines["static3"].Return,
        got.Baselines["random"].Return,got.ChangedPolicyParameters,
        got.ConflictingEvidence.Return,got.ReversedEvidence.Return)
    // This test verifies that training is factual, internal and reproducible
    // enough to report. It intentionally does not declare reasoning success:
    // promotion requires preregistered, independently repeated holdouts.
}

func TestReasoningFeedbackPenalizesConflictingEvidence(t *testing.T) {
    var layout reasoningScaffold
    layout.Chains[0][3][0]=k.NodeID(1)
    layout.Chains[1][3][0]=k.NodeID(2)
    state:=map[k.NodeID]float64{1:.8,2:.8}
    task:=reasoningTask{Truth:0}
    if v:=reasoningFeedback(state,task,layout); !(v<0) {
        t.Fatalf("co-activated contradictory outputs should not earn reward, got %.3f",v)
    }
}
