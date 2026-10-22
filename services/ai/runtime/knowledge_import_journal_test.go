package runtime

import (
	"bytes"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/project-horizon/horizon-core/services/ai/knowledge"
)

func journalKnowledgeDocument() knowledge.KnowledgeDocument {
	return knowledge.KnowledgeDocument{
		ID: "journalled-developmental-experience",
		Domain: "physics",
		Source: "controlled-sensor-document",
		SourceType: "test_input",
		Status: "unverified_hypothesis",
		Experiences: []knowledge.KnowledgeExperienceInput{
			{ID: "motion-transition",
				Before: []knowledge.KnowledgeDatum{{Value:"object at rest", Modality:"vision", Confidence:0.4}},
				Changes: []knowledge.KnowledgeDatum{{Value:"object begins moving", Modality:"vision", Confidence:0.4}},
				After: []knowledge.KnowledgeDatum{{Value:"object moves", Modality:"vision", Confidence:0.4}},
				Confidence: 0.4,
			},
			{ID: "second-transition",
				Before: []knowledge.KnowledgeDatum{{Value:"object moves", Modality:"vision", Confidence:0.4}},
				Changes: []knowledge.KnowledgeDatum{{Value:"object slows", Modality:"vision", Confidence:0.4}},
				After: []knowledge.KnowledgeDatum{{Value:"object at rest", Modality:"vision", Confidence:0.4}},
				Confidence: 0.4,
			},
		},
	}
}

func openKnowledgeJournal(t *testing.T) *EventLog {
	t.Helper()
	log,err:=OpenEventLog(filepath.Join(t.TempDir(),"knowledge-events.jsonl"))
	if err!=nil {t.Fatal(err)}
	t.Cleanup(func(){_ = log.Close()})
	return log
}

func assertRecoveredNeuralState(t *testing.T, before, after *BrainRuntime) {
	t.Helper()
	orig,err:=before.brain.CanonicalJSON()
	if err!=nil {t.Fatal(err)}
	got,err:=after.brain.CanonicalJSON()
	if err!=nil {t.Fatal(err)}
	if !bytes.Equal(orig,got) {
		t.Fatalf("replayed canonical neural substrate differs from accepted knowledge import: original=%d replayed=%d bytes, first-difference=%d, expected=%q, actual=%q",len(orig),len(got), firstJSONDifference(orig,got), diagnosticFragment(orig,firstJSONDifference(orig,got)), diagnosticFragment(got,firstJSONDifference(orig,got)))
	}
	if before.LastSequence()!=after.LastSequence() {
		t.Fatalf("replayed runtime sequence differs: live %d recovered %d",before.LastSequence(),after.LastSequence())
	}
	if !reflect.DeepEqual(before.lastCognitiveState,after.lastCognitiveState) {
		t.Fatal("replay did not reproduce last cognitive state")
	}
	if !reflect.DeepEqual(before.lastObservationPopulation,after.lastObservationPopulation) ||
		!before.lastObservationAt.Equal(after.lastObservationAt) {
		t.Fatal("replay did not reproduce knowledge temporal predecessor")
	}
	if !reflect.DeepEqual(before.processedEvents,after.processedEvents) {
		t.Fatal("replay did not reproduce processed event index")
	}
}

func TestKnowledgeImportOneDurableRevisionReplaysFullNeuralExperience(t *testing.T) {
	at:=time.Date(2026,10,18,8,0,0,0,time.UTC)
	log:=openKnowledgeJournal(t)
	live:=NewBrainRuntime(knowledge.NewBootstrapBrain())
	live.SetEventLog(log)
	document:=journalKnowledgeDocument()
	result,err:=live.ProcessKnowledgeDocument(document,at)
	if err!=nil {t.Fatalf("knowledge import: %v",err)}
	if len(result.Outputs)!=6 {t.Fatalf("knowledge import emitted %d exposure outputs, want 6",len(result.Outputs))}
	for _, output:=range result.Outputs {
		if output.Sequence!=1 {t.Fatalf("microstep leaked uncommitted revision %d",output.Sequence)}
		if len(output.Activations)==0 {t.Fatal("knowledge microstep had no activations")}
	}
	if live.LastSequence()!=1 {t.Fatalf("document expanded sequence to %d; one imported document must be one committed revision",live.LastSequence())}
	if len(live.brain.BrainState.DevelopmentalExperiences)<2 {t.Fatal("knowledge provenance not written to canonical brain")}
	for _, exp:=range live.brain.BrainState.DevelopmentalExperiences {
		if exp.ID=="motion-transition" && exp.Provenance.SourceHash!=result.Compilation.SourceHash {
			t.Fatal("source provenance hash changed during import")
		}
	}
	events,err:=ReadEventLog(log.file.Name())
	if err!=nil {t.Fatal(err)}
	if len(events)!=1 || events[0].Sequence!=1 || events[0].Type!=EventTypeKnowledgeImport ||
		events[0].KnowledgeDocument==nil || events[0].KnowledgeDocument.ID!=document.ID {
		t.Fatalf("committed journal did not preserve whole knowledge document: %+v",events)
	}
	if _,_,err:=live.Process(Event{ID:"after-knowledge",Stimulus:[]string{"object"},Cycles:1,Timestamp:at.Add(time.Second)});err!=nil {t.Fatalf("append after import: %v",err)}
	if live.LastSequence()!=2 {t.Fatalf("event after document used wrong sequence %d",live.LastSequence())}
	events,err=ReadEventLog(log.file.Name())
	if err!=nil {t.Fatal(err)}
	if len(events)!=2 || events[1].Sequence!=2 {t.Fatalf("journal discontinuity after import: %+v",events)}
	recovered,err:=RebuildRuntimeFromCommittedJournal(log,knowledge.NewBootstrapBrain)
	if err!=nil {t.Fatalf("replay committed import: %v",err)}
	assertRecoveredNeuralState(t,live,recovered)
	if recovered.eventLog!=nil {t.Fatal("rebuild published runtime with attached writer")}
}

