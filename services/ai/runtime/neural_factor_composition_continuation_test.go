package runtime

// Research-only factor-composition experiment. All learned patterns are stored
// on the SAME restored 488-pattern Horizon Brain. A three-step test-only
// inquiry sequence drives native PredictOrdered synaptic pathways. The teacher
// defines a separate numerical context factor, a separate structural factor,
// and an independent combination operation; it never teaches a context x
// structure pair. Nothing in this file changes production inference or main.
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

type factorCompositionCase struct {
    Group string `json:"group"`
    Context string `json:"context"`
    Structure string `json:"structure"`
    Evidence int `json:"evidence"`
    Expected int `json:"expected"`
    ContextFactor int `json:"context_factor"`
    StructureFactor int `json:"structure_factor"`
    Selected int `json:"selected"`
    ContextMargin float64 `json:"context_margin"`
    StructureMargin float64 `json:"structure_margin"`
    CompositionMargin float64 `json:"composition_margin"`
    Supported bool `json:"supported"`
}
type factorCompositionReport struct {
    Protocol string `json:"protocol"`
    ParentRun int64 `json:"parent_run"`
    ParentCheckpointSHA256 string `json:"parent_checkpoint_sha256"`
    ParentPatterns int `json:"parent_patterns"`
    ContinuedPatterns int `json:"continued_patterns"`
    Lessons int `json:"lessons"`
    OriginalMargins [2]float64 `json:"original_margins"`
    RetainedMargins [2]float64 `json:"retained_margins"`
    Baseline []factorCompositionCase `json:"baseline"`
    After []factorCompositionCase `json:"after"`
    ReversedContextTeacher []factorCompositionCase `json:"reversed_context_teacher"`
    Ablated []factorCompositionCase `json:"ablated"`
    Groups []struct {
        Name string `json:"name"`
        Before int `json:"before"`
        After int `json:"after"`
        Total int `json:"total"`
    } `json:"groups"`
    Limitations string `json:"limitations"`
}
type factorCompositionSpec struct {
    group,context,structure string
    evidence,expected int
}
type factorCompositionEnv struct {
    Base [][]knowledge.NodeID
    Actor []knowledge.NodeID
    Context map[string][]knowledge.NodeID
    Structure map[string][][]knowledge.NodeID
    Evidence [][]knowledge.NodeID
    Cues [][]knowledge.NodeID
    ContextFactors [][]knowledge.NodeID
    RelationFactors [][]knowledge.NodeID
    Results [][]knowledge.NodeID
}

