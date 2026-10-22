package runtime

import (
	"bytes"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"sync"
	"testing"
	"time"

	"github.com/project-horizon/horizon-core/services/ai/knowledge"
)

func managedVector(t *testing.T) knowledge.NeuralVector {
	t.Helper()
	return knowledge.NewNeuralVector([]float64{0.2,-0.1,0.7,0.25})
}

func TestManagedGroundingFollowsCheckpointHandoverWithoutStaleWrites(t *testing.T) {
	log,err:=OpenEventLog(filepath.Join(t.TempDir(),"events.jsonl"))
	if err!=nil {t.Fatal(err)}
	defer log.Close()
	original:=NewBrainRuntime(knowledge.NewBootstrapBrain())
	legacy:=NewGroundingIntegration(original.brain)
	original.SetEventLog(log)
	supervisor,err:=NewRuntimeSupervisor(original)
	if err!=nil {t.Fatal(err)}
	managed,err:=NewManagedGroundingIntegration(supervisor)
	if err!=nil {t.Fatal(err)}
	at:=time.Date(2026,10,18,12,0,0,0,time.UTC)
	vector:=managedVector(t)
	first,err:=managed.GroundObservationAt(vector,at)
	if err!=nil {t.Fatalf("first managed numeric grounding: %v",err)}
	if first.NodeID==0 || len(first.Population)!=4 || first.Existing {
		t.Fatalf("unexpected first neural population: %+v",first)
	}
	if _,err:=legacy.GroundObservation(vector);!errors.Is(err,ErrManagedGroundingRequired) {
		t.Fatalf("legacy adapter bypassed supervisor ownership: %v",err)
	}
	checkpoint:=filepath.Join(t.TempDir(),"checkpoint.json")
	if err:=original.Checkpoint(checkpoint);err!=nil {t.Fatal(err)}
	if _,err:=managed.GroundObservationAt(vector,at.Add(time.Second));err!=nil {t.Fatal(err)}
	if original.LastSequence()!=2 {t.Fatalf("grounding did not advance revisions: %d",original.LastSequence())}
	if err:=supervisor.RecoverFromCheckpoint(log,checkpoint);err!=nil {
		t.Fatalf("numeric grounding checkpoint handover: %v",err)
	}
	if !original.RecoveryRequired() {t.Fatal("old runtime was not retired")}
	if _,err:=original.GroundVectorAt(vector,at.Add(2*time.Second));!errors.Is(err,ErrRuntimeRecoveryRequired) {
		t.Fatalf("old runtime accepted numeric grounding: %v",err)
	}
	if _,err:=legacy.GroundObservationAt(vector,at.Add(2*time.Second));!errors.Is(err,ErrManagedGroundingRequired) {
		t.Fatalf("legacy adapter reached retired brain: %v",err)
	}
	third,err:=managed.GroundObservationAt(vector,at.Add(2*time.Second))
	if err!=nil {t.Fatalf("managed adapter kept stale runtime: %v",err)}
	if !third.Existing || third.NodeID!=first.NodeID || !reflect.DeepEqual(third.Population,first.Population) {
		t.Fatalf("recovered grounding lost distributed population: first=%+v third=%+v",first,third)
	}
	var newRuntime *BrainRuntime
	if err:=supervisor.With(func(rt *BrainRuntime) error {
		if rt==original || rt.brain==original.brain {t.Fatal("supervisor published original brain after handover")}
		newRuntime=rt
		if rt.LastSequence()!=3 {t.Fatalf("new runtime sequence %d",rt.LastSequence())}
		return nil
	});err!=nil {t.Fatal(err)}
	events,err:=ReadEventLog(log.file.Name())
	if err!=nil {t.Fatal(err)}
	if len(events)!=3 {t.Fatalf("journal contains %d records, want 3",len(events))}
	for i,event:=range events {
		if event.Type!=EventTypeVectorGround || event.Sequence!=uint64(i+1) || event.Vector==nil {
			t.Fatalf("grounding event %d malformed: %+v",i,event)
		}
	}
	rewound,err:=RebuildRuntimeFromCommittedJournal(log,knowledge.NewBootstrapBrain)
	if err!=nil {t.Fatal(err)}
	expected,err:=newRuntime.brain.CanonicalJSON()
	if err!=nil {t.Fatal(err)}
	actual,err:=rewound.brain.CanonicalJSON()
	if err!=nil {t.Fatal(err)}
	if !bytes.Equal(expected,actual) || rewound.LastSequence()!=3 {
		t.Fatal("genesis replay changed canonical numeric grounding state")
	}
	if err:=supervisor.Close();err!=nil {t.Fatal(err)}
	if _,err:=managed.GroundObservation(vector);!errors.Is(err,ErrRuntimeSupervisorClosed) {
		t.Fatalf("closed supervisor accepted grounding: %v",err)
	}
}

