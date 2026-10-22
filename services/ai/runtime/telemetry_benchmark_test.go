package runtime

import (
	"encoding/json"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"sort"
	"testing"
	"time"

	"github.com/project-horizon/horizon-core/services/ai/knowledge"
)

// Load a deterministic synthetic neural substrate before timing. Direct file
// construction avoids measuring quadratic projection recruitment as event cost.
// These 16D units have no populations/experience history; the measured physical
// sensor recruits and then reuses its own normal scoped populations.
func telemetryBenchmarkRuntime(b *testing.B, count, fanout int) *BrainRuntime {
	b.Helper()
	type unit struct {
		ID             knowledge.NodeID `json:"id"`
		Representation []float64        `json:"representation"`
		Threshold      float64          `json:"threshold"`
		Plasticity     float64          `json:"plasticity"`
	}
	type edge struct {
		SourceID   knowledge.NodeID       `json:"source_id"`
		TargetID   knowledge.NodeID       `json:"target_id"`
		Weight     float64                `json:"weight"`
		Confidence float64                `json:"confidence"`
		Dynamic    knowledge.DynamicState `json:"dynamic"`
	}
	units := make([]unit, count)
	edges := make([]edge, 0, count*fanout)
	for i := range units {
		values := make([]float64, 16)
		for j := range values {
			values[j] = math.Sin(float64((i+1)*(j+3)) * .731)
		}
		units[i] = unit{knowledge.NodeID(i + 1), values, .5, .1}
		for j := 1; j <= fanout; j++ {
			edges = append(edges, edge{knowledge.NodeID(i + 1), knowledge.NodeID((i+j)%count + 1), .2, .5, knowledge.DynamicState{Weight: .2, Confidence: .5, Frequency: 1}})
		}
	}
	data, err := json.Marshal(struct {
		SchemaVersion int    `json:"schema_version"`
		NeuralUnits   []unit `json:"neural_units"`
		Synapses      []edge `json:"synapses"`
	}{2, units, edges})
	if err != nil {
		b.Fatal(err)
	}
	path := filepath.Join(b.TempDir(), "foundation.json")
	if err := os.WriteFile(path, data, 0600); err != nil {
		b.Fatal(err)
	}
	brain := knowledge.NewBrain()
	if err := brain.Load(path); err != nil {
		b.Fatal(err)
	}
	r := NewBrainRuntime(brain)
	if r == nil {
		b.Fatal("runtime rejected fixture")
	}
	return r
}

func benchmarkTelemetrySubscription(h *TelemetryHub) *TelemetrySubscriber {
	// This benchmark also runs unchanged on PR #39. The old hub has only the
	// canonical stream; the optional interface selects the new monitor mode.
	if os.Getenv("HORIZON_BENCH_STREAM") != "canonical" {
		if revisions, ok := any(h).(interface{ SubscribeRevisions() *TelemetrySubscriber }); ok {
			return revisions.SubscribeRevisions()
		}
	}
	return h.Subscribe()
}

func BenchmarkTelemetryTransitions(b *testing.B) {
	for _, count := range []int{1000, 4000, 40000} {
		for _, fanout := range []int{0, 4} {
			for _, monitor := range []bool{false, true} {
				for _, journaled := range []bool{false, true} {
					b.Run(fmt.Sprintf("nodes=%d/fanout=%d/monitor=%t/journal=%t", count, fanout, monitor, journaled), func(b *testing.B) {
						setup := time.Now()
						r := telemetryBenchmarkRuntime(b, count, fanout)
						if journaled {
							log, err := OpenEventLog(filepath.Join(b.TempDir(), "events.jsonl"))
							if err != nil {
								b.Fatal(err)
							}
							b.Cleanup(func() { _ = log.Close() })
							r.SetEventLog(log)
						}
						var sub *TelemetrySubscriber
						if monitor {
							sub = benchmarkTelemetrySubscription(r.Telemetry())
							b.Cleanup(func() { r.telemetry.Unsubscribe(sub) })
						}
						sensor := factorizedSensor()
						sensor.Modality, sensor.Source = "motion", "benchmark-imu"
						sensor.Fields = []SensorField{{FieldID: "x", GroupID: "imu", Kind: "acceleration.mps2-v1", Values: []float64{.3}}}
						round := func(index int) (time.Duration, time.Duration) {
							sensor.SampleID = fmt.Sprintf("sample-%d", index)
							sensor.Timestamp = time.Date(2026, 10, 20, 12, 0, 0, 0, time.UTC).Add(time.Duration(index) * time.Second)
							if index > 0 {
								sensor.Provenance.ParentSampleIDs = []string{fmt.Sprintf("sample-%d", index-1)}
							}
							start := time.Now()
							if _, err := r.ProcessSensorObservation(sensor); err != nil {
								b.Fatal(err)
							}
							process := time.Since(start)
							r.SetClock(transactionClock{sensor.Timestamp.Add(time.Millisecond)})
							start = time.Now()
							if _, err := r.CognitiveThink(1); err != nil {
								b.Fatal(err)
							}
							think := time.Since(start)
							if sub != nil {
								<-sub.C()
								<-sub.C()
							}
							return process, think
						}
						for i := 0; i < 3; i++ {
							round(i)
						}
						setupTime := time.Since(setup)
						process, think := make([]time.Duration, b.N), make([]time.Duration, b.N)
						b.ReportAllocs()
						if os.Getenv("HORIZON_BENCH_READER") == "1" {
							stop, done := make(chan struct{}), make(chan struct{})
							go func() {
								defer close(done)
								clock := time.NewTicker(time.Millisecond)
								defer clock.Stop()
								for {
									select {
									case <-clock.C:
										_ = r.LastSequence()
									case <-stop:
										return
									}
								}
							}()
							b.Cleanup(func() { close(stop); <-done })
						}
						b.ResetTimer()
						for i := 0; i < b.N; i++ {
							process[i], think[i] = round(i + 3)
						}
						b.StopTimer()
						for name, samples := range map[string][]time.Duration{"sensor": process, "think": think} {
							sort.Slice(samples, func(i, j int) bool { return samples[i] < samples[j] })
							b.ReportMetric(float64(samples[(len(samples)-1)/2])/float64(time.Millisecond), name+"_p50_ms")
							b.ReportMetric(float64(samples[int(math.Ceil(.95*float64(len(samples))))-1])/float64(time.Millisecond), name+"_p95_ms")
						}
						b.ReportMetric(setupTime.Seconds(), "setup_s")
						b.ReportMetric(float64(len(r.brain.Registry.Nodes())), "final_units")
					})
				}
			}
		}
	}
}

