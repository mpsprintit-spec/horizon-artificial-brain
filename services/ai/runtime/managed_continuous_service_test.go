package runtime

import (
	"context"
	"errors"
	"path/filepath"
	"testing"
	"time"

	"github.com/project-horizon/horizon-core/services/ai/knowledge"
)

func TestManagedContinuousServiceResolvesRuntimeAfterHandover(t *testing.T) {
	log, err := OpenEventLog(filepath.Join(t.TempDir(),"events.jsonl"))
	if err != nil { t.Fatal(err) }
	defer log.Close()
	original := NewBrainRuntime(knowledge.NewBootstrapBrain())
	original.SetEventLog(log)
	at := time.Date(2026,10,17,12,0,0,0,time.UTC)
	if _,_,err:=original.Process(Event{ID:"seed",Stimulus:[]string{"saya"},Cycles:1,Timestamp:at});err!=nil { t.Fatal(err) }
	checkpointPath := filepath.Join(t.TempDir(),"checkpoint.json")
	if err:=original.Checkpoint(checkpointPath);err!=nil { t.Fatal(err) }
	supervisor,err:=NewRuntimeSupervisor(original)
	if err!=nil { t.Fatal(err) }
	service,err:=NewManagedContinuousService(supervisor,10*time.Millisecond,1)
	if err!=nil { t.Fatal(err) }
	if service.Runtime != nil { t.Fatal("managed loop cached raw runtime") }

	ctx,cancel:=context.WithCancel(context.Background())
	defer cancel()
	outputs := make(chan error,3)
	done := make(chan error,1)
	go func() {
		count:=0
		done<-service.Run(ctx,func(output CognitiveOutput,err error){
			if err!=nil { outputs<-err;cancel();return }
			count++
			if count==1 {
				if recoverErr:=supervisor.RecoverFromCheckpoint(log,checkpointPath);recoverErr!=nil {
					outputs<-recoverErr;cancel();return
				}
			}
			if count==2 { cancel() }
			outputs<-nil
		})
	}()
	for i:=0;i<2;i++ {
		select {
		case err:= <-outputs:
			if err!=nil { t.Fatalf("managed cognition/hand-over failed: %v",err) }
		case <-time.After(5*time.Second):
			t.Fatal("continuous service deadlocked on runtime handover")
		}
	}
	select {
	case err:= <-done:
		if !errors.Is(err,context.Canceled) { t.Fatalf("service returned %v",err) }
	case <-time.After(5*time.Second):t.Fatal("continuous service failed to stop")
	}
	if !original.RecoveryRequired() { t.Fatal("retired runtime still writable") }
	if err:=supervisor.With(func(r *BrainRuntime) error {
		if r==original { t.Fatal("service still uses old runtime") }
		if r.LastSequence()<3 { t.Fatalf("cognitive loop did not advance new runtime, seq=%d",r.LastSequence()) }
		return nil
	});err!=nil { t.Fatal(err) }
}

func TestManagedContinuousServiceClosesWithSupervisor(t *testing.T) {
	rt:=NewBrainRuntime(nil)
	supervisor,err:=NewRuntimeSupervisor(rt)
	if err!=nil {t.Fatal(err)}
	svc,err:=NewManagedContinuousService(supervisor,time.Hour,1)
	if err!=nil {t.Fatal(err)}
	if err:=supervisor.Close();err!=nil {t.Fatal(err)}
	err=svc.Run(context.Background(),nil)
	if !errors.Is(err,ErrRuntimeSupervisorClosed) {
		t.Fatalf("managed loop ignored closed supervisor: %v",err)
	}
}

func TestManagedContinuousServiceRejectsUnavailableSupervisor(t *testing.T) {
	if _,err:=NewManagedContinuousService(nil,time.Second,1);err==nil {t.Fatal("nil supervisor accepted")}
	supervisor,err:=NewRuntimeSupervisor(NewBrainRuntime(nil))
	if err!=nil {t.Fatal(err)}
	if err:=supervisor.Close();err!=nil {t.Fatal(err)}
	if _,err:=NewManagedContinuousService(supervisor,time.Second,1);!errors.Is(err,ErrRuntimeSupervisorClosed) {
		t.Fatalf("closed supervisor accepted: %v",err)
	}
}

func TestSupervisorEpochInvalidatesOldClients(t *testing.T) {
	log,err:=OpenEventLog(filepath.Join(t.TempDir(),"events.jsonl"))
	if err!=nil {t.Fatal(err)}
	defer log.Close()
	rt:=NewBrainRuntime(nil)
	rt.SetEventLog(log)
	checkpoint:=filepath.Join(t.TempDir(),"checkpoint.json")
	if err:=rt.Checkpoint(checkpoint);err!=nil {t.Fatal(err)}
	sup,err:=NewRuntimeSupervisor(rt)
	if err!=nil {t.Fatal(err)}
	var initial <-chan struct{}
	if err:=sup.WithEpoch(func(_ *BrainRuntime,epoch <-chan struct{}) error {initial=epoch;return nil});err!=nil {t.Fatal(err)}
	if err:=sup.RecoverFromCheckpoint(log,checkpoint);err!=nil {t.Fatal(err)}
	select {case <-initial:default:t.Fatal("old epoch not closed by handover")}
	var next <-chan struct{}
	if err:=sup.WithEpoch(func(_ *BrainRuntime,epoch <-chan struct{}) error {next=epoch;return nil});err!=nil {t.Fatal(err)}
	if next==initial {t.Fatal("handover reused epoch channel")}
	select {case <-next:t.Fatal("new epoch closed too early");default:}
	if err:=sup.Close();err!=nil {t.Fatal(err)}
	select {case <-next:default:t.Fatal("shutdown did not close current epoch")}
}
