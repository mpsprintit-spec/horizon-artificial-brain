package runtime

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"math/rand"
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/project-horizon/horizon-core/services/ai/knowledge"
	"github.com/project-horizon/horizon-core/services/ai/learning"
)

var numericWrittenAt = time.Date(2026, 10, 21, 1, 0, 0, 0, time.UTC)

func numericWrittenFixture(t *testing.T, seed int64, journal bool) (*BrainRuntime, map[string]knowledge.NodeID, *EventLog) {
	t.Helper()
	r := NewBrainRuntime(knowledge.NewBrain())
	r.SetClock(NewFixedClock(numericWrittenAt))
	var log *EventLog
	if journal {
		log = openKnowledgeJournal(t)
		r.SetEventLog(log)
	}
	words := []string{"naku", "melo", "tiva", "sopo", "keri", "luma"}
	rand.New(rand.NewSource(seed)).Shuffle(len(words), func(i, j int) { words[i], words[j] = words[j], words[i] })
	for _, word := range words {
		if _, err := r.LearnExperience(learning.Experience{Sequence: []string{word}}, numericWrittenAt); err != nil {
			t.Fatal(err)
		}
	}
	for i := 0; i < 4; i++ {
		for _, sentence := range []string{"naku melo tiva", "sopo keri luma"} {
			if _, err := r.LearnExperience(learning.Experience{Sequence: strings.Fields(sentence), Weight: .9, Confidence: .9, Source: "synthetic-teacher"}, numericWrittenAt); err != nil {
				t.Fatal(err)
			}
		}
	}
	ids := make(map[string]knowledge.NodeID)
	for _, word := range words {
		ids[word] = r.brain.Fetch(word).ID
	}
	return r, ids, log
}

func enableNumericWritten(t *testing.T, r *BrainRuntime) string {
	t.Helper()
	plan, err := r.PlanNeuralWrittenGrounding()
	if err != nil {
		t.Fatal(err)
	}
	backup := filepath.Join(t.TempDir(), "pre-migration.json")
	if err := r.ApplyNeuralWrittenGrounding(plan, backup, numericWrittenAt); err != nil {
		t.Fatal(err)
	}
	return backup
}

func numericWrittenQuery(t *testing.T, r *BrainRuntime, prompt string) CognitiveInterpretation {
	t.Helper()
	out, _, err := NewCognitiveOrchestrator(r).ProcessObservation(
		Event{ID: "numeric-written-query", Timestamp: numericWrittenAt.Add(time.Second), Cycles: 1},
		ObservationInput{Source: "diagnostic", Modality: "text", Tokens: strings.Fields(prompt)}, nil)
	if err != nil {
		t.Fatal(err)
	}
	return out
}

func scrambleAnnotations(t *testing.T, r *BrainRuntime, variant int) {
	t.Helper()
	annotations := r.brain.SurfaceAnnotationsSnapshot()
	switch variant {
	case 1:
		annotations = nil
	case 2:
		for i := range annotations {
			annotations[i].Surface = "wrong-" + annotations[i].Surface
			annotations[i].NodeID = annotations[(i+1)%len(annotations)].Population[0]
			annotations[i].Population = []knowledge.NodeID{annotations[i].NodeID}
		}
	}
	if err := r.brain.RestoreSurfaceAnnotations(annotations); err != nil {
		t.Fatal(err)
	}
}

