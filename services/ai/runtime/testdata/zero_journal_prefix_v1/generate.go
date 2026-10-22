//go:build ignore

// Run with the PR #44 module to capture the pre-fix writer, not the new reader.
package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"time"

	"github.com/project-horizon/horizon-core/services/ai/knowledge"
	"github.com/project-horizon/horizon-core/services/ai/learning"
	"github.com/project-horizon/horizon-core/services/ai/runtime"
)

func must(err error) {
	if err != nil {
		panic(err)
	}
}

func digest(data []byte) string {
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}

func read(path string) []byte {
	data, err := os.ReadFile(path)
	must(err)
	return data
}

func stateChecksum(r *runtime.BrainRuntime) string {
	dir, err := os.MkdirTemp("", "horizon-zero-prefix-state-")
	must(err)
	defer os.RemoveAll(dir)
	path := filepath.Join(dir, "state.json")
	must(r.Checkpoint(path))
	var s runtime.BrainSnapshot
	must(json.Unmarshal(read(path), &s))
	return s.Checksum
}

func main() {
	dir := os.Args[1]
	must(os.Mkdir(dir, 0700)) // Refuse to replace an existing fixture.
	at := time.Unix(100, 0).UTC()
	r := runtime.NewBrainRuntime(knowledge.NewBrain())
	r.SetClock(runtime.NewFixedClock(at))
	log, err := runtime.OpenEventLog(filepath.Join(dir, "events.jsonl"))
	must(err)
	defer log.Close()
	r.SetEventLog(log)
	zero := filepath.Join(dir, "zero-checkpoint.json")
	must(r.Checkpoint(zero))
	for i, tokens := range [][]string{{"naku", "melo", "tiva"}, {"sopo", "keri", "luma"}} {
		_, err := r.LearnExperience(learning.Experience{Sequence: tokens, Source: "zero-prefix-v1", Weight: .9, Confidence: .9}, at.Add(time.Duration(i)*time.Second))
		must(err)
	}
	// Reference the complete detached continuation checksum, as returned by
	// RebuildRuntimeFromCheckpointAndJournal. No recovery is called here:
	// that reader cannot handle this fixture on the pre-fix source.
	r.SetEventLog(nil)
	before := stateChecksum(r)
	next := at.Add(2 * time.Second)
	r.SetClock(runtime.NewFixedClock(next))
	out, err := r.CognitiveThink(1)
	must(err)
	output, err := json.Marshal(out)
	must(err)
	reference := map[string]any{
		"producer_commit":         "56afe5dd449eeebc6cb69e0e13561589fb54002f",
		"zero_checkpoint_sha256":  digest(read(zero)),
		"events_sha256":           digest(read(filepath.Join(dir, "events.jsonl"))),
		"before_thought_checksum": before,
		"after_thought_checksum":  stateChecksum(r),
		"next_thought_sha256":     digest(output),
		"next_thought_at":         next,
		"recovered_sequence":      2,
	}
	data, err := json.MarshalIndent(reference, "", "  ")
	must(err)
	must(os.WriteFile(filepath.Join(dir, "reference.json"), append(data, '\n'), 0600))
}
