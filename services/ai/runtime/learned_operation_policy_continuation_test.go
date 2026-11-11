package runtime

// Research only: train operation selection on the EXISTING 580-pattern Brain.
// The policy is stored in canonical learned neural patterns, not a Go
// task-to-action routing table. The experiment's executor interprets chosen
// operation populations and does not choose their order itself.
// The teacher gives operation trajectories ONLY in the training set.
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
 "strings"
 "testing"
 "time"

 "github.com/project-horizon/horizon-core/services/ai/knowledge"
 "github.com/project-horizon/horizon-core/services/ai/learning"
)

const (
 opReadContext=0
 opReadRelation=1
 opCompose=2
 opStop=3
)
type operationPolicyState struct {
 Goal [][]knowledge.NodeID
 Priority [][]knowledge.NodeID
 Status map[int][]knowledge.NodeID
 Actions [][]knowledge.NodeID
 Cue []knowledge.NodeID
}
type operationCaseSpec struct {
 Group,Context,Structure string
 Goal,Priority,Evidence int
}
type operationTrial struct {
 Group string `json:"group"`
 Context string `json:"context"`
 Structure string `json:"structure"`
 Goal int `json:"goal"`
 Priority int `json:"priority"`
 Evidence int `json:"evidence"`
 Target int `json:"target"`
 Selected int `json:"selected"`
 Sequence []int `json:"sequence"`
 Stopped bool `json:"stopped"`
 Unknown bool `json:"unknown"`
 GoalCorrect bool `json:"goal_correct"`
 Efficient bool `json:"efficient"`
 FirstOperationCorrect bool `json:"first_operation_correct"`
 Success bool `json:"success"`
}
type policyGroupMetric struct{
 Name string `json:"name"`
 Before int `json:"before"`
 After int `json:"after"`
 ReversedTeacher int `json:"reversed_teacher"`
 Total int `json:"total"`
}
type operationPolicyReport struct {
 Protocol string `json:"protocol"`
 ParentCheckpointSHA256 string `json:"parent_checkpoint_sha256"`
 ParentPatterns int `json:"parent_patterns"`
 LearnedPatterns int `json:"learned_patterns"`
 AddedLessons int `json:"added_lessons"`
 Before []operationTrial `json:"before"`
 After []operationTrial `json:"after"`
 ReversedTeacher []operationTrial `json:"reversed_teacher"`
 Groups []policyGroupMetric `json:"groups"`
 BeforeSuccess int `json:"before_success"`
 AfterSuccess int `json:"after_success"`
 ReversedSuccess int `json:"reversed_success"`
 Total int `json:"total"`
 UnknownFactorControls []operationTrial `json:"unknown_factor_controls"`
 Limits string `json:"limits"`
}