func TestNumericWrittenGroundingIsIndependentOfAdapterLabelsAndMemberDrift(t *testing.T) {
	for _, seed := range []int64{7, 17, 47} {
		r, ids, _ := numericWrittenFixture(t, seed, false)
		// Preserve legacy identity even when the learned vectors have moved.
		for _, node := range r.brain.Registry.Nodes() {
			for i := range node.Representation {
				node.Representation[i] = .9*node.Representation[i] + .05
			}
		}
		before := r.brain.Registry.Nodes()
		beforeBytes, _ := json.Marshal(before)
		annotations := r.brain.SurfaceAnnotationsSnapshot()
		backup := enableNumericWritten(t, r)
		afterBytes, _ := json.Marshal(r.brain.Registry.Nodes())
		if !bytes.Equal(beforeBytes, afterBytes) || !reflect.DeepEqual(annotations, r.brain.SurfaceAnnotationsSnapshot()) {
			t.Fatal("migration changed existing neural graph or removed labels")
		}
		legacy := orderedFixtureClone(t, backup)
		if legacy.neuralWrittenGrounding {
			t.Fatal("backup did not preserve the pre-migration contract")
		}
		checkpoint := orderedSave(t, r)
		for _, prompt := range []struct{ cue, target string }{{"naku melo", "tiva"}, {"sopo keri", "luma"}} {
			var reference CognitiveInterpretation
			var nextReference CognitiveOutput
			var hashReference string
			for variant := 0; variant < 3; variant++ {
				clone := orderedFixtureClone(t, checkpoint)
				scrambleAnnotations(t, clone, variant)
				out := numericWrittenQuery(t, clone, prompt.cue)
				if out.State.Activations[ids[prompt.target]] <= 0 || !slices.Contains(out.Answer.NodeIDs, ids[prompt.target]) {
					t.Fatalf("seed %d variant %d lost learned target %s", seed, variant, prompt.target)
				}
				for i, word := range strings.Fields(prompt.cue) {
					if out.GroundedRepresentations[i].NodeID != ids[word] || len(out.GroundedRepresentations[i].Population) != 1 {
						t.Fatal("numeric prototype lost the existing learned population")
					}
				}
				if len(clone.brain.Registry.Nodes()) != len(before) {
					t.Fatal("label ablation duplicated units")
				}
				clone.SetClock(NewFixedClock(numericWrittenAt.Add(2 * time.Second)))
				next, err := clone.CognitiveThink(1)
				if err != nil {
					t.Fatal(err)
				}
				hash, err := clone.brain.CanonicalHash()
				if err != nil {
					t.Fatal(err)
				}
				if variant == 0 {
					reference, nextReference, hashReference = out, next, hash
				} else if !reflect.DeepEqual(reference, out) || !reflect.DeepEqual(nextReference, next) || hashReference != hash {
					t.Fatal("adapter labels changed neural inference, subsequent thought or canonical state")
				}
			}
		}
	}
}

func TestLegacyWrittenGroundingLabelAblationExposesTheOriginalDependency(t *testing.T) {
	r, _, _ := numericWrittenFixture(t, 7, false)
	for _, node := range r.brain.Registry.Nodes() {
		for i := range node.Representation {
			node.Representation[i] = .9*node.Representation[i] + .05
		}
	}
	path := orderedSave(t, r)
	present, hidden := orderedFixtureClone(t, path), orderedFixtureClone(t, path)
	scrambleAnnotations(t, hidden, 1)
	numericWrittenQuery(t, present, "naku melo")
	numericWrittenQuery(t, hidden, "naku melo")
	if len(hidden.brain.Registry.Nodes()) <= len(present.brain.Registry.Nodes()) {
		t.Fatal("control did not reproduce old label-dependent population growth")
	}
	t.Logf("legacy units: labels=%d hidden=%d; this control is not a semantic accuracy test", len(present.brain.Registry.Nodes()), len(hidden.brain.Registry.Nodes()))
}

