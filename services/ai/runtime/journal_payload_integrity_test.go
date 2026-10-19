package runtime

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/project-horizon/horizon-core/services/ai/knowledge"
)

func TestJournalRejectsMixedCommandsAndContradictoryObservationProvenance(t *testing.T) {
	at := time.Date(2026, 10, 19, 9, 0, 0, 0, time.UTC)
	journal := openKnowledgeJournal(t)
	r := NewBrainRuntime(nil)
	r.SetEventLog(journal)
	if _, err := r.ProcessObservationEnvelope(transactionObservation("observation", at, "seen")); err != nil {
		t.Fatal(err)
	}
	events, err := ReadEventLog(journal.file.Name())
	if err != nil {
		t.Fatal(err)
	}
	base, err := json.Marshal(events[0])
	if err != nil {
		t.Fatal(err)
	}
	cases := map[string]func(*LoggedEvent){
		"unknown-type":     func(e *LoggedEvent) { e.Type = "unrecognized-command" },
		"mixed-binding":    func(e *LoggedEvent) { e.ActionBinding = &ActionBinding{RequestID: "foreign"} },
		"mixed-inquiry":    func(e *LoggedEvent) { e.InquiryOutcome = &InquiryOutcomeCommand{} },
		"mixed-experience": func(e *LoggedEvent) { e.Type = EventTypeLearn },
		"missing-input":    func(e *LoggedEvent) { e.ObservationCommand = nil },
		"post-grounding":   func(e *LoggedEvent) { e.Event.StimulusNodeIDs = []knowledge.NodeID{1} },
		"source-mismatch":  func(e *LoggedEvent) { e.ObservationCommand.Input.Source = "another-source" },
		"tokens-mismatch":  func(e *LoggedEvent) { e.ObservationCommand.Input.Tokens[0] = "another-surface" },
		"time-mismatch":    func(e *LoggedEvent) { e.Event.Timestamp = at.Add(time.Second) },
	}
	path := filepath.Join(t.TempDir(), "bad.jsonl")
	for name, mutate := range cases {
		t.Run(name, func(t *testing.T) {
			var event LoggedEvent
			if err := json.Unmarshal(base, &event); err != nil {
				t.Fatal(err)
			}
			mutate(&event)
			data, err := json.Marshal(event)
			if err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(path, append(data, '\n'), 0600); err != nil {
				t.Fatal(err)
			}
			if _, err := ReadEventLog(path); err == nil {
				t.Fatal("unreplayable or mixed record accepted")
			}
		})
	}
	think := LoggedEvent{SchemaVersion: 1, BrainIdentity: BrainIdentity, Sequence: 1, Type: EventTypeCognitiveThink, Timestamp: at, Event: &Event{Timestamp: at, Cycles: 1, Source: "ignored"}}
	if err := ReplayEventLog(NewBrainRuntime(nil), []LoggedEvent{think}); err == nil {
		t.Fatal("cognitive think silently discarded event fields")
	}
}

func TestSensorRecoveryCannotDiscardForeignCompoundPayload(t *testing.T) {
	for _, onCommit := range []bool{false, true} {
		journal := openKnowledgeJournal(t)
		intent := sensorJournalTestIntent()
		var records []LoggedEvent
		if onCommit {
			hash, err := sensorIntentHash(intent)
			if err != nil {
				t.Fatal(err)
			}
			records = []LoggedEvent{intent, {SchemaVersion: 1, BrainIdentity: BrainIdentity, Sequence: 1, Type: EventTypeSensorCommit, Timestamp: intent.Timestamp, IntentHash: hash, InquiryOutcome: &InquiryOutcomeCommand{}}}
		} else {
			intent.InquiryOutcome = &InquiryOutcomeCommand{}
			records = []LoggedEvent{intent}
		}
		for _, record := range records {
			if err := journal.Append(record); err != nil {
				t.Fatal(err)
			}
		}
		before, err := os.ReadFile(journal.file.Name())
		if err != nil {
			t.Fatal(err)
		}
		if _, err := ReadEventLog(journal.file.Name()); err == nil {
			t.Fatal("sensor framing ignored foreign compound payload")
		}
		if _, err := journal.RecoverTrailingSensorIntent(); err == nil {
			t.Fatal("sensor repair erased a compound command")
		}
		after, err := os.ReadFile(journal.file.Name())
		if err != nil {
			t.Fatal(err)
		}
		if !bytes.Equal(before, after) {
			t.Fatal("rejected recovery changed journal")
		}
	}
}

func TestObservationRejectsClaimedHashMismatchBeforeNormalization(t *testing.T) {
	r := NewBrainRuntime(nil)
	before := continuationBytes(t, r)
	event := transactionObservation("wrong-hash", time.Date(2026, 10, 19, 9, 0, 0, 0, time.UTC), "seen")
	event.Payload, event.PayloadHash = json.RawMessage(`{"sample":1}`), "sha256:not-the-payload"
	if _, err := r.ProcessObservationEnvelope(event); err == nil {
		t.Fatal("normalization hid a false payload digest")
	}
	if !bytes.Equal(before, continuationBytes(t, r)) {
		t.Fatal("bad payload digest changed state")
	}
}
