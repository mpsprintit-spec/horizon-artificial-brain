package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"strings"
	"time"

	"github.com/project-horizon/horizon-core/services/ai/config"
	"github.com/project-horizon/horizon-core/services/ai/core"
	"github.com/project-horizon/horizon-core/services/ai/runtime"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))

	cfg, err := config.Load()
	if err != nil {
		logger.Error("failed to load AI configuration", "error", err)
		os.Exit(1)
	}

	// BrainRuntime is the single cognitive authority for the production
	// executable. Cluster remains available as a legacy sensor/decision
	// compatibility component, but it is not invoked as the brain here.
	horizon := core.NewHorizonEngine()
	signals := bootstrapSignals()
	for _, signal := range signals {
		if err := signal.Validate(); err != nil {
			logger.Error("invalid bootstrap signal", "error", err)
			os.Exit(1)
		}
	}

	ctx, cancel := context.WithTimeout(context.Background(), cfg.DecisionTimeout)
	defer cancel()

	stimulus := signalStimulus(signals)
	output, err := horizon.Runtime.CognitiveProcess(ctx, runtime.Event{
		ID:        "bootstrap",
		Stimulus:  stimulus,
		Cycles:    8,
		Timestamp: time.Now().UTC(),
	})
	if err != nil {
		logger.Error("Horizon brain bootstrap failed", "error", err)
		os.Exit(1)
	}

	logger.Info(
		"Horizon brain ready",
		"service", cfg.ServiceName,
		"environment", cfg.Environment,
		"port", cfg.Port,
		"brain_identity", output.BrainIdentity,
		"sequence", output.Sequence,
		"resonance", output.Resonance,
		"prediction_error", output.PredictionError,
		"stimulus_count", len(stimulus),
	)
}

// signalStimulus is an input adapter only. It preserves signal provenance at
// the event boundary without assigning semantic relations or creating a
// second cognitive engine.
func signalStimulus(signals []Signal) []string {
	stimulus := make([]string, 0, len(signals)*2)
	for _, signal := range signals {
		stimulus = append(stimulus, string(signal.Type))
		if source := strings.TrimSpace(signal.Source); source != "" {
			stimulus = append(stimulus, fmt.Sprintf("source:%s", source))
		}
	}
	return stimulus
}

func bootstrapSignals() []Signal {
	now := time.Now().UTC()
	return []Signal{
		{Type: SignalVision, Source: "bootstrap_camera", Confidence: 0.86, Timestamp: now, Payload: map[string]float64{"objects": 1}},
		{Type: SignalTelemetry, Source: "bootstrap_drone", Confidence: 0.91, Timestamp: now, Payload: map[string]float64{"stability": 0.98}},
		{Type: SignalSensor, Source: "bootstrap_imu", Confidence: 0.88, Timestamp: now, Payload: map[string]float64{"variance": 0.04}},
		{Type: SignalLocation, Source: "bootstrap_gps", Confidence: 0.83, Timestamp: now, Payload: map[string]float64{"accuracy": 0.92}},
		{Type: SignalUser, Source: "bootstrap_profile", Confidence: 0.77, Timestamp: now, Payload: map[string]float64{"attention": 0.80}},
	}
}
