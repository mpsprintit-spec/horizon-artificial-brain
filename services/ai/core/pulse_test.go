package core

import (
	"context"
	"testing"
)

type recordingPlugin struct{ triggered []string }

func (p *recordingPlugin) Name() string { return "recording" }
func (p *recordingPlugin) Trigger(data string) {
	p.triggered = append(p.triggered, data)
}

func TestPulseNeuralPathDoesNotDispatchExecutionDirectly(t *testing.T) {
	previous := LegacyCognitionEnabled
	LegacyCognitionEnabled = false
	defer func() { LegacyCognitionEnabled = previous }()

	horizon := NewHorizonEngine()
	drone := &recordingPlugin{}
	horizon.Execution.RegisterPlugin("terbang", drone)
	horizon.Knowledge.Store("burung")
	horizon.Knowledge.Store("elang")
	horizon.Knowledge.Store("terbang")

	result := horizon.Pulse(context.Background(), TaskPulse{
		Stimulus: "burung elang",
		Context:  "terbang",
		Data:     "Koordinat Ketinggian 50m",
	})

	if result.Path != "neural_runtime" {
		t.Fatalf("expected neural runtime path, got %q", result.Path)
	}
	if len(drone.triggered) != 0 {
		t.Fatalf("neural cognition must not dispatch execution directly, got %#v", drone.triggered)
	}
}
