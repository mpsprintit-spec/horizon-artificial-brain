//go:build ignore

// Generate a synthetic, explicitly opted-in propagation fixture in a NEW dir.
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
func digest(data []byte) string { sum := sha256.Sum256(data); return hex.EncodeToString(sum[:]) }
func main() {
	dir := os.Args[1]
	must(os.Mkdir(dir, 0700))
	at := time.Unix(100, 0).UTC()
	r := runtime.NewCanonicalPropagationRuntime()
	r.SetClock(runtime.NewFixedClock(at))
	log, err := runtime.OpenEventLog(filepath.Join(dir, "events.jsonl"))
	must(err)
	defer log.Close()
	r.SetEventLog(log)
	var populations [][]knowledge.NodeID
	for i, text := range []string{"", "na 🌌", ""} {
		vector := make([]float64, 8)
		vector[i] = 1
		p, err := r.GroundVectorAt(knowledge.NewNeuralVector(vector), at)
		must(err)
		populations = append(populations, p.Population)
		if i > 0 {
			e, err := knowledge.NewSpellingExposure(p.Population, text, .9, .9)
			must(err)
			_, err = r.LearnSpelling(e, at)
			must(err)
		}
	}
	for i := 0; i < 6; i++ {
		for j := 0; j < 2; j++ {
			_, err := r.LearnExperience(learning.Experience{NeuralSequence: [][]knowledge.NodeID{populations[j], populations[j+1]}, Weight: .9, Confidence: .9}, at)
			must(err)
		}
	}
	_, err = r.CognitiveProcess(runtime.Event{ID: "numeric-context", Timestamp: at.Add(time.Second), Cycles: 2, EpisodeID: "fixture", ResetContext: true, NeuralTrace: &knowledge.ObservationTrace{Stimulus: [][]knowledge.NodeID{populations[0], populations[1]}}})
	must(err)
	journal, err := os.ReadFile(filepath.Join(dir, "events.jsonl"))
	must(err)
	events, err := runtime.ReadEventLog(filepath.Join(dir, "events.jsonl"))
	must(err)
	r.SetEventLog(nil)
	checkpoint := filepath.Join(dir, "scratch-checkpoint.json")
	stateDigest := func() string {
		must(r.Checkpoint(checkpoint))
		data, err := os.ReadFile(checkpoint)
		must(err)
		return digest(data)
	}
	initial := stateDigest()
	var future []map[string]any
	for i := 0; i < 8; i++ {
		now := at.Add(time.Duration(i+2) * time.Second)
		r.SetClock(runtime.NewFixedClock(now))
		out, err := r.CognitiveThink(1)
		must(err)
		data, err := json.Marshal(out)
		must(err)
		future = append(future, map[string]any{"at": now, "output_sha256": digest(data), "checkpoint_sha256": stateDigest()})
	}
	must(os.Remove(checkpoint))
	data, err := json.MarshalIndent(map[string]any{
		"parent_commit":     "3f3716cb283372a18cf12a0ae17fdf2ff515118f",
		"dynamics_encoding": knowledge.CanonicalPropagationDynamics,
		"journal_sha256":    digest(journal), "records": len(events),
		"initial_checkpoint_sha256": initial, "initial_clock": at, "future": future,
	}, "", "  ")
	must(err)
	must(os.WriteFile(filepath.Join(dir, "reference.json"), append(data, '\n'), 0600))
}
