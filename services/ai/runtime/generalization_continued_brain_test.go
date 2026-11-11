package runtime

// Generalization study on the actual PREVIOUSLY TRAINED neural checkpoint.
// Input dimensions and teacher outputs are experimental numerical conventions;
// no philosophical words, answer rules, alternate model or production changes.
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

type genPrior struct {
    Protocol string `json:"protocol"`
    ContinuedPatternCount int `json:"continued_pattern_count"`
    ParentSeed int64 `json:"parent_seed"`
}
type genCase struct {
    Kind string `json:"kind"`
    Context int `json:"context"`
    Evidence []int `json:"evidence"`
    Expected int `json:"expected"`
    Selected int `json:"selected"`
    Margin float64 `json:"margin"`
    Unknown float64 `json:"unknown"`
    Confidence float64 `json:"confidence"`
    Strength [3]float64 `json:"strength"`
    CorrectSupported bool `json:"correct_supported"`
}
type genReport struct{
    Protocol string `json:"protocol"`
    PriorRun int64 `json:"prior_run"`
    PriorCheckpointSHA256 string `json:"prior_checkpoint_sha256"`
    PatternCountBefore int `json:"pattern_count_before"`
    PatternCountAfter int `json:"pattern_count_after"`
    AdditionalLessons int `json:"additional_lessons"`
    InitialRelationalMargins [2]float64 `json:"initial_relational_margins"`
    RetainedRelationalMargins [2]float64 `json:"retained_relational_margins"`
    Baseline []genCase `json:"baseline"`
    Trained []genCase `json:"trained"`
    Ablated []genCase `json:"ablated"`
    Shuffled []genCase `json:"reversed_teacher_control"`
    GeneralizationBaseline int `json:"generalization_baseline"`
    GeneralizationAfter int `json:"generalization_after"`
    GeneralizationTotal int `json:"generalization_total"`
    StrictHeldoutCorrect int `json:"strict_heldout_correct"`
    StrictHeldoutTotal int `json:"strict_heldout_total"`
    Limitations string `json:"limitations"`
}
func genFrame(groups ...[]knowledge.NodeID) []knowledge.NodeID {
    var values []knowledge.NodeID
    for _,g:=range groups{values=append(values,g...)}
    slices.Sort(values)
    return slices.Compact(values)
}
func genEvidenceFrame(base [][]knowledge.NodeID, evidence []int) []knowledge.NodeID {
    var parts [][]knowledge.NodeID
    for _,i:=range evidence {parts=append(parts,base[i])}
    return genFrame(parts...)
}
func genProbe(t *testing.T, snapshot string, base, identity, disjoint [][]knowledge.NodeID, deferPop []knowledge.NodeID, ctx int, ev []int, expected, actor int, kind string, at time.Time, ablate bool) genCase {
    t.Helper()
    r:=orderedFixtureClone(t,snapshot)
    if ablate {r.brain.Patterns=knowledge.NewPatternIndex()}
    cue:=[][]knowledge.NodeID{genFrame(base[0],identity[actor]),base[1],base[2]}
    if kind=="disjoint" {
        cue=[][]knowledge.NodeID{disjoint[0],disjoint[1],disjoint[2]}
    }
    trace:=&knowledge.ObservationTrace{
        Stimulus:cue,
        Context:[][]knowledge.NodeID{base[ctx]},
        Data:[][]knowledge.NodeID{genEvidenceFrame(base,ev)},
    }
    out,err:=r.CognitiveProcess(Event{
        ID:fmt.Sprintf("gen-%s-%d-%v-%d-%t",kind,ctx,ev,actor,ablate),
        EpisodeID:"strict-heldout",
        Timestamp:at,
        NeuralTrace:trace,
        Cycles:1,
    })
    if err!=nil {t.Fatal(err)}
    pops:=[3][]knowledge.NodeID{base[6],base[7],deferPop}
    p:=r.activation.PredictionSnapshot().State
    ans:=genCase{Kind:kind,Context:ctx,Evidence:slices.Clone(ev),Expected:expected,Selected:-1,Unknown:1}
    high,runner:=math.Inf(-1),math.Inf(-1)
    for i,pop:=range pops{
        sum:=0.0
        for _,id:=range pop{sum+=p[id]}
        v:=sum/float64(len(pop))
        ans.Strength[i]=v
        if v>high {runner=high;high=v;ans.Selected=i
        } else if v>runner {runner=v}
    }
    ans.Margin=high-runner
    if out.PredictionAssessment!=nil {
        ans.Unknown=out.PredictionAssessment.UnknownProbability
        ans.Confidence=out.PredictionAssessment.Confidence
    }
    ans.CorrectSupported=ans.Selected==ans.Expected && ans.Margin>.05 && ans.Unknown<.8
    return ans
}

