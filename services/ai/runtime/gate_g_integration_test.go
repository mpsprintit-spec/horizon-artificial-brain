package runtime

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	k "github.com/project-horizon/horizon-core/services/ai/knowledge"
	"github.com/project-horizon/horizon-core/services/ai/learning"
	"os"
	"path/filepath"
	"reflect"
	"testing"
	"time"
)

func gateGTime(r *BrainRuntime) time.Time {
	return time.Date(2026, 11, 6, 0, 0, 0, 0, time.UTC).Add(time.Duration(r.LastSequence()+1) * time.Millisecond)
}
func gateGSensor(t *testing.T, r *BrainRuntime, id string) {
	t.Helper()
	s := testSensorObservation()
	s.SampleID = id
	s.Timestamp = gateGTime(r)
	if _, err := r.ProcessSensorObservation(s); err != nil {
		t.Fatal(err)
	}
}
func gateGText(t *testing.T, r *BrainRuntime, id string) {
	t.Helper()
	if _, err := r.ProcessObservationEnvelope(transactionObservation(id, gateGTime(r), "new", "surface")); err != nil {
		t.Fatal(err)
	}
}
func gateGInquiry(t *testing.T, r *BrainRuntime, id string) {
	t.Helper()
	at := gateGTime(r)
	req := InquiryRequest{BrainIdentity: BrainIdentity, RequestID: id, Action: InquiryFocus, Uncertainty: .7}
	if _, _, err := NewCognitiveOrchestrator(r).ProcessInquiryObservation(req, Event{ID: id, Timestamp: at, Cycles: 1}, ObservationInput{Source: "gate-g-observer", Modality: "vision", Tokens: []string{"new-evidence"}}, nil); err != nil {
		t.Fatal(err)
	}
}
func gateGNative(t *testing.T, r *BrainRuntime, id string) {
	t.Helper()
	before := r.activation.CurrentState()
	nativeApply(t, r, NativeControlCommand{ID: id + "-begin", Begin: &NativeBegin{EpisodeID: id, Training: true}})
	if !reflect.DeepEqual(before, r.activation.CurrentState()) {
		t.Fatal("begin replaced production activation")
	}
	for j := 0; j < 8; j++ {
		input := r.activation.CurrentState()
		out := nativeApply(t, r, NativeControlCommand{ID: fmt.Sprintf("%s-step-%d", id, j), Step: &NativeStep{EpisodeID: id, Sample: .01}})
		ep := r.brain.NativeControlSnapshot().Episode
		if !reflect.DeepEqual(input, ep.Decisions[j].State) {
			t.Fatal("native operation lost previous canonical state")
		}
		if out.Stopped {
			break
		}
	}
	out := nativeEnd(t, r, id+"-observed", .9, true)
	if out.Learning == nil || out.Learning.UpdatedSynapses == 0 {
		t.Fatal("no canonical synaptic learning")
	}
}
func gateGCompare(t *testing.T, a, b *BrainRuntime) {
	t.Helper()
	if !bytes.Equal(continuationBytes(t, a), continuationBytes(t, b)) {
		// Keep a local diagnostic that identifies the first complete-state component.
		x, y := []json.RawMessage{}, []json.RawMessage{}
		_ = json.Unmarshal(continuationBytes(t, a), &x)
		_ = json.Unmarshal(continuationBytes(t, b), &y)
		for i := range x {
			if !bytes.Equal(x[i], y[i]) {
				t.Logf("mismatching continuation component %d\nlive %s\nrecovered %s", i, x[i], y[i])
				break
			}
		}
		t.Fatal("complete canonical continuation mismatch")
	}
	if k.NativeDigest(a.activation.SnapshotState()) != k.NativeDigest(b.activation.SnapshotState()) {
		t.Fatal("activation checksum mismatch")
	}
}

