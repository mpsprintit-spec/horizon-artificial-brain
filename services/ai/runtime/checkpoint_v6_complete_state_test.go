package runtime

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/project-horizon/horizon-core/services/ai/knowledge"
)

func TestCheckpointV6PreservesCompleteNeuralStateAndSeparateAdapterSurfaces(t *testing.T) {
	rt:=NewBrainRuntime(knowledge.NewBootstrapBrain())
	at:=time.Date(2026,10,18,12,0,0,0,time.UTC)
	grounded,err:=rt.GroundObservationAt("surface-metadata-not-neural-identity","microphone","audio",at)
	if err!=nil {t.Fatal(err)}
	if grounded.NodeID==0 {t.Fatal("grounding returned zero node")}
	before,err:=rt.brain.CanonicalJSON()
	if err!=nil {t.Fatal(err)}
	if bytes.Contains(before,[]byte("surface-metadata-not-neural-identity")) {
		t.Fatal("canonical substrate leaked lexical adapter identity")
	}
	annotations:=rt.brain.SurfaceAnnotationsSnapshot()
	if len(annotations)==0 {t.Fatal("test grounding produced no adapter metadata")}
	checkpoint:=filepath.Join(t.TempDir(),"complete-checkpoint.json")
	if err:=rt.Checkpoint(checkpoint);err!=nil {t.Fatal(err)}
	snapshot,err:=readCheckpointForTest(checkpoint)
	if err!=nil {t.Fatal(err)}
	if snapshot.SchemaVersion!=6 {t.Fatalf("schema version %d",snapshot.SchemaVersion)}
	if len(snapshot.SurfaceAnnotations)!=len(annotations) {t.Fatal("checkpoint lost adapter surface annotations")}
	if !bytes.Contains(snapshot.Brain,[]byte("bootstrap_experiences")) || !bytes.Contains(snapshot.Brain,[]byte("episodes")) {
		t.Fatal("checkpoint omitted bootstrap experience and episodes")
	}
	if bytes.Contains(snapshot.Brain,[]byte("surface-metadata-not-neural-identity")) {
		t.Fatal("checkpoint canonical brain was polluted by lexical adapter identity")
	}
	restored:=NewBrainRuntime(knowledge.NewBootstrapBrain())
	if err:=restored.RestoreCheckpoint(checkpoint);err!=nil {t.Fatalf("restore complete checkpoint: %v",err)}
	after,err:=restored.brain.CanonicalJSON()
	if err!=nil {t.Fatal(err)}
	if !bytes.Equal(before,after) {t.Fatalf("canonical neural state changed across checkpoint: original=%d recovered=%d",len(before),len(after))}
	if !reflect.DeepEqual(annotations,restored.brain.SurfaceAnnotationsSnapshot()) {
		t.Fatal("separate adapter metadata was not restored exactly")
	}
}

func TestCheckpointV6RejectsCorruptedAdapterMetadata(t *testing.T) {
	rt:=NewBrainRuntime(nil)
	at:=time.Date(2026,10,18,12,0,0,0,time.UTC)
	if _,err:=rt.GroundObservationAt("visual-shape","camera","vision",at);err!=nil {t.Fatal(err)}
	path:=filepath.Join(t.TempDir(),"checkpoint.json")
	if err:=rt.Checkpoint(path);err!=nil {t.Fatal(err)}
	snapshot,err:=readCheckpointForTest(path)
	if err!=nil {t.Fatal(err)}
	if len(snapshot.SurfaceAnnotations)==0 {t.Fatal("test lacks adapter metadata")}
	snapshot.SurfaceAnnotations[0].Surface="attacker-relabelled"
	data,err:=json.Marshal(snapshot)
	if err!=nil {t.Fatal(err)}
	if err:=os.WriteFile(path,data,0600);err!=nil {t.Fatal(err)}
	restored:=NewBrainRuntime(nil)
	if err:=restored.RestoreCheckpoint(path);err==nil || !strings.Contains(err.Error(),"checksum mismatch") {
		t.Fatalf("tampered adapter annotation was accepted: %v",err)
	}
}

func TestCheckpointV5ReadCompatibilityButNoOffsetReplay(t *testing.T) {
	log:=openKnowledgeJournal(t)
	r:=NewBrainRuntime(knowledge.NewBootstrapBrain())
	r.SetEventLog(log)
	at:=time.Date(2026,10,18,12,0,0,0,time.UTC)
	if _,_,err:=r.Process(Event{ID:"prefix",Cycles:1,Timestamp:at});err!=nil {t.Fatal(err)}
	path:=filepath.Join(t.TempDir(),"checkpoint.json")
	if err:=r.Checkpoint(path);err!=nil {t.Fatal(err)}
	snapshot,err:=readCheckpointForTest(path)
	if err!=nil {t.Fatal(err)}
	snapshot.SchemaVersion=priorBrainSnapshotSchemaVersion
	snapshot.SurfaceAnnotations=nil
	snapshot.Checksum,err=checkpointChecksumV5(snapshot)
	if err!=nil {t.Fatal(err)}
	data,err:=json.Marshal(snapshot)
	if err!=nil {t.Fatal(err)}
	if err:=os.WriteFile(path,data,0600);err!=nil {t.Fatal(err)}
	restored:=NewBrainRuntime(nil)
	if err:=restored.RestoreCheckpoint(path);err!=nil {t.Fatalf("historical v5 snapshot lost compatibility: %v",err)}
	if _,err:=RebuildRuntimeFromCheckpointAndJournal(log,path);err==nil || !strings.Contains(err.Error(),"schema") {
		t.Fatalf("historical v5 used for unsafe offset recovery: %v",err)
	}
	snapshot.SurfaceAnnotations=[]knowledge.SurfaceAnnotation{{Surface:"unauthenticated",NodeID:1}}
	data,err=json.Marshal(snapshot)
	if err!=nil {t.Fatal(err)}
	if err:=os.WriteFile(path,data,0600);err!=nil {t.Fatal(err)}
	if err:=NewBrainRuntime(nil).RestoreCheckpoint(path);err==nil || !strings.Contains(err.Error(),"unauthenticated") {
		t.Fatalf("historical checkpoint accepted unattested annotations: %v",err)
	}
}