func TestThinkingGeneralizesFromContinuedBrain(t *testing.T) {
    prevCheckpoint:=os.Getenv("HORIZON_GENERALIZATION_PRIOR_CHECKPOINT")
    prevMetrics:=os.Getenv("HORIZON_GENERALIZATION_PRIOR_METRICS")
    if prevCheckpoint==""||prevMetrics==""{t.Fatal("the actual prior trained Brain and its metrics are mandatory")}
    b,err:=os.ReadFile(prevCheckpoint);if err!=nil{t.Fatal(err)}
    hash:=sha256.Sum256(b)
    m,err:=os.ReadFile(prevMetrics);if err!=nil{t.Fatal(err)}
    var prior genPrior
    if err=json.Unmarshal(m,&prior);err!=nil{t.Fatal(err)}
    if prior.Protocol!="philosophical-thinking-numeric-continuation-v1"||prior.ParentSeed!=17||prior.ContinuedPatternCount!=40 {
        t.Fatalf("wrong predecessor checkpoint: %+v",prior)
    }
    // Fresh runtime HANDLE only: no experiment runs until the previous
    // checkpoint has been restored and authenticated by the runtime.
    r:=orderedFixtureClone(t,prevCheckpoint)
    if len(r.brain.Patterns.All())!=40 || len(r.brain.SurfaceAnnotationsSnapshot())!=0 {
        t.Fatal("previous trained state was not restored as-is")
    }
    at:=time.Date(2026,11,11,12,0,0,0,time.UTC)
    probeAt:=at.Add(12*time.Second)
    // The preceding test stored 8 opaque populations, with 12-D zero-index
    // evidence-deferral representation in its continuation.
    // Reground existing numerical coordinates; don't invent a new outcome.
    base:=make([][]knowledge.NodeID,8)
    for i:=0;i<8;i++{
        // Previous experiment grounded permuted 8-D axes with seed 17.
        perm:=rand.New(rand.NewSource(17)).Perm(8)
        values:=make([]float64,8);values[perm[i]]=1
        g,ge:=r.GroundVectorAt(knowledge.NewNeuralVector(values),at.Add(6*time.Second))
        if ge!=nil {t.Fatal(ge)}
        if !g.Existing{t.Fatalf("previous learned numeric population %d absent",i)}
        base[i]=slices.Clone(g.Population)
    }
    deferVector:=make([]float64,12);deferVector[0]=1
    dg,err:=r.GroundVectorAt(knowledge.NewNeuralVector(deferVector),at.Add(6*time.Second))
    if err!=nil{t.Fatal(err)}
    if !dg.Existing{t.Fatal("previous learned deferral population not found in checkpoint")}
    deferPop:=slices.Clone(dg.Population)

    // New identity-specific numerical features; all sharing a common
    // previously learned relational frame in first stimulus population.
    // Identity 3 is held out from the new supervision entirely.
    identity:=make([][]knowledge.NodeID,4)
    for i:=range identity{
        v:=make([]float64,16);v[i+1]=1
        g,ge:=r.GroundVectorAt(knowledge.NewNeuralVector(v),at.Add(6*time.Second))
        if ge!=nil{t.Fatal(ge)}
        identity[i]=slices.Clone(g.Population)
    }
    // Fully disjoint triple: no shared relational stem; reports a real limit.
    disjoint:=make([][]knowledge.NodeID,3)
    for i:=range disjoint{
        v:=make([]float64,16);v[i+10]=1
        g,ge:=r.GroundVectorAt(knowledge.NewNeuralVector(v),at.Add(6*time.Second))
        if ge!=nil{t.Fatal(ge)}
        disjoint[i]=slices.Clone(g.Population)
    }
    if len(r.brain.SurfaceAnnotationsSnapshot())!=0{t.Fatal("numeric input created lexical labels")}
    preTraining:=orderedSave(t,r)
    priorRel:=func(snapshot string) [2]float64{
        results:=[2]float64{}
        for side,ctx:=range []int{3,4}{
            p:=orderedFixtureClone(t,snapshot)
            trace:=&knowledge.ObservationTrace{Stimulus:[][]knowledge.NodeID{base[0],base[1],base[2]},Context:[][]knowledge.NodeID{base[ctx]}}
            if _,pe:=p.CognitiveProcess(Event{ID:fmt.Sprintf("retention-%d",ctx),EpisodeID:"retention",Timestamp:probeAt,NeuralTrace:trace,Cycles:1});pe!=nil{t.Fatal(pe)}
            score:=p.activation.PredictionSnapshot().State
            avg:=func(ids []knowledge.NodeID)float64{n:=0.0;for _,id:=range ids{n+=score[id]};return n/float64(len(ids))}
            results[side]=avg(base[6])-avg(base[7])
        }
        return results
    }
    oldRel:=priorRel(preTraining)

    type heldConfig struct{ctx int;ev []int;expected int;kind string}
    held:=[]heldConfig{}
    // No new-subject 3 training. Reversals of [6,7] were not taught
    // as temporal sequences; this query models them as one simultaneous set.
    for _,ctx:=range []int{3,4}{
        for _,item:=range []struct{ev []int;answer int}{{[]int{6},0},{[]int{7},1},{[]int{7,6},2}}{
            held=append(held,heldConfig{ctx,item.ev,item.answer,"novel-identity"})
        }
    }
    // Untrained context and disjoint-structure cases are reported as controls.
    held=append(held,
        heldConfig{5,[]int{6},0,"novel-context"},
        heldConfig{5,[]int{7},1,"novel-context"},
        heldConfig{5,[]int{7,6},2,"novel-context"},
        heldConfig{3,[]int{6},0,"disjoint"},
        heldConfig{4,[]int{7},1,"disjoint"},
        heldConfig{5,[]int{7,6},2,"disjoint"},
    )
    measure:=func(snapshot string, ablate bool)[]genCase{
        var got []genCase
        for _,c:=range held {
            got=append(got,genProbe(t,snapshot,base,identity,disjoint,deferPop,c.ctx,c.ev,c.expected,3,c.kind,probeAt,ablate))
        }
        return got
    }
    before:=measure(preTraining,false)

    train:=func(reverse bool)(string,int,int){
        student:=orderedFixtureClone(t,preTraining)
        randOrder:=rand.New(rand.NewSource(157))
        lessons:=0
        for epoch:=0;epoch<8;epoch++{
            tasks:=[]struct{ctx int;ev []int;answer int;actor int;generic bool}{}
            for _,ctx:=range []int{3,4}{
                for _,i:=range []int{0,1,2}{
                    ev:=[]int{6};if i==1{ev=[]int{7}};if i==2{ev=[]int{6,7}}
                    answer:=i
                    if reverse&&i<2{answer=1-i}
                    for actor:=0;actor<3;actor++ {
                        tasks=append(tasks,struct{ctx int;ev []int;answer int;actor int;generic bool}{ctx,ev,answer,actor,false})
                    }
                    tasks=append(tasks,struct{ctx int;ev []int;answer int;actor int;generic bool}{ctx,ev,answer,0,true})
                }
            }
            for _,j:=range randOrder.Perm(len(tasks)){
                task:=tasks[j]
                first:=base[0]
                if !task.generic{first=genFrame(base[0],identity[task.actor])}
                // The lesson outcome is supervision, never part of the query.
                target:=base[6]
                if task.answer==1{target=base[7]}
                if task.answer==2{target=deferPop}
                experience:=learning.Experience{
                    NeuralSequence:[][]knowledge.NodeID{first,base[1],base[2],target},
                    NeuralContext:[][]knowledge.NodeID{base[task.ctx]},
                    NeuralData:[][]knowledge.NodeID{genEvidenceFrame(base,task.ev)},
                    Weight:.9,Confidence:.9,Reliability:.85,
                    Source:fmt.Sprintf("structural-training-%d",j),
                    IndependenceGroup:fmt.Sprintf("fixed-lab-source-%d",j),
                    EpisodeID:fmt.Sprintf("structural-%d-%d",epoch,j),
                    ExperienceID:fmt.Sprintf("continuation-%t-%d-%d",reverse,epoch,j),
                    Modality:"internal-numeric",
                    Timestamp:at.Add(8*time.Second),
                }
                if _,le:=student.LearnExperience(experience,at.Add(8*time.Second));le!=nil{t.Fatal(le)}
                lessons++
            }
        }
        if len(student.brain.Patterns.All())<=40 || len(student.brain.SurfaceAnnotationsSnapshot())!=0{
            t.Fatal("continued Brain did not learn structural patterns")
        }
        return orderedSave(t,student),lessons,len(student.brain.Patterns.All())
    }
    trainedPath,n,patterns:=train(false)
    revPath,_,_:=train(true)
    after:=measure(trainedPath,false)
    ablated:=measure(trainedPath,true)
    flipped:=measure(revPath,false)
    newRel:=priorRel(trainedPath)

    successes:=func(cases []genCase,low,high int)int{
        n:=0
        for _,c:=range cases[low:high]{if c.CorrectSupported{n++}}
        return n
    }
    beforeNovel:=successes(before,0,6)
    afterNovel:=successes(after,0,6)
    heldCorrect:=successes(after,0,len(after))

    t.Logf("PARENT trained checkpoint sha256=%s; 40 existing neural patterns",hex.EncodeToString(hash[:]))
    t.Logf("TRAINING lessons=%d patterns=%d retained_relations=%v -> %v",n,patterns,oldRel,newRel)
    t.Logf("NOVEL SHARED-STRUCTURE baseline=%d/6 after=%d/6; ALL heldout=%d/%d",
        beforeNovel,afterNovel,heldCorrect,len(after))
    for i:=range held{
        t.Logf("HELD[%02d] type=%s context=%d evidence=%v expected=%d baseline=%d/%.3f/%.3f after=%d/%.3f/%.3f ablated=%d/%.3f/%.3f reversed=%d/%.3f/%.3f",
            i,held[i].kind,held[i].ctx,held[i].ev,held[i].expected,
            before[i].Selected,before[i].Margin,before[i].Unknown,
            after[i].Selected,after[i].Margin,after[i].Unknown,
            ablated[i].Selected,ablated[i].Margin,ablated[i].Unknown,
            flipped[i].Selected,flipped[i].Margin,flipped[i].Unknown)
        if held[i].kind=="novel-identity" && len(held[i].ev)==2 {
            t.Logf("CONFLICT_DETAIL[%d] baselineStrength=%v trainedStrength=%v ablatedStrength=%v reversedStrength=%v",
                i,before[i].Strength,after[i].Strength,ablated[i].Strength,flipped[i].Strength)
        }
    }
    report:=genReport{
        Protocol:"structural-generalization-from-trained-brain-v1",
        PriorRun:43032495188,
        PriorCheckpointSHA256:hex.EncodeToString(hash[:]),
        PatternCountBefore:40,PatternCountAfter:patterns,
        AdditionalLessons:n,InitialRelationalMargins:oldRel,RetainedRelationalMargins:newRel,
        Baseline:before,Trained:after,Ablated:ablated,Shuffled:flipped,
        GeneralizationBaseline:beforeNovel,GeneralizationAfter:afterNovel,
        GeneralizationTotal:6,StrictHeldoutCorrect:heldCorrect,StrictHeldoutTotal:len(after),
        Limitations:"Training uses externally supplied shared relational coordinates and a simultaneous evidence population. Heldout identity is new but shares the original relational stem. Fully disjoint relational triples and withheld contexts are separate controls; this is not spontaneous discovery of universal logic.",
    }
    if dir:=os.Getenv("HORIZON_GENERALIZATION_OUT_DIR");dir!=""{
        if err:=os.MkdirAll(dir,0700);err!=nil{t.Fatal(err)}
        copyRt:=orderedFixtureClone(t,trainedPath)
        if err:=copyRt.Checkpoint(filepath.Join(dir,"horizon-brain-generalized.json"));err!=nil{t.Fatal(err)}
        data,err:=json.MarshalIndent(report,"","  ");if err!=nil{t.Fatal(err)}
        if err:=os.WriteFile(filepath.Join(dir,"generalization-metrics.json"),append(data,byte(10)),0600);err!=nil{t.Fatal(err)}
        t.Logf("EXPORT continued trained Brain and heldout diagnostics")
    }
    if newRel[0]<.07||newRel[1]>-.07{t.Fatalf("old contextual relational knowledge degraded: %v -> %v",oldRel,newRel)}
    if afterNovel<beforeNovel {t.Fatalf("heldout transfer regressed: before=%d after=%d",beforeNovel,afterNovel)}
}