func TestKnowledgeImportCheckpointSuffixRecoveryAndContinuation(t *testing.T) {
	at:=time.Date(2026,10,18,8,15,0,0,time.UTC)
	log:=openKnowledgeJournal(t)
	live:=NewBrainRuntime(knowledge.NewBootstrapBrain())
	live.SetEventLog(log)
	if _,_,err:=live.Process(Event{ID:"checkpoint-prefix",Stimulus:[]string{"object"},Cycles:1,Timestamp:at});err!=nil {t.Fatal(err)}
	cp:=filepath.Join(t.TempDir(),"checkpoint.json")
	if err:=live.Checkpoint(cp);err!=nil {t.Fatal(err)}
	if _,err:=live.ProcessKnowledgeDocument(journalKnowledgeDocument(),at.Add(time.Second));err!=nil {t.Fatal(err)}
	if _,_,err:=live.ThinkAt(1,at.Add(2*time.Second));err!=nil {t.Fatal(err)}
	if live.LastSequence()!=3 {t.Fatalf("sequence after knowledge import and thought %d",live.LastSequence())}

	fresh,err:=RebuildRuntimeFromCheckpointAndJournal(log,cp)
	if err!=nil {t.Fatalf("checkpoint offset replay with knowledge import: %v",err)}
	assertRecoveredNeuralState(t,live,fresh)
	if fresh.eventLog!=nil {t.Fatal("checkpoint rebuild attached active writer")}
	// The replayed knowledge import is now part of the verified journal prefix
	// when a later checkpoint is written by the active runtime.
	nextCheckpoint:=filepath.Join(t.TempDir(),"after-import-checkpoint.json")
	if err:=live.Checkpoint(nextCheckpoint);err!=nil {t.Fatalf("checkpoint after import: %v",err)}
	if _,_,err:=live.Process(Event{ID:"after-import-checkpoint",Cycles:1,Timestamp:at.Add(3*time.Second)});err!=nil {t.Fatal(err)}
	fresh,err=RebuildRuntimeFromCheckpointAndJournal(log,nextCheckpoint)
	if err!=nil {t.Fatalf("replay after an anchored knowledge import: %v",err)}
	assertRecoveredNeuralState(t,live,fresh)
}

func TestKnowledgeImportAppendFailurePoisonsWithoutNeuralMutation(t *testing.T) {
	at:=time.Date(2026,10,18,9,0,0,0,time.UTC)
	log:=openKnowledgeJournal(t)
	rt:=NewBrainRuntime(knowledge.NewBootstrapBrain())
	rt.SetEventLog(log)
	before,err:=rt.brain.CanonicalJSON()
	if err!=nil {t.Fatal(err)}
	log.mu.Lock()
	log.beforeAppend=func(event LoggedEvent) error {
		if event.Type==EventTypeKnowledgeImport {return errors.New("simulated fsync/append failure")}
		return nil
	}
	log.mu.Unlock()
	_,err=rt.ProcessKnowledgeDocument(journalKnowledgeDocument(),at)
	if !errors.Is(err,ErrRuntimeRecoveryRequired) {t.Fatalf("append failure did not poison runtime: %v",err)}
	after,err:=rt.brain.CanonicalJSON()
	if err!=nil {t.Fatal(err)}
	if !bytes.Equal(before,after) || rt.LastSequence()!=0 {
		t.Fatal("failed WAL append changed neural state or sequence")
	}
	if _,err:=rt.ProcessKnowledgeDocument(journalKnowledgeDocument(),at);!errors.Is(err,ErrRuntimeRecoveryRequired) {
		t.Fatalf("poisoned runtime accepted another knowledge import: %v",err)
	}
	events,err:=ReadEventLog(log.file.Name())
	if err!=nil || len(events)!=0 {t.Fatalf("fault injection unexpectedly changed log: count=%d err=%v",len(events),err)}
	recovered,err:=RebuildRuntimeFromCommittedJournal(log,knowledge.NewBootstrapBrain)
	if err!=nil {t.Fatalf("clean committed prefix unrecoverable: %v",err)}
	if recovered.RecoveryRequired() || recovered.LastSequence()!=0 {t.Fatal("recovery replayed uncommitted document")}
}