func TestNumericWrittenMigrationAndSuffixRecoverWithTheSameNextThought(t *testing.T) {
	r, _, log := numericWrittenFixture(t, 17, true)
	backup := enableNumericWritten(t, r)
	migrated := orderedSave(t, r)
	if _, err := r.LearnExperience(learning.Experience{Sequence: []string{"tiva", "new-codec-observation"}, Weight: .7, Confidence: .8}, numericWrittenAt); err != nil {
		t.Fatal(err)
	}
	numericWrittenQuery(t, r, "naku melo")
	events, err := ReadEventLog(log.file.Name())
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, e := range events {
		if e.Type == EventTypeWrittenGroundingMigration {
			found = true
			data, err := os.ReadFile(backup)
			if err != nil {
				t.Fatal(err)
			}
			hash := sha256.Sum256(data)
			if e.WrittenGroundingMigration.BackupHash != "sha256:"+hex.EncodeToString(hash[:]) {
				t.Fatal("migration receipt does not bind the backup bytes")
			}
			journalHash := sha256.Sum256(mustReadWrittenFile(t, backup+".events.jsonl"))
			if e.WrittenGroundingMigration.BackupJournalHash != "sha256:"+hex.EncodeToString(journalHash[:]) {
				t.Fatal("migration receipt does not bind the old journal prefix")
			}
			prefix, err := ReadEventLog(backup + ".events.jsonl")
			if err != nil || uint64(len(prefix)) != e.WrittenGroundingMigration.Plan.ReferenceRevision {
				t.Fatal("rollback journal is not the pre-migration prefix", err)
			}
		} else if found && e.SurfaceEncoding != knowledge.NeuralWrittenEncoding {
			t.Fatal("new written command lost its versioned grounding law")
		}
	}
	if !found {
		t.Fatal("migration was not journaled")
	}
	for _, source := range []string{"genesis", backup, migrated} {
		var recovered *BrainRuntime
		if source == "genesis" {
			recovered, err = RebuildRuntimeFromCommittedJournal(log, knowledge.NewBrain)
		} else {
			recovered, err = RebuildRuntimeFromCheckpointAndJournal(log, source)
		}
		if err != nil {
			t.Fatal(source, err)
		}
		if !recovered.neuralWrittenGrounding || recovered.surfaceEncoding != knowledge.NeuralWrittenEncoding {
			t.Fatal("recovery reverted to label-based grounding")
		}
		if !bytes.Equal(continuationBytes(t, r), continuationBytes(t, recovered)) {
			t.Fatal("migration recovery changed complete continuation")
		}
		recovered.SetClock(NewFixedClock(numericWrittenAt.Add(2 * time.Second)))
		expected := orderedFixtureClone(t, orderedSave(t, r))
		expected.SetClock(NewFixedClock(numericWrittenAt.Add(2 * time.Second)))
		want, err := expected.CognitiveThink(1)
		if err != nil {
			t.Fatal(err)
		}
		got, err := recovered.CognitiveThink(1)
		if err != nil || !reflect.DeepEqual(want, got) {
			t.Fatal("migration recovery changed the next thought", err)
		}
	}
	if _, err := log.RepairConflictingSequences(); err == nil {
		t.Fatal("legacy renumbering accepted a migration receipt")
	}
}

func TestNumericWrittenMigrationRejectsInvalidPlanOrBackupBeforeMutation(t *testing.T) {
	for _, mutation := range []string{"revision", "reference", "route", "existing-backup", "existing-journal-backup"} {
		t.Run(mutation, func(t *testing.T) {
			r, _, log := numericWrittenFixture(t, 7, true)
			plan, err := r.PlanNeuralWrittenGrounding()
			if err != nil {
				t.Fatal(err)
			}
			backup := filepath.Join(t.TempDir(), "reference.json")
			switch mutation {
			case "revision":
				plan.ReferenceRevision++
			case "reference":
				plan.ReferenceHash = "sha256:" + strings.Repeat("0", 64)
			case "route":
				plan.Projections[0].Units[0].NodeID = plan.Projections[1].Units[0].NodeID
			case "existing-backup":
				if err := os.WriteFile(backup, []byte("keep"), 0600); err != nil {
					t.Fatal(err)
				}
			case "existing-journal-backup":
				if err := os.WriteFile(backup+".events.jsonl", []byte("keep-journal"), 0600); err != nil {
					t.Fatal(err)
				}
			}
			before, journal := continuationBytes(t, r), mustReadWrittenFile(t, log.file.Name())
			if err := r.ApplyNeuralWrittenGrounding(plan, backup, numericWrittenAt); err == nil {
				t.Fatal("unsafe migration accepted")
			}
			if !bytes.Equal(before, continuationBytes(t, r)) || !bytes.Equal(journal, mustReadWrittenFile(t, log.file.Name())) {
				t.Fatal("rejected migration changed state/journal")
			}
			if mutation == "existing-backup" {
				if string(mustReadWrittenFile(t, backup)) != "keep" {
					t.Fatal("backup overwritten")
				}
			} else if _, err := os.Stat(backup); !os.IsNotExist(err) {
				t.Fatal("invalid plan wrote a backup")
			}
			if mutation == "existing-journal-backup" && string(mustReadWrittenFile(t, backup+".events.jsonl")) != "keep-journal" {
				t.Fatal("journal backup overwritten")
			}
		})
	}
}

