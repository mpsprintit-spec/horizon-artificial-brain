package runtime

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"testing"
	"time"

	"github.com/project-horizon/horizon-core/services/ai/knowledge"
)

func canonicalPropagationCheckpointBytes(t *testing.T, r *BrainRuntime) []byte {
	t.Helper()
	data, err := os.ReadFile(orderedSave(t, r))
	if err != nil {
		t.Fatal(err)
	}
	return data
}

func TestCanonicalPropagationExactRecoveryAndRepeatedNativeThought(t *testing.T) {
	for _, seed := range []int64{7, 17, 47} {
		t.Run(fmt.Sprint(seed), func(t *testing.T) {
			f := newContinuationFixtureRuntime(t, seed, true, NewCanonicalPropagationRuntime())
			f.teach(t, 3)
			middle := orderedSave(t, f.r)
			f.teach(t, 3)
			f.context(t, "object-a", "motion-b")
			want := f.read(t, 8)
			if want.Status != ContinuationComplete || want.Text != "Bola jatuh." {
				t.Fatalf("bounded control failed: %+v", want)
			}
			full := orderedSave(t, f.r)
			log := f.r.eventLog
			journalBefore, err := os.ReadFile(log.file.Name())
			if err != nil {
				t.Fatal(err)
			}
			genesis, err := RebuildRuntimeFromCommittedJournal(log, knowledge.NewBrain)
			if err != nil {
				t.Fatal(err)
			}
			zero, err := RebuildRuntimeFromCheckpointAndJournal(log, f.zero)
			if err != nil {
				t.Fatal(err)
			}
			suffix, err := RebuildRuntimeFromCheckpointAndJournal(log, middle)
			if err != nil {
				t.Fatal(err)
			}
			fullReplay, err := RebuildRuntimeFromCheckpointAndJournal(log, full)
			if err != nil {
				t.Fatal(err)
			}
			restored := NewBrainRuntime(knowledge.NewBrain())
			if err := restored.RestoreCheckpoint(full); err != nil {
				t.Fatal(err)
			}
			candidates := []*BrainRuntime{f.r, genesis, zero, suffix, fullReplay, restored}
			f.r.SetEventLog(nil)
			for _, r := range candidates {
				r.SetClock(NewFixedClock(f.at))
			}
			before := canonicalPropagationCheckpointBytes(t, f.r)
			for _, r := range candidates {
				if r.dynamicsEncoding != knowledge.CanonicalPropagationDynamics {
					t.Fatal("recovery lost the serving propagation law")
				}
				if !bytes.Equal(before, canonicalPropagationCheckpointBytes(t, r)) {
					t.Fatal("complete checkpoint differs before native continuation")
				}
				g := f
				g.r = r
				for i := 0; i < 3; i++ {
					if !reflect.DeepEqual(want, g.read(t, 8)) {
						t.Fatal("recovery changed the read-only continuation")
					}
				}
			}
			journalAfter, err := os.ReadFile(log.file.Name())
			if err != nil || !bytes.Equal(journalBefore, journalAfter) {
				t.Fatal("recovery or preview changed the source journal")
			}
			f.r.SetEventLog(nil)
			for step := 0; step < 8; step++ {
				at := f.at.Add(time.Duration(step+2) * time.Second)
				var output, checkpoint []byte
				for i, r := range candidates {
					r.SetClock(NewFixedClock(at))
					out, err := r.CognitiveThink(1)
					if err != nil {
						t.Fatal(err)
					}
					data, err := json.Marshal(out)
					if err != nil {
						t.Fatal(err)
					}
					state := canonicalPropagationCheckpointBytes(t, r)
					if i == 0 {
						output, checkpoint = data, state
					}
					if !bytes.Equal(output, data) || !bytes.Equal(checkpoint, state) {
						t.Fatalf("native thought %d candidate %d differs bit-for-bit", step, i)
					}
				}
			}
			t.Logf("seed=%d recovery_paths=5 native_steps=8 exact_output_and_full_checkpoint=true", seed)
		})
	}
}

