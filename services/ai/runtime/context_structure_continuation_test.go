package runtime

// Research-only continued training on the REAL 152-pattern Horizon checkpoint.
// A held-out actor, unseen context/structure combinations, and entirely new
// inputs are tested independently. No production code, separate model,
// natural-language answer labels, or default branch modifications.
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

type contextStructureTrial struct {
    Group string `json:"group"`
    Structure string `json:"structure"`
    Context string `json:"context"`
    Input int `json:"input"`
    Actor int `json:"actor"`
    Expected int `json:"expected"`
    Selected int `json:"selected"`
    Scores [3]float64 `json:"scores"`
    Margin float64 `json:"margin"`
    Unknown float64 `json:"unknown"`
    Supported bool `json:"supported"`
}
type contextStructureReport struct{
    Protocol string `json:"protocol"`
    ParentRun int64 `json:"parent_run"`
    ParentCheckpointSHA256 string `json:"parent_checkpoint_sha256"`
    PatternsBefore int `json:"patterns_before"`
    PatternsAfter int `json:"patterns_after"`
    Lessons int `json:"lessons"`
    OldMargins [2]float64 `json:"old_margins"`
    RetainedMargins [2]float64 `json:"retained_margins"`
    Baseline []contextStructureTrial `json:"baseline"`
    Trained []contextStructureTrial `json:"trained"`
    Ablated []contextStructureTrial `json:"ablated"`
    ReversedTeacher []contextStructureTrial `json:"reversed_teacher"`
    GroupMetrics []struct{
        Group string `json:"group"`
        Baseline int `json:"baseline"`
        Trained int `json:"trained"`
        Total int `json:"total"`
    } `json:"group_metrics"`
    Limitations string `json:"limitations"`
}
type contextStructureCase struct {
    Group, Structure, Context string
    Kind, Actor, Expected int
}
type contextStructureLab struct{
    Base [][]knowledge.NodeID
    IDs [][]knowledge.NodeID
    Alternate [][]knowledge.NodeID
    Novel [][]knowledge.NodeID
    Context map[string][]knowledge.NodeID
    Fused []knowledge.NodeID
    Deferral []knowledge.NodeID
}
func (env contextStructureLab) stem(kind string,actor int,generic bool) [][]knowledge.NodeID{
    var s [][]knowledge.NodeID
    switch kind{
    case "original":
        s=[][]knowledge.NodeID{env.Base[0],env.Base[1],env.Base[2]}
    case "alternate":
        s=[][]knowledge.NodeID{env.Alternate[0],env.Alternate[1],env.Alternate[2]}
    case "reverse":
        s=[][]knowledge.NodeID{env.Alternate[2],env.Alternate[1],env.Alternate[0]}
    case "novel":
        s=[][]knowledge.NodeID{env.Novel[0],env.Novel[1],env.Novel[2]}
    default:
        panic("unknown structural family in lab")
    }
    if !generic{s[0]=genFrame(s[0],env.IDs[actor])}
    return s
}
func (env contextStructureLab) datum(kind int) []knowledge.NodeID{
    switch kind{
    case 0:return env.Base[6]
    case 1:return env.Base[7]
    case 2:return env.Fused
    default:panic("unknown numerical observation channel")
    }
}
func (env contextStructureLab) answer(kind int) []knowledge.NodeID{
    switch kind{
    case 0:return env.Base[6]
    case 1:return env.Base[7]
    case 2:return env.Deferral
    default:panic("unknown teacher outcome")
    }
}
func evalContextStructure(t *testing.T, file string, env contextStructureLab, spec contextStructureCase, at time.Time, ablate bool) contextStructureTrial{
    t.Helper()
    r:=orderedFixtureClone(t,file)
    if ablate {r.brain.Patterns=knowledge.NewPatternIndex()}
    out,err:=r.CognitiveProcess(Event{
        ID:fmt.Sprintf("probe-%s-%s-%s-%d-%d-%t",spec.Group,spec.Structure,spec.Context,spec.Kind,spec.Actor,ablate),
        EpisodeID:"unseen-structure-and-context",
        Timestamp:at,
        NeuralTrace:&knowledge.ObservationTrace{
            Stimulus:env.stem(spec.Structure,spec.Actor,false),
            Context:[][]knowledge.NodeID{env.Context[spec.Context]},
            Data:[][]knowledge.NodeID{env.datum(spec.Kind)},
        },
        Cycles:1,
    })
    if err!=nil{t.Fatal(err)}
    p:=r.activation.PredictionSnapshot().State
    result:=contextStructureTrial{
        Group:spec.Group,Structure:spec.Structure,Context:spec.Context,
        Input:spec.Kind,Actor:spec.Actor,Expected:spec.Expected,Selected:-1,Unknown:1,
    }
    tops:=[3][]knowledge.NodeID{env.Base[6],env.Base[7],env.Deferral}
    high,next:=math.Inf(-1),math.Inf(-1)
    for i,ids:=range tops{
        mean:=0.0
        for _,id:=range ids{mean+=p[id]}
        mean/=float64(len(ids))
        result.Scores[i]=mean
        if mean>high {
            next=high
            high=mean
            result.Selected=i
        }else if mean>next{next=mean}
    }
    result.Margin=high-next
    if out.PredictionAssessment!=nil{result.Unknown=out.PredictionAssessment.UnknownProbability}
    result.Supported=result.Selected==spec.Expected&&result.Margin>.05&&result.Unknown<.8
    return result
}
func TestLearnUnseenContextAndAlternateRelationsFromTrainedBrain(t *testing.T){
    parentPath:=os.Getenv("HORIZON_CONTEXT_STRUCTURE_PRIOR_CHECKPOINT")
    metaPath:=os.Getenv("HORIZON_CONTEXT_STRUCTURE_PRIOR_METRICS")
    if parentPath==""||metaPath==""{
        t.Fatal("previous 152-pattern trained Brain checkpoint and manifest are MANDATORY")
    }
    priorBytes,err:=os.ReadFile(parentPath)
    if err!=nil{t.Fatal(err)}
    digest:=sha256.Sum256(priorBytes)
    metrics,err:=os.ReadFile(metaPath)
    if err!=nil{t.Fatal(err)}
    var prior fusionLabReport
    if err=json.Unmarshal(metrics,&prior);err!=nil{t.Fatal(err)}
    if prior.Protocol!="fused-evidence-neural-generalization-v1"||prior.ContinuedPatterns!=152||prior.ParentPatterns!=120{
        t.Fatalf("wrong trained predecessor: %+v",prior)
    }
    brain:=orderedFixtureClone(t,parentPath)
    if len(brain.brain.Patterns.All())!=152||len(brain.brain.SurfaceAnnotationsSnapshot())!=0{
        t.Fatal("the 152 previously trained neural patterns were not restored")
    }
    at:=time.Date(2026,11,11,12,0,0,0,time.UTC)
    env:=contextStructureLab{Context:make(map[string][]knowledge.NodeID)}
    perm:=rand.New(rand.NewSource(17)).Perm(8)
    for i:=0;i<8;i++{
        v:=make([]float64,8);v[perm[i]]=1
        g,e:=brain.GroundVectorAt(knowledge.NewNeuralVector(v),at.Add(32*time.Second))
        if e!=nil{t.Fatal(e)}
        if !g.Existing{t.Fatalf("old numerical basis %d missing",i)}
        env.Base=append(env.Base,slices.Clone(g.Population))
    }
    get:=func(dim,index int,mustExist bool)[]knowledge.NodeID{
        v:=make([]float64,dim);v[index]=1
        grounded,e:=brain.GroundVectorAt(knowledge.NewNeuralVector(v),at.Add(32*time.Second))
        if e!=nil{t.Fatal(e)}
        if mustExist&&!grounded.Existing{t.Fatalf("previous trained numeric vector %d:%d lost",dim,index)}
        if len(grounded.Population)==0{t.Fatal("numeric grounding returned no neurons")}
        return slices.Clone(grounded.Population)
    }
    env.Deferral=get(12,0,true)
    for i:=0;i<4;i++{env.IDs=append(env.IDs,get(16,i+1,true))}
    for i:=0;i<3;i++{env.Alternate=append(env.Alternate,get(16,i+10,true))}
    for i:=0;i<3;i++{env.Novel=append(env.Novel,get(16,i+13,false))}
    env.Context["old-a"]=env.Base[3]
    env.Context["old-b"]=env.Base[4]
    env.Context["previously-untrained"]=env.Base[5]
    env.Context["new-trained"]=get(16,7,false)
    env.Context["never-trained"]=get(16,8,false)
    f:=make([]float64,8)
    f[perm[6]]=.6;f[perm[7]]=.6;f[perm[5]]=-1
    fused,err:=brain.GroundVectorAt(knowledge.NewNeuralVector(f),at.Add(33*time.Second))
    if err!=nil{t.Fatal(err)}
    if !fused.Existing{t.Fatal("previously learned fused-evidence population lost")}
    env.Fused=slices.Clone(fused.Population)
    if len(brain.brain.SurfaceAnnotationsSnapshot())!=0{t.Fatal("neural grounding created lexical labels")}

    initial:=orderedSave(t,brain)
    snapshotMargin:=func(file string)[2]float64{
        var results [2]float64
        for i,k:=range []string{"old-a","old-b"}{
            probe:=orderedFixtureClone(t,file)
            out,pe:=probe.CognitiveProcess(Event{
                ID:"retention-"+k,EpisodeID:"retention",
                Timestamp:at.Add(39*time.Second),
                NeuralTrace:&knowledge.ObservationTrace{
                    Stimulus:[][]knowledge.NodeID{env.Base[0],env.Base[1],env.Base[2]},
                    Context:[][]knowledge.NodeID{env.Context[k]},
                },Cycles:1,
            })
            if pe!=nil{t.Fatal(pe)}
            _=out
            p:=probe.activation.PredictionSnapshot().State
            mean:=func(pop []knowledge.NodeID)float64{
                s:=0.0;for _,id:=range pop{s+=p[id]};return s/float64(len(pop))
            }
            results[i]=mean(env.Base[6])-mean(env.Base[7])
        }
        return results
    }
    marginBefore:=snapshotMargin(initial)
    // Trainer learns previously-untrained C, a genuinely new context N,
    // and two different ordered relational stems. Only three actors out of
    // four receive feedback, while the generic stem provides an invariant.
    // Crucial compositional tests: reverse under C or N NEVER trained.
    contexts:=map[string][]string{
        "original":{"previously-untrained"},
        "alternate":{"old-a","old-b","previously-untrained","new-trained"},
        "reverse":{"old-a","old-b"},
    }
    type teaching struct{stem,context string;kind,actor int;generic bool}
    lessons:=[]teaching{}
    for stem,keys:=range contexts{
        for _,context:=range keys{
            for kind:=0;kind<3;kind++{
                lessons=append(lessons,teaching{stem,context,kind,0,true})
                for actor:=0;actor<3;actor++{
                    lessons=append(lessons,teaching{stem,context,kind,actor,false})
                }
            }
        }
    }
    // Stable sorting is not required for the content, but guarantees
    // deterministic training order independent of Go map iteration.
    slices.SortFunc(lessons,func(a,b teaching)int{
        x:=fmt.Sprintf("%s|%s|%d|%d|%t",a.stem,a.context,a.kind,a.actor,a.generic)
        y:=fmt.Sprintf("%s|%s|%d|%d|%t",b.stem,b.context,b.kind,b.actor,b.generic)
        if x<y{return -1};if x>y{return 1};return 0
    })

    // Both previously unseen input dimensions and true out-of-distribution
    // controls must be measured as separate categories.
    trials:=[]contextStructureCase{}
    add:=func(group,stem,context string,actor int,kinds ...int){
        for _,kind:=range kinds{
            trials=append(trials,contextStructureCase{group,stem,context,kind,actor,kind})
        }
    }
    add("seen-context-new-actor","original","previously-untrained",3,0,1,2)
    for _,ctx:=range []string{"old-a","old-b","previously-untrained","new-trained"}{
        add("alternate-new-actor","alternate",ctx,3,0,1,2)
    }
    for _,ctx:=range []string{"old-a","old-b"}{
        add("reverse-new-actor","reverse",ctx,3,0,1,2)
    }
    // All of these structure/context PAIRS are strictly withheld.
    for _,ctx:=range []string{"previously-untrained","new-trained"}{
        add("heldout-structure-context-combination","reverse",ctx,3,0,1,2)
    }
    add("fully-new-context","alternate","never-trained",3,0,1,2)
    add("fully-new-structure","novel","old-a",3,0,1,2)
    add("fully-new-structure-new-context","novel","never-trained",3,0,1,2)

    evaluate:=func(file string,ablate bool)[]contextStructureTrial{
        out:=make([]contextStructureTrial,0,len(trials))
        for _,trial:=range trials{
            out=append(out,evalContextStructure(t,file,env,trial,at.Add(40*time.Second),ablate))
        }
        return out
    }
    baseline:=evaluate(initial,false)
    train:=func(flipped bool)(string,int,int){
        r:=orderedFixtureClone(t,initial)
        order:=rand.New(rand.NewSource(2831))
        seen:=0
        for epoch:=0;epoch<8;epoch++{
            for _,j:=range order.Perm(len(lessons)){
                lesson:=lessons[j]
                answer:=lesson.kind
                if flipped {
                    if lesson.kind==0{answer=1
                    }else{answer=0}
                }
                exp:=learning.Experience{
                    NeuralSequence:append(
                        env.stem(lesson.stem,lesson.actor,lesson.generic),
                        env.answer(answer),
                    ),
                    NeuralContext:[][]knowledge.NodeID{env.Context[lesson.context]},
                    NeuralData:[][]knowledge.NodeID{env.datum(lesson.kind)},
                    Weight:.9,Confidence:.9,Reliability:.85,
                    Source:fmt.Sprintf("context-structure-%d",j),
                    IndependenceGroup:fmt.Sprintf("origin-%d",j),
                    EpisodeID:fmt.Sprintf("learning-%t-%d-%d",flipped,epoch,j),
                    ExperienceID:fmt.Sprintf("continued-%t-%d-%d",flipped,epoch,j),
                    Modality:"internal-numeric",
                    Timestamp:at.Add(36*time.Second),
                }
                if _,e:=r.LearnExperience(exp,at.Add(36*time.Second));e!=nil{t.Fatal(e)}
                seen++
            }
        }
        if len(r.brain.Patterns.All())<=prior.ContinuedPatterns{
            t.Fatalf("no new neural patterns from learned contexts/structures: %d",len(r.brain.Patterns.All()))
        }
        if len(r.brain.SurfaceAnnotationsSnapshot())!=0{t.Fatal("new brain language labels appeared")}
        return orderedSave(t,r),seen,len(r.brain.Patterns.All())
    }
    trained,seen,patterns:=train(false)
    reversed,_,_:=train(true)
    after:=evaluate(trained,false)
    ablation:=evaluate(trained,true)
    flipped:=evaluate(reversed,false)
    marginAfter:=snapshotMargin(trained)
    report:=contextStructureReport{
        Protocol:"unseen-context-and-alternate-structure-continuation-v1",
        ParentRun:43035418753,
        ParentCheckpointSHA256:hex.EncodeToString(digest[:]),
        PatternsBefore:prior.ContinuedPatterns,
        PatternsAfter:patterns,Lessons:seen,
        OldMargins:marginBefore,RetainedMargins:marginAfter,
        Baseline:baseline,Trained:after,Ablated:ablation,ReversedTeacher:flipped,
        Limitations:"Learner sees externally separated numeric context, ordered stem, and evidence channels. Three known relational families receive teaching, with actor-3 withheld. Held-out context-structure combinations and fully novel structures receive no supervision. High performance on new actor with shared structural core does not prove universal structural transfer.",
    }
    groups:=[]string{
        "seen-context-new-actor","alternate-new-actor","reverse-new-actor",
        "heldout-structure-context-combination","fully-new-context",
        "fully-new-structure","fully-new-structure-new-context",
    }
    totalBase,totalAfter:=0,0
    for _,group:=range groups{
        pre,post,count:=0,0,0
        for i,x:=range baseline{
            if x.Group!=group{continue}
            count++
            if x.Supported{pre++;totalBase++}
            if after[i].Supported{post++;totalAfter++}
        }
        report.GroupMetrics=append(report.GroupMetrics,struct{
            Group string `json:"group"`
            Baseline int `json:"baseline"`
            Trained int `json:"trained"`
            Total int `json:"total"`
        }{group,pre,post,count})
        t.Logf("GROUP %s baseline=%d/%d after=%d/%d",group,pre,count,post,count)
    }
    t.Logf("PARENT sha256=%s patterns=%d; CONTINUATION lessons=%d patterns=%d",hex.EncodeToString(digest[:]),prior.ContinuedPatterns,seen,patterns)
    t.Logf("RETENTION previous=%v after=%v",marginBefore,marginAfter)
    t.Logf("OVERALL heldout baseline=%d/%d after=%d/%d",totalBase,len(trials),totalAfter,len(trials))
    for i,x:=range after{
        t.Logf("CASE[%02d] %s/%s/%s input=%d actor=%d pre=%d %.3f %.3f post=%d %.3f %.3f scores=%v ablate=%d %.3f %.3f reverse=%d %.3f %.3f",
            i,x.Group,x.Structure,x.Context,x.Input,x.Actor,
            baseline[i].Selected,baseline[i].Margin,baseline[i].Unknown,
            x.Selected,x.Margin,x.Unknown,x.Scores,
            ablation[i].Selected,ablation[i].Margin,ablation[i].Unknown,
            flipped[i].Selected,flipped[i].Margin,flipped[i].Unknown)
    }
    if dest:=os.Getenv("HORIZON_CONTEXT_STRUCTURE_OUTPUT_DIR");dest!=""{
        if err:=os.MkdirAll(dest,0700);err!=nil{t.Fatal(err)}
        restored:=orderedFixtureClone(t,trained)
        if err:=restored.Checkpoint(filepath.Join(dest,"horizon-brain-context-structure.json"));err!=nil{t.Fatal(err)}
        data,err:=json.MarshalIndent(report,"","  ");if err!=nil{t.Fatal(err)}
        if err:=os.WriteFile(filepath.Join(dest,"context-structure-metrics.json"),append(data,byte(10)),0600);err!=nil{t.Fatal(err)}
        t.Log("EXPORTED existing continued Brain checkpoint and strict diagnostics")
    }
    if marginAfter[0]<.07||marginAfter[1]>-.07{
        t.Fatalf("prior relational knowledge lost: %v -> %v",marginBefore,marginAfter)
    }
}
