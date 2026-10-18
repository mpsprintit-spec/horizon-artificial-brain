package main

import (
	"bytes"
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/project-horizon/horizon-core/services/ai/bridge"
	"github.com/project-horizon/horizon-core/services/ai/knowledge"
	"github.com/project-horizon/horizon-core/services/ai/runtime"
)

func newStartupJournal(t *testing.T, path string) *runtime.EventLog {
	t.Helper()
	log,err:=runtime.OpenEventLog(path)
	if err!=nil {t.Fatal(err)}
	t.Cleanup(func(){_ = log.Close()})
	return log
}

func getActiveStartupState(t *testing.T,h interface{
	WithActiveBrain(func(*knowledge.Brain)error)error
}) []byte {
	t.Helper()
	var data []byte
	if err:=h.WithActiveBrain(func(brain *knowledge.Brain)error{
		var err error
		data,err=brain.CanonicalJSON()
		return err
	});err!=nil {t.Fatal(err)}
	return data
}

func TestProductionStartupAnchorsNewBrainBeforeObservationAndRecoversIt(t *testing.T) {
	dir:=t.TempDir()
	journal:=filepath.Join(dir,"events.jsonl")
	memory:=filepath.Join(dir,"brain.json")
	checkpoint:=filepath.Join(dir,"checkpoint.json")
	log:=newStartupJournal(t,journal)
	h,boot,err:=initializeProductionHorizon(log,journal,memory,checkpoint)
	if err!=nil {t.Fatal(err)}
	if !boot {t.Fatal("empty new startup did not request canonical bootstrap")}
	if _,err:=os.Stat(checkpoint);err!=nil {t.Fatalf("sequence-zero checkpoint not anchored: %v",err)}
	if err:=h.Supervisor.WithMutation(func(rt *runtime.BrainRuntime)error{
		if _,err:=rt.CognitiveProcess(runtime.Event{
			ID:"startup-bootstrap",Stimulus:[]string{"light"},Cycles:1,
			Timestamp:time.Date(2026,10,18,13,0,0,0,time.UTC),
		});err!=nil {return err}
		if err:=rt.SaveBrain(memory);err!=nil {return err}
		return rt.Checkpoint(checkpoint)
	});err!=nil {t.Fatal(err)}
	before:=getActiveStartupState(t,h)
	if err:=h.Supervisor.Close();err!=nil {t.Fatal(err)}
	if err:=log.Close();err!=nil {t.Fatal(err)}

	restarted:=newStartupJournal(t,journal)
	next,boot,err:=initializeProductionHorizon(restarted,journal,memory,checkpoint)
	if err!=nil {t.Fatalf("restart from authenticated checkpoint: %v",err)}
	if boot {t.Fatal("restart tried to run bootstrap twice")}
	after:=getActiveStartupState(t,next)
	if !bytes.Equal(before,after) {t.Fatal("production startup changed neural state during anchored recovery")}
	if err:=next.Supervisor.With(func(rt *runtime.BrainRuntime)error{
		if rt.LastSequence()!=1 {t.Fatalf("restart sequence %d",rt.LastSequence())}
		return nil
	});err!=nil {t.Fatal(err)}

	// Check both production adapters share the supervisor's active neural
	// runtime without storing the old pointer.
	monitor,err:=bridge.NewManaged(next.Supervisor,bridge.Config{})
	if err!=nil || monitor==nil {t.Fatalf("managed HTTP Bridge not initialized: %v",err)}
	service,err:=runtime.NewManagedContinuousService(next.Supervisor,time.Hour,1)
	if err!=nil {t.Fatal(err)}
	ctx,cancel:=context.WithCancel(context.Background())
	defer cancel()
	err=service.Run(ctx,func(_ runtime.CognitiveOutput,err error){
		if err!=nil {t.Fatalf("managed cognition cycle failed: %v",err)}
		cancel()
	})
	if !errors.Is(err,context.Canceled) {t.Fatalf("managed service exit: %v",err)}
	if err:=next.Supervisor.With(func(rt *runtime.BrainRuntime)error{
		if rt.LastSequence()!=2 {t.Fatalf("managed cognition did not advance restarted brain: %d",rt.LastSequence())}
		return nil
	});err!=nil {t.Fatal(err)}
}