func mustReadWrittenFile(t *testing.T, path string) []byte {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return data
}

func TestNumericWrittenMigrationAppendFailureRetainsBackupAndRecoveryBoundary(t *testing.T) {
	r, _, log := numericWrittenFixture(t, 7, true)
	if _, err := r.CognitiveThink(1); err != nil {
		t.Fatal(err)
	}
	plan, err := r.PlanNeuralWrittenGrounding()
	if err != nil {
		t.Fatal(err)
	}
	before := continuationBytes(t, r)
	backup := filepath.Join(t.TempDir(), "reference.json")
	log.beforeAppend = func(e LoggedEvent) error {
		if e.Type != EventTypeWrittenGroundingMigration {
			t.Fatal("wrong fault boundary")
		}
		if len(mustReadWrittenFile(t, backup)) == 0 {
			t.Fatal("journal append preceded the durable backup")
		}
		if len(mustReadWrittenFile(t, backup+".events.jsonl")) == 0 {
			t.Fatal("journal append preceded the old journal backup")
		}
		return errors.New("uncertain migration fsync")
	}
	if err := r.ApplyNeuralWrittenGrounding(plan, backup, numericWrittenAt); !errors.Is(err, ErrRuntimeRecoveryRequired) {
		t.Fatal("append failure did not retire the runtime", err)
	}
	if !bytes.Equal(before, continuationBytes(t, r)) {
		t.Fatal("failed append changed neural state/mode")
	}
	if _, err := r.PlanNeuralWrittenGrounding(); !errors.Is(err, ErrRuntimeRecoveryRequired) {
		t.Fatal("retired migration API remained writable")
	}
	recovered := orderedFixtureClone(t, backup)
	if !bytes.Equal(before, continuationBytes(t, recovered)) {
		var a, b []json.RawMessage
		_ = json.Unmarshal(before, &a)
		_ = json.Unmarshal(continuationBytes(t, recovered), &b)
		for i := range a {
			if !bytes.Equal(a[i], b[i]) {
				t.Logf("field %d before=%s after=%s", i, a[i], b[i])
			}
		}
		t.Fatal("backup cannot recover pre-migration state")
	}
}