func TestCanonicalPropagationDefaultAndHistoricalLawIsolation(t *testing.T) {
	legacy := NewBrainRuntime(knowledge.NewBrain())
	canonical := NewCanonicalPropagationRuntime()
	canonical.SetClock(NewFixedClock(time.Unix(100, 0).UTC()))
	if knowledge.CurrentDynamics != knowledge.EvidenceProvenanceDynamics || legacy.dynamicsEncoding != knowledge.EvidenceProvenanceDynamics {
		t.Fatal("historical default changed")
	}
	old := orderedSave(t, legacy)
	var oldSnapshot BrainSnapshot
	oldBytes, err := os.ReadFile(old)
	if err != nil || json.Unmarshal(oldBytes, &oldSnapshot) != nil || oldSnapshot.SchemaVersion != 13 || oldSnapshot.DynamicsEncoding != "" {
		t.Fatal("default checkpoint format changed")
	}
	before := canonicalPropagationCheckpointBytes(t, canonical)
	if err := canonical.RestoreCheckpoint(old); err == nil || !bytes.Equal(before, canonicalPropagationCheckpointBytes(t, canonical)) {
		t.Fatal("historical checkpoint silently migrated an opt-in runtime")
	}
	for _, encoding := range []knowledge.DynamicsEncoding{knowledge.LegacyDynamics, knowledge.OrderedDynamics, knowledge.RecurrentDynamics, knowledge.PhysicalMemoryDynamics, knowledge.LocalActivityDynamics, knowledge.ObservedFeedbackDynamics, knowledge.EvidenceUncertaintyDynamics, knowledge.StableProjectionDynamics, knowledge.EvidenceProvenanceDynamics} {
		if encoding.Validate() != nil || encoding.HasCanonicalPropagation() {
			t.Fatal("historical law gained canonical ordering")
		}
	}
	if knowledge.CanonicalPropagationDynamics.Validate() != nil || !knowledge.CanonicalPropagationDynamics.HasEvidenceProvenance() {
		t.Fatal("opt-in law lost inherited behavior")
	}
}

func TestCanonicalPropagationRejectsLawTimelineChangesBeforeMutation(t *testing.T) {
	r := NewCanonicalPropagationRuntime()
	r.SetClock(NewFixedClock(time.Unix(100, 0).UTC()))
	r.SetEventLog(openKnowledgeJournal(t))
	for _, values := range [][]float64{{1, 0}, {0, 1}} {
		if _, err := r.GroundVectorAt(knowledge.NewNeuralVector(values), time.Unix(100, 0)); err != nil {
			t.Fatal(err)
		}
	}
	events, err := ReadEventLog(r.eventLog.file.Name())
	if err != nil {
		t.Fatal(err)
	}
	for _, mode := range []string{"wrong-reader", "mixed-upgrade", "mixed-downgrade"} {
		t.Run(mode, func(t *testing.T) {
			batch := append([]LoggedEvent(nil), events...)
			target := NewCanonicalPropagationRuntime()
			if mode == "wrong-reader" {
				target = NewBrainRuntime(knowledge.NewBrain())
			}
			if mode == "mixed-upgrade" {
				batch[0].DynamicsEncoding = knowledge.EvidenceProvenanceDynamics
			}
			if mode == "mixed-downgrade" {
				batch[1].DynamicsEncoding = knowledge.EvidenceProvenanceDynamics
			}
			target.SetClock(NewFixedClock(time.Unix(100, 0).UTC()))
			before := canonicalPropagationCheckpointBytes(t, target)
			if err := ReplayEventLog(target, batch); err == nil || !bytes.Equal(before, canonicalPropagationCheckpointBytes(t, target)) {
				t.Fatal("law mismatch applied a prefix before rejection")
			}
		})
	}
	if _, err := RebuildRuntimeFromCommittedJournal(r.eventLog, knowledge.NewBootstrapBrain); err == nil {
		t.Fatal("opt-in recovery adopted a different foundation")
	}
}

