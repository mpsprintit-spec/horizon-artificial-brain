// Run against the documented pre-H24 commit, never the candidate implementation:
// go run /path/to/generate.go /path/to/output
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
func main() {
	out := os.Args[1]
	must(os.MkdirAll(out, 0755))
	if _, err := os.Stat(filepath.Join(out, "commands.jsonl")); !os.IsNotExist(err) {
		panic("output already contains a journal")
	}
	at := time.Date(2026, 10, 19, 10, 0, 0, 0, time.UTC)
	r := runtime.NewBrainRuntime(knowledge.NewBrain())
	r.SetClock(runtime.NewFixedClock(at))
	log, err := runtime.OpenEventLog(filepath.Join(out, "commands.jsonl"))
	must(err)
	r.SetEventLog(log)
	_, err = r.LearnExperience(learning.Experience{Sequence: []string{"naku", "melo", "tiva"}, Weight: .8, Confidence: .8, Source: "legacy-teacher", Modality: "language"}, at)
	must(err)
	must(r.Checkpoint(filepath.Join(out, "prefix-checkpoint.json")))
	o := runtime.NewCognitiveOrchestrator(r)
	_, _, err = o.ProcessObservation(runtime.Event{ID: "old-text", Timestamp: at.Add(time.Second), Cycles: 1}, runtime.ObservationInput{Source: "user", Modality: "text", Tokens: []string{"NAKU", "melo"}, ContextTokens: []string{"old-context"}, DataTokens: []string{"old-data"}}, nil)
	must(err)
	_, _, err = o.ProcessObservation(runtime.Event{ID: "old-language", Timestamp: at.Add(2 * time.Second), Cycles: 1}, runtime.ObservationInput{Source: "keyboard", Modality: "language", Tokens: []string{"MiXeD"}}, &learning.Experience{Sequence: []string{"mixed", "target"}, Weight: .6, Confidence: .7})
	must(err)
	_, _, err = r.Process(runtime.Event{ID: "old-fallback", Stimulus: []string{"FALLBACK"}, Timestamp: at.Add(3 * time.Second), Cycles: 1})
	must(err)
	_, err = r.CognitiveProcess(runtime.Event{ID: "old-cognitive", Stimulus: []string{"COGNITIVE"}, Timestamp: at.Add(4 * time.Second), Cycles: 1})
	must(err)
	_, err = r.ProcessKnowledgeDocument(knowledge.KnowledgeDocument{ID: "legacy-doc", Domain: "test", Source: "archive", SourceType: "text", Experiences: []knowledge.KnowledgeExperienceInput{{ID: "old-step", Before: []knowledge.KnowledgeDatum{{Value: "BeforeValue", Modality: "text"}}, After: []knowledge.KnowledgeDatum{{Value: "AfterValue", Modality: "text"}}}}}, at.Add(5*time.Second))
	must(err)
	must(log.Close())
	r.SetEventLog(nil)
	checksum := func() string {
		path := filepath.Join(out, "temporary-checkpoint.json")
		must(r.Checkpoint(path))
		data, err := os.ReadFile(path)
		must(err)
		var s runtime.BrainSnapshot
		must(json.Unmarshal(data, &s))
		must(os.Remove(path))
		return s.Checksum
	}
	before := checksum()
	next, err := r.CognitiveThink(1)
	must(err)
	after := checksum()
	foundation, err := knowledge.NewBootstrapBrain().CanonicalJSON()
	must(err)
	sum := sha256.Sum256(foundation)
	data, err := json.MarshalIndent(struct {
		Commit     string                  `json:"commit"`
		Checksum   string                  `json:"checksum"`
		Next       runtime.CognitiveOutput `json:"next"`
		AfterNext  string                  `json:"after_next_checksum"`
		Foundation string                  `json:"bootstrap_brain_sha256"`
	}{"68708eb852ee900d636edec0fe3e4d8d0fbe1f59", before, next, after, hex.EncodeToString(sum[:])}, "", "  ")
	must(err)
	must(os.WriteFile(filepath.Join(out, "reference.json"), append(data, '\n'), 0644))
	must(os.Remove(filepath.Join(out, "commands.jsonl.writer.lock")))
}
