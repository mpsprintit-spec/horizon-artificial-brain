package main

import (
	"encoding/json"
	"fmt"
	"math/rand"
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
	dir, err := os.MkdirTemp("", "horizon-parent-roundoff-")
	must(err)
	defer os.RemoveAll(dir)
	at := time.Unix(100, 0).UTC()
	r := runtime.NewBrainRuntime(knowledge.NewBrain())
	r.SetClock(runtime.NewFixedClock(at))
	pops := map[string][]knowledge.NodeID{}
	coordinates := rand.New(rand.NewSource(7)).Perm(32)
	for i, name := range []string{"seed", "object-a", "object-b", "motion-a", "motion-b", "Bola", "Gelas", " ", "diam.", "jatuh.", ""} {
		vector := make([]float64, 32)
		vector[coordinates[i]] = 1
		p, err := r.GroundVectorAt(knowledge.NewNeuralVector(vector), at)
		must(err)
		pops[name] = p.Population
		if i >= 5 {
			e, err := knowledge.NewSpellingExposure(p.Population, name, .9, .9)
			must(err)
			_, err = r.LearnSpelling(e, at)
			must(err)
		}
	}
	for round := 0; round < 6; round++ {
		for _, lesson := range [][3]string{{"seed", "Bola", "object-a"}, {"seed", "Gelas", "object-b"}, {"Bola", " ", ""}, {"Gelas", " ", ""}, {" ", "diam.", "motion-a"}, {" ", "jatuh.", "motion-b"}, {"diam.", "", ""}, {"jatuh.", "", ""}} {
			e := learning.Experience{NeuralSequence: [][]knowledge.NodeID{pops[lesson[0]], pops[lesson[1]]}, Weight: .9, Confidence: .9}
			if lesson[2] != "" {
				e.NeuralContext = [][]knowledge.NodeID{pops[lesson[2]]}
			}
			_, err := r.LearnExperience(e, at)
			must(err)
		}
	}
	_, err = r.CognitiveProcess(runtime.Event{ID: "context", Timestamp: at.Add(time.Second), Cycles: 1, EpisodeID: "control", ResetContext: true, NeuralTrace: &knowledge.ObservationTrace{Stimulus: [][]knowledge.NodeID{pops["object-a"], pops["motion-b"]}}})
	must(err)
	cp := filepath.Join(dir, "full.json")
	must(r.Checkpoint(cp))
	copy := runtime.NewBrainRuntime(knowledge.NewBrain())
	must(copy.RestoreCheckpoint(cp))
	for _, candidate := range []*runtime.BrainRuntime{r, copy} {
		candidate.SetClock(runtime.NewFixedClock(at.Add(2 * time.Second)))
	}
	a, err := r.CognitiveThink(1)
	must(err)
	b, err := copy.CognitiveThink(1)
	must(err)
	aa, err := json.Marshal(a)
	must(err)
	bb, err := json.Marshal(b)
	must(err)
	i := 0
	for i < len(aa) && i < len(bb) && aa[i] == bb[i] {
		i++
	}
	lo := max(0, i-65)
	hiA := min(len(aa), i+85)
	hiB := min(len(bb), i+85)
	result := map[string]any{"producer_commit": "99820e0c7912440f84fa4e0a223a3880b88bbf7b", "preview_called": false, "byte_identical": string(aa) == string(bb), "first_difference": i, "original_fragment": string(aa[lo:hiA]), "restored_fragment": string(bb[lo:hiB])}
	data, err := json.MarshalIndent(result, "", "  ")
	must(err)
	fmt.Println(string(data))
}
