//go:build ignore

// Run with the H18 source tree as cwd. Imports must resolve to the parent,
// not the new implementation, so the fixture independently pins old behavior.
package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/project-horizon/horizon-core/services/ai/knowledge"
	"github.com/project-horizon/horizon-core/services/ai/learning"
	"github.com/project-horizon/horizon-core/services/ai/runtime"
)

const parent = "8a4f9ced05d715b338bab4d324011e8c4f6c8596"

var at = time.Date(2026, 10, 21, 2, 0, 0, 0, time.UTC)

func must(err error) {
	if err != nil {
		panic(err)
	}
}
func main() {
	r := runtime.NewBrainRuntime(knowledge.NewBrain())
	r.SetClock(runtime.NewFixedClock(at))
	if os.Args[1] == "restore" || os.Args[1] == "recover" {
		if os.Args[1] == "recover" {
			log, err := runtime.OpenEventLog(os.Args[2] + ".events.jsonl")
			must(err)
			defer log.Close()
			r, err = runtime.RebuildRuntimeFromCheckpointAndJournal(log, os.Args[2])
			must(err)
		} else {
			must(r.RestoreCheckpoint(os.Args[2]))
		}
		hash, err := r.InspectBrain()
		must(err)
		data := hash.Graph.CanonicalBytes()
		fmt.Println(string(data))
		return
	}
	dir := os.Args[2]
	must(os.MkdirAll(dir, 0755))
	for i := 0; i < 3; i++ {
		_, err := r.LearnExperience(learning.Experience{Sequence: []string{"naku", "melo", "tiva"}, Weight: .8, Confidence: .8, Source: "fixture"}, at)
		must(err)
	}
	_, err := r.CognitiveProcess(runtime.Event{ID: "v12-prefix", Timestamp: at, Cycles: 1, Modality: "text", Stimulus: []string{"naku", "melo"}})
	must(err)
	must(r.Checkpoint(filepath.Join(dir, "checkpoint.json")))
	before, err := r.MonitorHandshake(parent)
	must(err)
	r.SetClock(runtime.NewFixedClock(at.Add(time.Second)))
	next, err := r.CognitiveThink(1)
	must(err)
	after, err := r.MonitorHandshake(parent)
	must(err)
	reference := map[string]any{"commit": parent, "before": before.CanonicalStateHash, "next": next, "after": after.CanonicalStateHash}
	data, err := json.MarshalIndent(reference, "", "  ")
	must(err)
	must(os.WriteFile(filepath.Join(dir, "reference.json"), append(data, '\n'), 0644))
	fmt.Println("Created the independent schema-12 checkpoint and next-thought reference from H18.")
}
