package runtime

// Research-only continuation. Requires the actual synthetic checkpoint created
// by TestRelationalContradictionEmptyBrain, NOT an untrained Brain. The learner
// sees numeric populations, ordered experience and consequences; "thinking"
// tasks and labels exist only in this test evaluator, not production inference.
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

type philosophyParentManifest struct {
    Experiment string `json:"experiment"`
    Seed int64 `json:"seed"`
    Origin string `json:"origin"`
    PopulationIDs [][]knowledge.NodeID `json:"population_ids"`
    Lessons int `json:"lessons"`
    PatternCount int `json:"pattern_count"`
}

type philosophyLabProbe struct {
    Context int `json:"context"`
    EvidenceOrder []int `json:"evidence_order"`
    NovelSubject bool `json:"novel_subject"`
    Selected int `json:"selected"`
    Strengths [3]float64 `json:"strengths"`
    Margin float64 `json:"margin"`
    Confidence float64 `json:"confidence"`
    Ambiguity float64 `json:"ambiguity"`
    Unknown float64 `json:"unknown"`
}

type philosophyLabReport struct {
    Protocol string `json:"protocol"`
    ParentRunID int64 `json:"parent_run_id"`
    ParentSeed int64 `json:"parent_seed"`
    ParentCheckpointSHA256 string `json:"parent_checkpoint_sha256"`
    OriginalPatternCount int `json:"original_pattern_count"`
    ContinuedPatternCount int `json:"continued_pattern_count"`
    OriginalMargins [2]float64 `json:"original_relational_margins"`
    RetainedMargins [2]float64 `json:"retained_relational_margins"`
    NewSupervisedLessons int `json:"new_supervised_lessons"`
    Baseline []philosophyLabProbe `json:"baseline"`
    Trained []philosophyLabProbe `json:"trained"`
    Ablated []philosophyLabProbe `json:"ablated"`
    Heldout []philosophyLabProbe `json:"heldout"`
    SeenCorrect int `json:"seen_correct"`
    SeenTotal int `json:"seen_total"`
    HeldoutCorrect int `json:"heldout_correct"`
    HeldoutTotal int `json:"heldout_total"`
    Summary string `json:"summary"`
}

func philosophyReadProbe(t *testing.T, path string, base [][]knowledge.NodeID, novel [][]knowledge.NodeID, outputs [3][]knowledge.NodeID, ctx int, evidence []int, novelSubject bool, at time.Time, ablate bool) philosophyLabProbe {
    t.Helper()
    r := orderedFixtureClone(t, path)
    if ablate {
        r.brain.Patterns = knowledge.NewPatternIndex()
    }
    cue := [][]knowledge.NodeID{base[0],base[1],base[2]}
    if novelSubject {
        cue = [][]knowledge.NodeID{novel[1],novel[2],novel[3]}
    }
    data := make([][]knowledge.NodeID,0,len(evidence))
    for _, idx := range evidence {
        if idx < 0 || idx >= len(base) {
            t.Fatal("invalid evaluator-only evidence coordinate")
        }
        data = append(data,base[idx])
    }
    trace := &knowledge.ObservationTrace{
        Stimulus: cue,
        Context: [][]knowledge.NodeID{base[ctx]},
        Data: data,
    }
    out, err := r.CognitiveProcess(Event{
        ID: fmt.Sprintf("test-%d-%v-%t-%t",ctx,evidence,novelSubject,ablate),
        EpisodeID: "heldout-mental-probe",
        Timestamp: at,
        NeuralTrace: trace,
        Cycles: 1,
    })
    if err != nil {
        t.Fatal(err)
    }
    prediction := r.activation.PredictionSnapshot().State
    res := philosophyLabProbe{Context:ctx,EvidenceOrder:slices.Clone(evidence),NovelSubject:novelSubject,Selected:-1}
    best := math.Inf(-1)
    second := math.Inf(-1)
    for i,pop := range outputs {
        var value float64
        for _,id := range pop {
            value+=prediction[id]
        }
        value/=float64(len(pop))
        res.Strengths[i]=value
        if value>best {
            second=best
            best=value
            res.Selected=i
        } else if value>second {
            second=value
        }
    }
    res.Margin=best-second
    if out.PredictionAssessment!=nil {
        res.Confidence=out.PredictionAssessment.Confidence
        res.Ambiguity=out.PredictionAssessment.Ambiguity
        res.Unknown=out.PredictionAssessment.UnknownProbability
    }
    return res
}

