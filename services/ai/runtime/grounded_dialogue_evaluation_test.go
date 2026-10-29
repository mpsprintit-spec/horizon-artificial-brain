package runtime

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"sync"
	"testing"
	"time"

	"github.com/project-horizon/horizon-core/services/ai/knowledge"
)

func dialogueLoadModel(t *testing.T) (*BrainRuntime, DialogueCodec, string) {
	t.Helper()
	root := os.Getenv("HORIZON_DIALOGUE_MODEL")
	if root == "" {
		t.Skip("explicit trained language model required")
	}
	r := NewBrainRuntime(knowledge.NewBrain())
	compositionCheck(t, r.RestoreCheckpoint(filepath.Join(root, "language-brain.json")))
	var c DialogueCodec
	b, e := os.ReadFile(filepath.Join(root, "codec.json"))
	compositionCheck(t, e)
	compositionCheck(t, json.Unmarshal(b, &c))
	return r, c, root
}

func TestGroundedDialogueFreshEvaluation(t *testing.T) {
	r, codec, _ := dialogueLoadModel(t)
	path := os.Getenv("HORIZON_DIALOGUE_FINAL_CASES")
	if path == "" {
		t.Skip("sealed new evaluation required")
	}
	raw, e := os.ReadFile(path)
	compositionCheck(t, e)
	var cases []dialogueEvalCase
	compositionCheck(t, json.Unmarshal(raw, &cases))
	if len(cases) != 512 {
		t.Fatalf("expected the frozen 512-case evaluation, got %d", len(cases))
	}
	hash := sha256.Sum256(raw)
	counts := map[string]map[string]int{}
	results := []any{}
	samples := []any{}
	latencies := []float64{}
	before := continuationBytes(t, r)
	sequence := r.LastSequence()
	for i, c := range cases {
		at := time.Now()
		o, e := r.ReadGroundedDialogue(codec, c.Request)
		compositionCheck(t, e)
		latencies = append(latencies, float64(time.Since(at).Nanoseconds())/1e6)
		if counts[c.Split] == nil {
			counts[c.Split] = map[string]int{}
		}
		g := counts[c.Split]
		g["total"]++
		if c.Expected != "" && o.Text == c.Expected {
			g["exact"]++
		}
		meaning := o.MeaningValid == c.Valid && (!c.Valid || o.Focus == c.Focus && o.Temporal == c.Temporal && o.ObservedState == c.State)
		if meaning {
			g["meaning"]++
		}
		if c.Split == "outside-scope" {
			if o.MeaningValid {
				t.Errorf("%s invented a substantive answer outside the trained scope", c.ID)
			}
		} else if c.Expected == "" || o.Text != c.Expected || !meaning {
			t.Errorf("%s: got %q (meaning valid=%t), want %q with matching meaning", c.ID, o.Text, o.MeaningValid, c.Expected)
		}
		if o.Status == "untrained" {
			g["abstained"]++
		}
		if !o.MeaningValid {
			g["no-substantive-claim"]++
		}
		results = append(results, map[string]any{"id": c.ID, "split": c.Split, "expected": c.Expected, "actual": o.Text, "status": o.Status, "reason": o.Reason, "meaning_valid": o.MeaningValid, "focus": o.Focus, "temporal": o.Temporal, "state": o.ObservedState, "agreement": o.Agreement})
		if i%48 == 0 || c.Split == "must-clarify" || c.Split == "outside-scope" {
			samples = append(samples, map[string]any{"case": c, "output": o})
		}
	}
	if !bytes.Equal(before, continuationBytes(t, r)) || sequence != r.LastSequence() {
		t.Fatal("fresh evaluation mutated Brain")
	}
	dialogueSave(t, "fresh_evaluation.json", results)
	dialogueSave(t, "fresh_samples.json", samples)
	dialogueSave(t, "fresh_results.json", map[string]any{"sealed_sha256": hex.EncodeToString(hash[:]), "counts": counts, "p50_ms": mPct(latencies, .5), "p95_ms": mPct(latencies, .95), "readonly": true, "evaluated": len(cases)})
	t.Logf("Fresh language evaluation: %v", counts)
}

