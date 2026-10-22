//go:build ignore

// Generate an offline, synthetic codec-v1 journal and reference in a NEW directory.
package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"github.com/project-horizon/horizon-core/services/ai/knowledge"
	"github.com/project-horizon/horizon-core/services/ai/runtime"
	"os"
	"path/filepath"
	"time"
)

func must(err error) {
	if err != nil {
		panic(err)
	}
}
func digest(b []byte) string { h := sha256.Sum256(b); return hex.EncodeToString(h[:]) }
func canonical(r *runtime.BrainRuntime) []byte {
	v, e := r.InspectBrain()
	must(e)
	return v.Graph.CanonicalBytes()
}
func main() {
	dir := os.Args[1]
	must(os.Mkdir(dir, 0700))
	r := runtime.NewBrainRuntime(knowledge.NewBrain())
	at := time.Unix(100, 0).UTC()
	r.SetClock(runtime.NewFixedClock(at))
	log, e := runtime.OpenEventLog(filepath.Join(dir, "events.jsonl"))
	must(e)
	defer log.Close()
	r.SetEventLog(log)
	p, e := r.GroundVectorAt(knowledge.NewNeuralVector([]float64{.2, -.8, .7, -.3}), at)
	must(e)
	exposure, e := knowledge.NewSpellingExposure(p.Population, "A🌌", .8, .8)
	must(e)
	for i := 0; i < 4; i++ {
		_, e = r.LearnSpelling(exposure, at.Add(time.Duration(i)*time.Second))
		must(e)
	}
	spelling, e := r.ReadLearnedSpelling(p.Population)
	must(e)
	before := digest(canonical(r))
	next := at.Add(4 * time.Second)
	r.SetClock(runtime.NewFixedClock(next))
	out, e := r.CognitiveThink(1)
	must(e)
	output, e := json.Marshal(out)
	must(e)
	reference := map[string]any{"codec": knowledge.SpellingCodecV1, "population": p.Population, "spelling": spelling, "before_thought_sha256": before, "after_thought_sha256": digest(canonical(r)), "next_thought_sha256": digest(output), "next_thought_at": next}
	data, e := json.MarshalIndent(reference, "", "  ")
	must(e)
	must(os.WriteFile(filepath.Join(dir, "reference.json"), append(data, '\n'), 0600))
}