// Every case executes the genuine production input, without substituting a
// fixture observation at native begin. All owners retain one Brain pointer.
func TestGateGMixedReplay(t *testing.T) {
	for _, seed := range []int64{37, 83, 151} {
		for _, scenario := range []string{"sensor-native", "native-sensor", "sensor-inquiry-native", "inquiry-native-outcome", "written-native", "thought-native", "learning-checkpoint-sensor", "checkpoint-active-continuation", "native-thought-native"} {
			t.Run(fmt.Sprintf("seed%d/%s", seed, scenario), func(t *testing.T) {
				r, ids := nativeFixture(t)
				owner := r.brain
				foundation := gateGFoundation(t, r)
				wal, err := OpenEventLog(filepath.Join(t.TempDir(), "events.jsonl"))
				if err != nil {
					t.Fatal(err)
				}
				defer wal.Close()
				r.SetEventLog(wal)
				cp := filepath.Join(t.TempDir(), "prefix.json")
				if err = r.Checkpoint(cp); err != nil {
					t.Fatal(err)
				}
				// Always start through production, then explicitly opt into the controller.
				gateGSensor(t, r, "sensor-prefix")
				cfg := nativeControlConfig(ids)
				cfg.Version = k.NativeControlCausalVersion
				cfg.Encoder = "local-relational-v1"
				cfg.Seed = seed
				nativeApply(t, r, NativeControlCommand{ID: "initialize", Initialize: &cfg})
				switch scenario {
				case "sensor-native":
					gateGSensor(t, r, "sensor")
					gateGNative(t, r, "ep")
				case "native-sensor":
					gateGNative(t, r, "ep")
					gateGSensor(t, r, "sensor")
				case "sensor-inquiry-native":
					gateGSensor(t, r, "sensor")
					gateGInquiry(t, r, "inquiry")
					gateGNative(t, r, "ep")
				case "inquiry-native-outcome":
					gateGInquiry(t, r, "inquiry")
					rtClock := NewFixedClock(gateGTime(r))
					r.SetClock(rtClock)
					if err = r.RegisterActionBinding(ActionBinding{RequestID: "inquiry-12-focus", Intent: "inquiry:focus", TargetNodeIDs: ids[:1]}); err != nil {
						t.Fatal(err)
					}
					execution := transactionInquiry(t, r, gateGTime(r))
					gateGNative(t, r, "ep")
					at := gateGTime(r)
					if _, _, _, err = NewCognitiveOrchestrator(r).ProcessInquiryOutcome(InquiryResult{Execution: execution, Event: Event{ID: "actual-inquiry-outcome", Timestamp: at, Cycles: 1}, Observation: ObservationInput{Source: "observer", Modality: "vision", Tokens: []string{"unexpected"}}, Success: true, Reliability: .9, InformationGain: .6}, at); err != nil {
						t.Fatal(err)
					}
				case "written-native":
					gateGText(t, r, "written")
					gateGNative(t, r, "ep")
				case "thought-native":
					r.SetClock(NewFixedClock(gateGTime(r)))
					if _, err = r.CognitiveThink(2); err != nil {
						t.Fatal(err)
					}
					gateGNative(t, r, "ep")
				case "learning-checkpoint-sensor":
					gateGNative(t, r, "ep")
					if err = r.Checkpoint(cp); err != nil {
						t.Fatal(err)
					}
					gateGSensor(t, r, "sensor")
				case "native-thought-native":
					gateGNative(t, r, "first-episode")
					r.SetClock(NewFixedClock(gateGTime(r)))
					if _, err = r.CognitiveThink(2); err != nil {
						t.Fatal(err)
					}
					gateGNative(t, r, "second-episode")
				case "checkpoint-active-continuation":
					nativeApply(t, r, NativeControlCommand{ID: "begin", Begin: &NativeBegin{EpisodeID: "active", Training: true}})
					out := nativeApply(t, r, NativeControlCommand{ID: "step", Step: &NativeStep{EpisodeID: "active", Sample: .01}})
					if err = r.Checkpoint(cp); err != nil {
						t.Fatal(err)
					}
					if !out.Stopped {
						for j := 1; j < 8; j++ {
							out = nativeApply(t, r, NativeControlCommand{ID: fmt.Sprintf("step-%d", j), Step: &NativeStep{EpisodeID: "active", Sample: .01}})
							if out.Stopped {
								break
							}
						}
					}
					nativeEnd(t, r, "observed", .9, true)
				}
				if owner != r.brain || r.activation.Memory != owner || r.learning.Kb != owner || r.dnf.Brain() != owner {
					t.Fatal("canonical Brain ownership changed")
				}
				fromGenesis, err := RebuildRuntimeFromCommittedJournal(wal, foundation)
				if err != nil {
					t.Fatal("genesis", err)
				}
				gateGCompare(t, r, fromGenesis)
				fromCP, err := RebuildRuntimeFromCheckpointAndJournal(wal, cp)
				if err != nil {
					t.Fatal("checkpoint suffix", err)
				}
				gateGCompare(t, r, fromCP)
				r.SetClock(NewFixedClock(gateGTime(r)))
				final := filepath.Join(t.TempDir(), "final.json")
				if err = r.Checkpoint(final); err != nil {
					t.Fatal(err)
				}
				snapData, err := os.ReadFile(final)
				if err != nil {
					t.Fatal(err)
				}
				var snap BrainSnapshot
				if err = json.Unmarshal(snapData, &snap); err != nil {
					t.Fatal(err)
				}
				for _, rt := range []*BrainRuntime{fromGenesis, fromCP} {
					got, err := checkpointChecksumForSchema(BrainSnapshot{SchemaVersion: 20, BrainIdentity: BrainIdentity, Sequence: rt.LastSequence(), JournalPrefixHash: snap.JournalPrefixHash, Brain: mustGateGBrain(t, rt), Activation: rt.activation.SnapshotState(), CognitiveState: rt.lastCognitiveState, InquiryValence: rt.inquiryValence, LastObservationPopulation: rt.lastObservationPopulation, LastObservationAt: rt.lastObservationAt, ProcessedEvents: rt.processedEvents, SurfaceAnnotations: rt.brain.SurfaceAnnotationsSnapshot(), EvidenceLedger: ptrGateGLedger(rt), ActionBindings: rt.actions, Composition: rt.composition, NeuralWrittenGrounding: rt.neuralWrittenGrounding})
					if err != nil || got != snap.Checksum {
						t.Fatalf("checkpoint checksum %s != %s: %v", got, snap.Checksum, err)
					}
				}
				// Restore after all learning, then continue both runtimes with real input.
				restored, err := RebuildRuntimeFromCheckpointAndJournal(wal, final)
				if err != nil {
					t.Fatal(err)
				}
				gateGSensor(t, r, "next-input")
				gateGSensor(t, restored, "next-input")
				gateGNative(t, r, "next-episode")
				gateGNative(t, restored, "next-episode")
				gateGCompare(t, r, restored)
			})
		}
	}
}
func mustGateGBrain(t *testing.T, r *BrainRuntime) json.RawMessage {
	t.Helper()
	b, e := r.brain.CanonicalJSON()
	if e != nil {
		t.Fatal(e)
	}
	return b
}

