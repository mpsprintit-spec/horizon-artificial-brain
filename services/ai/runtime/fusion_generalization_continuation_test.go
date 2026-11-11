package runtime

// Isolated continuation of the 120-pattern Horizon Brain, not a fresh Brain.
// Fusion is an experimental NUMERIC sensory representation of co-observed
// evidence. The teacher does not insert lexical knowledge or reasoning rules.
import (
    "crypto/sha256"
    "encoding/hex"
    "encoding/json"
    "fmt"
    "math"
    "math/rand"
    "os"
    "path/filepath"
    "slices"
    "testing"
    "time"

    "github.com/project-horizon/horizon-core/services/ai/knowledge"
    "github.com/project-horizon/horizon-core/services/ai/learning"
)

type fusionLabReport struct {
    Protocol string `json:"protocol"`
    ParentRun int64 `json:"parent_run"`
    ParentSHA256 string `json:"parent_checkpoint_sha256"`
    ParentPatterns int `json:"parent_patterns"`
    ContinuedPatterns int `json:"continued_patterns"`
    AdditionalLessons int `json:"additional_lessons"`
    PreviousMargins [2]float64 `json:"previous_margins"`
    ContinuedMargins [2]float64 `json:"continued_margins"`
    Before []genCase `json:"before"`
    After []genCase `json:"after"`
    ReversedTeacher []genCase `json:"reversed_teacher"`
    Ablated []genCase `json:"ablated"`
    TransferBefore int `json:"transfer_before"`
    TransferAfter int `json:"transfer_after"`
    TransferTotal int `json:"transfer_total"`
    AllSupportedAfter int `json:"all_supported_after"`
    AllTotal int `json:"all_total"`
    Limits string `json:"limits"`
}

func fusionCheck(t *testing.T, snapshot string, base, ids, disjoint [][]knowledge.NodeID, fused, deferPop []knowledge.NodeID, ctx int, expected, actor int, kind string, at time.Time, ablate bool) genCase {
    t.Helper()
    r:=orderedFixtureClone(t,snapshot)
    if ablate{r.brain.Patterns=knowledge.NewPatternIndex()}
    cue:=[][]knowledge.NodeID{genFrame(base[0],ids[actor]),base[1],base[2]}
    if kind=="disjoint"{cue=[][]knowledge.NodeID{disjoint[0],disjoint[1],disjoint[2]}}
    if kind=="distractor" {cue[0]=genFrame(base[0],ids[actor],disjoint[0])}
    out,err:=r.CognitiveProcess(Event{
        ID:fmt.Sprintf("fusion-%s-%d-%d-%t",kind,ctx,actor,ablate),
        EpisodeID:"strict-fused-heldout",
        Timestamp:at,
        NeuralTrace:&knowledge.ObservationTrace{
            Stimulus:cue,
            Context:[][]knowledge.NodeID{base[ctx]},
            Data:[][]knowledge.NodeID{fused},
        },
        Cycles:1,
    })
    if err!=nil{t.Fatal(err)}
    ps:=[3][]knowledge.NodeID{base[6],base[7],deferPop}
    read:=r.activation.PredictionSnapshot().State
    ans:=genCase{Kind:kind,Context:ctx,Evidence:[]int{6,7},Expected:expected,Selected:-1,Unknown:1}
    best,second:=math.Inf(-1),math.Inf(-1)
    for i,pop:=range ps{
        v:=0.0
        for _,id:=range pop {v+=read[id]}
        v/=float64(len(pop))
        ans.Strength[i]=v
        if v>best {
            second=best
            best=v
            ans.Selected=i
        } else if v>second{second=v}
    }
    ans.Margin=best-second
    if out.PredictionAssessment!=nil{
        ans.Unknown=out.PredictionAssessment.UnknownProbability
        ans.Confidence=out.PredictionAssessment.Confidence
    }
    ans.CorrectSupported=ans.Selected==ans.Expected && ans.Margin>.05 && ans.Unknown<.8
    return ans
}

