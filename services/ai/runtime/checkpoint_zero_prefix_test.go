package runtime

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/project-horizon/horizon-core/services/ai/knowledge"
	"github.com/project-horizon/horizon-core/services/ai/learning"
)

// ReadEventLog has always returned nil for a valid empty journal. Keep the
// digest already authenticated by historical checkpoint checksums.
const emptyJournalDigest = "74234e98afe7498fb5daf1f36ac2d78acc339464f950703b8c019892f982b90b"

func TestJournalPrefixHashCanonicalEmptyAndUnchangedNonempty(t *testing.T) {
	events := []LoggedEvent{{Type: EventTypeCognitiveThink, Sequence: 1}}
	for name, prefix := range map[string][]LoggedEvent{
		"nil": nil, "allocated": {}, "reserved": make([]LoggedEvent, 0, 8), "slice-of-nonempty": events[:0],
	} {
		t.Run(name, func(t *testing.T) {
			got, err := journalPrefixHash(prefix)
			if err != nil || got != emptyJournalDigest {
				t.Fatalf("empty history digest: got %q, %v", got, err)
			}
		})
	}
	data, err := json.Marshal(events)
	if err != nil {
		t.Fatal(err)
	}
	sum := sha256.Sum256(data)
	got, err := journalPrefixHash(events)
	if err != nil || got != hex.EncodeToString(sum[:]) || got == emptyJournalDigest {
		t.Fatalf("nonempty history hash law changed: %q, %v", got, err)
	}
}

func zeroPrefixLesson() learning.Experience {
	return learning.Experience{Sequence: []string{"naku", "melo", "tiva"}, Source: "zero-prefix-test", Weight: .9, Confidence: .9}
}

func TestCheckpointJournalRecoveryFromRevisionZero(t *testing.T) {
	for name, factory := range map[string]func() *knowledge.Brain{
		"empty": knowledge.NewBrain, "bootstrap": knowledge.NewBootstrapBrain,
	} {
		t.Run(name, func(t *testing.T) {
			log := openKnowledgeJournal(t)
			live := NewBrainRuntime(factory())
			at := time.Unix(100, 0).UTC()
			live.SetClock(NewFixedClock(at))
			live.SetEventLog(log)
			path := filepath.Join(t.TempDir(), "zero.json")
			if err := live.Checkpoint(path); err != nil {
				t.Fatal(err)
			}
			original, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			snapshot, err := readCheckpointForTest(path)
			if err != nil || snapshot.Sequence != 0 || snapshot.JournalPrefixHash != emptyJournalDigest {
				t.Fatalf("unexpected initial checkpoint: %+v, %v", snapshot, err)
			}
			for step := 0; step < 3; step++ {
				if step > 0 {
					if _, err := live.LearnExperience(zeroPrefixLesson(), at.Add(time.Duration(step)*time.Second)); err != nil {
						t.Fatal(err)
					}
				}
				beforeJournal, err := os.ReadFile(log.file.Name())
				if err != nil {
					t.Fatal(err)
				}
				for retry := 0; retry < 2; retry++ {
					rebuilt, err := RebuildRuntimeFromCheckpointAndJournal(log, path)
					if err != nil {
						t.Fatalf("zero checkpoint, %d appended events: %v", step, err)
					}
					if rebuilt.eventLog != nil {
						t.Fatal("recovery published a writer")
					}
					assertCompleteContinuation(t, live, rebuilt)
				}
				afterJournal, err := os.ReadFile(log.file.Name())
				if err != nil || !bytes.Equal(beforeJournal, afterJournal) {
					t.Fatal("repeated recovery changed the journal")
				}
			}
			after, err := os.ReadFile(path)
			if err != nil || !bytes.Equal(original, after) {
				t.Fatal("recovery rewrote the authenticated checkpoint")
			}
		})
	}
}

func TestZeroPrefixRecoveryRetainsStrictAnchorAndChecksumValidation(t *testing.T) {
	for _, nonempty := range []bool{false, true} {
		for _, kind := range []string{"wrong-anchor", "array-empty-anchor", "wrong-checksum"} {
			t.Run(kind+map[bool]string{false: "/empty", true: "/appended"}[nonempty], func(t *testing.T) {
				log := openKnowledgeJournal(t)
				live := NewBrainRuntime(knowledge.NewBrain())
				live.SetEventLog(log)
				path := filepath.Join(t.TempDir(), "zero.json")
				if err := live.Checkpoint(path); err != nil {
					t.Fatal(err)
				}
				snapshot, err := readCheckpointForTest(path)
				if err != nil {
					t.Fatal(err)
				}
				if nonempty {
					if _, err := live.LearnExperience(zeroPrefixLesson(), time.Unix(100, 0).UTC()); err != nil {
						t.Fatal(err)
					}
				}
				wantError := "prefix hash mismatch"
				switch kind {
				case "wrong-anchor":
					snapshot.JournalPrefixHash = strings.Repeat("0", 64)
				case "array-empty-anchor":
					// Historical writers emitted SHA256(null), never SHA256([]).
					snapshot.JournalPrefixHash = "4f53cda18c2baa0c0354bb5f9a3ecbe5ed12ab4d8e11ba873c2f11161202b945"
				case "wrong-checksum":
					wantError = "checksum mismatch"
				}
				snapshot.Checksum, err = checkpointChecksumForSchema(snapshot)
				if err != nil {
					t.Fatal(err)
				}
				if kind == "wrong-checksum" {
					snapshot.Checksum = strings.Repeat("0", 64)
				}
				data, err := json.Marshal(snapshot)
				if err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(path, data, 0600); err != nil {
					t.Fatal(err)
				}
				before := continuationBytes(t, live)
				rebuilt, err := RebuildRuntimeFromCheckpointAndJournal(log, path)
				if rebuilt != nil || err == nil || !strings.Contains(err.Error(), wantError) {
					t.Fatalf("invalid boundary accepted: %v", err)
				}
				if !bytes.Equal(before, continuationBytes(t, live)) || live.RecoveryRequired() {
					t.Fatal("failed isolated recovery changed the live runtime")
				}
			})
		}
	}
}

