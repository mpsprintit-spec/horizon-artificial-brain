package runtime

import (
	"bytes"
	"errors"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/project-horizon/horizon-core/services/ai/knowledge"
)

func directMutationTestDocument() knowledge.KnowledgeDocument {
	return knowledge.KnowledgeDocument{
		ID: "retirement-test", Domain: "physics", Source: "controlled-test",
		SourceType: "test_input", Status: "initial_hypothesis",
		Experiences: []knowledge.KnowledgeExperienceInput{{
			ID: "before-after",
			Before: []knowledge.KnowledgeDatum{{Value: "body at height", Modality: "text", Confidence: 0.2}},
			Changes: []knowledge.KnowledgeDatum{{Value: "position changes", Modality: "text", Confidence: 0.2}},
			After: []knowledge.KnowledgeDatum{{Value: "body lower", Modality: "text", Confidence: 0.2}},
			Confidence: 0.2,
		}},
	}
}

func TestSupervisorRetirementRejectsDirectNeuralMutators(t *testing.T) {
	rt := NewBrainRuntime(nil)
	if rt == nil { t.Fatal("brain runtime missing") }
	node := rt.brain.Store("retirement-target")
	if node == nil { t.Fatal("test brain node missing") }
	now := time.Date(2026, 10, 18, 16, 30, 0, 0, time.UTC)
	supervisor, err := NewRuntimeSupervisor(rt)
	if err != nil { t.Fatal(err) }
	if err := supervisor.Close(); err != nil { t.Fatal(err) }

	before, err := rt.brain.CanonicalJSON()
	if err != nil { t.Fatal(err) }
	if _, err := rt.GroundObservation("retired-ground", "camera", "vision"); !errors.Is(err, ErrRuntimeRecoveryRequired) {
		t.Fatalf("retired GroundObservation accepted mutation: %v", err)
	}
	if _, err := rt.GroundObservationAt("retired-ground-at", "camera", "vision", now); !errors.Is(err, ErrRuntimeRecoveryRequired) {
		t.Fatalf("retired GroundObservationAt accepted mutation: %v", err)
	}
	if _, err := rt.ActivateBootstrap(now); !errors.Is(err, ErrRuntimeRecoveryRequired) {
		t.Fatalf("retired ActivateBootstrap accepted activation: %v", err)
	}
	if err := rt.RegisterActionBinding(ActionBinding{RequestID:"retired-binding", TargetNodeIDs:[]knowledge.NodeID{node.ID}}); !errors.Is(err, ErrRuntimeRecoveryRequired) {
		t.Fatalf("retired RegisterActionBinding accepted binding: %v", err)
	}
	if err := rt.RestoreEventIndex(nil); !errors.Is(err, ErrRuntimeRecoveryRequired) {
		t.Fatalf("retired RestoreEventIndex accepted sequence mutation: %v", err)
	}
	if _, err := rt.ProcessKnowledgeDocument(directMutationTestDocument(), now); !errors.Is(err, ErrRuntimeRecoveryRequired) {
		t.Fatalf("retired document import accepted neural mutation: %v", err)
	}
	agenda := InquiryAgenda{BrainIdentity:BrainIdentity, Sequence:1, CreatedAt:now, Selected:&InquiryEvaluation{Action:InquiryFocus, CandidateID:"focus"}}
	if _, err := rt.BuildInquiryRequest(agenda); !errors.Is(err, ErrRuntimeRecoveryRequired) {
		t.Fatalf("retired inquiry request accessed neural state: %v",err)
	}
	if _, err := rt.CaptureInquiryPrediction(InquiryExecution{}, now); !errors.Is(err, ErrRuntimeRecoveryRequired) {
		t.Fatalf("retired prediction captured activation: %v",err)
	}
	// This legacy setter has no error return; it must nevertheless never
	// reattach an event writer to a poisoned/retired runtime.
	log,err := OpenEventLog(filepath.Join(t.TempDir(),"events.jsonl"))
	if err != nil { t.Fatal(err) }
	defer log.Close()
	rt.SetEventLog(log)
	if rt.eventLog != nil { t.Fatal("retired runtime was reattached to journal writer") }
	if _, ok := rt.actions["retired-binding"]; ok { t.Fatal("retired runtime retained rejected binding") }
	if rt.LastSequence() != 0 { t.Fatal("retired runtime advanced sequence") }
	after,err := rt.brain.CanonicalJSON()
	if err != nil { t.Fatal(err) }
	if !bytes.Equal(before, after) { t.Fatal("retired runtime brain mutated through direct method") }
}

