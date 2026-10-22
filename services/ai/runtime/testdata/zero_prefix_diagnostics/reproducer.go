package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/project-horizon/horizon-core/services/ai/knowledge"
	"github.com/project-horizon/horizon-core/services/ai/learning"
	hr "github.com/project-horizon/horizon-core/services/ai/runtime"
)

func must(e error) {
	if e != nil {
		panic(e)
	}
}
func message(e error) string {
	if e == nil {
		return ""
	}
	return e.Error()
}
func main() {
	dir, e := os.MkdirTemp("", "horizon-empty-prefix-")
	must(e)
	defer os.RemoveAll(dir)
	r := hr.NewBrainRuntime(knowledge.NewBrain())
	at := time.Date(2026, 10, 21, 8, 0, 0, 0, time.UTC)
	r.SetClock(hr.NewFixedClock(at))
	log, e := hr.OpenEventLog(filepath.Join(dir, "events.jsonl"))
	must(e)
	defer log.Close()
	r.SetEventLog(log)
	zero := filepath.Join(dir, "zero.json")
	must(r.Checkpoint(zero))
	_, before := hr.RebuildRuntimeFromCheckpointAndJournal(log, zero)
	_, e = r.LearnExperience(learning.Experience{Sequence: []string{"naku", "melo", "tiva"}, Source: "empty-prefix-probe", Weight: .9, Confidence: .9}, at)
	must(e)
	_, after := hr.RebuildRuntimeFromCheckpointAndJournal(log, zero)
	one := filepath.Join(dir, "one.json")
	must(r.Checkpoint(one))
	_, e = r.LearnExperience(learning.Experience{Sequence: []string{"sopo", "keri", "luma"}, Source: "empty-prefix-probe", Weight: .9, Confidence: .9}, at)
	must(e)
	recovered, control := hr.RebuildRuntimeFromCheckpointAndJournal(log, one)
	sequence := uint64(0)
	if recovered != nil {
		sequence = recovered.LastSequence()
	}
	data, e := os.ReadFile(zero)
	must(e)
	var snapshot map[string]any
	must(json.Unmarshal(data, &snapshot))
	result := map[string]any{"zero_checkpoint_sequence": snapshot["sequence"], "zero_checkpoint_prefix_hash": snapshot["journal_prefix_hash"], "empty_journal_restore_error": message(before), "restore_zero_after_first_event_error": message(after), "restore_one_after_second_event_error": message(control), "nonempty_recovered_revision": sequence, "observed_empty_prefix_failure": before == nil && after != nil && control == nil}
	b, e := json.MarshalIndent(result, "", "  ")
	must(e)
	fmt.Println(string(b))
}
