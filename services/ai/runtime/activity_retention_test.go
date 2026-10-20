package runtime

import (
	"fmt"
	"math"
	"math/rand"
	"testing"
	"time"

	"github.com/project-horizon/horizon-core/services/ai/activation"
	"github.com/project-horizon/horizon-core/services/ai/knowledge"
	"github.com/project-horizon/horizon-core/services/ai/learning"
)

type retentionTask struct{ cue, target []knowledge.NodeID }
type retentionReadout struct {
	Mean                  float64
	Above, Correct, Total int
}

func retentionFoundation(t *testing.T, seed int64) (*BrainRuntime, []retentionTask, []retentionTask, []knowledge.NodeID, time.Time) {
	t.Helper()
	r := NewBrainRuntime(knowledge.NewBrain())
	at := time.Date(2026, 10, 20, 3, 0, 0, 123, time.UTC)
	r.SetClock(NewFixedClock(at))
	// Two-member cues/outcomes are synthetic distributed neural populations.
	// Insertion permutations prevent an accidental preference for low IDs.
	nodes := make([]knowledge.NodeID, 40*8+2)
	for _, i := range rand.New(rand.NewSource(seed)).Perm(len(nodes)) {
		nodes[i] = r.brain.StoreAt(fmt.Sprintf("unit-%d", i), at).ID
	}
	oldTask, newTask := make([]retentionTask, 40), make([]retentionTask, 40)
	for i := range oldTask {
		base := i * 8
		oldTask[i] = retentionTask{nodes[base : base+2], nodes[base+2 : base+4]}
		newTask[i] = retentionTask{nodes[base+4 : base+6], nodes[base+6 : base+8]}
	}
	trainRetentionTask(t, r, oldTask, at)
	return r, oldTask, newTask, nodes[320:], at
}

func trainRetentionTask(t *testing.T, r *BrainRuntime, task []retentionTask, at time.Time) {
	t.Helper()
	for repeat := 0; repeat < 3; repeat++ {
		for _, pair := range task {
			for _, source := range pair.cue {
				if _, err := r.LearnExperience(learning.Experience{NeuralSequence: [][]knowledge.NodeID{{source}, pair.target}, Weight: .7, Confidence: .8, Source: "retention-probe"}, at); err != nil {
					t.Fatal(err)
				}
			}
		}
	}
}

func readRetentionTask(r *BrainRuntime, task, alternatives []retentionTask, at time.Time) retentionReadout {
	var readout retentionReadout
	for i, pair := range task {
		// A fresh read-only probe excludes incidental residual recurrent activity.
		prediction := activation.NewEngine(r.brain).PredictOutcomeFromNodes(pair.cue, at, 1)
		competitor := 0.0
		for _, id := range alternatives[i].target {
			competitor += prediction.State[id] / float64(len(alternatives[i].target))
		}
		for _, id := range pair.target {
			score := prediction.State[id]
			readout.Mean += score
			readout.Total++
			if score >= .25 {
				readout.Above++
			}
			if score > competitor {
				readout.Correct++
			}
		}
	}
	readout.Mean /= float64(readout.Total)
	return readout
}

