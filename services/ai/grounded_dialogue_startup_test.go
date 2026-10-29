package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/project-horizon/horizon-core/services/ai/runtime"
)

// Exercise the actual production owner/recovery path, not a second standalone
// interpreter. The opt-in bundle is unpacked by scripts/prepare_grounded_dialogue.py.
func TestProductionGroundedDialogueBundle(t *testing.T) {
	model := os.Getenv("HORIZON_DIALOGUE_MODEL")
	if model == "" {
		t.Skip("explicit trained dialogue bundle required")
	}
	dir := t.TempDir()
	for _, name := range []string{"language-brain.json", "events.jsonl", "codec.json", "evaluation.json"} {
		data, err := os.ReadFile(filepath.Join(model, name))
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dir, name), data, 0600); err != nil {
			t.Fatal(err)
		}
	}
	var codec runtime.DialogueCodec
	data, err := os.ReadFile(filepath.Join(dir, "codec.json"))
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(data, &codec); err != nil {
		t.Fatal(err)
	}
	var cases []struct {
		ID       string                          `json:"id"`
		Split    string                          `json:"split"`
		Request  runtime.GroundedDialogueRequest `json:"request"`
		Expected string                          `json:"expected"`
		Valid    bool                            `json:"valid"`
	}
	data, err = os.ReadFile(filepath.Join(dir, "evaluation.json"))
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(data, &cases); err != nil {
		t.Fatal(err)
	}
	if len(cases) != 512 {
		t.Fatalf("bundle evaluation has %d cases", len(cases))
	}
	journal := filepath.Join(dir, "events.jsonl")
	checkpoint := filepath.Join(dir, "language-brain.json")
	memory := filepath.Join(dir, "brain_memory.json")
	log := newStartupJournal(t, journal)
	h, bootstrap, err := initializeProductionHorizon(log, journal, memory, checkpoint)
	if err != nil {
		t.Fatal(err)
	}
	if bootstrap {
		t.Fatal("trained bundle requested fresh bootstrap")
	}
	check := func() {
		t.Helper()
		seen := map[string]bool{}
		for _, c := range cases {
			if seen[c.Split] || c.Split == "outside-scope" {
				continue
			}
			seen[c.Split] = true
			out, err := h.ReadGroundedDialogue(codec, c.Request)
			if err != nil {
				t.Fatal(err)
			}
			if out.Text != c.Expected || out.MeaningValid != c.Valid {
				t.Fatalf("%s after production recovery: got %q (%s), want %q", c.ID, out.Text, out.Reason, c.Expected)
			}
		}
		if len(seen) != 11 {
			t.Fatalf("checked %d supported/clarification categories", len(seen))
		}
	}
	check()
	for n := 0; n < 3; n++ {
		if err := h.Supervisor().WithMutation(func(rt *runtime.BrainRuntime) error {
			rt.SetClock(runtime.NewFixedClock(cases[0].Request.At.Add(time.Duration(n) * time.Hour)))
			_, err := rt.CognitiveThink(1)
			return err
		}); err != nil {
			t.Fatal(err)
		}
		check()
	}
	if err := h.Supervisor().WithMutation(func(rt *runtime.BrainRuntime) error {
		return rt.Checkpoint(checkpoint)
	}); err != nil {
		t.Fatal(err)
	}
	if err := h.Supervisor().Close(); err != nil {
		t.Fatal(err)
	}
	if err := log.Close(); err != nil {
		t.Fatal(err)
	}
	log = newStartupJournal(t, journal)
	h, bootstrap, err = initializeProductionHorizon(log, journal, memory, checkpoint)
	if err != nil {
		t.Fatal(err)
	}
	defer h.Supervisor().Close()
	if bootstrap {
		t.Fatal("trained bundle reboot requested bootstrap")
	}
	check()
}
