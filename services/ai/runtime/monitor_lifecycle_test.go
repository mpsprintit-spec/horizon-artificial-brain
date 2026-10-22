package runtime

import (
	"errors"
	"path/filepath"
	"testing"
	"time"

	"github.com/project-horizon/horizon-core/services/ai/knowledge"
)

func TestRetiredAndPoisonedRuntimeStopsMonitorAccess(t *testing.T) {
	for _, scenario := range []string{
		"shutdown", "checkpoint-handover", "genesis-handover",
		"numeric-append-failure", "knowledge-append-failure",
		"sensor-intent-failure", "sensor-commit-failure",
	} {
		t.Run(scenario, func(t *testing.T) {
			journal, err := OpenEventLog(filepath.Join(t.TempDir(), "events.jsonl"))
			if err != nil {
				t.Fatal(err)
			}
			t.Cleanup(func() { _ = journal.Close() })
			original := NewBrainRuntime(nil)
			original.SetEventLog(journal)
			supervisor, err := NewRuntimeSupervisor(original)
			if err != nil {
				t.Fatal(err)
			}
			t.Cleanup(func() { _ = supervisor.Close() })
			hub := original.Telemetry()
			subscriber := hub.Subscribe()
			t.Cleanup(func() { hub.Unsubscribe(subscriber) })
			at := time.Date(2026, 10, 18, 13, 30, 0, 0, time.UTC)

			switch scenario {
			case "shutdown":
				err = supervisor.Close()
			case "checkpoint-handover":
				checkpoint := filepath.Join(t.TempDir(), "checkpoint.json")
				if err = original.Checkpoint(checkpoint); err == nil {
					err = supervisor.RecoverFromCheckpoint(journal, checkpoint)
				}
			case "genesis-handover":
				err = supervisor.RecoverFromCommittedJournal(journal, knowledge.NewBootstrapBrain)
			default:
				journal.beforeAppend = func(event LoggedEvent) error {
					if scenario == "sensor-commit-failure" && event.Type != EventTypeSensorCommit {
						return nil
					}
					return errors.New("injected uncertain append")
				}
				switch scenario {
				case "numeric-append-failure":
					_, err = original.GroundVectorAt(knowledge.NewNeuralVector([]float64{0.2, -0.1, 0.7}), at)
				case "knowledge-append-failure":
					_, err = original.ProcessKnowledgeDocument(directMutationTestDocument(), at)
				default:
					_, err = NewCognitiveOrchestrator(original).ProcessSensorObservation(testSensorObservation())
				}
				if !errors.Is(err, ErrRuntimeRecoveryRequired) {
					t.Fatalf("append failure did not require recovery: %v", err)
				}
				err = nil
			}
			if err != nil {
				t.Fatal(err)
			}

			if _, err := original.MonitorSnapshot(); !errors.Is(err, ErrRuntimeRecoveryRequired) {
				t.Errorf("unavailable runtime exposed a monitor snapshot: %v", err)
			}
			if _, err := original.MonitorHandshake("stale"); !errors.Is(err, ErrRuntimeRecoveryRequired) {
				t.Errorf("unavailable runtime exposed a monitor handshake: %v", err)
			}
			if original.Telemetry() != nil {
				t.Error("unavailable runtime exposed a telemetry publisher")
			}
			original.PublishObservationTelemetry(ObservationEnvelope{EventID: "stale"}, CognitiveOutput{Timestamp: at})
			select {
			case event, open := <-subscriber.C():
				if open {
					t.Errorf("unavailable runtime published telemetry: %+v", event)
				}
			default:
				t.Error("runtime invalidation left telemetry subscription open")
			}
			if late := hub.Subscribe(); late != nil {
				hub.Unsubscribe(late)
				t.Error("cached retired hub accepted a new subscription")
			}
			if scenario == "checkpoint-handover" || scenario == "genesis-handover" {
				if err := supervisor.With(func(active *BrainRuntime) error {
					if active == original || active.Telemetry() == hub {
						t.Fatal("handover retained retired runtime telemetry")
					}
					_, err := active.MonitorSnapshot()
					return err
				}); err != nil {
					t.Fatalf("replacement runtime unavailable: %v", err)
				}
			}
		})
	}
}

func TestRejectedRecoveryPreservesActiveMonitorSubscriptions(t *testing.T) {
	original := NewBrainRuntime(nil)
	journal, err := OpenEventLog(filepath.Join(t.TempDir(), "events.jsonl"))
	if err != nil {
		t.Fatal(err)
	}
	defer journal.Close()
	original.SetEventLog(journal)
	supervisor, err := NewRuntimeSupervisor(original)
	if err != nil {
		t.Fatal(err)
	}
	defer supervisor.Close()
	hub := original.Telemetry()
	subscriber := hub.Subscribe()
	defer hub.Unsubscribe(subscriber)
	if err := supervisor.RecoverFromCheckpoint(journal, filepath.Join(t.TempDir(), "missing.json")); err == nil {
		t.Fatal("missing checkpoint accepted")
	}
	if _, err := original.MonitorSnapshot(); err != nil {
		t.Fatalf("failed handover invalidated the still-active monitor: %v", err)
	}
	hub.Publish(TelemetryEvent{Type: "still-active"})
	select {
	case event, open := <-subscriber.C():
		if !open || event.Type != "still-active" {
			t.Fatal("failed handover closed active telemetry")
		}
	default:
		t.Fatal("active subscriber did not receive telemetry after rejected handover")
	}
}