func TestNumericWrittenCheckpointAuthenticatesServingModeAndRejectsDowngrade(t *testing.T) {
	r, _, _ := numericWrittenFixture(t, 7, false)
	enableNumericWritten(t, r)
	path := orderedSave(t, r)
	var snapshot BrainSnapshot
	if err := json.Unmarshal(mustReadWrittenFile(t, path), &snapshot); err != nil {
		t.Fatal(err)
	}
	snapshot.NeuralWrittenGrounding = false
	data, _ := json.Marshal(snapshot)
	bad := filepath.Join(t.TempDir(), "mode-tamper.json")
	if err := os.WriteFile(bad, data, 0600); err != nil {
		t.Fatal(err)
	}
	target := NewBrainRuntime(knowledge.NewBrain())
	before := continuationBytes(t, target)
	if err := target.RestoreCheckpoint(bad); err == nil || !bytes.Equal(before, continuationBytes(t, target)) {
		t.Fatal("mode tampering changed the target")
	}
	snapshot.NeuralWrittenGrounding, snapshot.SchemaVersion = true, evidenceProvenanceBrainSnapshotSchemaVersion
	snapshot.Checksum, _ = checkpointChecksumV7(snapshot)
	data, _ = json.Marshal(snapshot)
	if err := os.WriteFile(bad, data, 0600); err != nil {
		t.Fatal(err)
	}
	if err := target.RestoreCheckpoint(bad); err == nil || !bytes.Equal(before, continuationBytes(t, target)) {
		t.Fatal("unauthenticated old-schema mode accepted")
	}
	if err := r.LoadBrain(path); err == nil {
		t.Fatal("Brain-only load bypassed numeric mode recovery")
	}
	raw := filepath.Join(t.TempDir(), "brain-only.json")
	if err := r.SaveBrain(raw); err != nil {
		t.Fatal(err)
	}
	if err := target.LoadBrain(raw); err == nil || !bytes.Equal(before, continuationBytes(t, target)) {
		t.Fatal("unmigrated runtime silently loaded a v2 Brain-only graph")
	}
	detached := knowledge.NewBrain()
	if err := detached.Load(raw); err != nil {
		t.Fatal(err)
	}
	if NewBrainRuntime(detached) != nil {
		t.Fatal("constructor bypassed authenticated v2 recovery")
	}
	// Even a recomputed digest cannot make scoped v2 routes a valid v1 state.
	snapshot.SchemaVersion, snapshot.NeuralWrittenGrounding = BrainSnapshotSchemaVersion, false
	snapshot.Checksum, _ = checkpointChecksumV8(snapshot)
	data, _ = json.Marshal(snapshot)
	if err := os.WriteFile(bad, data, 0600); err != nil {
		t.Fatal(err)
	}
	if err := target.RestoreCheckpoint(bad); err == nil || !bytes.Equal(before, continuationBytes(t, target)) {
		t.Fatal("scoped routes accepted under the legacy serving mode")
	}
	// Equivalent JSON spellings must not bypass validation's fast path.
	snapshot.Brain = bytes.ReplaceAll(snapshot.Brain, []byte("written-neural-v2"), []byte(`written-neural-v\u0032`))
	snapshot.Checksum, _ = checkpointChecksumV8(snapshot)
	data, _ = json.Marshal(snapshot)
	if err := os.WriteFile(bad, data, 0600); err != nil {
		t.Fatal(err)
	}
	if err := target.RestoreCheckpoint(bad); err == nil || !bytes.Equal(before, continuationBytes(t, target)) {
		t.Fatal("Unicode-escaped routes bypassed the serving contract before mutation")
	}
	snapshot.NeuralWrittenGrounding = true
	snapshot.Checksum, _ = checkpointChecksumV8(snapshot)
	data, _ = json.Marshal(snapshot)
	if err := os.WriteFile(bad, data, 0600); err != nil {
		t.Fatal(err)
	}
	if err := target.RestoreCheckpoint(bad); err != nil {
		t.Fatal("valid escaped v2 checkpoint rejected", err)
	}
	wantHash, _ := r.brain.CanonicalHash()
	gotHash, _ := target.brain.CanonicalHash()
	if wantHash != gotHash || target.surfaceEncoding != knowledge.NeuralWrittenEncoding {
		t.Fatal("escaped v2 checkpoint changed the numerical state or mode")
	}
}