func factorChoice(p knowledge.OrderedPatternPrediction, choices [][]knowledge.NodeID)(int,float64) {
    selected:=-1
    best,runner:=0.0,0.0
    for i,ids:=range choices {
        var score float64
        for _,id:=range ids{
            score=math.Max(score,p.Strength[id])
        }
        if score>best {
            runner=best
            best=score
            selected=i
        }else if score>runner{runner=score}
    }
    margin:=best-runner
    if selected<0||margin<.015||best<.04{return -1,margin}
    return selected,margin
}
func factorRead(t *testing.T,r *BrainRuntime,env factorCompositionEnv,s factorCompositionSpec) factorCompositionCase{
    t.Helper()
    answer:=factorCompositionCase{
        Group:s.group,Context:s.context,Structure:s.structure,
        Evidence:s.evidence,Expected:s.expected,
        ContextFactor:-1,StructureFactor:-1,Selected:-1,
    }
    cue:=genFrame(env.Cues[0],env.Actor)
    // First neural operation: learned context -> internal contextual state.
    cTrace:=&knowledge.ObservationTrace{
        Stimulus:[][]knowledge.NodeID{cue},
        Context:[][]knowledge.NodeID{env.Context[s.context]},
    }
    contextPrediction:=r.brain.Patterns.PredictOrdered(cTrace)
    answer.ContextFactor,answer.ContextMargin=factorChoice(contextPrediction,env.ContextFactors)
    if answer.ContextFactor<0{return answer}
    // Second neural operation: separately learned ordered structure + evidence
    // -> intermediate relation state. Neither the context nor final answer is
    // provided in this step.
    stem:=knowledge.ClonePopulationFrames(env.Structure[s.structure])
    stem[0]=genFrame(stem[0],env.Actor)
    rTrace:=&knowledge.ObservationTrace{
        Stimulus:stem,
        Data:[][]knowledge.NodeID{env.Evidence[s.evidence]},
    }
    relationPrediction:=r.brain.Patterns.PredictOrdered(rTrace)
    answer.StructureFactor,answer.StructureMargin=factorChoice(relationPrediction,env.RelationFactors)
    if answer.StructureFactor<0{return answer}
    // Third operation: the two SELECTED neural states are presented to an
    // independently trained composition synapse. No target/output label is
    // ever supplied in the query. All mappings live in Brain patterns.
    c:=env.ContextFactors[answer.ContextFactor]
    v:=env.RelationFactors[answer.StructureFactor]
    combo:=&knowledge.ObservationTrace{Stimulus:[][]knowledge.NodeID{c,v}}
    final:=r.brain.Patterns.PredictOrdered(combo)
    answer.Selected,answer.CompositionMargin=factorChoice(final,env.Results)
    answer.Supported=answer.Selected==answer.Expected && answer.Selected>=0 &&
        answer.ContextMargin>=.015 && answer.StructureMargin>=.015 && answer.CompositionMargin>=.015
    return answer
}
func TestComposeTwoSeparatelyLearnedNeuralFactors(t *testing.T){
    parentFile:=os.Getenv("HORIZON_FACTOR_COMPOSITION_PRIOR_CHECKPOINT")
    priorMetricsFile:=os.Getenv("HORIZON_FACTOR_COMPOSITION_PRIOR_METRICS")
    if parentFile==""||priorMetricsFile==""{
        t.Fatal("previously trained 488-pattern Brain checkpoint and metrics are required")
    }
    raw,err:=os.ReadFile(parentFile);if err!=nil{t.Fatal(err)}
    h:=sha256.Sum256(raw)
    metricBytes,err:=os.ReadFile(priorMetricsFile);if err!=nil{t.Fatal(err)}
    var prior contextStructureReport
    if err=json.Unmarshal(metricBytes,&prior);err!=nil{t.Fatal(err)}
    if prior.Protocol!="unseen-context-and-alternate-structure-continuation-v1"||prior.PatternsAfter!=488{
        t.Fatalf("unexpected predecessor: %+v",prior)
    }
    brain:=orderedFixtureClone(t,parentFile)
    if len(brain.brain.Patterns.All())!=488||len(brain.brain.SurfaceAnnotationsSnapshot())!=0{
        t.Fatal("previous 488-pattern neural substrate is missing")
    }
    at:=time.Date(2026,11,11,12,0,0,0,time.UTC)
    env:=factorCompositionEnv{
        Context:make(map[string][]knowledge.NodeID),
        Structure:make(map[string][][]knowledge.NodeID),
    }
    ground:=func(dim,axis int,expectKnown bool)[]knowledge.NodeID{
        values:=make([]float64,dim);values[axis]=1
        g,e:=brain.GroundVectorAt(knowledge.NewNeuralVector(values),at.Add(61*time.Second))
        if e!=nil{t.Fatal(e)}
        if expectKnown&&!g.Existing{t.Fatalf("cannot recover previously grounded vector %d:%d",dim,axis)}
        if len(g.Population)==0{t.Fatal("numeric grounding failed")}
        return slices.Clone(g.Population)
    }
    perm:=rand.New(rand.NewSource(17)).Perm(8)
    for i:=0;i<8;i++{env.Base=append(env.Base,ground(8,perm[i],true))}
    env.Actor=ground(16,4,true)
    env.Context["old-a"]=env.Base[3]
    env.Context["old-b"]=env.Base[4]
    env.Context["previous-c"]=env.Base[5]
    env.Context["new-n"]=ground(16,7,true)
    env.Context["unpaired-u"]=ground(16,8,true)
    env.Context["unlearned-z"]=ground(20,10,false)

    env.Structure["original"]=[][]knowledge.NodeID{env.Base[0],env.Base[1],env.Base[2]}
    env.Structure["alternate"]=[][]knowledge.NodeID{ground(16,10,true),ground(16,11,true),ground(16,12,true)}
    env.Structure["reverse"]=[][]knowledge.NodeID{env.Structure["alternate"][2],env.Structure["alternate"][1],env.Structure["alternate"][0]}
    env.Structure["novel-r"]=[][]knowledge.NodeID{ground(16,13,true),ground(16,14,true),ground(16,15,true)}
    env.Structure["unlearned-x"]=[][]knowledge.NodeID{ground(20,1,false),ground(20,2,false),ground(20,3,false)}
    env.Evidence=[][]knowledge.NodeID{env.Base[6],env.Base[7]}
    fusedVector:=make([]float64,8)
    fusedVector[perm[6]]=.6;fusedVector[perm[7]]=.6;fusedVector[perm[5]]=-1
    fg,err:=brain.GroundVectorAt(knowledge.NewNeuralVector(fusedVector),at.Add(62*time.Second))
    if err!=nil{t.Fatal(err)}
    if !fg.Existing{t.Fatal("previous fused evidence population lost")}
    env.Evidence=append(env.Evidence,slices.Clone(fg.Population))
    deferPop:=ground(12,0,true)
    env.Results=[][]knowledge.NodeID{env.Base[6],env.Base[7],deferPop}

    // New latent neural populations are NUMERIC and share the canonical Brain.
    // These are not names or categories inserted into production.
    for i:=0;i<7;i++ {
        g:=ground(24,i,false)
        if i==0{env.Cues=append(env.Cues,g)
        }else if i<=2{env.ContextFactors=append(env.ContextFactors,g)
        }else{env.RelationFactors=append(env.RelationFactors,g)}
    }
    // Only the first THREE relation factor units are used; the fourth is an
    // unused, independently grounded negative-control neuron population.
    env.RelationFactors=env.RelationFactors[:3]
    if len(brain.brain.SurfaceAnnotationsSnapshot())!=0{t.Fatal("lexical knowledge appeared")}
    original:=orderedSave(t,brain)

    oldMargins:=func(file string)[2]float64{
        var v [2]float64
        for i,ctx:=range []int{3,4}{
            probe:=orderedFixtureClone(t,file)
            score,_:=relationLabScore(t,orderedSave(t,probe),env.Base,ctx,at.Add(71*time.Second),false)
            v[i]=score
        }
        return v
    }
    old:=oldMargins(original)
    // Hold out ALL combinations from the new curriculum. Earlier parent
    // checkpoints also never observed the strict reverse x C/N cases,
    // or novel-r x any real context.
    cases:=[]factorCompositionSpec{}
    add:=func(group,relation string,contexts ...string){
        for _,ctx:=range contexts{
            for evidence:=0;evidence<3;evidence++{
                // Evaluation rule is the teacher's mathematical definition,
                // completely outside the neural inference path.
                structureValue:=evidence
                if relation=="reverse"&&evidence<2{structureValue=1-evidence}
                if relation=="novel-r"{
                    structureValue=[]int{2,0,1}[evidence]
                }
                parity:=map[string]int{
                    "old-a":0,"old-b":1,"previous-c":1,"new-n":0,
                    "unpaired-u":1,"unlearned-z":0,
                }[ctx]
                expected:=structureValue
                if parity==1&&expected<2{expected=1-expected}
                cases=append(cases,factorCompositionSpec{
                    group:group,context:ctx,structure:relation,evidence:evidence,expected:expected,
                })
            }
        }
    }
    add("never-paired-reverse-context","reverse","previous-c","new-n")
    add("previously-unknown-context","alternate","unpaired-u")
    add("new-structure-compositions","novel-r","old-a","old-b","previous-c","new-n","unpaired-u")
    add("strict-new-structure-control","unlearned-x","previous-c")
    add("strict-new-context-control","novel-r","unlearned-z")
    add("strict-both-new-control","unlearned-x","unlearned-z")
    eval:=func(file string,ablate bool)[]factorCompositionCase{
        r:=orderedFixtureClone(t,file)
        if ablate{r.brain.Patterns=knowledge.NewPatternIndex()}
        out:=make([]factorCompositionCase,0,len(cases))
        for _,c:=range cases{out=append(out,factorRead(t,r,env,c))}
        return out
    }
    before:=eval(original,false)

    // Train TWO factor projections separately and ONE generic neural binary
    // operator. The learner never sees any context/structure pair.
    contexts:=[]struct{name string;latent int}{
        {"old-a",0},{"old-b",1},{"previous-c",1},{"new-n",0},{"unpaired-u",1},
    }
    structureKinds:=[]string{"original","alternate","reverse","novel-r"}
    structureTarget:=func(name string,evidence int)int{
        if name=="reverse"&&evidence<2{return 1-evidence}
        if name=="novel-r"{return []int{2,0,1}[evidence]}
        return evidence
    }
    train:=func(reverseContext bool)(string,int,int){
        r:=orderedFixtureClone(t,original)
        rng:=rand.New(rand.NewSource(8871))
        count:=0
        for epoch:=0;epoch<8;epoch++{
            for _,i:=range rng.Perm(len(contexts)){
                c:=contexts[i]
                choice:=c.latent
                if reverseContext{choice=1-choice}
                exp:=learning.Experience{
                    NeuralSequence:[][]knowledge.NodeID{env.Cues[0],env.ContextFactors[choice]},
                    NeuralContext:[][]knowledge.NodeID{env.Context[c.name]},
                    Weight:.9,Confidence:.9,Reliability:.85,Modality:"internal-numeric",
                    Source:fmt.Sprintf("isolated-context-factor-%d",i),
                    IndependenceGroup:fmt.Sprintf("context-origin-%d",i),
                    EpisodeID:fmt.Sprintf("factor-c-%t-%d-%d",reverseContext,epoch,i),
                    ExperienceID:fmt.Sprintf("factor-c-%t-%d-%d",reverseContext,epoch,i),
                    Timestamp:at.Add(64*time.Second),
                }
                if _,e:=r.LearnExperience(exp,at.Add(64*time.Second));e!=nil{t.Fatal(e)}
                count++
            }
            for _,index:=range rng.Perm(len(structureKinds)*3){
                name:=structureKinds[index/3]
                evidence:=index%3
                result:=structureTarget(name,evidence)
                exp:=learning.Experience{
                    NeuralSequence:append(
                        knowledge.ClonePopulationFrames(env.Structure[name]),
                        env.RelationFactors[result],
                    ),
                    NeuralData:[][]knowledge.NodeID{env.Evidence[evidence]},
                    Weight:.9,Confidence:.9,Reliability:.85,Modality:"internal-numeric",
                    Source:fmt.Sprintf("isolated-structure-factor-%d",index),
                    IndependenceGroup:fmt.Sprintf("structure-origin-%d",index),
                    EpisodeID:fmt.Sprintf("factor-r-%t-%d-%d",reverseContext,epoch,index),
                    ExperienceID:fmt.Sprintf("factor-r-%t-%d-%d",reverseContext,epoch,index),
                    Timestamp:at.Add(65*time.Second),
                }
                if _,e:=r.LearnExperience(exp,at.Add(65*time.Second));e!=nil{t.Fatal(e)}
                count++
            }
            // Generic learned combination operator: 2 latent contextual
            // states x 3 latent structural states. NO raw context node or
            // relational triple appears in any of these lessons.
            for _,i:=range rng.Perm(6){
                c,j:=i/3,i%3
                result:=j
                if c==1&&j<2{result=1-j}
                exp:=learning.Experience{
                    NeuralSequence:[][]knowledge.NodeID{
                        env.ContextFactors[c],env.RelationFactors[j],env.Results[result],
                    },
                    Weight:.9,Confidence:.9,Reliability:.85,Modality:"internal-numeric",
                    Source:fmt.Sprintf("independent-composer-%d",i),
                    IndependenceGroup:fmt.Sprintf("operator-origin-%d",i),
                    EpisodeID:fmt.Sprintf("factor-compose-%t-%d-%d",reverseContext,epoch,i),
                    ExperienceID:fmt.Sprintf("factor-compose-%t-%d-%d",reverseContext,epoch,i),
                    Timestamp:at.Add(66*time.Second),
                }
                if _,e:=r.LearnExperience(exp,at.Add(66*time.Second));e!=nil{t.Fatal(e)}
                count++
            }
        }
        if len(r.brain.Patterns.All())<=prior.PatternsAfter{
            t.Fatal("training did not add learned neural composition pathways")
        }
        if len(r.brain.SurfaceAnnotationsSnapshot())!=0{t.Fatal("unexpected linguistic surface annotations")}
        return orderedSave(t,r),count,len(r.brain.Patterns.All())
    }
    trained,lessons,patternCount:=train(false)
    swapped,_,_:=train(true)
    diag:=orderedFixtureClone(t,trained)
    for _,contextName:=range []string{"old-a","previous-c","unpaired-u"} {
        noActor:=&knowledge.ObservationTrace{
            Stimulus:[][]knowledge.NodeID{env.Cues[0]},
            Context:[][]knowledge.NodeID{env.Context[contextName]},
        }
        withActor:=&knowledge.ObservationTrace{
            Stimulus:[][]knowledge.NodeID{genFrame(env.Cues[0],env.Actor)},
            Context:[][]knowledge.NodeID{env.Context[contextName]},
        }
        pa:=diag.brain.Patterns.PredictOrdered(noActor)
        pb:=diag.brain.Patterns.PredictOrdered(withActor)
        ca,ma:=factorChoice(pa,env.ContextFactors)
        cb,mb:=factorChoice(pb,env.ContextFactors)
        t.Logf("DIAG context=%s predicted(noactor=%d/%.5f/%v withactor=%d/%.5f/%v) cue=%v actor=%v ctx=%v ctxfactors=%v",
            contextName,ca,ma,pa.Strength,cb,mb,pb.Strength,
            env.Cues[0],env.Actor,env.Context[contextName],env.ContextFactors)
        found:=0
        for _,p:=range diag.brain.Patterns.All() {
            if p.Observation==nil||len(p.Observation.Stimulus)!=2||len(p.Observation.Context)!=1{continue}
            if relationLabPopulationEquals(p.Observation.Stimulus[0],env.Cues[0])&&
                relationLabPopulationEquals(p.Observation.Context[0],env.Context[contextName]) {
                found++
                if found<=3 {
                    t.Logf("DIAG candidate id=%d result=%d weight=%.6f confidence=%.6f freq=%d",p.ID,p.Result,p.Weight,p.Confidence,p.Frequency)
                }
            }
        }
        t.Logf("DIAG context pattern count=%d",found)
    }
    after:=eval(trained,false)
    ablated:=eval(trained,true)
    reversedTeacher:=eval(swapped,false)
    retained:=oldMargins(trained)
    result:=factorCompositionReport{
        Protocol:"learned-two-factor-neural-composition-v1",
        ParentRun:43037032979,ParentCheckpointSHA256:hex.EncodeToString(h[:]),
        ParentPatterns:prior.PatternsAfter,ContinuedPatterns:patternCount,
        Lessons:lessons,OriginalMargins:old,RetainedMargins:retained,
        Baseline:before,After:after,ReversedContextTeacher:reversedTeacher,Ablated:ablated,
        Limitations:"All 3 operations are independently trained native neural patterns on one pre-trained Brain; an experimental test-only scheduler selects intermediate neural population maxima and feeds them to the next learned operation. It is not an autonomous thought planner or arbitrary unseen factor discovery. The operator's 2x3 truth table was taught; no raw context x relation pair was taught.",
    }
    groups:=[]string{
        "never-paired-reverse-context",
        "previously-unknown-context",
        "new-structure-compositions",
        "strict-new-structure-control",
        "strict-new-context-control",
        "strict-both-new-control",
    }
    totalBefore,totalAfter:=0,0
    for _,group:=range groups{
        baseline,trainedOK,n:=0,0,0
        for i,c:=range cases{
            if c.group!=group{continue}
            n++
            if before[i].Supported{baseline++;totalBefore++}
            if after[i].Supported{trainedOK++;totalAfter++}
        }
        result.Groups=append(result.Groups,struct {
            Name string `json:"name"`
            Before int `json:"before"`
            After int `json:"after"`
            Total int `json:"total"`
        }{group,baseline,trainedOK,n})
        t.Logf("GROUP %s baseline=%d/%d after=%d/%d",group,baseline,n,trainedOK,n)
    }
    t.Logf("PARENT checkpoint sha256=%s patterns=%d; FACTORIZATION lessons=%d total_patterns=%d",hex.EncodeToString(h[:]),prior.PatternsAfter,lessons,patternCount)
    t.Logf("RETENTION original=%v continued=%v",old,retained)
    t.Logf("ALL cases before=%d/%d after=%d/%d",totalBefore,len(cases),totalAfter,len(cases))
    for i,c:=range cases{
        t.Logf("CASE[%02d] %s context=%s structure=%s evidence=%d expected=%d pre=%d after=%d factors=%d/%d margins=%.3f/%.3f/%.3f ablate=%d swapped=%d",
            i,c.group,c.context,c.structure,c.evidence,c.expected,
            before[i].Selected,after[i].Selected,after[i].ContextFactor,after[i].StructureFactor,
            after[i].ContextMargin,after[i].StructureMargin,after[i].CompositionMargin,
            ablated[i].Selected,reversedTeacher[i].Selected)
    }
    if dir:=os.Getenv("HORIZON_FACTOR_COMPOSITION_OUTPUT_DIR");dir!=""{
        if e:=os.MkdirAll(dir,0700);e!=nil{t.Fatal(e)}
        restored:=orderedFixtureClone(t,trained)
        if e:=restored.Checkpoint(filepath.Join(dir,"horizon-brain-factor-composition.json"));e!=nil{t.Fatal(e)}
        blob,e:=json.MarshalIndent(result,"","  ");if e!=nil{t.Fatal(e)}
        if e:=os.WriteFile(filepath.Join(dir,"factor-composition-metrics.json"),append(blob,byte(10)),0600);e!=nil{t.Fatal(e)}
        t.Log("EXPORTED same trained Brain plus complete evaluation metrics")
    }
    if retained[0]<.07||retained[1]>-.07{
        t.Fatalf("old relational knowledge degraded: %v -> %v",old,retained)
    }
    if totalAfter<totalBefore{t.Fatalf("compositional transfer regressed: %d -> %d",totalBefore,totalAfter)}
}