func TestProductionStartupGenesisReplayMustMatchLegacyBrainFile(t *testing.T) {
	dir:=t.TempDir()
	journal:=filepath.Join(dir,"events.jsonl")
	memory:=filepath.Join(dir,"brain.json")
	checkpoint:=filepath.Join(dir,"checkpoint.json")
	log:=newStartupJournal(t,journal)
	rt:=runtime.NewBrainRuntime(knowledge.NewBootstrapBrain())
	rt.SetEventLog(log)
	if _,_,err:=rt.Process(runtime.Event{ID:"genesis",Stimulus:[]string{"air"},Cycles:1,Timestamp:time.Date(2026,10,18,13,0,0,0,time.UTC)});err!=nil {t.Fatal(err)}
	if err:=rt.SaveBrain(memory);err!=nil {t.Fatal(err)}
	want,err:=os.ReadFile(memory)
	if err!=nil {t.Fatal(err)}
	h,boot,err:=initializeProductionHorizon(log,journal,memory,checkpoint)
	if err!=nil {t.Fatalf("genesis replay refused matching brain file: %v",err)}
	if boot {t.Fatal("journal replay must not bootstrap a second time")}
	if _,err:=os.Stat(checkpoint);err!=nil {t.Fatalf("genesis replay lacked anchored checkpoint: %v",err)}
	got:=getActiveStartupState(t,h)
	if !bytes.Equal(want,got) {t.Fatal("genesis replay did not reconstruct saved neural state")}

	// An incomplete/out-of-band memory file must never be silently
	// adopted as the substrate for a nonempty durable event journal.
	foreign:=knowledge.NewBrain()
	if err:=foreign.Save(memory);err!=nil {t.Fatal(err)}
	if err:=os.Remove(checkpoint);err!=nil {t.Fatal(err)}
	_,_,err=initializeProductionHorizon(log,journal,memory,checkpoint)
	if err==nil || !strings.Contains(err.Error(),"disagrees with committed journal") {
		t.Fatalf("startup accepted diverging memory/journal pairing: %v",err)
	}
	if _,err:=os.Stat(checkpoint);!errors.Is(err,os.ErrNotExist) {
		t.Fatal("failed startup wrote an unauthenticated checkpoint")
	}
}

func TestProductionStartupLegacyMemoryOnlyIsAnchoredAndRestartable(t *testing.T) {
	dir:=t.TempDir()
	journal:=filepath.Join(dir,"events.jsonl")
	memory:=filepath.Join(dir,"legacy.json")
	checkpoint:=filepath.Join(dir,"checkpoint.json")
	legacy:=knowledge.NewBrain()
	legacy.Store("legacy-local-state")
	if err:=legacy.Save(memory);err!=nil {t.Fatal(err)}
	log:=newStartupJournal(t,journal)
	h,boot,err:=initializeProductionHorizon(log,journal,memory,checkpoint)
	if err!=nil {t.Fatal(err)}
	if boot {t.Fatal("memory-only migration must not rerun bootstrap")}
	anchored:=getActiveStartupState(t,h)
	if err:=h.Supervisor.Close();err!=nil {t.Fatal(err)}
	if err:=log.Close();err!=nil {t.Fatal(err)}
	reopened:=newStartupJournal(t,journal)
	recovered,boot,err:=initializeProductionHorizon(reopened,journal,memory,checkpoint)
	if err!=nil {t.Fatalf("anchored legacy memory recovery: %v",err)}
	if boot || !bytes.Equal(anchored,getActiveStartupState(t,recovered)) {
		t.Fatal("initial memory-only migration lost canonical state")
	}
}

func TestProductionStartupFailsClosedOnInvalidCheckpointAndJournal(t *testing.T) {
	dir:=t.TempDir()
	journal:=filepath.Join(dir,"events.jsonl")
	memory:=filepath.Join(dir,"brain.json")
	checkpoint:=filepath.Join(dir,"checkpoint.json")
	log:=newStartupJournal(t,journal)
	_,_,err:=initializeProductionHorizon(log,journal,memory,checkpoint)
	if err!=nil {t.Fatal(err)}
	if err:=os.WriteFile(checkpoint,[]byte("invalid checkpoint"),0600);err!=nil {t.Fatal(err)}
	if _,_,err:=initializeProductionHorizon(log,journal,memory,checkpoint);err==nil {
		t.Fatal("corrupted checkpoint was ignored")
	}
	if err:=os.Remove(checkpoint);err!=nil {t.Fatal(err)}
	if err:=os.WriteFile(journal,[]byte("{invalid durable record}\n"),0600);err!=nil {t.Fatal(err)}
	if _,_,err:=initializeProductionHorizon(log,journal,memory,checkpoint);err==nil {
		t.Fatal("corrupted journal was ignored")
	}
}

func TestProductionStartupRejectsMissingRequiredPaths(t *testing.T) {
	if _,_,err:=initializeProductionHorizon(nil,"","","");err==nil {
		t.Fatal("nil writer and empty required paths accepted")
	}
}