func TestNumericWrittenFreshLearningImportAndRepeatedExposureReusePrototypes(t *testing.T) {
	r := NewBrainRuntime(knowledge.NewBrain())
	r.SetClock(NewFixedClock(numericWrittenAt))
	enableNumericWritten(t, r)
	for i := 0; i < 3; i++ {
		scrambleAnnotations(t, r, 1)
		if _, err := r.LearnExperience(learning.Experience{Sequence: []string{"naku", "melo", "tiva"}, Weight: .8, Confidence: .8}, numericWrittenAt); err != nil {
			t.Fatal(err)
		}
		if len(r.brain.Registry.Nodes()) != 3 {
			t.Fatal("numeric sequence learning duplicated units after label ablation")
		}
	}
	ids := []knowledge.NodeID{}
	for _, unit := range r.brain.Registry.Nodes() {
		ids = append(ids, unit.ID)
	}
	checkpoint := orderedSave(t, r)
	for _, modality := range []string{"text", "language", " TEXT ", "web-search"} {
		clone := orderedFixtureClone(t, checkpoint)
		scrambleAnnotations(t, clone, 1)
		out, _, err := NewCognitiveOrchestrator(clone).ProcessObservation(
			Event{ID: "aliases", Timestamp: numericWrittenAt, Cycles: 1},
			ObservationInput{Source: "changed-transport", Modality: modality, Tokens: []string{"NAKU", "melo"}}, nil)
		if err != nil || out.GroundedRepresentations[0].NodeID != ids[0] || out.GroundedRepresentations[1].NodeID != ids[1] {
			t.Fatal("numeric written aliases lost learned units", err)
		}
	}
	doc := knowledge.KnowledgeDocument{ID: "numeric-written-import", Domain: "diagnostic", Source: "teacher", SourceType: "text",
		Experiences: []knowledge.KnowledgeExperienceInput{{Before: []knowledge.KnowledgeDatum{{Value: "naku", Modality: "text"}},
			After: []knowledge.KnowledgeDatum{{Value: "tiva", Modality: "text"}}}}}
	if _, err := r.ProcessKnowledgeDocument(doc, numericWrittenAt); err != nil {
		t.Fatal(err)
	}
	scrambleAnnotations(t, r, 1)
	out := numericWrittenQuery(t, r, "naku melo")
	if out.GroundedRepresentations[0].NodeID != ids[0] || out.GroundedRepresentations[1].NodeID != ids[1] {
		t.Fatal("document import redirected numeric written grounding")
	}
}

func TestH18Schema12FixtureRetainsExactHashAndNextThought(t *testing.T) {
	dir := "testdata/legacy_written_grounding_v1"
	var reference struct {
		Commit string          `json:"commit"`
		Before string          `json:"before"`
		Next   CognitiveOutput `json:"next"`
		After  string          `json:"after"`
	}
	if err := json.Unmarshal(mustReadWrittenFile(t, filepath.Join(dir, "reference.json")), &reference); err != nil {
		t.Fatal(err)
	}
	if reference.Commit != "8a4f9ced05d715b338bab4d324011e8c4f6c8596" {
		t.Fatal("unexpected independent fixture provenance")
	}
	r := orderedFixtureClone(t, filepath.Join(dir, "checkpoint.json"))
	hash, err := r.brain.CanonicalHash()
	if err != nil || hash != reference.Before || r.neuralWrittenGrounding {
		t.Fatal("schema-12 fixture changed its hash or serving contract", err)
	}
	r.SetClock(NewFixedClock(time.Date(2026, 10, 21, 2, 0, 1, 0, time.UTC)))
	next, err := r.CognitiveThink(1)
	if err != nil || !reflect.DeepEqual(next, reference.Next) {
		t.Fatal("H18 next-thought reference changed", err)
	}
	hash, err = r.brain.CanonicalHash()
	if err != nil || hash != reference.After {
		t.Fatal("H18 continuation state changed", err)
	}
}

