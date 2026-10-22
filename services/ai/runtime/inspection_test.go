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
	"sort"
	"sync"
	"testing"
	"time"

	"github.com/project-horizon/horizon-core/services/ai/knowledge"
)

func compactInspection(t *testing.T, data []byte) string {
	t.Helper()
	var b bytes.Buffer
	if err := json.Compact(&b, data); err != nil {
		t.Fatal(err)
	}
	return b.String()
}
func TestInspectionPaginationCoversEntireCanonicalStateWithoutMutation(t *testing.T) {
	r := NewBrainRuntime(nil)
	sensor := factorizedSensor()
	if _, err := r.ProcessSensorObservation(sensor); err != nil {
		t.Fatal(err)
	}
	before := continuationBytes(t, r)
	canonical, err := r.brain.CanonicalJSON()
	if err != nil {
		t.Fatal(err)
	}
	var document map[string]json.RawMessage
	if err := json.Unmarshal(canonical, &document); err != nil {
		t.Fatal(err)
	}
	seen := map[string]bool{}
	for _, section := range knowledge.InspectionSections {
		seen[section] = true
		var actual []string
		cursor := ""
		for pages := 0; ; pages++ {
			if pages > 1000 {
				t.Fatal("cursor failed to advance")
			}
			page, err := r.MonitorInspectionPage(InspectionPageRequest{Section: section, Limit: 7, Cursor: cursor}, "test")
			if err != nil {
				t.Fatalf("%s: %v", section, err)
			}
			hash := page.PageHash
			page.PageHash = ""
			data, _ := json.Marshal(page)
			sum := sha256.Sum256(data)
			if hash != "sha256:"+hex.EncodeToString(sum[:]) {
				t.Fatal("page hash excludes payload")
			}
			if len(data) > knowledge.MaxInspectionBytes {
				t.Fatal("unbounded page")
			}
			for _, item := range page.Data.Items {
				actual = append(actual, compactInspection(t, item))
			}
			if page.NextCursor == "" {
				break
			}
			if page.NextCursor == cursor {
				t.Fatal("cursor repeated")
			}
			cursor = page.NextCursor
		}
		var expected []string
		raw := document[section]
		if len(raw) == 0 || string(raw) == "null" {
			if len(actual) != 0 {
				t.Fatalf("%s should be empty", section)
			}
			continue
		}
		if raw[0] == '[' {
			var items []json.RawMessage
			_ = json.Unmarshal(raw, &items)
			for _, item := range items {
				expected = append(expected, compactInspection(t, item))
			}
		} else if section == "schema_version" || section == "brain_identity" || section == "memory_decay" || section == "learning_policy_state" || section == "inquiry_state" {
			expected = append(expected, compactInspection(t, raw))
		} else {
			var values map[string]float64
			if err := json.Unmarshal(raw, &values); err != nil {
				t.Fatal(err)
			}
			for key, value := range values {
				item, _ := json.Marshal(struct {
					Key   string  `json:"key"`
					Value float64 `json:"value"`
				}{key, value})
				expected = append(expected, string(item))
			}
		}
		sort.Strings(expected)
		sort.Strings(actual)
		if !reflect.DeepEqual(expected, actual) {
			t.Fatalf("%s pagination differs from complete canonical data: %d vs %d", section, len(actual), len(expected))
		}
	}
	for section := range document {
		if !seen[section] {
			t.Fatalf("canonical field %s has no inspection route", section)
		}
	}
	if !bytes.Equal(before, continuationBytes(t, r)) {
		t.Fatal("inspection changed complete continuation state")
	}
}

func TestInspectionFencesRevisionRestoreAndRuntimeLifetime(t *testing.T) {
	r := NewBrainRuntime(nil)
	request := InspectionPageRequest{Section: "neural_units", Limit: 1}
	first, err := r.MonitorInspectionPage(request, "")
	if err != nil {
		t.Fatal(err)
	}
	checkpoint := orderedSave(t, r)
	request.Cursor = first.NextCursor
	if _, err := r.MonitorInspectionPage(request, ""); err != nil {
		t.Fatal(err)
	}
	if _, err := r.CognitiveThink(1); err != nil {
		t.Fatal(err)
	}
	if _, err := r.MonitorInspectionPage(request, ""); !errors.Is(err, ErrInspectionChanged) {
		t.Fatal("cross-revision cursor accepted", err)
	}
	if err := r.RestoreCheckpoint(checkpoint); err != nil {
		t.Fatal(err)
	}
	if _, err := r.MonitorInspectionPage(request, ""); !errors.Is(err, ErrInspectionChanged) {
		t.Fatal("same-revision restore reused prior read epoch", err)
	}
	other := NewBrainRuntime(nil)
	if _, err := other.MonitorInspectionPage(request, ""); !errors.Is(err, ErrInspectionChanged) {
		t.Fatal("cross-runtime cursor accepted", err)
	}
	request.Section = "synapses"
	if _, err := r.MonitorInspectionPage(request, ""); !errors.Is(err, ErrInspectionCursor) {
		t.Fatal("cross-section cursor accepted", err)
	}
	r.mu.Lock()
	r.markRecoveryRequiredLocked()
	r.mu.Unlock()
	if _, err := r.MonitorInspectionHandshake(""); !errors.Is(err, ErrRuntimeRecoveryRequired) {
		t.Fatal("retired inspection remained available")
	}
}

