package core

import (
	"context"
	"fmt"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/project-horizon/horizon-core/services/ai/execution"
	"github.com/project-horizon/horizon-core/services/ai/knowledge"
	"github.com/project-horizon/horizon-core/services/ai/perception"
	"github.com/project-horizon/horizon-core/services/ai/runtime"
)

type compositionTestMotor struct {
	calls  int
	values []float64
}

func (m *compositionTestMotor) ExecuteMotor(_ context.Context, _ string, values []float64) error {
	m.calls++
	m.values = slices.Clone(values)
	return nil
}

func TestCompositionFacadePulseThinkAndPersistence(t *testing.T) {
	r := runtime.NewBrainRuntime(knowledge.NewBrain())
	owner, err := runtime.NewRuntimeSupervisor(r)
	if err != nil {
		t.Fatal(err)
	}
	h := &HorizonEngine{supervisor: owner, Perception: perception.UserInputPerception{}}
	at := time.Now().UTC().Add(-time.Second)
	r.SetClock(runtime.NewFixedClock(at))
	if err := h.InitializeNeuralComposition(knowledge.DefaultCompositionConfig(), filepath.Join(t.TempDir(), "rollback.json"), at); err != nil {
		t.Fatal(err)
	}
	makeScene := func() knowledge.CompositionScene {
		return knowledge.CompositionScene{ObservedAt: at, Point: []float64{0, 0, -1}, Patches: []knowledge.CompositionPatch{{Values: []float64{-.8, -.8, .8, -1, -1}, XY: []float64{-.5, 0}}, {Values: []float64{.8, .8, -.8, -1, -1}, XY: []float64{.5, 0}}}}
	}
	step := 0
	teach := func(words []string, target int, on bool) {
		step++
		before := makeScene()
		before.ObservedAt = at.Add(time.Duration(step) * time.Microsecond)
		after := makeScene()
		after.ObservedAt = before.ObservedAt.Add(time.Nanosecond)
		state := -1.0
		if on {
			state = 1
		}
		after.Patches[target].Values[3] = state
		motor := append(slices.Clone(before.Patches[target].XY), state, 1)
		input := knowledge.CompositionInput{Scene: before}
		for _, word := range words {
			input.Words = append(input.Words, knowledge.EncodeObservation(word, "language").Values)
		}
		e := knowledge.CompositionExposure{ID: fmt.Sprintf("demo-%d", step), Source: "teacher", EpisodeID: "two-patch-room", IndependenceGroup: "one-teacher", CausalGroup: fmt.Sprintf("contact-%d", step), Reliability: 1, Input: input, Motor: motor, After: after}
		if _, err := h.LearnComposition(e, after.ObservedAt); err != nil {
			t.Fatal(err)
		}
	}
	for i := 0; i < 16; i++ {
		teach(nil, i%2, i%2 == 0)
	}
	for i := 0; i < 40; i++ {
		teach([]string{"nyalakan", "lampu"}, 0, true)
		teach([]string{"matikan", "lampu"}, 0, false)
		teach([]string{"matikan", "televisi"}, 1, false)
	}
	scene := makeScene()
	// This is a new physical observation after teaching, not the old
	// pre-training frame. Race instrumentation may take longer than its TTL.
	scene.ObservedAt = time.Now().UTC()
	scene.Patches[0].XY, scene.Patches[1].XY = scene.Patches[1].XY, scene.Patches[0].XY
	if _, err := h.ObserveCompositionScene("moved-scene", "simulator", scene); err != nil {
		t.Fatal(err)
	}
	r.SetClock(runtime.NewFixedClock(scene.ObservedAt.Add(time.Millisecond)))
	out, err := h.Pulse("nyalakan televisi")
	if err != nil {
		t.Fatal(err)
	}
	if out.Composition == nil || !out.Composition.Accepted || !slices.Equal(out.Composition.Motor, []float64{-.5, 0, 1, 1}) {
		t.Fatalf("ordinary facade Pulse did not compose and bind current location: %+v", out.Composition)
	}
	if out.Recommendation != nil || h.Gateway != nil {
		t.Fatal("composition unexpectedly created action authorization or execution")
	}
	thought, err := h.ThinkActive(1)
	if err != nil || thought.Composition == nil {
		t.Fatalf("native autonomous thought lost composition state: %v", err)
	}
	h.Gateway = execution.NewExecutionGateway(nil)
	motor := &compositionTestMotor{}
	if err := h.Gateway.RegisterMotorActuator("room-contact", motor); err != nil {
		t.Fatal(err)
	}
	if err := h.RequestCompositionAction(*out.Composition, "room-contact", "stale-action", "test-operator", time.Now().Add(time.Minute)); err == nil {
		t.Fatal("stale pre-thought proposal was accepted")
	}
	fresh, err := h.Pulse("nyalakan televisi")
	if err != nil {
		t.Fatal(err)
	}
	if err := h.RequestCompositionAction(*fresh.Composition, "room-contact", "no-approval", "", time.Now().Add(time.Minute)); err == nil || motor.calls != 0 {
		t.Fatal("confidence authorized physical execution")
	}
	if err := h.RequestCompositionAction(*fresh.Composition, "room-contact", "approved-action", "test-operator", time.Now().Add(time.Minute)); err != nil {
		t.Fatal(err)
	}
	if motor.calls != 1 || !slices.Equal(motor.values, []float64{-.5, 0, 1, 1}) {
		t.Fatal("authorized numerical action did not reach the physical adapter")
	}
	path := filepath.Join(t.TempDir(), "serving-memory.json")
	if err := h.SaveActiveBrain(path); err != nil {
		t.Fatal(err)
	}
	r2 := runtime.NewBrainRuntime(knowledge.NewBrain())
	owner2, err := runtime.NewRuntimeSupervisor(r2)
	if err != nil {
		t.Fatal(err)
	}
	h2 := &HorizonEngine{supervisor: owner2, Perception: perception.UserInputPerception{}}
	if err := h2.LoadActiveBrain(path); err != nil {
		t.Fatal(err)
	}
	state, err := h2.CompositionStatus()
	if err != nil || state == nil || state.Scene == nil {
		t.Fatalf("facade startup lost full neural serving state: %v", err)
	}
	if err := h2.LoadActiveBrain(path); err == nil || !strings.Contains(err.Error(), "supervisor recovery") {
		t.Fatal("live checkpoint reset bypassed recovery")
	}
}

func TestCompositionFacadeEmptyBackupDoesNotPartiallyMigrate(t *testing.T) {
	h := NewHorizonEngine()
	revision := uint64(0)
	if err := h.Supervisor().With(func(r *runtime.BrainRuntime) error { revision = r.LastSequence(); return nil }); err != nil {
		t.Fatal(err)
	}
	if err := h.InitializeNeuralComposition(knowledge.DefaultCompositionConfig(), "", time.Time{}); err == nil {
		t.Fatal("empty backup accepted")
	}
	if err := h.Supervisor().With(func(r *runtime.BrainRuntime) error {
		if r.LastSequence() != revision {
			t.Fatal("bad activation partially migrated Brain")
		}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
}