func TestCanonicalPropagationCheckpointAuthenticatesLawAndRejectsDowngrade(t *testing.T) {
	r := NewCanonicalPropagationRuntime()
	r.SetClock(NewFixedClock(time.Unix(100, 0).UTC()))
	data := canonicalPropagationCheckpointBytes(t, r)
	var snapshot BrainSnapshot
	if err := json.Unmarshal(data, &snapshot); err != nil {
		t.Fatal(err)
	}
	if snapshot.SchemaVersion != 14 || snapshot.DynamicsEncoding != knowledge.CanonicalPropagationDynamics {
		t.Fatal("checkpoint omitted explicit propagation law")
	}
	want, err := checkpointChecksumV9(snapshot)
	if err != nil || want != snapshot.Checksum {
		t.Fatal("streaming and materialized checksums differ")
	}
	altered := snapshot
	altered.DynamicsEncoding = knowledge.EvidenceProvenanceDynamics
	got, err := checkpointChecksumV9(altered)
	if err != nil || got == want {
		t.Fatal("checksum does not cover the propagation law")
	}
	for _, mutate := range []func(*BrainSnapshot){
		func(s *BrainSnapshot) { s.DynamicsEncoding = "future" },
		func(s *BrainSnapshot) { s.DynamicsEncoding = "" },
		func(s *BrainSnapshot) { s.SchemaVersion = 13; s.Checksum, _ = checkpointChecksumV8(*s) },
		func(s *BrainSnapshot) { s.Checksum = "invalid" },
	} {
		x := snapshot
		mutate(&x)
		path := filepath.Join(t.TempDir(), "invalid.json")
		b, err := json.Marshal(x)
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, b, 0600); err != nil {
			t.Fatal(err)
		}
		target := NewBrainRuntime(knowledge.NewBrain())
		target.SetClock(NewFixedClock(time.Unix(100, 0).UTC()))
		before := canonicalPropagationCheckpointBytes(t, target)
		if err := target.RestoreCheckpoint(path); err == nil || !bytes.Equal(before, canonicalPropagationCheckpointBytes(t, target)) {
			t.Fatal("malformed law/checksum changed live state")
		}
	}
}

func TestCanonicalPropagationAppendRejectsCrossingLawsIncludingReopen(t *testing.T) {
	for _, canonicalFirst := range []bool{false, true} {
		t.Run(fmt.Sprint(canonicalFirst), func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "events.jsonl")
			log, err := OpenEventLog(path)
			if err != nil {
				t.Fatal(err)
			}
			first, second := knowledge.EvidenceProvenanceDynamics, knowledge.CanonicalPropagationDynamics
			if canonicalFirst {
				first, second = second, first
			}
			event := LoggedEvent{SchemaVersion: 3, BrainIdentity: BrainIdentity, Sequence: 1, Type: EventTypeVectorGround, Timestamp: time.Unix(100, 0).UTC(), Vector: &knowledge.NeuralVector{Values: []float64{1, 0}}, DynamicsEncoding: first}
			if err := log.Append(event); err != nil {
				t.Fatal(err)
			}
			before, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			for _, reopen := range []bool{false, true} {
				if reopen {
					if err := log.Close(); err != nil {
						t.Fatal(err)
					}
					log, err = OpenEventLog(path)
					if err != nil {
						t.Fatal(err)
					}
				}
				event.Sequence, event.DynamicsEncoding = 2, second
				if err := log.Append(event); err == nil {
					t.Fatal("append accepted a propagation migration")
				}
				after, err := os.ReadFile(path)
				if err != nil || !bytes.Equal(before, after) {
					t.Fatal("rejected append changed durable bytes")
				}
			}
			defer log.Close()
			event.DynamicsEncoding = first
			if err := log.Append(event); err != nil {
				t.Fatal("matching law could not continue after rejection", err)
			}
		})
	}
}