func TestInspectionActivityWindowIsDetachedAndBounded(t *testing.T) {
	b := knowledge.NewBootstrapBrain()
	nodes := b.Registry.Nodes()
	nodes[len(nodes)-1].Activation = .99
	r := NewBrainRuntime(b)
	page, err := r.MonitorInspectionPage(InspectionPageRequest{Section: "neural_units", Order: "activity", Limit: 1}, "")
	if err != nil {
		t.Fatal(err)
	}
	var unit struct {
		ID knowledge.NodeID `json:"id"`
	}
	_ = json.Unmarshal(page.Data.Items[0], &unit)
	if unit.ID != nodes[len(nodes)-1].ID {
		t.Fatal("active unit missing from bounded window")
	}
	page.Data.Items[0][0] = '!'
	if _, err := r.brain.CanonicalJSON(); err != nil {
		t.Fatal("page aliases live neural state")
	}
	// Preflight must reject huge individual records before building huge JSON.
	nodes[len(nodes)-1].Representation = make([]float64, knowledge.MaxInspectionBytes)
	if _, err := r.MonitorInspectionPage(InspectionPageRequest{Section: "neural_units", Order: "activity", Limit: 1}, ""); !errors.Is(err, knowledge.ErrInspectionTooLarge) {
		t.Fatal("oversized unit not rejected", err)
	}
}

func TestStreamingCheckpointPreservesLogicalChecksumAndContinuation(t *testing.T) {
	r := NewBrainRuntime(nil)
	at := time.Date(2026, 10, 20, 12, 0, 0, 0, time.UTC)
	if _, err := r.ProcessObservationEnvelope(transactionObservation("checkpoint-stream", at, "naku", "melo")); err != nil {
		t.Fatal(err)
	}
	path := orderedSave(t, r)
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Contains(data, []byte("\n")) {
		t.Fatal("checkpoint should be compact")
	}
	var snapshot BrainSnapshot
	if err := json.Unmarshal(data, &snapshot); err != nil {
		t.Fatal(err)
	}
	want, err := checkpointChecksumForSchema(snapshot)
	if err != nil || want != snapshot.Checksum {
		t.Fatal("streaming changed logical checksum", err)
	}
	restored := orderedFixtureClone(t, path)
	if !bytes.Equal(continuationBytes(t, r), continuationBytes(t, restored)) {
		t.Fatal("streaming lost checkpoint fields")
	}
	// A failure after header emission must preserve the previous file.
	r.brain.BrainState.MemoryState["bad"] = math.NaN()
	if err := r.Checkpoint(path); err == nil {
		t.Fatal("invalid state checkpoint succeeded")
	}
	after, _ := os.ReadFile(path)
	if !bytes.Equal(data, after) {
		t.Fatal("failed streaming replaced complete checkpoint")
	}
	temps, _ := filepath.Glob(filepath.Join(filepath.Dir(path), ".horizon-checkpoint-*.tmp"))
	if len(temps) != 0 {
		t.Fatal("checkpoint temp file leaked")
	}
}

func TestInspectionConcurrentOwnerCommands(t *testing.T) {
	r := NewBrainRuntime(nil)
	var workers sync.WaitGroup
	for i := 0; i < 3; i++ {
		workers.Add(1)
		go func() {
			defer workers.Done()
			for j := 0; j < 12; j++ {
				page, err := r.MonitorInspectionPage(InspectionPageRequest{Section: "neural_units", Limit: 16}, "")
				if err != nil {
					t.Error(err)
					return
				}
				if len(page.Data.Items) > 16 {
					t.Error("unbounded read")
				}
			}
		}()
	}
	for i := 0; i < 12; i++ {
		if _, err := r.CognitiveThink(1); err != nil {
			t.Fatal(err)
		}
	}
	workers.Wait()
}
