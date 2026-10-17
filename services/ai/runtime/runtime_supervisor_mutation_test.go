package runtime

import (
	"errors"
	"testing"
	"time"
)

func TestSupervisorMutationExcludesConcurrentManagedRead(t *testing.T) {
	supervisor,err:=NewRuntimeSupervisor(NewBrainRuntime(nil))
	if err!=nil {t.Fatal(err)}
	started:=make(chan struct{})
	release:=make(chan struct{})
	mutationDone:=make(chan error,1)
	go func(){
		mutationDone<-supervisor.WithMutation(func(*BrainRuntime) error {
			close(started)
			<-release
			return nil
		})
	}()
	select {case <-started:case <-time.After(3*time.Second):t.Fatal("mutating lease never started")}
	readDone:=make(chan error,1)
	go func(){readDone<-supervisor.With(func(*BrainRuntime) error{return nil})}()
	select {
	case err:= <-readDone: t.Fatalf("concurrent read overtook mutating lease: %v",err)
	case <-time.After(25*time.Millisecond):
	}
	close(release)
	if err:= <-mutationDone;err!=nil {t.Fatal(err)}
	select {
	case err:= <-readDone:if err!=nil {t.Fatal(err)}
	case <-time.After(3*time.Second): t.Fatal("read never resumed after mutation")
	}
}

func TestSupervisorRejectsReadsMutationsAndStreamsOnPoisonedRuntime(t *testing.T) {
	rt:=NewBrainRuntime(nil)
	supervisor,err:=NewRuntimeSupervisor(rt)
	if err!=nil {t.Fatal(err)}
	rt.mu.Lock()
	rt.recoveryRequired=true
	rt.mu.Unlock()

	if err:=supervisor.With(func(*BrainRuntime) error{return nil});!errors.Is(err,ErrRuntimeRecoveryRequired) {
		t.Fatalf("managed read accepted poisoned runtime: %v",err)
	}
	if err:=supervisor.WithEpoch(func(*BrainRuntime,<-chan struct{}) error{return nil});!errors.Is(err,ErrRuntimeRecoveryRequired) {
		t.Fatalf("managed WebSocket access accepted poisoned runtime: %v",err)
	}
	if err:=supervisor.WithMutation(func(*BrainRuntime) error{return nil});!errors.Is(err,ErrRuntimeRecoveryRequired) {
		t.Fatalf("managed mutation accepted poisoned runtime: %v",err)
	}
}
