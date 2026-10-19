package runtime

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"testing"
	"time"

	"github.com/project-horizon/horizon-core/services/ai/knowledge"
	"github.com/project-horizon/horizon-core/services/ai/learning"
)

const legacySurfaceFixture = "testdata/legacy_surface_v1"

func legacySurfaceJournal(t *testing.T) *EventLog {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(legacySurfaceFixture, "commands.jsonl"))
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), "commands.jsonl")
	if err := os.WriteFile(path, data, 0600); err != nil {
		t.Fatal(err)
	}
	log, err := OpenEventLog(path)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = log.Close() })
	return log
}

func surfaceCheckpointChecksum(t *testing.T, r *BrainRuntime) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "checkpoint.json")
	if err := r.Checkpoint(path); err != nil {
		t.Fatal(err)
	}
	s, err := readCheckpointForTest(path)
	if err != nil {
		t.Fatal(err)
	}
	return s.Checksum
}

func TestLegacySurfaceJournalMatchesPreChangeReferenceAndNextThought(t *testing.T) {
	data, err := os.ReadFile(filepath.Join(legacySurfaceFixture, "reference.json"))
	if err != nil {
		t.Fatal(err)
	}
	var reference struct {
		Commit     string          `json:"commit"`
		Checksum   string          `json:"checksum"`
		Next       CognitiveOutput `json:"next"`
		AfterNext  string          `json:"after_next_checksum"`
		Foundation string          `json:"bootstrap_brain_sha256"`
	}
	if err := json.Unmarshal(data, &reference); err != nil {
		t.Fatal(err)
	}
	if reference.Commit != "68708eb852ee900d636edec0fe3e4d8d0fbe1f59" {
		t.Fatal("unexpected fixture provenance")
	}
	foundation, err := knowledge.NewBootstrapBrain().CanonicalJSON()
	if err != nil {
		t.Fatal(err)
	}
	sum := sha256.Sum256(foundation)
	if hex.EncodeToString(sum[:]) != reference.Foundation {
		t.Fatal("historical genesis foundation changed")
	}
	log := legacySurfaceJournal(t)
	genesis, err := RebuildRuntimeFromCommittedJournal(log, knowledge.NewBrain)
	if err != nil {
		t.Fatal(err)
	}
	suffix, err := RebuildRuntimeFromCheckpointAndJournal(log, filepath.Join(legacySurfaceFixture, "prefix-checkpoint.json"))
	if err != nil {
		t.Fatal(err)
	}
	for _, r := range []*BrainRuntime{genesis, suffix} {
		if r.surfaceEncoding != knowledge.WrittenTextEncoding {
			t.Fatal("historical replay leaked its encoding into serving operations")
		}
		if surfaceCheckpointChecksum(t, r) != reference.Checksum {
			t.Fatal("historical full-state checksum changed")
		}
		r.SetClock(NewFixedClock(time.Date(2026, 10, 19, 10, 0, 0, 0, time.UTC)))
		next, err := r.CognitiveThink(1)
		if err != nil {
			t.Fatal(err)
		}
		if !reflect.DeepEqual(next, reference.Next) {
			t.Fatal("historical replay changed the next recurrent thought")
		}
		if surfaceCheckpointChecksum(t, r) != reference.AfterNext {
			t.Fatal("historical continuation state changed")
		}
	}
}

func TestUnknownSurfaceEncodingRejectedBeforeReplayMutation(t *testing.T) {
	events, err := ReadEventLog(filepath.Join(legacySurfaceFixture, "commands.jsonl"))
	if err != nil {
		t.Fatal(err)
	}
	events[len(events)-1].SurfaceEncoding = "future-unknown-contract"
	r := NewBrainRuntime(knowledge.NewBrain())
	before := continuationBytes(t, r)
	if err := ReplayEventLog(r, events); err == nil {
		t.Fatal("unknown encoding replayed")
	}
	if !reflect.DeepEqual(before, continuationBytes(t, r)) || r.surfaceEncoding != knowledge.WrittenTextEncoding {
		t.Fatal("invalid contract changed runtime before rejection")
	}
}

