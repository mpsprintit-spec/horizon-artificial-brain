package runtime

import (
	"bytes"
	"path/filepath"
	"testing"
	"time"

	"github.com/project-horizon/horizon-core/services/ai/knowledge"
)

func TestNativeCompositionImportGroundingAndSpellingKeepJournalContract(t *testing.T) {
	r, log, initial := newCompositionTestRuntime(t, 7, "separated", true)
	_, err := r.ProcessKnowledgeDocument(journalKnowledgeDocument(), compositionTestBase.Add(time.Second))
	compositionCheck(t, err)
	grounded, err := r.GroundVectorAt(knowledge.NewNeuralVector([]float64{.11, .22}), compositionTestBase.Add(2*time.Second))
	compositionCheck(t, err)
	exposure := knowledge.SpellingExposure{Codec: knowledge.SpellingCodecV1, Population: grounded.Population, Scalars: []uint32{65}, Weight: .5, Confidence: .3}
	_, err = r.LearnSpelling(exposure, compositionTestBase.Add(3*time.Second))
	compositionCheck(t, err)
	events, err := ReadEventLog(log.file.Name())
	compositionCheck(t, err)
	for _, event := range events[2:] {
		if event.SchemaVersion != compositionJournalSchemaVersion || event.CompositionEncoding != knowledge.NeuralCompositionV1 {
			t.Fatalf("native command %s lost its active composition contract", event.Type)
		}
	}
	checkpoint := filepath.Join(t.TempDir(), "after-native-import.json")
	compositionCheck(t, r.Checkpoint(checkpoint))
	for _, path := range []string{initial, checkpoint} {
		rebuilt, err := RebuildRuntimeFromCheckpointAndJournal(log, path)
		compositionCheck(t, err)
		if !bytes.Equal(continuationBytes(t, r), continuationBytes(t, rebuilt)) {
			t.Fatal("mixed native command replay lost complete composition continuation")
		}
	}
}