func TestGateGAdmissionRejectsBeforeMutation(t *testing.T) {
	for _, stage := range []string{"active", "awaiting-outcome", "pending-contrast"} {
		t.Run(stage, func(t *testing.T) {
			r, ids := nativeFixture(t)
			w := openKnowledgeJournal(t)
			r.SetEventLog(w)
			cfg := nativeControlConfig(ids)
			cfg.Version = k.NativeControlCausalVersion
			cfg.Encoder = "local-relational-v1"
			nativeApply(t, r, NativeControlCommand{ID: "init", Initialize: &cfg})
			nativeApply(t, r, NativeControlCommand{ID: "begin", Begin: &NativeBegin{EpisodeID: "ep", Training: true, Intervention: true, MatchKey: "world", Observation: &NativeObservation{Source: "fixture", SampleID: "first", State: map[k.NodeID]float64{ids[0]: .8, ids[6]: .8}}}})
			if stage != "active" {
				a := 3
				nativeApply(t, r, NativeControlCommand{ID: "stop", Step: &NativeStep{EpisodeID: "ep", ExperimentAction: &a}})
			}
			if stage == "pending-contrast" {
				ep := r.brain.NativeControlSnapshot().Episode
				nativeApply(t, r, NativeControlCommand{ID: "out", Outcome: &NativeOutcome{EpisodeID: "ep", TraceDigest: k.NativeDigest(ep), Source: "fixture", OutcomeID: "out", Provenance: "observed-sensor", Reward: .8, Learn: true, ContrastID: "pair", ContrastMode: "action"}})
			}
			ops := map[string]func() error{
				"sensor": func() error {
					s := testSensorObservation()
					s.SampleID = "blocked"
					s.Timestamp = gateGTime(r)
					_, e := r.ProcessSensorObservation(s)
					return e
				},
				"vector": func() error { _, e := r.GroundVectorAt(k.NewNeuralVector([]float64{.1, .7}), gateGTime(r)); return e },
				"written": func() error {
					_, e := r.ProcessObservationEnvelope(transactionObservation("blocked", gateGTime(r), "never-ground"))
					return e
				},
				"think": func() error { _, e := r.CognitiveThink(1); return e },
				"process": func() error {
					_, _, e := r.Process(Event{ID: "blocked", Timestamp: gateGTime(r), StimulusNodeIDs: ids[:1], Cycles: 1})
					return e
				},
				"binding": func() error {
					return r.RegisterActionBinding(ActionBinding{RequestID: "blocked", TargetNodeIDs: ids[:1]})
				},
				"import": func() error { _, e := r.ProcessKnowledgeDocument(directMutationTestDocument(), gateGTime(r)); return e },
			}
			for name, op := range ops {
				before := continuationBytes(t, r)
				raw, _ := os.ReadFile(w.file.Name())
				if e := op(); !errors.Is(e, ErrCommandAdmission) {
					t.Fatalf("%s: %v", name, e)
				}
				after, _ := os.ReadFile(w.file.Name())
				if !bytes.Equal(before, continuationBytes(t, r)) || !bytes.Equal(raw, after) || r.RecoveryRequired() {
					t.Fatalf("%s was not rejected atomically", name)
				}
			}
			cp := filepath.Join(t.TempDir(), "pending.json")
			if e := r.Checkpoint(cp); e != nil {
				t.Fatal(e)
			}
			got, e := RebuildRuntimeFromCheckpointAndJournal(w, cp)
			if e != nil {
				t.Fatal(e)
			}
			gateGCompare(t, r, got)
		})
	}
}
func ptrGateGLedger(r *BrainRuntime) *learning.EvidenceLedgerSnapshot {
	v := r.evidence.Snapshot()
	return &v
}

func gateGFoundation(t *testing.T, r *BrainRuntime) func() *k.Brain {
	t.Helper()
	p := filepath.Join(t.TempDir(), "genesis.json")
	if e := os.WriteFile(p, mustGateGBrain(t, r), 0600); e != nil {
		t.Fatal(e)
	}
	return func() *k.Brain {
		b := k.NewBrain()
		if e := b.Load(p); e != nil {
			t.Fatal(e)
		}
		return b
	}
}