func TestLocalActivityRetainsOldTaskWhileLearningNewTask(t *testing.T) {
	for _, seed := range []int64{7, 17, 47} {
		t.Run(fmt.Sprintf("seed-%d", seed), func(t *testing.T) {
			foundation, oldTask, newTask, noise, at := retentionFoundation(t, seed)
			path := orderedSave(t, foundation)
			before := readRetentionTask(foundation, oldTask, newTask, at)
			finalAt := at.Add(time.Second)
			idle := orderedFixtureClone(t, path)
			idle.brain.ApplyMemoryDynamicsWithEncoding(finalAt, knowledge.CurrentDynamics)
			idleReadout := readRetentionTask(idle, oldTask, newTask, finalAt)
			for _, protocol := range []string{"empty-activity", "unrelated", "overlapping-output"} {
				active := orderedFixtureClone(t, path)
				for i := 1; i <= 96; i++ {
					tick := at.Add(time.Duration(i) * time.Second / 96)
					if protocol == "empty-activity" {
						active.brain.ApplyMemoryDynamicsWithEncoding(tick, knowledge.CurrentDynamics)
						active.activation.ApplyActivityPlasticityWithDynamics(nil, nil, tick, knowledge.CurrentDynamics)
						continue
					}
					stimulus := noise
					if protocol == "overlapping-output" && i%2 == 1 {
						stimulus = nil
						for _, pair := range oldTask {
							stimulus = append(stimulus, pair.target...)
						}
					}
					if _, err := active.CognitiveProcess(Event{ID: fmt.Sprintf("noise-%d", i), EpisodeID: "unrelated", StimulusNodeIDs: stimulus, Timestamp: tick, Cycles: 1}); err != nil {
						t.Fatal(err)
					}
				}
				retained := readRetentionTask(active, oldTask, newTask, finalAt)
				untrained := readRetentionTask(active, newTask, oldTask, finalAt)
				trainRetentionTask(t, active, newTask, finalAt)
				acquired := readRetentionTask(active, newTask, oldTask, finalAt)
				afterNew := readRetentionTask(active, oldTask, newTask, finalAt)
				t.Logf("%s before=%+v idle=%+v interference=%+v old-after-new=%+v new-before=%+v new-after=%+v", protocol, before, idleReadout, retained, afterNew, untrained, acquired)
				if math.Abs(retained.Mean-idleReadout.Mean) > 1e-12 || retained.Above != retained.Total || retained.Correct != retained.Total || math.Abs(afterNew.Mean-idleReadout.Mean) > 1e-12 || afterNew.Above != afterNew.Total {
					t.Error("unrelated activity erased old predictive support")
				}
				if untrained.Above != 0 || acquired.Above != acquired.Total || acquired.Correct != acquired.Total || acquired.Mean < .25 {
					t.Error("retention blocked new task acquisition")
				}
			}
		})
	}
}

func TestLocalActivityRelevantContradictionStillRevisesOldTask(t *testing.T) {
	for _, seed := range []int64{7, 17, 47} {
		r, oldTask, newTask, _, at := retentionFoundation(t, seed)
		competing := make([]retentionTask, len(oldTask))
		predicted, actual := map[knowledge.NodeID]float64{}, map[knowledge.NodeID]float64{}
		for i, pair := range oldTask {
			competing[i] = retentionTask{pair.cue, newTask[i].target}
			for _, id := range pair.cue {
				predicted[id] = 1
				actual[id] = 1
			}
			for _, id := range pair.target {
				predicted[id] = 1
			}
			for _, id := range newTask[i].target {
				actual[id] = 1
			}
		}
		before := readRetentionTask(r, oldTask, competing, at)
		trainRetentionTask(t, r, competing, at)
		patterns := len(r.brain.Patterns.All())
		// Controlled negative evidence: the same sources participate, their old
		// outcomes fail, and the competing outcomes occur. This isolates causal
		// error credit; it is not a claim of natural-language contradiction skill.
		for i := 1; i <= 8; i++ {
			tick := at.Add(time.Duration(i) * time.Second / 8)
			r.brain.ApplyMemoryDynamicsWithEncoding(tick, knowledge.CurrentDynamics)
			r.activation.ApplyPredictionErrorPlasticityWithDynamics(predicted, actual, 1, tick, knowledge.CurrentDynamics)
		}
		after := readRetentionTask(r, oldTask, competing, at.Add(time.Second))
		alternative := readRetentionTask(r, competing, oldTask, at.Add(time.Second))
		t.Logf("seed=%d contradiction old-before=%+v old-after=%+v competing=%+v", seed, before, after, alternative)
		if after.Mean >= before.Mean*.5 || alternative.Above != alternative.Total || alternative.Correct != alternative.Total {
			t.Fatal("inactive-path protection froze relevant contradiction learning")
		}
		if len(r.brain.Patterns.All()) != patterns {
			t.Fatal("contradiction deleted historical patterns")
		}
		for _, pair := range oldTask {
			for _, source := range pair.cue {
				for _, target := range pair.target {
					s := r.brain.Registry.GetByID(source).FindDynamicSynapse(target, false)
					if s == nil || s.Dynamic.Frequency != 3 {
						t.Fatal("contradiction deleted the learned connection or its experience count")
					}
				}
			}
		}
	}
}
