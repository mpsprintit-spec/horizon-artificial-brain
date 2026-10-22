package core

import (
	"errors"
	"path/filepath"
	"testing"
	"time"

	"github.com/project-horizon/horizon-core/services/ai/knowledge"
	"github.com/project-horizon/horizon-core/services/ai/perception"
	"github.com/project-horizon/horizon-core/services/ai/runtime"
)

func TestManagedHorizonPulseFollowsCheckpointHandover(t *testing.T) {
	h := NewHorizonEngine()
	if h == nil || h.Supervisor() == nil { t.Fatal("Horizon must own a managed runtime supervisor") }
	legacy := runtimeForTest(t,h)
	log, err := runtime.OpenEventLog(filepath.Join(t.TempDir(), "events.jsonl"))
	if err != nil { t.Fatal(err) }
	defer log.Close()
	attachJournalForTest(t,h,log)
	at := time.Date(2026,10,18,7,0,0,0,time.UTC)
	signals := []perception.PerceptionSignal{{Source:"horizon-test", Modality:"language", Tokens:[]string{"cahaya"}, ObservedAt:at}}
	result, err := h.PulseSignals(signals)
	if err != nil { t.Fatalf("first managed pulse: %v", err) }
	if len(result)!=1 || result[0].State.Sequence!=1 {t.Fatalf("managed pulse did not commit: %+v",result)}
	checkpoint := filepath.Join(t.TempDir(),"checkpoint.json")
	if err:=legacy.Checkpoint(checkpoint);err!=nil {t.Fatalf("create anchored checkpoint: %v",err)}
	if err:=h.Supervisor().RecoverFromCheckpoint(log,checkpoint);err!=nil {t.Fatalf("handover core facade: %v",err)}
	if !legacy.RecoveryRequired() {t.Fatal("old public alias unexpectedly writable after handover")}
	if _,err:=legacy.CognitiveProcess(runtime.Event{ID:"stale-alias",Cycles:1,Timestamp:at});!errors.Is(err,runtime.ErrRuntimeRecoveryRequired) {
		t.Fatalf("stale runtime accepted cognition: %v",err)
	}
	next := []perception.PerceptionSignal{{Source:"horizon-test",Modality:"language",Tokens:[]string{"objek"},ObservedAt:at.Add(time.Second)}}
	continued,err:=h.PulseSignals(next)
	if err!=nil {t.Fatalf("facade still used retired orchestrator: %v",err)}
	if len(continued)!=1 || continued[0].State.Sequence!=2 {t.Fatalf("unexpected continuation %+v",continued)}
	if err:=h.Supervisor().With(func(rt *runtime.BrainRuntime) error {
		if rt==legacy {t.Fatal("supervisor retained stale runtime")}
		if rt.LastSequence()!=2 {t.Fatalf("new brain sequence=%d",rt.LastSequence())}
		return nil
	});err!=nil {t.Fatal(err)}
}

func TestManagedHorizonPulseLabelsResolveFromCurrentBrain(t *testing.T) {
	h := NewHorizonEngine()
	at:=time.Date(2026,10,18,7,5,0,0,time.UTC)
	h.Perception=fixedPerception{signal:perception.PerceptionSignal{Source:"mic",Modality:"language",Tokens:[]string{"terang","cahaya"},ObservedAt:at}}
	result, labels,err:=h.PulseWithLabels("ignored")
	if err!=nil {t.Fatal(err)}
	if result.State.Sequence!=1 || len(labels)==0 {t.Fatalf("managed labels missing: state=%d labels=%v",result.State.Sequence,labels)}
	snapshot := inspectForTest(t,h)
	for _, id := range result.Interpretation.RankedNodeIDs { if !snapshot.Graph.HasNode(id) { t.Fatalf("ranked id %d is absent from active brain", id) } }
}

func TestManagedHorizonFailsClosedAfterSupervisorRetirement(t *testing.T) {
	h:=NewHorizonEngine()
	legacy := runtimeForTest(t,h)
	if err:=h.Supervisor().Close();err!=nil {t.Fatal(err)}
	at:=time.Date(2026,10,18,7,0,0,0,time.UTC)
	_,err:=h.PulseSignals([]perception.PerceptionSignal{{Source:"sensor",Modality:"text",Tokens:[]string{"air"},ObservedAt:at}})
	if !errors.Is(err,runtime.ErrRuntimeSupervisorClosed) {t.Fatalf("closed supervisor permitted pulse: %v",err)}
	_,err=h.ThinkActive(1)
	if !errors.Is(err,runtime.ErrRuntimeSupervisorClosed) {t.Fatalf("closed supervisor permitted thought: %v",err)}
	if err:=h.SaveActiveBrain(filepath.Join(t.TempDir(),"memory.json"));!errors.Is(err,runtime.ErrRuntimeSupervisorClosed) {t.Fatalf("closed supervisor permitted save: %v",err)}
	if err:=h.WithActiveBrain(func(*knowledge.Brain) error {t.Fatal("brain view ran after retirement");return nil});!errors.Is(err,runtime.ErrRuntimeSupervisorClosed) {
		t.Fatalf("closed supervisor permitted brain view: %v",err)
	}
	if legacy==nil || !legacy.RecoveryRequired(){t.Fatal("retired raw runtime unexpectedly healthy")}
}

func TestManagedHorizonSingleMutationLeaseForMultimodalBatch(t *testing.T) {
	h:=NewHorizonEngine()
	at:=time.Date(2026,10,18,7,0,0,0,time.UTC)
	signals:=[]perception.PerceptionSignal{
		{Source:"mic",Modality:"audio",Tokens:[]string{"bunyi"},ObservedAt:at},
		{Source:"camera",Modality:"vision",Tokens:[]string{"cahaya"},ObservedAt:at.Add(time.Millisecond)},
	}
	output,err:=h.PulseSignals(signals)
	if err!=nil {t.Fatal(err)}
	if len(output)!=2 || output[0].State.Sequence!=1 || output[1].State.Sequence!=2 {
		t.Fatalf("managed multimodal batch not ordered: %+v",output)
	}
	if len(inspectForTest(t,h).Graph.NeuralUnits()) == 0 { t.Fatal("no neural state available") }
}