func TestPhilosophyNeuralThinkingContinuesTrainedRelationalBrain(t *testing.T) {
    checkpoint := os.Getenv("HORIZON_PHILOSOPHY_PRIOR_CHECKPOINT")
    manifestPath := os.Getenv("HORIZON_PHILOSOPHY_PRIOR_MANIFEST")
    if checkpoint=="" || manifestPath=="" {
        t.Fatal("the PREVIOUS TRAINED Brain checkpoint and manifest are mandatory; never silently start an empty Brain")
    }
    bytes,err := os.ReadFile(checkpoint)
    if err!=nil {t.Fatal(err)}
    digest:=sha256.Sum256(bytes)
    manifestBytes,err:=os.ReadFile(manifestPath)
    if err!=nil {t.Fatal(err)}
    var previous philosophyParentManifest
    if err=json.Unmarshal(manifestBytes,&previous);err!=nil {t.Fatal(err)}
    if previous.Seed!=17 || previous.Experiment!="numeric-conditional-relational-contradiction-v1" ||
        len(previous.PopulationIDs)!=8 || previous.PatternCount!=16 || previous.Lessons!=40 {
        t.Fatalf("unexpected predecessor: %+v",previous)
    }
    // A new runtime object is required by RestoreCheckpoint, but it is never
    // used to teach or infer until the exact earlier checkpoint is loaded.
    r:=NewBrainRuntime(knowledge.NewBrain())
    if r==nil {t.Fatal("runtime creation failed")}
    if err=r.RestoreCheckpoint(checkpoint);err!=nil {t.Fatal(err)}
    if len(r.brain.Patterns.All())!=previous.PatternCount {
        t.Fatalf("prior trained neural patterns were not restored: got %d want %d",len(r.brain.Patterns.All()),previous.PatternCount)
    }
    if len(r.brain.SurfaceAnnotationsSnapshot())!=0 {
        t.Fatal("previously learned brain unexpectedly has lexical knowledge")
    }
    base:=previous.PopulationIDs
    for _,pop:=range base{
        for _,id:=range pop{
            if r.brain.Registry.GetByID(id)==nil {t.Fatal("parent population lost on checkpoint restore")}
        }
    }

    at:=time.Date(2026,11,11,12,0,0,0,time.UTC)
    probeAt:=at.Add(5*time.Second)
    original:=orderedSave(t,r)
    beforeA:=philosophyReadProbe(t,original,base,nil,[3][]knowledge.NodeID{base[6],base[7],base[6]},3,nil,false,probeAt,false)
    beforeB:=philosophyReadProbe(t,original,base,nil,[3][]knowledge.NodeID{base[6],base[7],base[6]},4,nil,false,probeAt,false)
    originalMargins:=[2]float64{beforeA.Strengths[0]-beforeA.Strengths[1],beforeB.Strengths[0]-beforeB.Strengths[1]}

    // Add numeric populations to the SAME restored Brain, not a second model.
    // The additional 12-D observations are not lexical concepts or rules.
    novel:=make([][]knowledge.NodeID,5)
    for i:=range novel{
        values:=make([]float64,12)
        values[i]=1
        g,groundErr:=r.GroundVectorAt(knowledge.NewNeuralVector(values),at.Add(2*time.Second))
        if groundErr!=nil {t.Fatal(groundErr)}
        novel[i]=slices.Clone(g.Population)
        if len(novel[i])==0 {t.Fatal("no numeric neural population")}
        for j:=0;j<i;j++ {
            if relationLabPopulationEquals(novel[i],novel[j]) {
                t.Fatal("new populations are not distinguishable")
            }
        }
    }
    // Output classes are evaluator-only: support-6, support-7, defer.
    // Brain sees only existing outcome populations and one new numeric outcome.
    outputs:=[3][]knowledge.NodeID{base[6],base[7],novel[0]}

    type caseDef struct{context int;evidence []int;output int}
    train:=[]caseDef{}
    for _,ctx:=range []int{3,4}{
        train=append(train,
            caseDef{ctx,[]int{6},0},
            caseDef{ctx,[]int{7},1},
            caseDef{ctx,[]int{6,7},2},
        )
    }
    // The teacher derives consequences from evidence configurations, never
    // from words or hardcoded production answers. The learner does not receive
    // its expected class in any inference input.
    preTraining:=orderedSave(t,r)
    measure:=func(path string, ablate bool, novelSubject bool, cases []caseDef) []philosophyLabProbe{
        out:=make([]philosophyLabProbe,0,len(cases))
        for _,c:=range cases{
            out=append(out,philosophyReadProbe(t,path,base,novel,outputs,c.context,c.evidence,novelSubject,probeAt,ablate))
        }
        return out
    }
    baseline:=measure(preTraining,false,false,train)

    order:=rand.New(rand.NewSource(1317))
    const epochs=12
    for epoch:=0;epoch<epochs;epoch++{
        for _,i:=range order.Perm(len(train)){
            c:=train[i]
            data:=make([][]knowledge.NodeID,0,len(c.evidence))
            for _,idx:=range c.evidence {data=append(data,base[idx])}
            exp:=learning.Experience{
                NeuralContext:[][]knowledge.NodeID{base[c.context]},
                NeuralData:data,
                NeuralSequence:[][]knowledge.NodeID{base[0],base[1],base[2],outputs[c.output]},
                Weight:.9, Confidence:.9, Reliability:.85,
                Source:fmt.Sprintf("synthetic-reasoning-channel-%d",i),
                Modality:"internal-numeric",
                IndependenceGroup:fmt.Sprintf("fixed-synthetic-origin-%d",i),
                EpisodeID:fmt.Sprintf("thinking-%d-%d",epoch,i),
                ExperienceID:fmt.Sprintf("continuation-%d-%d",epoch,i),
                Timestamp:at.Add(3*time.Second),
            }
            if _,err:=r.LearnExperience(exp,at.Add(3*time.Second));err!=nil{t.Fatal(err)}
        }
    }
    if len(r.brain.SurfaceAnnotationsSnapshot())!=0{
        t.Fatal("reasoning training introduced language labels")
    }
    if len(r.brain.Patterns.All())<=previous.PatternCount {
        t.Fatal("continuation learned no new patterns in the existing Brain")
    }
    for _,pop:=range base{
        for _,id:=range pop{
            if r.brain.Registry.GetByID(id)==nil{t.Fatal("continuation overwrote parent neurons")}
        }
    }
    trainedPath:=orderedSave(t,r)
    afterA:=philosophyReadProbe(t,trainedPath,base,novel,outputs,3,nil,false,probeAt,false)
    afterB:=philosophyReadProbe(t,trainedPath,base,novel,outputs,4,nil,false,probeAt,false)
    retained:=[2]float64{afterA.Strengths[0]-afterA.Strengths[1],afterB.Strengths[0]-afterB.Strengths[1]}

    trained:=measure(trainedPath,false,false,train)
    ablated:=measure(trainedPath,true,false,train)
    // Held out from the new teacher: order reversal, a previously CONFLICTED
    // context, and an entirely different numerical relational triple.
    heldDefs:=[]caseDef{
        {3,[]int{7,6},2},
        {4,[]int{7,6},2},
        {5,[]int{6,7},2},
    }
    held:=measure(trainedPath,false,false,heldDefs)
    novelHeld:=measure(trainedPath,false,true,[]caseDef{
        {3,[]int{6},0},
        {4,[]int{7},1},
        {5,[]int{6,7},2},
    })
    held=append(held,novelHeld...)
    correct:=0
    for i,c:=range train{
        if trained[i].Selected==c.output && trained[i].Margin>.05 && trained[i].Unknown<.8 {correct++}
    }
    heldCorrect:=0
    for i,c:=range heldDefs {
        if held[i].Selected==c.output && held[i].Margin>.05 && held[i].Unknown<.8{heldCorrect++}
    }
    for i,c:=range []caseDef{{3,[]int{6},0},{4,[]int{7},1},{5,[]int{6,7},2}}{
        p:=novelHeld[i]
        if p.Selected==c.output && p.Margin>.05 && p.Unknown<.8{heldCorrect++}
    }

    t.Logf("PARENT checkpoint_sha256=%s seed=%d patterns=%d",hex.EncodeToString(digest[:]),previous.Seed,previous.PatternCount)
    t.Logf("RETAINED prior=%.5f/%.5f after=%.5f/%.5f new_patterns=%d",originalMargins[0],originalMargins[1],retained[0],retained[1],len(r.brain.Patterns.All()))
    for i,c:=range train{
        t.Logf("TRAIN[%d] ctx=%d evidence=%v expected=%d blank=%v learned=%v unknown=%.5f ablated=%v",
            i,c.context,c.evidence,c.output,baseline[i].Strengths,trained[i].Strengths,trained[i].Unknown,ablated[i].Strengths)
    }
    for i,p:=range held {
        t.Logf("HELD[%d] novel=%t ctx=%d evidence=%v selected=%d scores=%v margin=%.5f unknown=%.5f",
            i,p.NovelSubject,p.Context,p.EvidenceOrder,p.Selected,p.Strengths,p.Margin,p.Unknown)
    }
    t.Logf("OUTCOME trained=%d/%d heldout=%d/%d (selection margin > .05 and unknown < .8; not semantic correctness)",
        correct,len(train),heldCorrect,len(held))

    report:=philosophyLabReport{
        Protocol:"philosophical-thinking-numeric-continuation-v1",
        ParentRunID:43029895990,ParentSeed:previous.Seed,
        ParentCheckpointSHA256:hex.EncodeToString(digest[:]),
        OriginalPatternCount:previous.PatternCount,
        ContinuedPatternCount:len(r.brain.Patterns.All()),
        OriginalMargins:originalMargins,RetainedMargins:retained,
        NewSupervisedLessons:epochs*len(train),
        Baseline:baseline,Trained:trained,Ablated:ablated,Heldout:held,
        SeenCorrect:correct,SeenTotal:len(train),
        HeldoutCorrect:heldCorrect,HeldoutTotal:len(held),
        Summary:"Conditional consistency, evidential conflict and counterfactual evidence changes are tested on the existing trained Brain. The supervised input factorization and outcomes were provided by the teacher. Held-out transfer is separately reported; no philosophical semantics or generalized reasoning is claimed.",
    }
    if outDir:=os.Getenv("HORIZON_PHILOSOPHY_OUTPUT_DIR");outDir!=""{
        if err:=os.MkdirAll(outDir,0700);err!=nil{t.Fatal(err)}
        target:=filepath.Join(outDir,"horizon-brain-after-thinking.json")
        if err:=r.Checkpoint(target);err!=nil{t.Fatal(err)}
        data,err:=json.MarshalIndent(report,"","  ")
        if err!=nil{t.Fatal(err)}
        if err:=os.WriteFile(filepath.Join(outDir,"thinking-metrics.json"),append(data,byte(10)),0600);err!=nil{t.Fatal(err)}
        t.Logf("EXPORTED continued existing Brain and metrics to artifact directory")
    }
    // Demonstrate the mechanism is truly learned on the existing Brain.
    if correct<4 {
        t.Fatalf("failed to learn controlled evidence-operations: %d/%d",correct,len(train))
    }
    if retained[0]<.07 || retained[1]>-.07{
        t.Fatalf("continuation erased the earlier relational discrimination: %v",retained)
    }
}
