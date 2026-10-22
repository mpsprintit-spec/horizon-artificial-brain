package runtime

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"math"
	"os"
	"path/filepath"
	"reflect"
	"sync"
	"testing"
	"time"

	"github.com/project-horizon/horizon-core/services/ai/knowledge"
)

func runtimeSpellingExposure(t *testing.T, r *BrainRuntime, at time.Time) knowledge.SpellingExposure {
	t.Helper()
	grounded, err := r.GroundVectorAt(knowledge.NewNeuralVector([]float64{.2, -.8, .7, -.3}), at)
	if err != nil {
		t.Fatal(err)
	}
	e, err := knowledge.NewSpellingExposure(grounded.Population, "lantai🌌", .8, .8)
	if err != nil {
		t.Fatal(err)
	}
	return e
}

func requireRuntimeSpelling(t *testing.T, r *BrainRuntime, e knowledge.SpellingExposure) knowledge.SpellingReadout {
	t.Helper()
	before := continuationBytes(t, r)
	out, err := r.ReadLearnedSpelling(e.Population)
	if err != nil || out.Status != knowledge.SpellingDecoded || out.Text != "lantai🌌" {
		t.Fatalf("numeric spelling was lost: %+v %v", out, err)
	}
	if !bytes.Equal(before, continuationBytes(t, r)) {
		t.Fatal("reading spelling changed neural continuation")
	}
	return out
}

func TestLearnedSpellingJournalCheckpointGenesisSuffixAndNextThought(t *testing.T) {
	at := time.Date(2026, 10, 21, 6, 0, 0, 0, time.UTC)
	r := NewBrainRuntime(knowledge.NewBrain())
	r.SetClock(NewFixedClock(at))
	log := openKnowledgeJournal(t)
	r.SetEventLog(log)
	e := runtimeSpellingExposure(t, r, at)
	for i := 0; i < 2; i++ {
		if _, err := r.LearnSpelling(e, at.Add(time.Duration(i+1)*time.Second)); err != nil {
			t.Fatal(err)
		}
	}
	cp := filepath.Join(t.TempDir(), "codec.json")
	if err := r.Checkpoint(cp); err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 2; i++ {
		if _, err := r.LearnSpelling(e, at.Add(time.Duration(i+3)*time.Second)); err != nil {
			t.Fatal(err)
		}
	}
	want := requireRuntimeSpelling(t, r, e)
	genesis, err := RebuildRuntimeFromCommittedJournal(log, knowledge.NewBrain)
	if err != nil {
		t.Fatal(err)
	}
	suffix, err := RebuildRuntimeFromCheckpointAndJournal(log, cp)
	if err != nil {
		t.Fatal(err)
	}
	for _, recovered := range []*BrainRuntime{genesis, suffix} {
		assertCompleteContinuation(t, r, recovered)
		if !reflect.DeepEqual(want, requireRuntimeSpelling(t, recovered, e)) {
			t.Fatal("recovery changed numeric decoding")
		}
	}
	final := filepath.Join(t.TempDir(), "final.json")
	if err := r.Checkpoint(final); err != nil {
		t.Fatal(err)
	}
	restored := NewBrainRuntime(knowledge.NewBrain())
	if err := restored.RestoreCheckpoint(final); err != nil {
		t.Fatal(err)
	}
	assertCompleteContinuation(t, r, restored)
	requireRuntimeSpelling(t, restored, e)
	for _, candidate := range []*BrainRuntime{r, genesis, suffix, restored} {
		candidate.SetClock(NewFixedClock(at.Add(5 * time.Second)))
	}
	wantThought, err := r.CognitiveThink(1)
	if err != nil {
		t.Fatal(err)
	}
	for _, candidate := range []*BrainRuntime{genesis, suffix, restored} {
		got, err := candidate.CognitiveThink(1)
		if err != nil || !reflect.DeepEqual(wantThought, got) {
			t.Fatalf("codec recovery changed internal thought: %v", err)
		}
		requireRuntimeSpelling(t, candidate, e)
	}
	events, err := ReadEventLog(log.file.Name())
	if err != nil {
		t.Fatal(err)
	}
	for _, event := range events {
		if event.Type != EventTypeSpellingExposure {
			continue
		}
		packet, err := json.Marshal(event)
		if err != nil || bytes.Contains(packet, []byte("lantai")) || bytes.Contains(packet, []byte("🌌")) {
			t.Fatal("typed codec journal stored a spelling string")
		}
	}
}