func policyFactorTarget(name string,evidence int)int {
 if name=="reverse"&&evidence<2{return 1-evidence}
 if name=="novel-r"{return []int{2,0,1}[evidence]}
 return evidence
}
func policyContextTarget(name string)int{
 switch name{
 case "old-a","new-n": return 0
 case "old-b","previous-c","unpaired-u":return 1
 default:return -1
 }
}
func policyExpected(s operationCaseSpec)int{
 ctx,rel:=policyContextTarget(s.Context),policyFactorTarget(s.Structure,s.Evidence)
 switch s.Goal {
 case 0:return ctx
 case 1:return rel
 case 2:
  if ctx==1&&rel<2{return 1-rel}
  return rel
 default:return -1
 }
}
func (env operationPolicyState) predictAction(r *BrainRuntime,goal,status,priority int)(int,float64){
 trace:=&knowledge.ObservationTrace{
  Stimulus:[][]knowledge.NodeID{env.Cue},
  Context:[][]knowledge.NodeID{env.Goal[goal],env.Status[status],env.Priority[priority]},
 }
 return factorChoice(r.brain.Patterns.PredictOrdered(trace),env.Actions)
}
func operationPolicyEvaluate(t *testing.T,r *BrainRuntime, factors factorCompositionEnv,policy operationPolicyState,s operationCaseSpec)operationTrial{
 t.Helper()
 v:=operationTrial{
  Group:s.Group,Context:s.Context,Structure:s.Structure,Goal:s.Goal,
  Priority:s.Priority,Evidence:s.Evidence,Target:policyExpected(s),Selected:-1,
 }
 var ctxIdx,relIdx,combined int=-1,-1,-1
 status:=0
 for cycle:=0;cycle<6;cycle++{
  op,_:=policy.predictAction(r,s.Goal,status,s.Priority)
  if op<0{v.Unknown=true;break}
  v.Sequence=append(v.Sequence,op)
  switch op{
  case opReadContext:
   if status&1!=0 {v.Unknown=true;break}
   trace:=&knowledge.ObservationTrace{
    Stimulus:[][]knowledge.NodeID{genFrame(factors.Cues[0],factors.Actor)},
    Context:[][]knowledge.NodeID{factors.Context[s.Context]},
   }
   idx,_:=factorChoice(r.brain.Patterns.PredictOrdered(trace),factors.ContextFactors)
   if idx<0{v.Unknown=true;break}
   ctxIdx=idx;status|=1
  case opReadRelation:
   if status&2!=0{v.Unknown=true;break}
   stem:=knowledge.ClonePopulationFrames(factors.Structure[s.Structure])
   stem[0]=genFrame(stem[0],factors.Actor)
   trace:=&knowledge.ObservationTrace{
    Stimulus:stem, Data:[][]knowledge.NodeID{factors.Evidence[s.Evidence]},
   }
   idx,_:=factorChoice(r.brain.Patterns.PredictOrdered(trace),factors.RelationFactors)
   if idx<0{v.Unknown=true;break}
   relIdx=idx;status|=2
  case opCompose:
   if ctxIdx<0||relIdx<0||status&4!=0{v.Unknown=true;break}
   trace:=&knowledge.ObservationTrace{Stimulus:[][]knowledge.NodeID{
    factors.ContextFactors[ctxIdx],factors.RelationFactors[relIdx],
   }}
   idx,_:=factorChoice(r.brain.Patterns.PredictOrdered(trace),factors.Results)
   if idx<0{v.Unknown=true;break}
   combined=idx;status|=4
  case opStop:
   v.Stopped=true
   switch s.Goal{
   case 0:v.Selected=ctxIdx
   case 1:v.Selected=relIdx
   case 2:v.Selected=combined
   }
  default:v.Unknown=true
  }
  if v.Stopped||v.Unknown{break}
 }
 v.GoalCorrect=v.Selected==v.Target&&v.Stopped&&!v.Unknown
 expectedFirst:=opReadContext
 if s.Goal==1 || s.Goal==2&&s.Priority==1 {expectedFirst=opReadRelation}
 v.FirstOperationCorrect=len(v.Sequence)>0&&v.Sequence[0]==expectedFirst
 minSteps:=2
 if s.Goal==2{minSteps=4}
 v.Efficient=len(v.Sequence)==minSteps
 v.Success=v.GoalCorrect&&v.FirstOperationCorrect&&v.Efficient
 return v
}

