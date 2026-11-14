package main

import (
	"bytes"
	"encoding/json"
	k "github.com/project-horizon/horizon-core/services/ai/knowledge"
	r "github.com/project-horizon/horizon-core/services/ai/runtime"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestGateGActualProductionStartupContinuesNativeEpisode(t *testing.T) {
	dir := t.TempDir()
	journal, memory := filepath.Join(dir, "events.jsonl"), filepath.Join(dir, "brain_memory.json")
	cp := memory + ".checkpoint.json"
	log := newStartupJournal(t, journal)
	first, bootstrap, e := initializeProductionHorizon(log, journal, memory, cp)
	if e != nil {
		t.Fatal(e)
	}
	if _, _, e = finishProductionStartup(first, bootstrap, memory, cp); e != nil {
		t.Fatal(e)
	}
	var stopped r.NativeControlResult
	e = first.Supervisor().WithMutation(func(rt *r.BrainRuntime) error {
		at := time.Date(2026, 11, 13, 0, 0, 0, 0, time.UTC)
		sensor := r.SensorObservation{Modality: "motion", Source: "production-mock-imu", SampleID: "before-restart", Timestamp: at, ChannelStates: []float64{.3, .7}, Confidence: .9, Provenance: r.SensorProvenance{Adapter: "production-mock", CalibrationID: "cal"}}
		out, e := rt.ProcessSensorObservation(sensor)
		if e != nil {
			return e
		}
		cfg := k.NativeControlConfig{MaxActiveNodes: 4096, MaxSynapseReads: 100000, Version: k.NativeControlCausalVersion, Encoder: "local-relational-v1", Sources: out.GroundedRepresentations[0].Population, Operations: []k.NativeOperation{{Kind: "propagate", Gain: .5, Cost: .08}, {Kind: "relax", Gain: .1, Cost: .08}, {Kind: "focus", Gain: .5, Cost: .08}, {Kind: "stop"}}, Seed: 37, LearningRate: .2, Gamma: 1, Lambda: 1, Temperature: .15, MaxSteps: 2}
		apply := func(c r.NativeControlCommand) (r.NativeControlResult, error) {
			return rt.ApplyNativeControl(c, at.Add(time.Duration(rt.LastSequence()+1)*time.Millisecond))
		}
		if _, e = apply(r.NativeControlCommand{ID: "initialize", Initialize: &cfg}); e != nil {
			return e
		}
		if _, e = apply(r.NativeControlCommand{ID: "begin", Begin: &r.NativeBegin{EpisodeID: "episode", Training: true}}); e != nil {
			return e
		}
		stopped, e = apply(r.NativeControlCommand{ID: "first-step", Step: &r.NativeStep{EpisodeID: "episode", Sample: 0}})
		if e != nil {
			return e
		}
		if stopped.Stopped {
			t.Fatal("first production operation unexpectedly stopped")
		}
		return rt.Checkpoint(cp)
	})
	if e != nil {
		t.Fatal(e)
	}
	before, e := os.ReadFile(cp)
	if e != nil {
		t.Fatal(e)
	}
	memoryBefore, e := os.ReadFile(memory)
	if e != nil {
		t.Fatal(e)
	}
	if e = first.Supervisor().Close(); e != nil {
		t.Fatal(e)
	}
	if e = log.Close(); e != nil {
		t.Fatal(e)
	}
	log = newStartupJournal(t, journal)
	defer log.Close()
	second, bootstrap, e := initializeProductionHorizon(log, journal, memory, cp)
	if e != nil || bootstrap {
		t.Fatal("production resume", bootstrap, e)
	}
	if _, count, e := finishProductionStartup(second, bootstrap, memory, cp); e != nil || count != 0 {
		t.Fatal("bootstrap repeated", count, e)
	}
	memoryAfter, e := os.ReadFile(memory)
	if e != nil || !bytes.Equal(memoryBefore, memoryAfter) {
		t.Fatal("production resume overwrote memory")
	}
	restoredCP := filepath.Join(dir, "restored.json")
	if e = second.Supervisor().WithMutation(func(rt *r.BrainRuntime) error { return rt.Checkpoint(restoredCP) }); e != nil {
		t.Fatal(e)
	}
	after, e := os.ReadFile(restoredCP)
	if e != nil {
		t.Fatal(e)
	}
	var a, b r.BrainSnapshot
	_ = json.Unmarshal(before, &a)
	_ = json.Unmarshal(after, &b)
	if a.Checksum != b.Checksum || a.Sequence != b.Sequence {
		t.Fatal("production restart lost complete active neural state")
	}
	e = second.Supervisor().WithMutation(func(rt *r.BrainRuntime) error {
		at := time.Date(2026, 11, 13, 0, 0, 1, 0, time.UTC)
		out, e := rt.ApplyNativeControl(r.NativeControlCommand{ID: "second-step", Step: &r.NativeStep{EpisodeID: "episode", Sample: .99}}, at)
		if e != nil {
			return e
		}
		if !out.Stopped {
			t.Fatal("bounded episode did not stop")
		}
		learned, e := rt.ApplyNativeControl(r.NativeControlCommand{ID: "observed", Outcome: &r.NativeOutcome{EpisodeID: "episode", TraceDigest: out.TraceDigest, Source: "production-mock-sensor", OutcomeID: "observed-feedback", Provenance: "observed-sensor", Reward: .8, Learn: true}}, at.Add(time.Millisecond))
		if e != nil {
			return e
		}
		if learned.Learning == nil || learned.Learning.UpdatedSynapses == 0 {
			t.Fatal("resumed production episode did not learn")
		}
		return rt.Checkpoint(cp)
	})
	if e != nil {
		t.Fatal(e)
	}
	if e = second.Supervisor().RecoverFromCheckpoint(log, restoredCP); e != nil {
		t.Fatal("production learned suffix recovery", e)
	}
	t.Logf("recovered active checkpoint checksum=%s sequence=%d; continued observed learning; bootstrap events added=0", b.Checksum, b.Sequence)
}