func TestFusionContinuationGeneralizesNovelConflicts(t *testing.T) {
    parent:=os.Getenv("HORIZON_FUSION_PRIOR_CHECKPOINT")
    metrics:=os.Getenv("HORIZON_FUSION_PRIOR_METRICS")
    if parent==""||metrics==""{t.Fatal("trained 120-pattern predecessor is REQUIRED")}
    b,err:=os.ReadFile(parent);if err!=nil{t.Fatal(err)}
    digest:=sha256.Sum256(b)
    mb,err:=os.ReadFile(metrics);if err!=nil{t.Fatal(err)}
    var prior genReport
    if err=json.Unmarshal(mb,&prior);err!=nil{t.Fatal(err)}
    if prior.Protocol!="structural-generalization-from-trained-brain-v1"||prior.PatternCountAfter!=120||prior.AdditionalLessons!=192{
        t.Fatalf("wrong predecessor: %+v",prior)
    }
    student:=orderedFixtureClone(t,parent)
    if len(student.brain.Patterns.All())!=prior.PatternCountAfter ||len(student.brain.SurfaceAnnotationsSnapshot())!=0{
        t.Fatal("checkpoint did not restore previous trained Brain")
    }
    at:=time.Date(2026,11,11,12,0,0,0,time.UTC)
    probeAt:=at.Add(24*time.Second)
    perm:=rand.New(rand.NewSource(17)).Perm(8)
    base:=make([][]knowledge.NodeID,8)
    for i:=range base {
        v:=make([]float64,8);v[perm[i]]=1
        g,ge:=student.GroundVectorAt(knowledge.NewNeuralVector(v),at.Add(13*time.Second))
        if ge!=nil{t.Fatal(ge)}
        if !g.Existing{t.Fatalf("missing learned base population %d",i)}
        base[i]=slices.Clone(g.Population)
    }
    stopVector:=make([]float64,12);stopVector[0]=1
    g,err:=student.GroundVectorAt(knowledge.NewNeuralVector(stopVector),at.Add(13*time.Second))
    if err!=nil{t.Fatal(err)}
    if !g.Existing{t.Fatal("parent's learned uncertainty/deferral population is absent")}
    deferPop:=slices.Clone(g.Population)

    // Identities 0-2 already occurred in stage-2 supervision; identity 3
    // was NEVER a stage-2 training actor, and is held out again here.
    ids:=make([][]knowledge.NodeID,4)
    for i:=range ids{
        v:=make([]float64,16);v[i+1]=1
        x,ge:=student.GroundVectorAt(knowledge.NewNeuralVector(v),at.Add(13*time.Second))
        if ge!=nil{t.Fatal(ge)}
        if !x.Existing{t.Fatalf("identity %d missing from parent",i)}
        ids[i]=slices.Clone(x.Population)
    }
    disjoint:=make([][]knowledge.NodeID,3)
    for i:=range disjoint{
        v:=make([]float64,16);v[i+10]=1
        x,ge:=student.GroundVectorAt(knowledge.NewNeuralVector(v),at.Add(13*time.Second))
        if ge!=nil{t.Fatal(ge)}
        if !x.Existing{t.Fatalf("disjoint probe population %d missing",i)}
        disjoint[i]=slices.Clone(x.Population)
    }

    // The fused 8-D sensory field retains two measured channels and a shared
    // multi-source flag. It does NOT contain the word "conflict" or the target
    // action. The third independent numerical coordinate prevents accidental
    // aliasing with the previously trained one-hot single-evidence units.
    fusedVector:=make([]float64,8)
    fusedVector[perm[6]]=.6
    fusedVector[perm[7]]=.6
    fusedVector[perm[5]]=-1
    fx,err:=student.GroundVectorAt(knowledge.NewNeuralVector(fusedVector),at.Add(14*time.Second))
    if err!=nil{t.Fatal(err)}
    fused:=slices.Clone(fx.Population)
    if relationLabPopulationEquals(fused,base[6])||relationLabPopulationEquals(fused,base[7]) {
        t.Fatal("sensor-fusion pattern accidentally aliased to single evidence")
    }
    beforeSnapshot:=orderedSave(t,student)
    retention:=func(snapshot string)[2]float64{
        var m [2]float64
        for side,ctx:=range []int{3,4}{
            probe:=orderedFixtureClone(t,snapshot)
            _,pe:=probe.CognitiveProcess(Event{
                ID:fmt.Sprintf("retained-%d",ctx),EpisodeID:"retention",
                Timestamp:probeAt,NeuralTrace:&knowledge.ObservationTrace{
                    Stimulus:[][]knowledge.NodeID{base[0],base[1],base[2]},
                    Context:[][]knowledge.NodeID{base[ctx]},
                },Cycles:1,
            })
            if pe!=nil{t.Fatal(pe)}
            read:=probe.activation.PredictionSnapshot().State
            avg:=func(pop []knowledge.NodeID)float64{
                v:=0.0;for _,id:=range pop{v+=read[id]};return v/float64(len(pop))
            }
            m[side]=avg(base[6])-avg(base[7])
        }
        return m
    }
    old:=retention(beforeSnapshot)

    // Cases 0-1: withheld actor with familiar relational scaffold and a
    // genuinely NEW fused input. Cases 2-4 are strict out-of-distribution
    // controls, including an untrained context and a disjoint triple.
    defs:=[]struct{ctx,actor,expected int;kind string}{
        {3,3,2,"novel-fused"},
        {4,3,2,"novel-fused"},
        {3,3,2,"distractor"},
        {5,3,2,"novel-context"},
        {3,3,2,"disjoint"},
        {4,3,2,"disjoint"},
    }
    measure:=func(snapshot string,ablate bool)[]genCase{
        out:=[]genCase{}
        for _,d:=range defs {
            out=append(out,fusionCheck(t,snapshot,base,ids,disjoint,fused,deferPop,d.ctx,d.expected,d.actor,d.kind,probeAt,ablate))
        }
        return out
    }
    before:=measure(beforeSnapshot,false)
    train:=func(reverse bool)(string,int,int){
        r:=orderedFixtureClone(t,beforeSnapshot)
        randOrder:=rand.New(rand.NewSource(1701))
        count:=0
        for epoch:=0;epoch<10;epoch++{
            tasks:=[]struct{ctx,actor int;generic bool}{}
            for _,ctx:=range []int{3,4}{
                for actor:=0;actor<3;actor++{
                    tasks=append(tasks,struct{ctx,actor int;generic bool}{ctx,actor,false})
                }
                tasks=append(tasks,struct{ctx,actor int;generic bool}{ctx,0,true})
            }
            for _,j:=range randOrder.Perm(len(tasks)){
                task:=tasks[j]
                first:=base[0]
                if !task.generic{first=genFrame(base[0],ids[task.actor])}
                answer:=deferPop
                if reverse{answer=base[6]}
                lesson:=learning.Experience{
                    NeuralSequence:[][]knowledge.NodeID{first,base[1],base[2],answer},
                    NeuralContext:[][]knowledge.NodeID{base[task.ctx]},
                    NeuralData:[][]knowledge.NodeID{fused},
                    Weight:.9,Confidence:.9,Reliability:.85,
                    Source:fmt.Sprintf("fused-source-%d",j),
                    IndependenceGroup:fmt.Sprintf("fused-origin-%d",j),
                    EpisodeID:fmt.Sprintf("fused-%t-%d-%d",reverse,epoch,j),
                    ExperienceID:fmt.Sprintf("fused-continuation-%t-%d-%d",reverse,epoch,j),
                    Modality:"internal-numeric",
                    Timestamp:at.Add(18*time.Second),
                }
                if _,e:=r.LearnExperience(lesson,at.Add(18*time.Second));e!=nil{t.Fatal(e)}
                count++
            }
        }
        if len(r.brain.Patterns.All())<=prior.PatternCountAfter ||len(r.brain.SurfaceAnnotationsSnapshot())!=0{
            t.Fatal("fusion did not learn on the existing neural substrate")
        }
        return orderedSave(t,r),count,len(r.brain.Patterns.All())
    }
    trained,n,patterns:=train(false)
    reversed,_,_:=train(true)
    after:=measure(trained,false)
    control:=measure(reversed,false)
    ablated:=measure(trained,true)
    retained:=retention(trained)

    successes:=func(xs []genCase,from,to int)int{
        n:=0;for _,x:=range xs[from:to] {if x.CorrectSupported{n++}};return n
    }
    baseline:=successes(before,0,2)
    improved:=successes(after,0,2)
    all:=successes(after,0,len(after))
    t.Logf("RESTORED parent patterns=%d checkpoint_sha256=%s",prior.PatternCountAfter,hex.EncodeToString(digest[:]))
    t.Logf("FUSION new_lessons=%d patterns=%d retained=%v->%v",n,patterns,old,retained)
    t.Logf("TRANSFER novel_identity before=%d/2 after=%d/2 all=%d/%d",baseline,improved,all,len(after))
    for i,d:=range defs{
        t.Logf("FUSED_HELD[%d] %s ctx=%d before=%d/%.4f/%.4f after=%d/%.4f/%.4f afterScores=%v reverse=%d/%.4f/%.4f ablate=%d/%.4f/%.4f",
            i,d.kind,d.ctx,
            before[i].Selected,before[i].Margin,before[i].Unknown,
            after[i].Selected,after[i].Margin,after[i].Unknown,after[i].Strength,
            control[i].Selected,control[i].Margin,control[i].Unknown,
            ablated[i].Selected,ablated[i].Margin,ablated[i].Unknown)
    }
    report:=fusionLabReport{
        Protocol:"fused-evidence-neural-generalization-v1",
        ParentRun:43034669302,
        ParentSHA256:hex.EncodeToString(digest[:]),
        ParentPatterns:prior.PatternCountAfter,
        ContinuedPatterns:patterns,
        AdditionalLessons:n,PreviousMargins:old,ContinuedMargins:retained,
        Before:before,After:after,ReversedTeacher:control,Ablated:ablated,
        TransferBefore:baseline,TransferAfter:improved,TransferTotal:2,
        AllSupportedAfter:all,AllTotal:len(after),
        Limits:"Joint evidence is externally encoded into a new numeric vector, without an answer bit. Tests preserve a shared relational stem: transfer across completely disjoint triples or untrained contexts is not established by this representation. Any success is conditional, not autonomous discovery of universal reasoning.",
    }
    if dest:=os.Getenv("HORIZON_FUSION_OUTPUT_DIR");dest!=""{
        if err:=os.MkdirAll(dest,0700);err!=nil{t.Fatal(err)}
        restored:=orderedFixtureClone(t,trained)
        if err:=restored.Checkpoint(filepath.Join(dest,"horizon-brain-fused-generalization.json"));err!=nil{t.Fatal(err)}
        data,err:=json.MarshalIndent(report,"","  ");if err!=nil{t.Fatal(err)}
        if err:=os.WriteFile(filepath.Join(dest,"fused-generalization-metrics.json"),append(data,byte(10)),0600);err!=nil{t.Fatal(err)}
        t.Log("EXPORT trained Brain checkpoint + strict heldout metrics")
    }
    if retained[0]<.07||retained[1]>-.07{t.Fatalf("prior relational knowledge regressed: %v -> %v",old,retained)}
    if improved<baseline{t.Fatalf("novel-identity transfer regressed: %d -> %d",baseline,improved)}
}
