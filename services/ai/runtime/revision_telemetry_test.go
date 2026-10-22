package runtime

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"path/filepath"
	"testing"
	"time"

	"github.com/project-horizon/horizon-core/services/ai/knowledge"
)

func TestTelemetryBuildIsDemandDrivenAndKeepsLegacyHashes(t *testing.T) {
	h := NewTelemetryHub(2)
	defer h.Close()
	calls := 0
	build := func(canonical bool) (TelemetryEvent, error) {
		calls++
		hash := ""
		if canonical {
			hash = "sha256:full"
		}
		return TelemetryEvent{StateRevision: 1, EventHash: "sha256:event", CanonicalStateHash: hash}, nil
	}
	h.publishBuilt(build)
	if calls != 0 {
		t.Fatal("no-subscriber publication built an event")
	}
	revision := h.SubscribeRevisions()
	h.publishBuilt(build)
	e := <-revision.C()
	encoded, _ := json.Marshal(e)
	if calls != 1 || e.SchemaVersion != 2 || bytes.Contains(encoded, []byte("canonical_state_hash")) {
		t.Fatalf("revision contract: %s", encoded)
	}
	canonical := h.Subscribe()
	h.publishBuilt(build)
	r, c := <-revision.C(), <-canonical.C()
	if calls != 2 || r.CanonicalStateHash != "" || c.CanonicalStateHash != "sha256:full" || c.SchemaVersion != 0 || r.EventHash != c.EventHash {
		t.Fatal("mixed subscribers changed integrity contracts")
	}
	h.Unsubscribe(canonical)
	h.Unsubscribe(revision)
	h.publishBuilt(build)
	h.Close()
	h.publishBuilt(build)
	if calls != 2 {
		t.Fatal("empty/closed hub still builds events")
	}
}

func TestTelemetryRecipientSelectionDoesNotBlockCancellationOrMislabelLateJoin(t *testing.T) {
	h := NewTelemetryHub(2)
	defer h.Close()
	revision := h.SubscribeRevisions()
	started, release, done := make(chan struct{}), make(chan struct{}), make(chan struct{})
	go func() {
		defer close(done)
		h.publishBuilt(func(canonical bool) (TelemetryEvent, error) {
			if canonical {
				t.Error("revision-only selection requested full graph hash")
			}
			close(started)
			<-release
			return TelemetryEvent{StateRevision: 1}, nil
		})
	}()
	<-started
	canonical := h.Subscribe()
	h.Unsubscribe(revision)
	close(release)
	<-done
	select {
	case <-canonical.C():
		t.Fatal("late canonical subscriber received an unhashed event")
	default:
	}
	h.publishBuilt(func(required bool) (TelemetryEvent, error) {
		if !required {
			t.Fatal("canonical subscriber lost full hash requirement")
		}
		return TelemetryEvent{StateRevision: 2, CanonicalStateHash: "sha256:second"}, nil
	})
	if e := <-canonical.C(); e.StateRevision != 2 || e.CanonicalStateHash == "" {
		t.Fatal("next publication did not include late subscriber")
	}
}

func TestTelemetryEncodingFailureClosesSelectedSubscriptions(t *testing.T) {
	h := NewTelemetryHub(1)
	a, b := h.Subscribe(), h.SubscribeRevisions()
	h.publishBuilt(func(bool) (TelemetryEvent, error) { return TelemetryEvent{}, errors.New("invalid canonical state") })
	for _, s := range []*TelemetrySubscriber{a, b} {
		if _, open := <-s.C(); open {
			t.Fatal("encoding failure published an integrity digest")
		}
	}
}

