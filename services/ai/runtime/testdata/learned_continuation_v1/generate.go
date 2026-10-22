//go:build ignore

// Generate a synthetic v1 continuation fixture in a new directory.
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
	r := runtime.NewBrainRuntime(knowledge.NewBrain())
	r.SetClock(runtime.NewFixedClock(at))
	log, err := runtime.OpenEventLog(filepath.Join(dir, "events.jsonl"))
	must(err)
	defer log.Close()
	r.SetEventLog(log)
	var populations [][]knowledge.NodeID
	for i, text := range []string{"", "na ", "lu", ""} {
		vector := make([]float64, 32)
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
		for j := 0; j < 3; j++ {
			_, err := r.LearnExperience(learning.Experience{NeuralSequence: [][]knowledge.NodeID{populations[j], populations[j+1]}, Weight: .9, Confidence: .9}, at)
			must(err)
		}
	}
	before, err := r.InspectBrain()
	must(err)
	out, err := r.ReadLearnedContinuation(&knowledge.ObservationTrace{Stimulus: [][]knowledge.NodeID{populations[0]}}, 8, at)
	must(err)
	if out.Status != runtime.ContinuationComplete || out.Text != "na lu" {
		panic("fixture failed to complete its learned pieces")
	}
	after, err := r.InspectBrain()
	must(err)
	if string(before.Graph.CanonicalBytes()) != string(after.Graph.CanonicalBytes()) {
		panic("preview changed the Brain")
	}
	journal, err := os.ReadFile(filepath.Join(dir, "events.jsonl"))
	must(err)
	events, err := runtime.ReadEventLog(filepath.Join(dir, "events.jsonl"))
	must(err)
	data, err := json.MarshalIndent(map[string]any{
		"parent_commit": "99820e0c7912440f84fa4e0a223a3880b88bbf7b",
		"encoding":      runtime.ContinuationEncodingV1, "population": populations[0], "at": at,
		"readout": out, "canonical_sha256": digest(before.Graph.CanonicalBytes()),
		"journal_sha256": digest(journal), "records": len(events),
	}, "", "  ")
	must(err)
	must(os.WriteFile(filepath.Join(dir, "reference.json"), append(data, '\n'), 0600))
}