// Full-state reads remain explicit expensive boundaries. This benchmark makes
// their cost visible rather than counting only cheap revision notifications.
func BenchmarkTelemetryReadBoundary(b *testing.B) {
	for _, operation := range []string{"snapshot", "handshake"} {
		b.Run(operation, func(b *testing.B) {
			r := telemetryBenchmarkRuntime(b, 4000, 4)
			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				if operation == "snapshot" {
					if _, err := r.MonitorSnapshot(); err != nil {
						b.Fatal(err)
					}
				} else if _, err := r.MonitorHandshake("benchmark"); err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}

// A bounded closed-loop command scenario: physical sample -> internal thought
// -> action binding to the observed population -> measured encoder outcome.
// It exercises four committed publications per cycle, not just an idle Think.
func BenchmarkTelemetrySensorimotor(b *testing.B) {
	for _, monitor := range []bool{false, true} {
		for _, journaled := range []bool{false, true} {
			b.Run(fmt.Sprintf("monitor=%t/journal=%t", monitor, journaled), func(b *testing.B) {
				r := telemetryBenchmarkRuntime(b, 4000, 4)
				if journaled {
					log, err := OpenEventLog(filepath.Join(b.TempDir(), "events.jsonl"))
					if err != nil {
						b.Fatal(err)
					}
					b.Cleanup(func() { _ = log.Close() })
					r.SetEventLog(log)
				}
				var sub *TelemetrySubscriber
				if monitor {
					sub = benchmarkTelemetrySubscription(r.Telemetry())
					b.Cleanup(func() { r.telemetry.Unsubscribe(sub) })
				}
				round := func(index int) {
					at := time.Date(2026, 10, 20, 12, 0, 0, 0, time.UTC).Add(time.Duration(index) * time.Second)
					r.SetClock(NewFixedClock(at))
					sensor := factorizedSensor()
					sensor.SampleID = fmt.Sprintf("cycle-%d", index)
					sensor.Timestamp = at
					sensor.Modality, sensor.Source = "motion", "benchmark-imu"
					sensor.Fields = []SensorField{{FieldID: "x", GroupID: "imu", Kind: "acceleration.mps2-v1", Values: []float64{.3}}}
					input, err := r.ProcessSensorObservation(sensor)
					if err != nil {
						b.Fatal(err)
					}
					if _, err := r.CognitiveThink(1); err != nil {
						b.Fatal(err)
					}
					request := fmt.Sprintf("motor-%d", index)
					if err := r.RegisterActionBinding(ActionBinding{RequestID: request, TargetNodeIDs: input.SensorBindings[0].Population}); err != nil {
						b.Fatal(err)
					}
					if _, err := r.ObserveOutcome(OutcomeEvent{BrainIdentity: BrainIdentity, RequestID: request, Observation: []string{"encoder-terminal"}, Source: "benchmark-encoder", EpisodeID: "closed-loop-session", Success: true, Reliability: .9, ObservedAt: at.Add(2 * time.Millisecond)}); err != nil {
						b.Fatal(err)
					}
					if sub != nil {
						for i := 0; i < 4; i++ {
							<-sub.C()
						}
					}
				}
				for i := 0; i < 3; i++ {
					round(i)
				}
				samples := make([]time.Duration, b.N)
				b.ReportAllocs()
				b.ResetTimer()
				for i := 0; i < b.N; i++ {
					start := time.Now()
					round(i + 3)
					samples[i] = time.Since(start)
				}
				b.StopTimer()
				sort.Slice(samples, func(i, j int) bool { return samples[i] < samples[j] })
				b.ReportMetric(float64(samples[(len(samples)-1)/2])/float64(time.Millisecond), "cycle_p50_ms")
				b.ReportMetric(float64(samples[int(math.Ceil(.95*float64(len(samples))))-1])/float64(time.Millisecond), "cycle_p95_ms")
			})
		}
	}
}