func TestNumericWrittenJournalContractTimelineRejectsBeforeAnyReplayMutation(t *testing.T) {
	r, _, log := numericWrittenFixture(t, 7, true)
	enableNumericWritten(t, r)
	numericWrittenQuery(t, r, "naku melo")
	events, err := ReadEventLog(log.file.Name())
	if err != nil {
		t.Fatal(err)
	}
	for _, mutation := range []string{"missing-migration", "revert-contract", "foreign-payload"} {
		data, _ := json.Marshal(events)
		var changed []LoggedEvent
		_ = json.Unmarshal(data, &changed)
		switch mutation {
		case "missing-migration":
			for i := range changed {
				if changed[i].Type == EventTypeWrittenGroundingMigration {
					changed = append(changed[:i], changed[i+1:]...)
					break
				}
			}
		case "revert-contract":
			changed[len(changed)-1].SurfaceEncoding = knowledge.WrittenTextEncoding
		case "foreign-payload":
			changed[len(changed)-1].WrittenGroundingMigration = changed[len(changed)-2].WrittenGroundingMigration
		}
		target := NewBrainRuntime(knowledge.NewBrain())
		before := continuationBytes(t, target)
		if err := ReplayEventLog(target, changed); err == nil {
			t.Fatal(mutation, "accepted invalid contract timeline")
		}
		if !bytes.Equal(before, continuationBytes(t, target)) {
			t.Fatal(mutation, "changed target before rejection")
		}
	}
}

func TestNumericWrittenOwnerCommandsAndInspectionSerializeWithoutDuplication(t *testing.T) {
	r, _, _ := numericWrittenFixture(t, 7, false)
	enableNumericWritten(t, r)
	var workers sync.WaitGroup
	for worker := 0; worker < 3; worker++ {
		workers.Add(1)
		go func() {
			defer workers.Done()
			for i := 0; i < 6; i++ {
				if _, err := r.CognitiveProcess(Event{Timestamp: numericWrittenAt, Stimulus: []string{"naku", "melo"}, Modality: "text", Cycles: 1}); err != nil {
					t.Error(err)
				}
				if _, err := r.WrittenGroundingContract(); err != nil {
					t.Error(err)
				}
			}
		}()
	}
	workers.Wait()
	if len(r.brain.Registry.Nodes()) != 6 {
		t.Fatal("concurrent owner commands duplicated representations")
	}
}

func TestNumericWrittenPrefixPredictionDependsOnLearnedConnectionsAndPatterns(t *testing.T) {
	r, ids, _ := numericWrittenFixture(t, 17, false)
	enableNumericWritten(t, r)
	checkpoint := orderedSave(t, r)
	for _, prompt := range []struct{ cue, target, other string }{{"naku melo", "tiva", "luma"}, {"sopo keri", "luma", "tiva"}} {
		trained, ablated := orderedFixtureClone(t, checkpoint), orderedFixtureClone(t, checkpoint)
		scrambleAnnotations(t, trained, 1)
		scrambleAnnotations(t, ablated, 1)
		// Controlled experimental ablation on an isolated recovered copy.
		ablated.brain.Patterns = knowledge.NewPatternIndex()
		for _, unit := range ablated.brain.Registry.Nodes() {
			unit.Synapses = nil
		}
		numericWrittenQuery(t, trained, prompt.cue)
		numericWrittenQuery(t, ablated, prompt.cue)
		learned := trained.activation.PredictionSnapshot().State
		control := ablated.activation.PredictionSnapshot().State
		if learned[ids[prompt.target]] <= learned[ids[prompt.other]]+.1 || learned[ids[prompt.target]] <= control[ids[prompt.target]]+.1 {
			t.Fatal("prefix scores do not separate learned retrieval from the ablated control")
		}
		t.Logf("%s: target prediction=%g alternative=%g ablated=%g (scores, not semantic-answer probabilities)", prompt.cue, learned[ids[prompt.target]], learned[ids[prompt.other]], control[ids[prompt.target]])
	}
}