func TestRestoreEventIndexRejectsPartialInvalidIndexAndAttachedWriter(t *testing.T) {
	rt := NewBrainRuntime(nil)
	timestamp:=time.Date(2026,10,18,16,31,0,0,time.UTC)
	makeEvent:=func(seq uint64, id string) LoggedEvent {
		return LoggedEvent{SchemaVersion:EventLogSchemaVersion,BrainIdentity:BrainIdentity,Sequence:seq,Type:EventTypeProcess,Timestamp:timestamp,Event:&Event{ID:id,Cycles:1,Timestamp:timestamp}}
	}
	invalid:=[]LoggedEvent{makeEvent(1,"valid"),makeEvent(3,"gap")}
	if err:=rt.RestoreEventIndex(invalid);err==nil || !strings.Contains(err.Error(),"discontinuity") {
		t.Fatalf("sequence gap unexpectedly accepted: %v",err)
	}
	if rt.LastSequence()!=0 || len(rt.processedEvents)!=0 {
		t.Fatal("invalid event index partially changed runtime state")
	}
	duplicate:=[]LoggedEvent{makeEvent(1,"same"),makeEvent(2,"same")}
	if err:=rt.RestoreEventIndex(duplicate);err==nil || !strings.Contains(err.Error(),"duplicate") {
		t.Fatalf("duplicate event ID unexpectedly accepted: %v",err)
	}
	if rt.LastSequence()!=0 || len(rt.processedEvents)!=0 {
		t.Fatal("duplicate event index partially changed runtime")
	}
	log,err:=OpenEventLog(filepath.Join(t.TempDir(),"journal.jsonl"))
	if err!=nil {t.Fatal(err)}
	defer log.Close()
	rt.SetEventLog(log)
	if err:=rt.RestoreEventIndex([]LoggedEvent{makeEvent(1,"valid")});err==nil {
		t.Fatal("journal-attached index overwrite was accepted")
	}
	if rt.LastSequence()!=0 {t.Fatal("journal guard partially updated sequence")}
	rt.SetEventLog(nil)
	if err:=rt.RestoreEventIndex([]LoggedEvent{makeEvent(1,"valid")});err!=nil {
		t.Fatalf("standalone bootstrap index was rejected: %v",err)
	}
	if rt.LastSequence()!=1 || !rt.HasProcessedEvent("valid") {
		t.Fatal("valid standalone event index was not restored")
	}
	if err:=rt.RestoreEventIndex(nil);err==nil {
		t.Fatal("nonempty runtime accepted a second index restore")
	}
}

func TestJournalAttachedKnowledgeImportIsCommittedBeforeNeuralMutation(t *testing.T) {
	rt:=NewBrainRuntime(nil)
	log,err:=OpenEventLog(filepath.Join(t.TempDir(),"events.jsonl"))
	if err!=nil {t.Fatal(err)}
	defer log.Close()
	rt.SetEventLog(log)
	before,err:=rt.brain.CanonicalJSON()
	if err!=nil {t.Fatal(err)}
	now:=time.Date(2026,10,18,16,33,0,0,time.UTC)
	if _,err:=rt.ProcessKnowledgeDocument(directMutationTestDocument(),now);err!=nil {
		t.Fatalf("journalled knowledge import: %v",err)
	}
	after,err:=rt.brain.CanonicalJSON()
	if err!=nil {t.Fatal(err)}
	if bytes.Equal(before,after) || rt.LastSequence()!=1 {
		t.Fatal("committed knowledge import did not mutate neural state exactly once")
	}
	events,err:=ReadEventLog(log.file.Name())
	if err!=nil || len(events)!=1 || events[0].Type!=EventTypeKnowledgeImport {
		t.Fatalf("knowledge import was not captured as one durable revision: events=%d err=%v",len(events),err)
	}
}

func TestDirectGroundingAndCognitiveThinkSerializeRuntimeMutations(t *testing.T) {
	rt:=NewBrainRuntime(nil)
	const workers=8
	var wg sync.WaitGroup
	errs:=make(chan error,workers*2)
	at:=time.Date(2026,10,18,16,33,0,0,time.UTC)
	for i:=0;i<workers;i++ {
		wg.Add(2)
		go func(i int) {
			defer wg.Done()
			_,err:=rt.GroundObservationAt("vision object", "camera", "vision", at.Add(time.Duration(i)*time.Second))
			if err!=nil {errs<-err}
		}(i)
		go func() {
			defer wg.Done()
			_,err:=rt.CognitiveThink(1)
			if err!=nil {errs<-err}
		}()
	}
	wg.Wait()
	close(errs)
	for err:=range errs {t.Fatalf("concurrent direct mutation failed: %v",err)}
	if rt.LastSequence()!=workers {t.Fatalf("cognitive sequence=%d want=%d",rt.LastSequence(),workers)}
}
