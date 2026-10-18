package main

import (
	"bytes"
	"context"
	"errors"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/project-horizon/horizon-core/services/ai/runtime"
)

func TestCLICommandsAndPulseUseRecoveredActiveBrain(t *testing.T) {
	var output bytes.Buffer
	cli := newCLI(strings.NewReader(""), &output, filepath.Join(t.TempDir(),"brain_memory.json"))
	old := cli.horizon.Runtime
	log,err:=runtime.OpenEventLog(filepath.Join(t.TempDir(),"events.jsonl"))
	if err!=nil {t.Fatal(err)}
	defer log.Close()
	old.SetEventLog(log)

	first:=cli.process(context.Background(),"api menyebabkan panas")
	if strings.Contains(first.Answer,"Input tidak dapat diproses") {
		t.Fatalf("initial CLI cognition failed: %+v",first)
	}
	if old.LastSequence()!=1 {t.Fatalf("initial CLI sequence %d",old.LastSequence())}
	checkpoint:=filepath.Join(t.TempDir(),"checkpoint.json")
	if err:=old.Checkpoint(checkpoint);err!=nil {t.Fatalf("checkpoint CLI state: %v",err)}
	if err:=cli.horizon.Supervisor.RecoverFromCheckpoint(log,checkpoint);err!=nil {t.Fatalf("recover CLI runtime: %v",err)}
	if !old.RecoveryRequired() {t.Fatal("CLI old runtime is still writable")}
	if _,_,err:=old.Process(runtime.Event{ID:"stale-cli-process",Cycles:1,Timestamp:time.Now().UTC()});!errors.Is(err,runtime.ErrRuntimeRecoveryRequired) {
		t.Fatalf("retired runtime accepted direct process: %v",err)
	}

	second:=cli.process(context.Background(),"api menghasilkan cahaya")
	if strings.Contains(second.Answer,"Input tidak dapat diproses") {t.Fatalf("CLI followed stale orchestrator: %+v",second)}
	if err:=cli.horizon.Supervisor.With(func(active *runtime.BrainRuntime) error {
		if active==old {t.Fatal("CLI did not switch to new runtime")}
		if active.LastSequence()!=2 {t.Fatalf("CLI did not write into recovered runtime: %d",active.LastSequence())}
		return nil
	});err!=nil {t.Fatal(err)}

	for _,command:=range []string{"memory","nodes","synapses","save","load","history","debug on","debug off"} {
		if !cli.handleCommand(command) {t.Fatalf("command %q was not handled",command)}
	}
	if !strings.Contains(output.String(),"Memory saved.") || !strings.Contains(output.String(),"Load failed:") {
		t.Fatalf("managed CLI must save, but reject manual load while journal is attached: %s",output.String())
	}
	if err:=cli.horizon.Supervisor.With(func(active *runtime.BrainRuntime) error {
		if active.LastSequence()!=2 {t.Fatalf("rejected manual import changed the committed journal sequence: %d",active.LastSequence())}
		return nil
	});err!=nil {t.Fatal(err)}
}

func TestCLIBrainMonitorUsesActiveSupervisorAndExitsWhenRetired(t *testing.T) {
	var output bytes.Buffer
	cli:=newCLI(strings.NewReader(""),&output,filepath.Join(t.TempDir(),"memory.json"))
	old:=cli.horizon.Runtime
	log,err:=runtime.OpenEventLog(filepath.Join(t.TempDir(),"events.jsonl"))
	if err!=nil {t.Fatal(err)}
	defer log.Close()
	old.SetEventLog(log)
	checkpoint:=filepath.Join(t.TempDir(),"checkpoint.json")
	if err:=old.Checkpoint(checkpoint);err!=nil {t.Fatal(err)}
	if err:=cli.horizon.Supervisor.RecoverFromCheckpoint(log,checkpoint);err!=nil {t.Fatal(err)}

	cli.startBrainMonitor()
	defer cli.stopBrainMonitor()
	// The monitor has a 250ms interval. Wait for a committed tick from
	// the recovered runtime, with a bounded timeout.
	deadline:=time.Now().Add(3*time.Second)
	for {
		var sequence uint64
		if err:=cli.horizon.Supervisor.With(func(r *runtime.BrainRuntime) error {
			if r==old {t.Fatal("CLI monitor published old runtime")}
			sequence=r.LastSequence()
			return nil
		});err!=nil {t.Fatal(err)}
		if sequence>0 {break}
		if time.Now().After(deadline) {t.Fatal("managed CLI monitor never advanced recovered runtime")}
		time.Sleep(10*time.Millisecond)
	}
	if old.LastSequence()!=0 {t.Fatalf("CLI monitor wrote retired brain: %d",old.LastSequence())}
	cli.stopBrainMonitor()
	if !strings.Contains(output.String(),"[BRAIN] seq=") {t.Fatalf("no managed monitor event: %s",output.String())}
}

func TestCLICommandsFailClosedAfterSupervisorClose(t *testing.T) {
	var output bytes.Buffer
	cli:=newCLI(strings.NewReader(""),&output,filepath.Join(t.TempDir(),"memory.json"))
	if err:=cli.horizon.Supervisor.Close();err!=nil {t.Fatal(err)}
	if !cli.handleCommand("memory") {t.Fatal("closed CLI command should report error")}
	if !strings.Contains(output.String(),"Brain command unavailable") {t.Fatalf("closed supervisor error not surfaced: %s",output.String())}
	result:=cli.process(context.Background(),"air")
	if !strings.Contains(result.Answer,"Input tidak dapat diproses") {t.Fatalf("closed supervisor permitted CLI pulse: %+v",result)}
}
