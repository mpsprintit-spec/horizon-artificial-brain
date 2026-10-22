// Run on PR37, b5b697f5570495094c9a2f2db7432708c3d166c1, with Go 1.22.12.
package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/project-horizon/horizon-core/services/ai/knowledge"
	"github.com/project-horizon/horizon-core/services/ai/runtime"
)

func must(err error) {
	if err != nil {
		panic(err)
	}
}
func main() {
	out := os.Args[1]
	if _, err := os.Stat(filepath.Join(out, "commands.jsonl")); !os.IsNotExist(err) {
		panic("journal already exists")
	}
	at := time.Date(2026, 10, 20, 9, 0, 0, 0, time.UTC)
	r := runtime.NewBrainRuntime(knowledge.NewBrain())
	r.SetClock(runtime.NewFixedClock(at))
	log, err := runtime.OpenEventLog(filepath.Join(out, "commands.jsonl"))
	must(err)
	r.SetEventLog(log)
	g, err := r.GroundVectorAt(knowledge.NewNeuralVector([]float64{.2, .8}), at)
	must(err)
	for i := 1; i <= 3; i++ {
		must(r.RegisterActionBinding(runtime.ActionBinding{RequestID: fmt.Sprintf("request-%d", i), TargetNodeIDs: []knowledge.NodeID{g.NodeID}}))
	}
	_, err = r.ObserveOutcome(runtime.OutcomeEvent{RequestID: "request-1", Source: "same-camera", Success: true, Observation: []string{"seen"}, Reliability: .9, ObservedAt: at.Add(time.Second)})
	must(err)
	must(r.Checkpoint(filepath.Join(out, "prefix-checkpoint.json")))
	_, err = r.ObserveOutcome(runtime.OutcomeEvent{RequestID: "request-2", Source: "same-camera", Success: true, Observation: []string{"seen"}, Reliability: .9, ObservedAt: at.Add(2 * time.Second)})
	must(err)
	for i := 0; i < 4; i++ {
		v := []float64{.2, .4, 0}
		if i%2 == 1 {
			v = []float64{.8, 0, .4}
		}
		_, err = r.ProcessSensorObservation(runtime.SensorObservation{SampleID: fmt.Sprintf("sample-%d", i), SessionID: "stream", Source: "camera", Modality: "vision", Timestamp: at.Add(time.Duration(i+3) * time.Second), ChannelStates: v, Confidence: .9, NoiseTolerance: .02})
		must(err)
	}
	must(log.Close())
	r.SetEventLog(nil)
	snapshot := func() runtime.BrainSnapshot {
		path := filepath.Join(out, "temporary.json")
		must(r.Checkpoint(path))
		data, e := os.ReadFile(path)
		must(e)
		var s runtime.BrainSnapshot
		must(json.Unmarshal(data, &s))
		must(os.Remove(path))
		return s
	}
	before := snapshot()
	next := runtime.OutcomeEvent{RequestID: "request-3", Source: "same-camera", Success: true, Observation: []string{"seen-again"}, Reliability: .9, ObservedAt: at.Add(10 * time.Second)}
	_, err = r.ObserveOutcome(next)
	must(err)
	after := snapshot()
	data, err := json.MarshalIndent(struct {
		Commit   string               `json:"commit"`
		Checksum string               `json:"checksum"`
		Next     runtime.OutcomeEvent `json:"next_outcome"`
		After    string               `json:"after_next_checksum"`
		Evidence any                  `json:"evidence"`
	}{"b5b697f5570495094c9a2f2db7432708c3d166c1", before.Checksum, next, after.Checksum, before.EvidenceLedger}, "", "  ")
	must(err)
	must(os.WriteFile(filepath.Join(out, "reference.json"), append(data, '\n'), 0644))
	must(os.Remove(filepath.Join(out, "commands.jsonl.writer.lock")))
	fmt.Println(string(data))
}
