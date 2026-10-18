package main

import (
	"bytes"
	"path/filepath"
	"strings"
	"testing"

	"github.com/project-horizon/horizon-core/services/ai/runtime"
)

func TestCLIInspectionCommandsUseDetachedCurrentBrainAfterHandover(t *testing.T) {
	var output bytes.Buffer
	app:=newCLI(strings.NewReader(""),&output,filepath.Join(t.TempDir(),"brain.json"))
	initial:=app.horizon.Runtime
	log,err:=runtime.OpenEventLog(filepath.Join(t.TempDir(),"events.jsonl"))
	if err!=nil {t.Fatal(err)}
	defer log.Close()
	initial.SetEventLog(log)
	before,err:=app.horizon.InspectActiveBrain()
	if err!=nil {t.Fatal(err)}
	cp:=filepath.Join(t.TempDir(),"checkpoint.json")
	if err:=initial.Checkpoint(cp);err!=nil {t.Fatal(err)}
	if err:=app.horizon.Supervisor.RecoverFromCheckpoint(log,cp);err!=nil {t.Fatal(err)}
	// All queries, including the more expensive statistics and clustering,
	// must work without passing a mutable *knowledge.Brain to the CLI.
	for _,cmd:=range []string{
		"memory","nodes","synapses","clusters","patterns",
		"stats nothing","relations nothing","name 1 pending",
		"activation","context","history","help","debug on","debug off",
	} {
		if !app.handleCommand(cmd) {t.Fatalf("inspection command %q was rejected",cmd)}
	}
	text:=output.String()
	if !strings.Contains(text,"Memory:") ||
		!strings.Contains(text,"Nodes (") ||
		!strings.Contains(text,"Synapses:") ||
		!strings.Contains(text,"Node \"nothing\" belum ada.") {
		t.Fatalf("inspection output missing expected detached views: %s",text)
	}
	after,err:=app.horizon.InspectActiveBrain()
	if err!=nil {t.Fatal(err)}
	if before.Revision!=after.Revision || !bytes.Equal(before.Graph.CanonicalBytes(),after.Graph.CanonicalBytes()) {
		t.Fatal("CLI read-only inspection mutated neural state")
	}
	if !initial.RecoveryRequired() {t.Fatal("legacy CLI runtime remained writable")}
	if err:=app.horizon.Supervisor.Close();err!=nil {t.Fatal(err)}
	output.Reset()
	if !app.handleCommand("clusters") || !strings.Contains(output.String(),"Brain command unavailable:") {
		t.Fatalf("closed supervisor exposed brain to CLI inspector: %s",output.String())
	}
}

func TestCLISnapshotLabelsAndCountsProgressAfterLearning(t *testing.T) {
	var output bytes.Buffer
	app:=newCLI(strings.NewReader(""),&output,filepath.Join(t.TempDir(),"brain.json"))
	_,err:=app.horizon.Pulse("foton")
	if err!=nil {t.Fatal(err)}
	app.handleCommand("memory")
	first:=output.String()
	if !strings.Contains(first,"nodes") {t.Fatalf("missing neural counts: %s",first)}
	inspected,err:=app.horizon.InspectActiveBrain()
	if err!=nil {t.Fatal(err)}
	if inspected.Revision!=1 {t.Fatalf("unexpected revision %d",inspected.Revision)}
	if len(inspected.Graph.NeuralUnits())==0 {t.Fatal("empty detached neural result")}
}