func TestLearnedSpellingRejectsInvalidInputBeforeAppendOrMutation(t *testing.T) {
	r := NewBrainRuntime(knowledge.NewBrain())
	log := openKnowledgeJournal(t)
	r.SetEventLog(log)
	at := time.Unix(100, 0).UTC()
	e := runtimeSpellingExposure(t, r, at)
	before := continuationBytes(t, r)
	journal, err := os.ReadFile(log.file.Name())
	if err != nil {
		t.Fatal(err)
	}
	cases := map[string]func(*knowledge.SpellingExposure){
		"codec": func(e *knowledge.SpellingExposure) { e.Codec = "future" },
		"empty": func(e *knowledge.SpellingExposure) { e.Population = nil },
		"duplicate": func(e *knowledge.SpellingExposure) {
			e.Population = []knowledge.NodeID{e.Population[0], e.Population[0]}
		},
		"missing":    func(e *knowledge.SpellingExposure) { e.Population = []knowledge.NodeID{999999} },
		"scalar":     func(e *knowledge.SpellingExposure) { e.Scalars = []uint32{0xd800} },
		"length":     func(e *knowledge.SpellingExposure) { e.Scalars = make([]uint32, knowledge.MaxSpellingScalars+1) },
		"weight":     func(e *knowledge.SpellingExposure) { e.Weight = math.NaN() },
		"confidence": func(e *knowledge.SpellingExposure) { e.Confidence = 0 },
	}
	for name, mutate := range cases {
		t.Run(name, func(t *testing.T) {
			bad := e
			mutate(&bad)
			if _, err := r.LearnSpelling(bad, at); err == nil {
				t.Fatal("invalid exposure accepted")
			}
			afterJournal, err := os.ReadFile(log.file.Name())
			if err != nil || !bytes.Equal(journal, afterJournal) || !bytes.Equal(before, continuationBytes(t, r)) || r.RecoveryRequired() {
				t.Fatal("preflight failure changed state/journal or retired the runtime")
			}
		})
	}
}

func TestLearnedSpellingUncertainAppendRetiresAndRecovers(t *testing.T) {
	for _, committed := range []bool{false, true} {
		t.Run(map[bool]string{false: "absent", true: "durable-lost-ack"}[committed], func(t *testing.T) {
			r := NewBrainRuntime(knowledge.NewBrain())
			log := openKnowledgeJournal(t)
			r.SetEventLog(log)
			at := time.Unix(100, 0).UTC()
			e := runtimeSpellingExposure(t, r, at)
			before := continuationBytes(t, r)
			log.beforeAppend = func(event LoggedEvent) error {
				if committed {
					data, err := json.Marshal(event)
					if err != nil {
						return err
					}
					if _, err := log.file.Write(append(data, '\n')); err != nil {
						return err
					}
					if err := log.file.Sync(); err != nil {
						return err
					}
				}
				return errors.New("uncertain write acknowledgment")
			}
			if _, err := r.LearnSpelling(e, at); !errors.Is(err, ErrRuntimeRecoveryRequired) {
				t.Fatalf("append failure did not retire runtime: %v", err)
			}
			if !bytes.Equal(before, continuationBytes(t, r)) {
				t.Fatal("codec mutated before the failed append")
			}
			if _, err := r.ReadLearnedSpelling(e.Population); !errors.Is(err, ErrRuntimeRecoveryRequired) {
				t.Fatal("retired codec read remained available")
			}
			log.beforeAppend = nil
			recovered, err := RebuildRuntimeFromCommittedJournal(log, knowledge.NewBrain)
			if err != nil {
				t.Fatal(err)
			}
			if !committed {
				if _, err := recovered.LearnSpelling(e, at); err != nil {
					t.Fatal(err)
				}
			} else if recovered.LastSequence() != 2 {
				t.Fatal("durable codec exposure was not recovered exactly once")
			}
			requireRuntimeSpelling(t, recovered, e)
		})
	}
}