func TestKnowledgeImportInvalidDocumentFailsBeforeWAL(t *testing.T) {
	log:=openKnowledgeJournal(t)
	rt:=NewBrainRuntime(nil)
	rt.SetEventLog(log)
	invalid:=journalKnowledgeDocument()
	invalid.Experiences[0].Before[0].Confidence=42
	before,err:=rt.brain.CanonicalJSON()
	if err!=nil {t.Fatal(err)}
	if _,err:=rt.ProcessKnowledgeDocument(invalid,time.Date(2026,10,18,9,0,0,0,time.UTC));err==nil {
		t.Fatal("invalid knowledge document accepted")
	}
	if rt.RecoveryRequired() || rt.LastSequence()!=0 {t.Fatal("preflight validation poisoned or advanced an unmodified runtime")}
	after,err:=rt.brain.CanonicalJSON()
	if err!=nil {t.Fatal(err)}
	if !bytes.Equal(before,after) {t.Fatal("invalid document mutated brain")}
	events,err:=ReadEventLog(log.file.Name())
	if err!=nil || len(events)!=0 {t.Fatalf("invalid document altered journal: %d events, err=%v",len(events),err)}
}

func TestKnowledgeImportRejectsMalformedAndMixedJournalPayload(t *testing.T) {
	dir:=t.TempDir()
	path:=filepath.Join(dir,"invalid.jsonl")
	at:=time.Date(2026,10,18,9,0,0,0,time.UTC)
	good:=LoggedEvent{SchemaVersion:EventLogSchemaVersion,BrainIdentity:BrainIdentity,Sequence:1,Type:EventTypeKnowledgeImport,Timestamp:at,KnowledgeDocument:func() *knowledge.KnowledgeDocument {d:=journalKnowledgeDocument();return &d}()}
	cases:=[]struct{
		name string
		mutate func(*LoggedEvent)
	}{
		{"missing-document",func(e *LoggedEvent){e.KnowledgeDocument=nil}},
		{"zero-time",func(e *LoggedEvent){e.Timestamp=time.Time{}}},
		{"mixed-process",func(e *LoggedEvent){e.Event=&Event{ID:"smuggled",Cycles:1,Timestamp:at}}},
		{"invalid-document",func(e *LoggedEvent){d:=*e.KnowledgeDocument;d.Source="";e.KnowledgeDocument=&d}},
		{"mixed-intent-hash",func(e *LoggedEvent){e.IntentHash="unexpected"}},
	}
	for _,tc:=range cases {
		t.Run(tc.name,func(t *testing.T){
			event:=good
			tc.mutate(&event)
			data,err:=json.Marshal(event)
			if err!=nil {t.Fatal(err)}
			if err:=os.WriteFile(path,append(data,'\n'),0600);err!=nil {t.Fatal(err)}
			if _,err:=ReadEventLog(path);err==nil {t.Fatal("invalid knowledge event accepted by strict journal reader")}
		})
	}
	// A schema-valid knowledge document must retain enough data for replay.
	if strings.TrimSpace(good.KnowledgeDocument.ID)=="" {t.Fatal("invalid fixture")}
}

func TestKnowledgeImportReplayRejectsUnknownDocumentSequence(t *testing.T) {
	rt:=NewBrainRuntime(nil)
	at:=time.Date(2026,10,18,9,0,0,0,time.UTC)
	d:=journalKnowledgeDocument()
	events:=[]LoggedEvent{{SchemaVersion:EventLogSchemaVersion,BrainIdentity:BrainIdentity,Sequence:2,Type:EventTypeKnowledgeImport,Timestamp:at,KnowledgeDocument:&d}}
	if err:=ReplayEventLog(rt,events);err==nil {t.Fatal("replay skipped missing knowledge revision")}
	if rt.LastSequence()!=0 {t.Fatal("invalid replay changed runtime sequence")}
}

func firstJSONDifference(a,b []byte) int {
	i:=0
	for i<len(a) && i<len(b) && a[i]==b[i] {i++}
	return i
}

func diagnosticFragment(data []byte, index int) string {
	lo:=index-40
	if lo<0 {lo=0}
	hi:=index+120
	if hi>len(data) {hi=len(data)}
	return string(data[lo:hi])
}