func TestCheckpointJournalNonemptyPrefixStillRejectsChangedHistory(t *testing.T) {
	log := openKnowledgeJournal(t)
	live := NewBrainRuntime(knowledge.NewBrain())
	live.SetEventLog(log)
	path := filepath.Join(t.TempDir(), "one.json")
	at := time.Unix(100, 0).UTC()
	if _, err := live.LearnExperience(zeroPrefixLesson(), at); err != nil {
		t.Fatal(err)
	}
	if err := live.Checkpoint(path); err != nil {
		t.Fatal(err)
	}
	if _, err := live.LearnExperience(zeroPrefixLesson(), at.Add(time.Second)); err != nil {
		t.Fatal(err)
	}
	snapshot, err := readCheckpointForTest(path)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := RebuildRuntimeFromCheckpointAndJournal(log, path); err != nil {
		t.Fatalf("nonempty control failed: %v", err)
	}
	t.Run("changed-committed-prefix", func(t *testing.T) {
		events, err := ReadEventLog(log.file.Name())
		if err != nil {
			t.Fatal(err)
		}
		events[0].Experience.Weight = .1
		changed := openKnowledgeJournal(t)
		for _, event := range events {
			if err := json.NewEncoder(changed.file).Encode(event); err != nil {
				t.Fatal(err)
			}
		}
		if _, err := ReadEventLog(changed.file.Name()); err != nil {
			t.Fatalf("test needs a valid committed journal: %v", err)
		}
		if rebuilt, err := RebuildRuntimeFromCheckpointAndJournal(changed, path); rebuilt != nil || err == nil || !strings.Contains(err.Error(), "prefix hash mismatch") {
			t.Fatalf("changed history accepted: %v", err)
		}
	})
	t.Run("empty-anchor-at-nonzero-sequence", func(t *testing.T) {
		snapshot.JournalPrefixHash = emptyJournalDigest
		snapshot.Checksum, err = checkpointChecksumForSchema(snapshot)
		if err != nil {
			t.Fatal(err)
		}
		data, err := json.Marshal(snapshot)
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, data, 0600); err != nil {
			t.Fatal(err)
		}
		if rebuilt, err := RebuildRuntimeFromCheckpointAndJournal(log, path); rebuilt != nil || err == nil || !strings.Contains(err.Error(), "prefix hash mismatch") {
			t.Fatalf("nonempty prefix accepted an empty-history digest: %v", err)
		}
	})
}

func TestZeroPrefixSupervisorHandoverPreservesLearnedSpelling(t *testing.T) {
	log := openKnowledgeJournal(t)
	live := NewBrainRuntime(knowledge.NewBrain())
	live.SetEventLog(log)
	owner, err := NewRuntimeSupervisor(live)
	if err != nil {
		t.Fatal(err)
	}
	defer owner.Close()
	path := filepath.Join(t.TempDir(), "zero.json")
	var exposure knowledge.SpellingExposure
	at := time.Unix(100, 0).UTC()
	if err := owner.WithMutation(func(r *BrainRuntime) error {
		if err := r.Checkpoint(path); err != nil {
			return err
		}
		exposure = runtimeSpellingExposure(t, r, at)
		_, err := r.LearnSpelling(exposure, at.Add(time.Second))
		return err
	}); err != nil {
		t.Fatal(err)
	}
	before := continuationBytes(t, live)
	want := requireRuntimeSpelling(t, live, exposure)
	if err := owner.RecoverFromCheckpoint(log, path); err != nil {
		t.Fatal(err)
	}
	if !live.RecoveryRequired() || live.eventLog != nil {
		t.Fatal("handover did not retire the old runtime")
	}
	if err := owner.WithMutation(func(r *BrainRuntime) error {
		if !bytes.Equal(before, continuationBytes(t, r)) || r.eventLog != log {
			t.Fatal("handover lost learned state or writer")
		}
		if !reflect.DeepEqual(want, requireRuntimeSpelling(t, r, exposure)) {
			t.Fatal("handover changed spelling")
		}
		_, err := r.LearnSpelling(exposure, at.Add(2*time.Second))
		return err
	}); err != nil {
		t.Fatal(err)
	}
	if _, err := live.LearnSpelling(exposure, at); !errors.Is(err, ErrRuntimeRecoveryRequired) {
		t.Fatal("retired writer accepted another lesson")
	}
	events, err := ReadEventLog(log.file.Name())
	if err != nil || len(events) != 3 || events[2].Sequence != 3 {
		t.Fatalf("resumed writer lost or duplicated a lesson: %d, %v", len(events), err)
	}
}
