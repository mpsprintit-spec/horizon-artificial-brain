package execution

import (
	"bytes"
	"encoding/json"
	"fmt"
	k "github.com/project-horizon/horizon-core/services/ai/knowledge"
	r "github.com/project-horizon/horizon-core/services/ai/runtime"
	"os"
	"path/filepath"
	"reflect"
	"testing"
	"time"
)

func TestGateGRecoveryNeverDispatchesPhysicalActions(t *testing.T) {
	at := time.Date(2026, 11, 6, 0, 0, 0, 0, time.UTC)
	rt := r.NewBrainRuntime(k.NewBrain())
	wal, e := r.OpenEventLog(filepath.Join(t.TempDir(), "events.jsonl"))
	if e != nil {
		t.Fatal(e)
	}
	defer wal.Close()
	rt.SetEventLog(wal)
	sensor := r.SensorObservation{Modality: "motion", Source: "mock-imu", SampleID: "sample", Timestamp: at, ChannelStates: []float64{.3, .7}, Confidence: .9, Provenance: r.SensorProvenance{Adapter: "mock-adapter", CalibrationID: "calibration"}}
	sensed, e := rt.ProcessSensorObservation(sensor)
	if e != nil {
		t.Fatal(e)
	}
	ids := sensed.GroundedRepresentations[0].Population
	cfg := k.NativeControlConfig{MaxActiveNodes: 4096, MaxSynapseReads: 100000, Version: k.NativeControlCausalVersion, Encoder: "local-relational-v1", Sources: ids, Operations: []k.NativeOperation{{Kind: "propagate", Gain: 1, Cost: .08}, {Kind: "relax", Gain: .1, Cost: .08}, {Kind: "focus", Gain: .5, Cost: .08}, {Kind: "stop"}}, Seed: 37, LearningRate: .2, Gamma: 1, Lambda: 1, Temperature: .15, MaxSteps: 8}
	apply := func(c r.NativeControlCommand) r.NativeControlResult {
		t.Helper()
		o, e := rt.ApplyNativeControl(c, at.Add(time.Duration(rt.LastSequence()+1)*time.Millisecond))
		if e != nil {
			t.Fatal(e)
		}
		return o
	}
	apply(r.NativeControlCommand{ID: "initialize", Initialize: &cfg})
	core := NewExecutionCore()
	mock := &gate8TestPlugin{}
	core.RegisterPlugin("action", mock)
	gateway := NewExecutionGateway(core)
	gateway.Now = func() time.Time { return at }
	request := buildLowRiskRequest(t, at, "observed-action")
	if e = rt.RegisterActionBinding(r.ActionBinding{RequestID: request.RequestID, TargetNodeIDs: ids, Intent: request.Intent}); e != nil {
		t.Fatal(e)
	}
	if e = gateway.Execute(request); e != nil {
		t.Fatal(e)
	}
	if e = gateway.Execute(request); e == nil || mock.Count() != 1 {
		t.Fatal("physical dispatch was duplicated")
	}
	apply(r.NativeControlCommand{ID: "begin", Begin: &r.NativeBegin{EpisodeID: "ep", Training: true}})
	var result r.NativeControlResult
	for j := 0; j < 8; j++ {
		result = apply(r.NativeControlCommand{ID: fmt.Sprintf("step%d", j), Step: &r.NativeStep{EpisodeID: "ep", Sample: .01}})
		if result.Stopped {
			break
		}
	}
	apply(r.NativeControlCommand{ID: "outcome", Outcome: &r.NativeOutcome{EpisodeID: "ep", Source: "mock-sensor", OutcomeID: "observed-consequence", TraceDigest: result.TraceDigest, Provenance: "observed-sensor", Reward: .8, Learn: true}})
	cp := filepath.Join(t.TempDir(), "before-outcome.json")
	if e = rt.Checkpoint(cp); e != nil {
		t.Fatal(e)
	}
	if _, e = rt.ObserveOutcome(r.OutcomeEvent{RequestID: request.RequestID, Success: true, Source: "mock-sensor", Reliability: .9, Observation: []string{"observed-consequence"}, ObservedAt: at.Add(time.Second)}); e != nil {
		t.Fatal(e)
	}
	dispatches := mock.Count()
	genesis, e := r.RebuildRuntimeFromCommittedJournal(wal, k.NewBrain)
	if e != nil {
		t.Fatal(e)
	}
	suffix, e := r.RebuildRuntimeFromCheckpointAndJournal(wal, cp)
	if e != nil {
		t.Fatal(e)
	}
	for _, rebuilt := range []*r.BrainRuntime{genesis, suffix} {
		for _, target := range []*r.BrainRuntime{rt, rebuilt} {
			target.SetClock(r.NewFixedClock(at.Add(2 * time.Second)))
		}
		a, b := filepath.Join(t.TempDir(), "live.json"), filepath.Join(t.TempDir(), "recovered.json")
		if e = rt.Checkpoint(a); e != nil {
			t.Fatal(e)
		}
		if e = rebuilt.Checkpoint(b); e != nil {
			t.Fatal(e)
		}
		read := func(p string) r.BrainSnapshot {
			raw, e := os.ReadFile(p)
			if e != nil {
				t.Fatal(e)
			}
			var s r.BrainSnapshot
			if e = json.Unmarshal(raw, &s); e != nil {
				t.Fatal(e)
			}
			return s
		}
		live, recovered := read(a), read(b)
		if !bytes.Equal(live.Brain, recovered.Brain) || !reflect.DeepEqual(live.Activation, recovered.Activation) || live.Sequence != recovered.Sequence || !reflect.DeepEqual(live.ProcessedEvents, recovered.ProcessedEvents) {
			t.Fatal("physical feedback recovery state mismatch")
		}
	}
	if mock.Count()-dispatches != 0 {
		t.Fatal("recovery dispatched external actions")
	}
	if e = gateway.Execute(r.ExecutionRequest{RequestID: "native-choice", BrainIdentity: r.BrainIdentity, Intent: "action", Expiry: at.Add(time.Minute)}); e == nil {
		t.Fatal("internal decision bypassed authorization")
	}
	if mock.Count() != 1 {
		t.Fatal("unauthorized native decision caused physical dispatch")
	}
	t.Logf("live_dispatches=%d new_recovery_dispatches=%d", dispatches, mock.Count()-dispatches)
}