func TestTelemetryModesPreserveCompleteStateCheckpointAndReplay(t *testing.T) {
	for _, journaled := range []bool{false, true} {
		t.Run(map[bool]string{false: "standalone", true: "journaled"}[journaled], func(t *testing.T) {
			type run struct {
				r    *BrainRuntime
				subs []*TelemetrySubscriber
				path string
			}
			runs := make([]run, 4)
			at := time.Date(2026, 10, 20, 12, 0, 0, 0, time.UTC)
			for i := range runs {
				r := NewBrainRuntime(knowledge.NewBrain())
				r.SetClock(NewFixedClock(at))
				runs[i].r = r
				if i == 1 || i == 3 {
					runs[i].subs = append(runs[i].subs, r.Telemetry().SubscribeRevisions())
				}
				if i == 2 || i == 3 {
					runs[i].subs = append(runs[i].subs, r.Telemetry().Subscribe())
				}
				if journaled {
					runs[i].path = filepath.Join(t.TempDir(), "events.jsonl")
					log, err := OpenEventLog(runs[i].path)
					if err != nil {
						t.Fatal(err)
					}
					t.Cleanup(func() { _ = log.Close() })
					r.SetEventLog(log)
				}
			}
			operations := []func(*BrainRuntime) error{
				func(r *BrainRuntime) error {
					_, _, e := r.Process(Event{ID: "raw", Stimulus: []string{"naku"}, Timestamp: at, Cycles: 1})
					return e
				},
				func(r *BrainRuntime) error { _, _, e := r.ThinkAt(1, at.Add(time.Second)); return e },
				func(r *BrainRuntime) error {
					_, e := r.CognitiveProcess(Event{ID: "cognitive", Stimulus: []string{"melo"}, Timestamp: at.Add(2 * time.Second), Cycles: 1})
					return e
				},
				func(r *BrainRuntime) error {
					_, e := r.ProcessObservationEnvelope(transactionObservation("written", at.Add(3*time.Second), "naku", "melo"))
					return e
				},
				func(r *BrainRuntime) error {
					s := factorizedSensor()
					s.Timestamp = at.Add(4 * time.Second)
					_, e := r.ProcessSensorObservation(s)
					return e
				},
				func(r *BrainRuntime) error {
					return r.RegisterActionBinding(ActionBinding{RequestID: "act", TargetNodeIDs: []knowledge.NodeID{1}})
				},
				func(r *BrainRuntime) error {
					_, e := r.ObserveOutcome(OutcomeEvent{RequestID: "act", BrainIdentity: BrainIdentity, Success: true, Observation: []string{"seen"}, Source: "camera", Reliability: .9, ObservedAt: at.Add(5 * time.Second)})
					return e
				},
				func(r *BrainRuntime) error {
					r.SetClock(NewFixedClock(at.Add(6 * time.Second)))
					_, e := r.CognitiveThink(2)
					return e
				},
			}
			for step, operation := range operations {
				for i := range runs {
					r := runs[i].r
					if err := operation(r); err != nil {
						t.Fatalf("mode %d step %d: %v", i, step, err)
					}
					brain, err := r.brain.CanonicalJSON()
					if err != nil {
						t.Fatal(err)
					}
					digest := sha256.Sum256(brain)
					hash := "sha256:" + hex.EncodeToString(digest[:])
					var eventHash string
					for _, sub := range runs[i].subs {
						if step == 1 { // Legacy raw ThinkAt has no telemetry publication.
							select {
							case <-sub.C():
								t.Fatal("raw ThinkAt changed its publication contract")
							default:
							}
							continue
						}
						select {
						case event := <-sub.C():
							if event.StateRevision != r.LastSequence() {
								t.Fatal("telemetry revision is not completed command revision")
							}
							if sub.revisionsOnly {
								if event.CanonicalStateHash != "" || event.SchemaVersion != 2 {
									t.Fatal("revision mode hashed full state")
								}
							} else if event.CanonicalStateHash != hash {
								t.Fatal("legacy event hash differs from exact canonical bytes")
							}
							if eventHash != "" && event.EventHash != eventHash {
								t.Fatal("mode changed event hash")
							}
							eventHash = event.EventHash
						default:
							t.Fatalf("mode %d step %d missing event", i, step)
						}
					}
					if !bytes.Equal(continuationBytes(t, runs[0].r), continuationBytes(t, r)) {
						t.Fatalf("telemetry changed complete continuation at step %d mode %d", step, i)
					}
				}
			}
			for i := range runs {
				r := runs[i].r
				path := orderedSave(t, r)
				restored := orderedFixtureClone(t, path)
				if !bytes.Equal(continuationBytes(t, r), continuationBytes(t, restored)) {
					t.Fatal("checkpoint state differs")
				}
				if journaled {
					events, err := ReadEventLog(runs[i].path)
					if err != nil {
						t.Fatal(err)
					}
					replayed := NewBrainRuntime(knowledge.NewBrain())
					if err := ReplayEventLog(replayed, events); err != nil {
						t.Fatal(err)
					}
					if !bytes.Equal(continuationBytes(t, r), continuationBytes(t, replayed)) {
						t.Fatal("journal replay state differs")
					}
				}
				r.SetClock(NewFixedClock(at.Add(7 * time.Second)))
				restored.SetClock(NewFixedClock(at.Add(7 * time.Second)))
				if _, err := r.CognitiveThink(2); err != nil {
					t.Fatal(err)
				}
				if _, err := restored.CognitiveThink(2); err != nil {
					t.Fatal(err)
				}
				if !bytes.Equal(continuationBytes(t, r), continuationBytes(t, restored)) {
					t.Fatal("checkpoint changed next thought")
				}
			}
		})
	}
}