func TestGroundedDialoguePersistenceContextAndSynapticAblation(t *testing.T) {
	r, codec, root := dialogueLoadModel(t)
	cases := dialogueSealedCases()
	log, e := OpenEventLog(filepath.Join(root, "events.jsonl"))
	compositionCheck(t, e)
	defer log.Close()
	replayed, e := RebuildRuntimeFromCheckpointAndJournal(log, filepath.Join(root, "start.json"))
	compositionCheck(t, e)
	for _, i := range []int{0, 2, 3, 4, 5, 6, 7, 128, 260} {
		want, e := r.ReadGroundedDialogue(codec, cases[i].Request)
		compositionCheck(t, e)
		got, e := replayed.ReadGroundedDialogue(codec, cases[i].Request)
		compositionCheck(t, e)
		if !reflect.DeepEqual(want, got) {
			t.Fatal("native language changed after full WAL replay")
		}
	}
	request := cases[5].Request
	baseline, e := r.ReadGroundedDialogue(codec, request)
	compositionCheck(t, e)
	if baseline.Text != cases[5].Expected || !baseline.MeaningValid {
		t.Fatal("reference-bearing model did not answer the native fixture")
	}
	request.History = nil
	missing, e := r.ReadGroundedDialogue(codec, request)
	compositionCheck(t, e)
	if missing.MeaningValid || missing.Status != "clarification-needed" {
		t.Fatal("missing conversational reference was invented")
	}
	request = cases[5].Request
	request.History = []string{"Bagaimana keadaan pintu?"}
	changed, e := r.ReadGroundedDialogue(codec, request)
	compositionCheck(t, e)
	if changed.Focus != .75 || changed.Text == baseline.Text {
		t.Fatal("changing raw context did not change reference")
	}
	// Removing output-form synapses on a disposable restored copy must remove
	// the complete answer; no label or teacher sentence may supply a fallback.
	damaged := NewBrainRuntime(knowledge.NewBrain())
	compositionCheck(t, damaged.RestoreCheckpoint(filepath.Join(root, "language-brain.json")))
	for _, id := range codec.Output[0].Population {
		for _, links := range damaged.brain.Registry.GetByID(id).Synapses {
			for _, link := range links {
				link.Dynamic.Weight = 0
			}
		}
	}
	ablated, e := damaged.ReadGroundedDialogue(codec, cases[0].Request)
	compositionCheck(t, e)
	if ablated.Text != "" || ablated.Reason != "untrained-spelling" {
		t.Fatal("text survived removal of its learned output synapses")
	}
	dialogueSave(t, "recovery_and_ablation.json", map[string]any{"full_replay_equal": true, "context_changes_reference": true, "context_absence_clarifies": true, "normal": baseline, "changed_context": changed, "missing_context": missing, "spelling_ablation": ablated})
}

func TestGroundedDialogueStandaloneLatency(t *testing.T) {
	if os.Getenv("HORIZON_DIALOGUE_TIMING") == "" {
		t.Skip("explicit isolated timing required")
	}
	r, codec, _ := dialogueLoadModel(t)
	cases := dialogueSealedCases()
	indices := []int{0, 2, 3, 4, 5, 6, 7, 128, 260}
	for _, i := range indices {
		_, e := r.ReadGroundedDialogue(codec, cases[i].Request)
		compositionCheck(t, e)
	}
	values := []float64{}
	for n := 0; n < 108; n++ {
		c := cases[indices[n%len(indices)]]
		start := time.Now()
		o, e := r.ReadGroundedDialogue(codec, c.Request)
		compositionCheck(t, e)
		values = append(values, float64(time.Since(start).Nanoseconds())/1e6)
		if o.Text != c.Expected {
			t.Fatal("timed native answer differs")
		}
	}
	dialogueSave(t, "standalone_timing.json", map[string]any{"unit": "milliseconds per complete utterance", "calls": len(values), "warmup_calls": len(indices), "p50_ms": mPct(values, .5), "p95_ms": mPct(values, .95), "includes_checkpoint_load": false, "includes_trace_and_spelling": true, "durations_ms": values})
}

func TestGroundedDialogueConcurrentCanonicalRead(t *testing.T) {
	r, codec, _ := dialogueLoadModel(t)
	cases := dialogueSealedCases()
	before := continuationBytes(t, r)
	sequence := r.LastSequence()
	indices := []int{0, 3, 5, 6}
	want := make([]GroundedDialogueReadout, len(indices))
	for i, index := range indices {
		var e error
		want[i], e = r.ReadGroundedDialogue(codec, cases[index].Request)
		compositionCheck(t, e)
	}
	var workers sync.WaitGroup
	failures := make(chan string, 16)
	for worker := 0; worker < 4; worker++ {
		workers.Add(1)
		go func() {
			defer workers.Done()
			for i, index := range indices {
				out, e := r.ReadGroundedDialogue(codec, cases[index].Request)
				if e != nil || !reflect.DeepEqual(out, want[i]) {
					failures <- "concurrent read changed interpretation or spelling"
				}
			}
		}()
	}
	workers.Wait()
	close(failures)
	for failure := range failures {
		t.Error(failure)
	}
	if !bytes.Equal(before, continuationBytes(t, r)) || sequence != r.LastSequence() {
		t.Fatal("concurrent language read mutated canonical Brain")
	}
}