func TestManagedGroundingConcurrentInputIsSerializable(t *testing.T) {
	log,err:=OpenEventLog(filepath.Join(t.TempDir(),"events.jsonl"))
	if err!=nil {t.Fatal(err)}
	defer log.Close()
	original:=NewBrainRuntime(nil)
	original.SetEventLog(log)
	sup,err:=NewRuntimeSupervisor(original)
	if err!=nil {t.Fatal(err)}
	adapter,err:=NewManagedGroundingIntegration(sup)
	if err!=nil {t.Fatal(err)}
	at:=time.Date(2026,10,18,12,0,0,0,time.UTC)
	const workers=12
	var wg sync.WaitGroup
	failures:=make(chan error,workers)
	for i:=0;i<workers;i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			vec:=knowledge.NewNeuralVector([]float64{0.2+float64(i)/100,-0.2,0.9})
			_,err:=adapter.GroundObservationAt(vec,at.Add(time.Duration(i)*time.Second))
			if err!=nil {failures<-err}
		}(i)
	}
	wg.Wait()
	close(failures)
	for err:=range failures {t.Fatal(err)}
	if original.LastSequence()!=workers {t.Fatalf("sequence %d != workers %d",original.LastSequence(),workers)}
	events,err:=ReadEventLog(log.file.Name())
	if err!=nil || len(events)!=workers {t.Fatalf("journal count=%d err=%v",len(events),err)}
}

func TestManagedGroundingAppendFailureIsFailClosed(t *testing.T) {
	log,err:=OpenEventLog(filepath.Join(t.TempDir(),"events.jsonl"))
	if err!=nil {t.Fatal(err)}
	defer log.Close()
	rt:=NewBrainRuntime(nil)
	rt.SetEventLog(log)
	sup,err:=NewRuntimeSupervisor(rt)
	if err!=nil {t.Fatal(err)}
	adapter,err:=NewManagedGroundingIntegration(sup)
	if err!=nil {t.Fatal(err)}
	before,err:=rt.brain.CanonicalJSON()
	if err!=nil {t.Fatal(err)}
	log.mu.Lock()
	log.beforeAppend=func(e LoggedEvent) error {
		if e.Type==EventTypeVectorGround {return errors.New("inject numeric grounding write failure")}
		return nil
	}
	log.mu.Unlock()
	if _,err:=adapter.GroundObservationAt(managedVector(t),time.Date(2026,10,18,12,0,0,0,time.UTC));!errors.Is(err,ErrRuntimeRecoveryRequired) {
		t.Fatalf("append uncertainty did not poison runtime: %v",err)
	}
	after,err:=rt.brain.CanonicalJSON()
	if err!=nil {t.Fatal(err)}
	if !bytes.Equal(before,after) || rt.LastSequence()!=0 {t.Fatal("failed numeric grounding mutated canonical state")}
	if _,err:=adapter.GroundObservation(managedVector(t));!errors.Is(err,ErrRuntimeRecoveryRequired) {t.Fatalf("poisoned adapter accepted data: %v",err)}
}

func TestNumericJournalRejectsMalformedAndMixedGrounding(t *testing.T) {
	path:=filepath.Join(t.TempDir(),"events.jsonl")
	at:=time.Date(2026,10,18,12,0,0,0,time.UTC)
	vector:=managedVector(t)
	good:=LoggedEvent{SchemaVersion:EventLogSchemaVersion,BrainIdentity:BrainIdentity,Sequence:1,Type:EventTypeVectorGround,Timestamp:at,Vector:&vector}
	tests:=[]struct{name string;tamper func(*LoggedEvent)}{
		{"no-vector",func(e *LoggedEvent){e.Vector=nil}},
		{"zero-timestamp",func(e *LoggedEvent){e.Timestamp=time.Time{}}},
		{"non-finite",func(e *LoggedEvent){e.Vector=&knowledge.NeuralVector{Values:[]float64{2}}}},
		{"process-smuggling",func(e *LoggedEvent){e.Event=&Event{ID:"smuggle",Timestamp:at}}},
		{"knowledge-smuggling",func(e *LoggedEvent){e.KnowledgeDocument=&knowledge.KnowledgeDocument{ID:"smuggle"}}},
		{"intent-smuggling",func(e *LoggedEvent){e.IntentHash="not-allowed"}},
	}
	for _,tt:=range tests {
		t.Run(tt.name,func(t *testing.T) {
			event:=good
			tt.tamper(&event)
			data,err:=json.Marshal(event)
			if err!=nil {t.Fatal(err)}
			if err:=os.WriteFile(path,append(data,'\n'),0600);err!=nil {t.Fatal(err)}
			if _,err:=ReadEventLog(path);err==nil {t.Fatal("invalid vector journal event accepted")}
		})
	}
}

func TestManagedGroundingConstructorValidatesSupervisor(t *testing.T) {
	if _,err:=NewManagedGroundingIntegration(nil);err==nil {t.Fatal("nil supervisor accepted")}
	sup,err:=NewRuntimeSupervisor(NewBrainRuntime(nil))
	if err!=nil {t.Fatal(err)}
	if err:=sup.Close();err!=nil {t.Fatal(err)}
	if _,err:=NewManagedGroundingIntegration(sup);!errors.Is(err,ErrRuntimeSupervisorClosed) {
		t.Fatalf("closed supervisor was accepted: %v",err)
	}
	if _,err:=NewGroundingIntegration(nil).GroundObservation(managedVector(t));!errors.Is(err,ErrGroundingIntegrationNotInitialized) {
		t.Fatalf("nil standalone brain error: %v",err)
	}
}