func TestLearnedSpellingCompoundJournalAndOwnerConcurrency(t *testing.T) {
	at := time.Unix(100, 0).UTC()
	r := NewBrainRuntime(knowledge.NewBrain())
	log := openKnowledgeJournal(t)
	r.SetEventLog(log)
	e := runtimeSpellingExposure(t, r, at)
	owner, err := NewRuntimeSupervisor(r)
	if err != nil {
		t.Fatal(err)
	}
	defer owner.Close()
	var wg sync.WaitGroup
	for i := 0; i < 4; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if err := owner.WithMutation(func(active *BrainRuntime) error { _, err := active.LearnSpelling(e, at); return err }); err != nil {
				t.Error(err)
			}
		}()
	}
	wg.Wait()
	if records, err := ReadEventLog(log.file.Name()); err != nil || len(records) != 5 {
		t.Fatalf("concurrent learning lost the grounded prefix or a revision: %d, %v", len(records), err)
	}
	if err := owner.With(func(active *BrainRuntime) error { requireRuntimeSpelling(t, active, e); return nil }); err != nil {
		t.Fatal(err)
	}
	base := LoggedEvent{Type: EventTypeSpellingExposure, SchemaVersion: 3, BrainIdentity: BrainIdentity, Sequence: 1, Timestamp: at, SpellingExposure: &e}
	for _, field := range []string{"event", "experience", "vector", "sensor-commit"} {
		bad := base
		switch field {
		case "event":
			bad.Event = &Event{}
		case "experience":
			bad.Type = EventTypeLearn
		case "vector":
			bad.Vector = &knowledge.NeuralVector{Values: []float64{.1}}
		case "sensor-commit":
			bad.Type, bad.IntentHash = EventTypeSensorCommit, "foreign"
		}
		if err := validateJournalPayload(bad); err == nil {
			t.Fatalf("compound codec/%s accepted", field)
		}
	}
}

func TestLearnedSpellingFrozenV1JournalAndNextThought(t *testing.T) {
	var reference struct {
		Codec      string                    `json:"codec"`
		Population []knowledge.NodeID        `json:"population"`
		Spelling   knowledge.SpellingReadout `json:"spelling"`
		Before     string                    `json:"before_thought_sha256"`
		After      string                    `json:"after_thought_sha256"`
		NextOutput string                    `json:"next_thought_sha256"`
		At         time.Time                 `json:"next_thought_at"`
	}
	dir := filepath.Join("testdata", "learned_spelling_v1")
	data, err := os.ReadFile(filepath.Join(dir, "reference.json"))
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(data, &reference); err != nil {
		t.Fatal(err)
	}
	events, err := ReadEventLog(filepath.Join(dir, "events.jsonl"))
	if err != nil || len(events) != 6 || reference.Codec != knowledge.SpellingCodecV1 {
		t.Fatalf("invalid v1 fixture: %v", err)
	}
	r := NewBrainRuntime(knowledge.NewBrain())
	if err := ReplayEventLog(r, events[:5]); err != nil {
		t.Fatal(err)
	}
	digest := func(b []byte) string { h := sha256.Sum256(b); return hex.EncodeToString(h[:]) }
	canonical, err := r.brain.CanonicalJSON()
	if err != nil || digest(canonical) != reference.Before {
		t.Fatal("codec-v1 replay changed its frozen neural substrate")
	}
	spelling, err := r.ReadLearnedSpelling(reference.Population)
	if err != nil || !reflect.DeepEqual(spelling, reference.Spelling) {
		t.Fatal("codec-v1 learned spelling changed")
	}
	r.SetClock(NewFixedClock(reference.At))
	out, err := r.CognitiveThink(1)
	if err != nil {
		t.Fatal(err)
	}
	packet, err := json.Marshal(out)
	if err != nil || digest(packet) != reference.NextOutput {
		t.Fatal("codec-v1 next thought changed")
	}
	canonical, err = r.brain.CanonicalJSON()
	if err != nil || digest(canonical) != reference.After {
		t.Fatal("codec-v1 next neural state changed")
	}
}
