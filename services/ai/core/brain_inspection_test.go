package core

import (
	"bytes"
	"errors"
	"path/filepath"
	"testing"

	"github.com/project-horizon/horizon-core/services/ai/runtime"
)

func TestActiveReadSnapshotFollowsHandoverAndOldSnapshotRemainsStable(t *testing.T) {
	h:=NewHorizonEngine()
	if h==nil || h.Supervisor==nil {t.Fatal("canonical supervised brain not initialized")}
	log,err:=runtime.OpenEventLog(filepath.Join(t.TempDir(),"events.jsonl"))
	if err!=nil {t.Fatal(err)}
	defer log.Close()
	legacy:=h.Runtime
	legacy.SetEventLog(log)
	if _,err:=h.Pulse("cahaya dan foton");err!=nil {t.Fatal(err)}
	before,err:=h.InspectActiveBrain()
	if err!=nil {t.Fatal(err)}
	if before.Revision!=1 {t.Fatalf("unexpected initial sequence %d",before.Revision)}
	neuralBefore:=before.Graph.CanonicalBytes()
	if len(neuralBefore)==0 {t.Fatal("empty neural snapshot")}
	checkpoint:=filepath.Join(t.TempDir(),"checkpoint.json")
	if err:=legacy.Checkpoint(checkpoint);err!=nil {t.Fatal(err)}
	if err:=h.Supervisor.RecoverFromCheckpoint(log,checkpoint);err!=nil {t.Fatal(err)}
	after,err:=h.InspectActiveBrain()
	if err!=nil {t.Fatal(err)}
	if after.Revision!=1 || !bytes.Equal(neuralBefore,after.Graph.CanonicalBytes()) {
		t.Fatalf("checkpoint handover changed read-only substrate at revision %d",after.Revision)
	}
	_,labels,err:=h.PulseWithLabels("pola baru")
	if err!=nil || len(labels)==0 {t.Fatalf("supervised ranked labels unavailable: %v labels=%v",err,labels)}
	continued,err:=h.InspectActiveBrain()
	if err!=nil {t.Fatal(err)}
	if continued.Revision!=2 || bytes.Equal(continued.Graph.CanonicalBytes(),neuralBefore) {
		t.Fatal("new runtime did not progress after checkpoint handover")
	}
	if !bytes.Equal(before.Graph.CanonicalBytes(),neuralBefore) {
		t.Fatal("previous neural inspection snapshot changed during handover")
	}
	if err:=h.Supervisor.Close();err!=nil {t.Fatal(err)}
	if _,err:=h.InspectActiveBrain();!errors.Is(err,runtime.ErrRuntimeSupervisorClosed) {
		t.Fatalf("retired engine exposed neural snapshot: %v",err)
	}
}

func TestReadSnapshotAccessDoesNotAdvanceNeuralRuntime(t *testing.T) {
	h:=NewHorizonEngine()
	first,err:=h.InspectActiveBrain()
	if err!=nil {t.Fatal(err)}
	neural:=first.Graph.CanonicalBytes()
	for i:=0;i<3;i++ {
		s,err:=h.InspectActiveBrain()
		if err!=nil {t.Fatal(err)}
		if s.Revision!=first.Revision || !bytes.Equal(s.Graph.CanonicalBytes(),neural) {
			t.Fatal("read-only inspection mutated recurrent or canonical neural state")
		}
	}
}