func TestContinuedBrainLearnsOperationSelectionAndStopping(t *testing.T){
 parent:=os.Getenv("HORIZON_POLICY_PRIOR_CHECKPOINT")
 manifest:=os.Getenv("HORIZON_POLICY_PRIOR_METRICS")
 sums:=os.Getenv("HORIZON_POLICY_PRIOR_SHA256")
 if parent==""||manifest==""||sums==""{t.Fatal("missing REQUIRED archived 580-pattern trained Brain or its integrity manifest")}
 contents,e:=os.ReadFile(parent);if e!=nil{t.Fatal(e)}
 hash:=sha256.Sum256(contents)
 expectedSums,e:=os.ReadFile(sums);if e!=nil{t.Fatal(e)}
 if !strings.Contains(string(expectedSums),hex.EncodeToString(hash[:])+"  horizon-brain-factor-composition.json"){
  t.Fatal("archived trained Brain differs from recorded SHA256")
 }
 metricsRaw,e:=os.ReadFile(manifest);if e!=nil{t.Fatal(e)}
 var preceding factorCompositionReport
 if e:=json.Unmarshal(metricsRaw,&preceding);e!=nil{t.Fatal(e)}
 if preceding.Protocol!="learned-two-factor-neural-composition-v1"||preceding.ContinuedPatterns!=580||preceding.Lessons!=184{
  t.Fatalf("invalid trained parent provenance: %+v",preceding)
 }
 brain:=orderedFixtureClone(t,parent)
 if len(brain.brain.Patterns.All())!=580||len(brain.brain.SurfaceAnnotationsSnapshot())!=0{
  t.Fatal("not the trained 580-pattern canonical brain")
 }
 at:=time.Date(2026,11,11,12,0,0,0,time.UTC)
 g:=func(dim,axis int,existing bool)[]knowledge.NodeID{
  vector:=make([]float64,dim);vector[axis]=1
  x,e:=brain.GroundVectorAt(knowledge.NewNeuralVector(vector),at.Add(100*time.Second))
  if e!=nil{t.Fatal(e)}
  if existing&&!x.Existing{t.Fatalf("parent lost existing numerical coordinate %d:%d",dim,axis)}
  if len(x.Population)==0{t.Fatal("grounding failed")}
  return slices.Clone(x.Population)
 }
 env:=factorCompositionEnv{
  Context:make(map[string][]knowledge.NodeID),
  Structure:make(map[string][][]knowledge.NodeID),
 }
 perm:=rand.New(rand.NewSource(17)).Perm(8)
 for i:=0;i<8;i++{env.Base=append(env.Base,g(8,perm[i],true))}
 env.Actor=g(16,4,true)
 env.Context["old-a"]=env.Base[3]
 env.Context["old-b"]=env.Base[4]
 env.Context["previous-c"]=env.Base[5]
 env.Context["new-n"]=g(16,7,true)
 env.Context["unpaired-u"]=g(16,8,true)
 env.Context["unlearned-z"]=g(20,10,true)
 env.Structure["original"]=[][]knowledge.NodeID{env.Base[0],env.Base[1],env.Base[2]}
 env.Structure["alternate"]=[][]knowledge.NodeID{g(16,10,true),g(16,11,true),g(16,12,true)}
 env.Structure["reverse"]=[][]knowledge.NodeID{
  env.Structure["alternate"][2],env.Structure["alternate"][1],env.Structure["alternate"][0],
 }
 env.Structure["novel-r"]=[][]knowledge.NodeID{g(16,13,true),g(16,14,true),g(16,15,true)}
 env.Structure["unlearned-x"]=[][]knowledge.NodeID{g(20,1,true),g(20,2,true),g(20,3,true)}
 fv:=make([]float64,8);fv[perm[6]]=.6;fv[perm[7]]=.6;fv[perm[5]]=-1
 fused,e:=brain.GroundVectorAt(knowledge.NewNeuralVector(fv),at.Add(100*time.Second))
 if e!=nil||!fused.Existing{t.Fatal("parent fused evidence unavailable",e)}
 env.Evidence=[][]knowledge.NodeID{env.Base[6],env.Base[7],slices.Clone(fused.Population)}
 env.Results=[][]knowledge.NodeID{env.Base[6],env.Base[7],g(12,0,true)}
 env.Cues=[][]knowledge.NodeID{g(24,0,true)}
 env.ContextFactors=[][]knowledge.NodeID{g(24,1,true),g(24,2,true)}
 env.RelationFactors=[][]knowledge.NodeID{g(24,3,true),g(24,4,true),g(24,5,true)}
 // The available operations are a fixed research ABI, but the selection
 // weights and stopping policy are all learned; no code picks "context first".
 policy:=operationPolicyState{Status:make(map[int][]knowledge.NodeID)}
 policy.Cue=g(40,0,false)
 for i:=0;i<3;i++{policy.Goal=append(policy.Goal,g(40,2+i,false))}
 for i:=0;i<2;i++{policy.Priority=append(policy.Priority,g(40,6+i,false))}
 for i,k:=range []int{0,1,2,3,7}{policy.Status[k]=g(40,10+i,false)}
 for i:=0;i<4;i++{policy.Actions=append(policy.Actions,g(40,20+i,false))}
 if len(brain.brain.SurfaceAnnotationsSnapshot())!=0{t.Fatal("lexical labels inserted")}
 foundation:=orderedSave(t,brain)

 // The raw context x structure pairs below NEVER appear in policy lessons.
 // Some were held out during previous stage-3 experiments; all policy
 // lessons use only numerical goals, readiness bits and proposed actions.
 defs:=[]operationCaseSpec{}
 add:=func(group string,ctxs,stems []string){
  for _,ctx:=range ctxs{
   for _,stem:=range stems{
    for evidence:=0;evidence<3;evidence++{
     for goal:=0;goal<3;goal++{
      for priority:=0;priority<2;priority++{
       defs=append(defs,operationCaseSpec{group,ctx,stem,goal,priority,evidence})
      }
     }
    }
   }
  }
 }
 add("strict-unseen-reverse-context",[]string{"previous-c","new-n"},[]string{"reverse"})
 add("strict-unseen-alternate-context",[]string{"unpaired-u"},[]string{"alternate"})
 add("separate-novel-structure",[]string{"old-a","old-b","previous-c","new-n","unpaired-u"},[]string{"novel-r"})
 add("existing-factors-retention",[]string{"old-a","old-b"},[]string{"original","alternate"})
 evaluate:=func(file string)[]operationTrial{
  b:=orderedFixtureClone(t,file)
  result:=make([]operationTrial,0,len(defs))
  for _,d:=range defs{result=append(result,operationPolicyEvaluate(t,b,env,policy,d))}
  return result
 }
 baseline:=evaluate(foundation)

 type teaching struct{goal,priority,status,action int}
 curriculum:=make([]teaching,0,40)
 for goal:=0;goal<3;goal++{
  for priority:=0;priority<2;priority++{
   switch goal {
   case 0:
    for _,s:=range []int{0,2}{curriculum=append(curriculum,teaching{goal,priority,s,opReadContext})}
    for _,s:=range []int{1,3,7}{curriculum=append(curriculum,teaching{goal,priority,s,opStop})}
   case 1:
    for _,s:=range []int{0,1}{curriculum=append(curriculum,teaching{goal,priority,s,opReadRelation})}
    for _,s:=range []int{2,3,7}{curriculum=append(curriculum,teaching{goal,priority,s,opStop})}
   case 2:
    first:=opReadContext
    if priority==1{first=opReadRelation}
    curriculum=append(curriculum,
     teaching{goal,priority,0,first},
     teaching{goal,priority,1,opReadRelation},
     teaching{goal,priority,2,opReadContext},
     teaching{goal,priority,3,opCompose},
     teaching{goal,priority,7,opStop},
    )
   }
  }
 }
 train:=func(reverse bool)(string,int,int){
  r:=orderedFixtureClone(t,foundation)
  order:=rand.New(rand.NewSource(13117))
  total:=0
  for epoch:=0;epoch<9;epoch++{
   for _,index:=range order.Perm(len(curriculum)){
    item:=curriculum[index]
    action:=item.action
    if reverse{action=(action+1)%4}
    exp:=learning.Experience{
     NeuralSequence:[][]knowledge.NodeID{policy.Cue,policy.Actions[action]},
     NeuralContext:[][]knowledge.NodeID{
      policy.Goal[item.goal],policy.Status[item.status],policy.Priority[item.priority],
     },
     Weight:.9,Confidence:.9,Reliability:.85,
     Source:fmt.Sprintf("teacher-policy-%d",index),
     IndependenceGroup:fmt.Sprintf("policy-origin-%d",index),
     EpisodeID:fmt.Sprintf("policy-%t-%d-%d",reverse,epoch,index),
     ExperienceID:fmt.Sprintf("policy-%t-%d-%d",reverse,epoch,index),
     Modality:"internal-numeric",Timestamp:at.Add(111*time.Second),
    }
    if _,e:=r.LearnExperience(exp,at.Add(111*time.Second));e!=nil{t.Fatal(e)}
    total++
   }
  }
  if len(r.brain.SurfaceAnnotationsSnapshot())!=0{t.Fatal("lexical label in learned policy")}
  if len(r.brain.Patterns.All())<=580{t.Fatal("no learned neural policy changes")}
  return orderedSave(t,r),total,len(r.brain.Patterns.All())
 }
 checkpoint,lessons,patternCount:=train(false)
 reversal,_,_:=train(true)
 after:=evaluate(checkpoint)
 reverseControl:=evaluate(reversal)

 // Independent control: missing factors may not be fabricated by the
 // scheduler. Valid unrelated factors may still answer single-factor goals.
 controls:=[]operationCaseSpec{
  {"unknown-structure-context-only","old-a","unlearned-x",0,0,0},
  {"unknown-context-relation-only","unlearned-z","reverse",1,1,0},
  {"unknown-structure-composite","old-a","unlearned-x",2,0,0},
  {"unknown-context-composite","unlearned-z","novel-r",2,1,1},
  {"both-factors-unlearned","unlearned-z","unlearned-x",2,0,2},
 }
 controlBrain:=orderedFixtureClone(t,checkpoint)
 unknownControls:=[]operationTrial{}
 for _,c:=range controls{unknownControls=append(unknownControls,operationPolicyEvaluate(t,controlBrain,env,policy,c))}

 success:=func(data []operationTrial)int{n:=0;for _,x:=range data{if x.Success{n++}};return n}
 countBefore,countAfter,countReverse:=success(baseline),success(after),success(reverseControl)
 report:=operationPolicyReport{
  Protocol:"learned-native-operation-policy-continuation-v1",
  ParentCheckpointSHA256:hex.EncodeToString(hash[:]),
  ParentPatterns:580,LearnedPatterns:patternCount,AddedLessons:lessons,
  Before:baseline,After:after,ReversedTeacher:reverseControl,
  BeforeSuccess:countBefore,AfterSuccess:countAfter,
  ReversedSuccess:countReverse,Total:len(defs),
  UnknownFactorControls:unknownControls,
  Limits:"The operation scheduler is experiment code using learned native PatternIndex action predictions from the SAME 580-pattern Brain; it does not hardcode a fixed sequence. Goal/priority/status numerical features and 4-action ABI are externally provided. Trajectories are supervised; no independently learned cost or deep metareasoning. This is not integrated into production native policy, nor general autonomy.",
 }
 for _,group:=range []string{
  "strict-unseen-reverse-context",
  "strict-unseen-alternate-context",
  "separate-novel-structure",
  "existing-factors-retention",
 }{
  a,b,c,n:=0,0,0,0
  for i,def:=range defs{
   if def.Group!=group{continue}
   n++
   if baseline[i].Success{a++}
   if after[i].Success{b++}
   if reverseControl[i].Success{c++}
  }
  report.Groups=append(report.Groups,policyGroupMetric{group,a,b,c,n})
  t.Logf("POLICY_GROUP %s before=%d/%d after=%d/%d reversed=%d/%d",group,a,n,b,n,c,n)
 }
 t.Logf("PARENT_SHA256=%s restored_patterns=580 added_lessons=%d learned_patterns=%d",hex.EncodeToString(hash[:]),lessons,patternCount)
 t.Logf("POLICY overall before=%d/%d after=%d/%d reversed=%d/%d",countBefore,len(defs),countAfter,len(defs),countReverse,len(defs))
 for i,c:=range unknownControls{
  t.Logf("UNKNOWN_FACTOR[%d] %s goal=%d selected=%d target=%d actions=%v success=%t unknown=%t",
   i,c.Group,c.Goal,c.Selected,c.Target,c.Sequence,c.Success,c.Unknown)
 }
 for i:=0;i<min(12,len(defs));i++{
  c:=after[i]
  t.Logf("POLICY_SAMPLE[%d] goal=%d priority=%d selected=%d expected=%d operations=%v stop=%t efficient=%t success=%t",
   i,c.Goal,c.Priority,c.Selected,c.Target,c.Sequence,c.Stopped,c.Efficient,c.Success)
 }
 if dest:=os.Getenv("HORIZON_POLICY_OUTPUT_DIR");dest!=""{
  if e:=os.MkdirAll(dest,0700);e!=nil{t.Fatal(e)}
  if e:=orderedFixtureClone(t,checkpoint).Checkpoint(filepath.Join(dest,"horizon-brain-learned-operation-policy.json"));e!=nil{t.Fatal(e)}
  blob,e:=json.MarshalIndent(report,"","  ");if e!=nil{t.Fatal(e)}
  if e:=os.WriteFile(filepath.Join(dest,"learned-operation-policy-metrics.json"),append(blob,'
'),0600);e!=nil{t.Fatal(e)}
  t.Log("PERSISTED trained 580-pattern Brain continuation and strict policy audit")
 }
 if countAfter<countBefore{t.Fatalf("learned policy regressed: %d<%d",countAfter,countBefore)}
}
