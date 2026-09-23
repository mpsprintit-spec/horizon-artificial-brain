package runtime

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/project-horizon/horizon-core/services/ai/knowledge"
	"github.com/project-horizon/horizon-core/services/ai/learning"
)

func TestEventLogAppendsAndReplaysLearning(t *testing.T) {
	path := filepath.Join(t.TempDir(), "events.jsonl")
	log, err := OpenEventLog(path)
	if err != nil { t.Fatalf("open log: %v", err) }
	defer log.Close()

	brain := knowledge.NewBrain()
	r := NewBrainRuntime(brain)
	r.SetEventLog(log)
	now := time.Date(2026, 9, 13, 12, 0, 0, 0, time.UTC)
	experience := learning.Experience{Sequence: []string{"saya", "ingin", "belajar"}, Weight: 0.8, Confidence: 0.6, ExperienceID: "replay-1", Source: "test", Modality: "language", Reliability: 0.9, IndependenceGroup: "g1"}
	if _, err := r.LearnExperience(experience, now); err != nil { t.Fatalf("learn: %v", err) }
	if _, _, err := r.Process(Event{ID: "p1", Stimulus: []string{"saya"}, Cycles: 2, Timestamp: now.Add(time.Second)}); err != nil { t.Fatalf("process: %v", err) }

	events, err := ReadEventLog(path)
	if err != nil { t.Fatalf("read log: %v", err) }
	if len(events) != 2 { t.Fatalf("event count: got %d want 2", len(events)) }
	if events[0].Sequence != 1 || events[1].Sequence != 2 { t.Fatalf("sequence mismatch: %d, %d", events[0].Sequence, events[1].Sequence) }
	if events[0].Type != EventTypeLearn || events[1].Type != EventTypeProcess { t.Fatalf("unexpected event types: %q, %q", events[0].Type, events[1].Type) }

	replayBrain := knowledge.NewBrain()
	replay := NewBrainRuntime(replayBrain)
	if err := ReplayEventLog(replay, events); err != nil { t.Fatalf("replay: %v", err) }
	if replayBrain.Fetch("saya") == nil || replayBrain.Fetch("belajar") == nil { t.Fatal("replay did not reconstruct learned nodes") }
	if len(replayBrain.Fetch("saya").OutboundAll()) == 0 { t.Fatal("replay did not reconstruct learned connection") }
	if replay.LastSequence() != 2 { t.Fatalf("replay sequence: got %d want 2", replay.LastSequence()) }
}

func TestReadEventLogRejectsTruncatedRecord(t *testing.T) {
	path := filepath.Join(t.TempDir(), "events.jsonl")
	if err := os.WriteFile(path, []byte(`{"schema_version":1,"brain_identity":"horizon-primary-brain","sequence":1,"type":"learn"`+"\n"), 0600); err != nil { t.Fatalf("write: %v", err) }
	if _, err := ReadEventLog(path); err == nil { t.Fatal("expected malformed event failure") }
}


func TestEventLogReplaysInquiryConsequencePreference(t *testing.T) {
	path := filepath.Join(t.TempDir(), "events.jsonl")
	log, err := OpenEventLog(path)
	if err != nil { t.Fatalf("open log: %v", err) }
	defer log.Close()

	r := NewBrainRuntime(knowledge.NewBrain())
	r.SetEventLog(log)
	if _, err := r.RecordInquiryConsequenceEvent(InquiryConsequenceEvent{Action: InquiryFocus, Valence: 1}, time.Date(2026, 9, 23, 12, 0, 0, 0, time.UTC)); err != nil {
		t.Fatalf("record consequence: %v", err)
	}
	before := r.InquiryPriorExperience(InquiryFocus, 0.50)

	events, err := ReadEventLog(path)
	if err != nil { t.Fatalf("read log: %v", err) }
	if len(events) != 1 || events[0].Type != EventTypeConsequence {
		t.Fatalf("unexpected consequence event log: %+v", events)
	}

	replay := NewBrainRuntime(knowledge.NewBrain())
	if err := ReplayEventLog(replay, events); err != nil { t.Fatalf("replay: %v", err) }
	after := replay.InquiryPriorExperience(InquiryFocus, 0.50)
	if after != before {
		t.Fatalf("replayed preference mismatch: got %v want %v", after, before)
	}
	if replay.LastSequence() != 1 {
		t.Fatalf("replay sequence: got %d want 1", replay.LastSequence())
	}
}


func TestInquiryConsequencePersistsCausalOutcomeTrace(t *testing.T) {
	path := filepath.Join(t.TempDir(), "events.jsonl")
	log, err := OpenEventLog(path)
	if err != nil { t.Fatalf("open log: %v", err) }
	defer log.Close()

	brain := knowledge.NewBrain()
	target := brain.Store("action-target")
	outcome := brain.Store("observed-outcome")
	r := NewBrainRuntime(brain)
	r.SetEventLog(log)
	event := InquiryConsequenceEvent{
		Action: InquiryFocus, Valence: 0.8, InformationGain: 0.9, Reliability: 0.8,
		TargetNodeIDs: []knowledge.NodeID{target.ID},
		OutcomeNodeIDs: []knowledge.NodeID{outcome.ID},
		CausalLink: "inquiry:test-causal",
	}
	at := time.Date(2026, 9, 23, 12, 0, 0, 0, time.UTC)
	if _, err := r.RecordInquiryConsequenceEvent(event, at); err != nil {
		t.Fatalf("record consequence: %v", err)
	}
	matches := brain.Patterns.MatchTrace([]knowledge.PatternStep{
		{NodeID: target.ID, Position: 0, Activation: 1},
		{NodeID: outcome.ID, Position: 1, Activation: 1},
	}, nil)
	if len(matches) == 0 {
		t.Fatal("causal outcome trace was not retained in brain patterns")
	}
	if matches[0].Evidence[0].CausalLink != "inquiry:test-causal" {
		t.Fatalf("causal link mismatch: %+v", matches[0].Evidence)
	}

	events, err := ReadEventLog(path)
	if err != nil { t.Fatalf("read log: %v", err) }
	replayBrain := knowledge.NewBrain()
	replayTarget := replayBrain.Store("action-target")
	replayOutcome := replayBrain.Store("observed-outcome")
	replayEvent := events[0]
	replayEvent.Consequence.TargetNodeIDs = []knowledge.NodeID{replayTarget.ID}
	replayEvent.Consequence.OutcomeNodeIDs = []knowledge.NodeID{replayOutcome.ID}
	replay := NewBrainRuntime(replayBrain)
	if err := ReplayEventLog(replay, []LoggedEvent{replayEvent}); err != nil {
		t.Fatalf("replay: %v", err)
	}
	replayMatches := replayBrain.Patterns.MatchTrace([]knowledge.PatternStep{
		{NodeID: replayTarget.ID, Position: 0, Activation: 1},
		{NodeID: replayOutcome.ID, Position: 1, Activation: 1},
	}, nil)
	if len(replayMatches) == 0 {
		t.Fatal("replay did not reconstruct causal outcome trace")
	}
}
