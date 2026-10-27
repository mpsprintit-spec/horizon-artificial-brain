package core

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/project-horizon/horizon-core/services/ai/knowledge"
	"github.com/project-horizon/horizon-core/services/ai/runtime"
	"github.com/project-horizon/horizon-core/services/ai/teaching"
)

func TestHNLFacadeUsesCanonicalBrainAndRecovery(t *testing.T) {
	at := time.Now().UTC().Add(-time.Second)
	rt := runtime.NewBrainRuntime(knowledge.NewBrain())
	rt.SetClock(runtime.NewFixedClock(at))
	owner, err := runtime.NewRuntimeSupervisor(rt)
	if err != nil {
		t.Fatal(err)
	}
	h := &HorizonEngine{supervisor: owner}
	cfg := knowledge.DefaultCompositionConfig()
	cfg.Version = knowledge.NeuralCompositionResolutionV5
	cfg.Initialization = "random"
	if err := h.InitializeNeuralComposition(cfg, filepath.Join(t.TempDir(), "backup.json"), at); err != nil {
		t.Fatal(err)
	}
	q := teaching.Question{Domain: 7, Relation: teaching.Sign, A: teaching.Ref{Lag: 0, Channel: 0}}
	p := teaching.Packet{Version: teaching.ExtendedVersion, Kind: teaching.Teach, ID: 1, Source: 1, Stream: 1, At: at.Add(time.Millisecond).UnixNano(), Question: q}
	for set, xs := range [][]float64{{-.8, -.4, .4, .8}, {-.6, -.2, .2, .6}} {
		for i, x := range xs {
			y := -1.
			if x > 0 {
				y = 1
			}
			p.Examples = append(p.Examples, teaching.Example{Role: teaching.Role(set + 1), Episode: uint64(set*10 + i + 1), ObservedAt: at.UnixNano(), Samples: []teaching.Sample{{Present: true, Values: [6]float64{x}}}, Outcome: &teaching.Evidence{Value: y, ObservedAt: at.Add(time.Nanosecond).UnixNano()}})
		}
	}
	wire, err := teaching.Marshal(p)
	if err != nil {
		t.Fatal(err)
	}
	response, err := h.ExchangeHNL(wire)
	if err != nil {
		t.Fatal(err)
	}
	receipt, err := teaching.Unmarshal(response)
	if err != nil {
		t.Fatal(err)
	}
	if receipt.Reply == nil || !receipt.Reply.Committed || receipt.Reply.LearnedN != 1 {
		t.Fatal("managed HNL teaching did not reach canonical numerical learning")
	}
	p.Kind = teaching.Ask
	p.ID = 2
	p.At = at.Add(2 * time.Millisecond).UnixNano()
	p.Examples = []teaching.Example{{Role: teaching.Probe, Episode: 20, ObservedAt: p.At, Samples: []teaching.Sample{{Present: true, Values: [6]float64{.33}}}}}
	wire, err = teaching.Marshal(p)
	if err != nil {
		t.Fatal(err)
	}
	saved := filepath.Join(t.TempDir(), "memory.json")
	if err = h.SaveActiveBrain(saved); err != nil {
		t.Fatal(err)
	}
	before, err := os.ReadFile(saved)
	if err != nil {
		t.Fatal(err)
	}
	response, err = h.ExchangeHNL(wire)
	if err != nil {
		t.Fatal(err)
	}
	answer, err := teaching.Unmarshal(response)
	if err != nil {
		t.Fatal(err)
	}
	if answer.Reply.Status != 3 || answer.Reply.Value <= 0 || h.Gateway != nil {
		t.Fatal("managed HNL query did not read the learned Brain without action execution")
	}
	if err = h.SaveActiveBrain(saved); err != nil {
		t.Fatal(err)
	}
	after, _ := os.ReadFile(saved)
	if !bytes.Equal(before, after) {
		t.Fatal("HNL query changed managed Brain state")
	}
	if _, err = h.ExchangeHNL(response); err == nil {
		t.Fatal("a reply was accepted as an instruction")
	}
	second := runtime.NewBrainRuntime(knowledge.NewBrain())
	owner2, err := runtime.NewRuntimeSupervisor(second)
	if err != nil {
		t.Fatal(err)
	}
	h2 := &HorizonEngine{supervisor: owner2}
	if err = h2.LoadActiveBrain(saved); err != nil {
		t.Fatal(err)
	}
	restored, err := h2.ExchangeHNL(wire)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(response, restored) {
		t.Fatal("managed restore changed HNL reply")
	}
	input, err := teaching.CompileAsk(p)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = h.ObserveCompositionInput(runtime.CompositionPerception{ID: "core-numeric-frame", Source: "hnl-core-test", Input: input}, at.Add(3*time.Millisecond)); err != nil {
		t.Fatal(err)
	}
	if err = owner.WithMutation(func(active *runtime.BrainRuntime) error {
		active.SetClock(runtime.NewFixedClock(at.Add(3 * time.Millisecond)))
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	thought, err := h.ThinkActive(1)
	if err != nil {
		t.Fatal(err)
	}
	if thought.Composition == nil || thought.Composition.NResolution == nil || thought.Composition.NResolution.Value <= 0 {
		t.Fatal("managed thinking lost actual numerical sequence")
	}
}