func TestCanonicalPropagationCheckpointAndJournalMustHaveSameLaw(t *testing.T) {
	for _, canonicalCheckpoint := range []bool{false, true} {
		t.Run(fmt.Sprint(canonicalCheckpoint), func(t *testing.T) {
			checkpointRuntime, writer := NewBrainRuntime(knowledge.NewBrain()), NewCanonicalPropagationRuntime()
			if canonicalCheckpoint {
				checkpointRuntime, writer = writer, checkpointRuntime
			}
			checkpointRuntime.SetClock(NewFixedClock(time.Unix(100, 0).UTC()))
			log := openKnowledgeJournal(t)
			checkpointRuntime.SetEventLog(log)
			cp := orderedSave(t, checkpointRuntime)
			checkpointRuntime.SetEventLog(nil)
			writer.SetEventLog(log)
			if _, err := writer.GroundVectorAt(knowledge.NewNeuralVector([]float64{1, 0}), time.Unix(100, 0)); err != nil {
				t.Fatal(err)
			}
			before, err := os.ReadFile(log.file.Name())
			if err != nil {
				t.Fatal(err)
			}
			if _, err := RebuildRuntimeFromCheckpointAndJournal(log, cp); err == nil {
				t.Fatal("zero checkpoint accepted a different law from its journal")
			}
			after, err := os.ReadFile(log.file.Name())
			if err != nil || !bytes.Equal(before, after) {
				t.Fatal("rejected law recovery changed the journal")
			}
		})
	}
}

func TestCanonicalPropagationWrittenMigrationRefusesHistoricalBackup(t *testing.T) {
	r := NewCanonicalPropagationRuntime()
	r.SetClock(NewFixedClock(time.Unix(100, 0).UTC()))
	before := canonicalPropagationCheckpointBytes(t, r)
	if _, err := r.PlanNeuralWrittenGrounding(); err == nil {
		t.Fatal("plan offered an unsupported historical backup")
	}
	path := filepath.Join(t.TempDir(), "backup.json")
	if err := r.ApplyNeuralWrittenGrounding(WrittenGroundingPlan{}, path, time.Unix(100, 0)); err == nil {
		t.Fatal("migration accepted a propagation downgrade")
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) || !bytes.Equal(before, canonicalPropagationCheckpointBytes(t, r)) {
		t.Fatal("rejected migration created a backup or changed the Brain")
	}
}

func TestCanonicalPropagationRejectsLegacyMutableImports(t *testing.T) {
	r := NewCanonicalPropagationRuntime()
	r.SetClock(NewFixedClock(time.Unix(100, 0).UTC()))
	before := canonicalPropagationCheckpointBytes(t, r)
	legacy := NewBrainRuntime(knowledge.NewBrain())
	path := filepath.Join(t.TempDir(), "graph.json")
	if err := legacy.SaveBrain(path); err != nil {
		t.Fatal(err)
	}
	if err := r.LoadBrain(path); err == nil {
		t.Fatal("unversioned graph import adopted an unrecorded law")
	}
	if err := r.RestoreEventIndex(nil); err == nil {
		t.Fatal("index-only recovery bypassed the propagation contract")
	}
	if r.DNF() != nil {
		t.Fatal("opt-in runtime exposed unversioned activation")
	}
	if err := r.WithBrainView(func(*knowledge.Brain) error { t.Error("mutable legacy callback ran"); return nil }); err == nil {
		t.Fatal("opt-in runtime exposed legacy mutable view")
	}
	if !bytes.Equal(before, canonicalPropagationCheckpointBytes(t, r)) {
		t.Fatal("rejected legacy import changed state")
	}
}

func TestCanonicalPropagationSupervisorHandoverKeepsLaw(t *testing.T) {
	at := time.Unix(100, 0).UTC()
	original := NewCanonicalPropagationRuntime()
	original.SetClock(NewFixedClock(at))
	owner, err := NewRuntimeSupervisor(original)
	if err != nil {
		t.Fatal(err)
	}
	defer owner.Close()
	log := openKnowledgeJournal(t)
	if err := owner.AttachEventLog(log); err != nil {
		t.Fatal(err)
	}
	cp := filepath.Join(t.TempDir(), "prefix.json")
	if err := owner.WithMutation(func(r *BrainRuntime) error {
		if _, err := r.GroundVectorAt(knowledge.NewNeuralVector([]float64{1, 0}), at); err != nil {
			return err
		}
		return r.Checkpoint(cp)
	}); err != nil {
		t.Fatal(err)
	}
	if err := owner.WithMutation(func(r *BrainRuntime) error {
		_, err := r.GroundVectorAt(knowledge.NewNeuralVector([]float64{0, 1}), at)
		return err
	}); err != nil {
		t.Fatal(err)
	}
	if err := owner.RecoverFromCheckpoint(log, cp); err != nil {
		t.Fatal(err)
	}
	if !original.RecoveryRequired() {
		t.Fatal("handover did not retire old owner")
	}
	if err := owner.WithMutation(func(r *BrainRuntime) error {
		if r.dynamicsEncoding != knowledge.CanonicalPropagationDynamics || r.LastSequence() != 2 {
			t.Fatal("handover changed the law or suffix revision")
		}
		_, err := r.CognitiveThink(2)
		return err
	}); err != nil {
		t.Fatal(err)
	}
	events, err := ReadEventLog(log.file.Name())
	if err != nil || len(events) != 3 {
		t.Fatal("continued owner journal invalid", err)
	}
	for _, event := range events {
		if event.DynamicsEncoding != knowledge.CanonicalPropagationDynamics {
			t.Fatal("handover wrote a historical serving law")
		}
	}
}