func TestWrittenEncodingRequiresNewJournalSchemaAndUnknownCheckpointFailsUnchanged(t *testing.T) {
	at := time.Unix(100, 0).UTC()
	r := NewBrainRuntime(knowledge.NewBrain())
	log := openKnowledgeJournal(t)
	r.SetEventLog(log)
	if _, err := r.LearnExperience(learning.Experience{Sequence: []string{"naku", "tiva"}}, at); err != nil {
		t.Fatal(err)
	}
	events, err := ReadEventLog(log.file.Name())
	if err != nil {
		t.Fatal(err)
	}
	if len(events) != 1 || events[0].SchemaVersion != 2 || events[0].SurfaceEncoding != knowledge.WrittenTextEncoding {
		t.Fatal("new written command is not explicitly versioned")
	}
	events[0].SchemaVersion = 1
	if err := validateJournalPayload(events[0]); err == nil {
		t.Fatal("new encoding silently downgraded to an old-reader-compatible journal")
	}
	path := filepath.Join(t.TempDir(), "checkpoint.json")
	if err := r.Checkpoint(path); err != nil {
		t.Fatal(err)
	}
	s, err := readCheckpointForTest(path)
	if err != nil {
		t.Fatal(err)
	}
	s.SurfaceAnnotations[0].Encoding = "future-unknown-contract"
	s.Checksum, err = checkpointChecksumV7(s)
	if err != nil {
		t.Fatal(err)
	}
	data, err := json.Marshal(s)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, data, 0600); err != nil {
		t.Fatal(err)
	}
	candidate := NewBrainRuntime(knowledge.NewBrain())
	before := continuationBytes(t, candidate)
	if err := candidate.RestoreCheckpoint(path); err == nil {
		t.Fatal("unknown checkpoint adapter encoding accepted")
	}
	if !reflect.DeepEqual(before, continuationBytes(t, candidate)) {
		t.Fatal("unknown checkpoint contract partially changed the target")
	}
}

func TestWrittenKnowledgeImportUsesSamePopulationAsActiveTextAndRecovers(t *testing.T) {
	at := time.Unix(100, 0).UTC()
	r := NewBrainRuntime(knowledge.NewBrain())
	log := openKnowledgeJournal(t)
	r.SetEventLog(log)
	doc := knowledge.KnowledgeDocument{ID: "written-import", Domain: "test", Source: "teacher", SourceType: "text", Experiences: []knowledge.KnowledgeExperienceInput{{Before: []knowledge.KnowledgeDatum{{Value: "naku", Modality: "text"}}, Changes: []knowledge.KnowledgeDatum{{Value: "melo", Modality: "text"}}, After: []knowledge.KnowledgeDatum{{Value: "tiva", Modality: "text"}}}}}
	imported, err := r.ProcessKnowledgeDocument(doc, at)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(imported.Compilation.Experiences[0].SymbolExposures[0].Representation, knowledge.EncodeObservation("naku", "language").Values) {
		t.Fatal("import compiler uses a different written vector")
	}
	before, err := r.InspectBrain()
	if err != nil {
		t.Fatal(err)
	}
	id, ok := before.Graph.NodeForSurface("naku")
	if !ok {
		t.Fatal("imported prefix has no surface binding")
	}
	checkpoint := filepath.Join(t.TempDir(), "imported.json")
	if err := r.Checkpoint(checkpoint); err != nil {
		t.Fatal(err)
	}
	out, _, err := NewCognitiveOrchestrator(r).ProcessObservation(Event{ID: "import-cue", Timestamp: at.Add(time.Second), Cycles: 1}, ObservationInput{Source: "new-user", Modality: "language", Tokens: []string{"NAKU"}}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(out.GroundedRepresentations) != 1 || out.GroundedRepresentations[0].NodeID != id {
		t.Fatal("active text did not reuse imported written population")
	}
	if _, err := r.LearnExperience(learning.Experience{Sequence: []string{"naku", "melo", "tiva"}, Modality: "text"}, at.Add(2*time.Second)); err != nil {
		t.Fatal(err)
	}
	after, err := r.InspectBrain()
	if err != nil {
		t.Fatal(err)
	}
	bn, _, _ := before.Graph.Counts()
	an, _, _ := after.Graph.Counts()
	if bn != an {
		t.Fatal("sequence learning duplicated imported written units")
	}
	genesis, err := RebuildRuntimeFromCommittedJournal(log, knowledge.NewBrain)
	if err != nil {
		t.Fatal(err)
	}
	suffix, err := RebuildRuntimeFromCheckpointAndJournal(log, checkpoint)
	if err != nil {
		t.Fatal(err)
	}
	assertCompleteContinuation(t, r, genesis)
	assertCompleteContinuation(t, r, suffix)
}