func TestCanonicalPropagationOrphanOnlyJournalClearsCachedLaw(t *testing.T) {
	log := openKnowledgeJournal(t)
	intent := sensorJournalTestIntent()
	intent.DynamicsEncoding = knowledge.CanonicalPropagationDynamics
	if err := log.Append(intent); err != nil {
		t.Fatal(err)
	}
	before, err := os.ReadFile(log.file.Name())
	if err != nil {
		t.Fatal(err)
	}
	backup, err := log.RecoverTrailingSensorIntent()
	if err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(backup)
	if err != nil || !bytes.Equal(before, data) {
		t.Fatal("orphan backup was not exact")
	}
	// No committed record remains to select a law. A fresh historical runtime
	// may start its own journal; the removed intent must not pin a cached law.
	r := NewBrainRuntime(knowledge.NewBrain())
	r.SetEventLog(log)
	if _, err := r.GroundVectorAt(knowledge.NewNeuralVector([]float64{1, 0}), time.Unix(100, 0)); err != nil {
		t.Fatal("empty repaired journal retained its removed law", err)
	}
	events, err := ReadEventLog(log.file.Name())
	if err != nil || len(events) != 1 || events[0].DynamicsEncoding != knowledge.CurrentDynamics {
		t.Fatal("repaired journal has the wrong new law", err)
	}
}

func TestCanonicalPropagationRejectsWrongFullPrefixWithValidDigests(t *testing.T) {
	r := NewCanonicalPropagationRuntime()
	r.SetClock(NewFixedClock(time.Unix(100, 0).UTC()))
	log := openKnowledgeJournal(t)
	r.SetEventLog(log)
	if _, err := r.GroundVectorAt(knowledge.NewNeuralVector([]float64{1, 0}), time.Unix(100, 0)); err != nil {
		t.Fatal(err)
	}
	cp := orderedSave(t, r)
	data, err := os.ReadFile(cp)
	if err != nil {
		t.Fatal(err)
	}
	var snapshot BrainSnapshot
	if err := json.Unmarshal(data, &snapshot); err != nil {
		t.Fatal(err)
	}
	events, err := ReadEventLog(log.file.Name())
	if err != nil {
		t.Fatal(err)
	}
	events[0].DynamicsEncoding = knowledge.EvidenceProvenanceDynamics
	snapshot.JournalPrefixHash, err = journalPrefixHash(events)
	if err != nil {
		t.Fatal(err)
	}
	snapshot.Checksum, err = checkpointChecksumV9(snapshot)
	if err != nil {
		t.Fatal(err)
	}
	data, err = json.Marshal(snapshot)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(cp, data, 0600); err != nil {
		t.Fatal(err)
	}
	data, err = json.Marshal(events[0])
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(log.file.Name(), append(data, '\n'), 0600); err != nil {
		t.Fatal(err)
	}
	// Both digests are valid and the suffix is empty. The law in the full
	// prefix must still agree with the complete checkpoint serving law.
	if _, err := RebuildRuntimeFromCheckpointAndJournal(log, cp); err == nil {
		t.Fatal("matching hashes concealed a propagation-law mismatch")
	}
}
