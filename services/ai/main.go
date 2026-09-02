package main

import (
	"context"
	"log/slog"
	"os"
	"time"

	"github.com/project-horizon/horizon-core/services/ai/config"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))

	cfg, err := config.Load()
	if err != nil {
		logger.Error("failed to load AI cluster configuration", "error", err)
		os.Exit(1)
	}

	cluster := NewCluster()
	ctx, cancel := context.WithTimeout(context.Background(), cfg.DecisionTimeout)
	defer cancel()

	decision, err := cluster.Decide(ctx, bootstrapSignals(), ContextInformation{
		MissionID:        "bootstrap",
		UserID:           "system",
		EnvironmentRisk:  0.20,
		BatteryLevel:     0.95,
		Connectivity:     0.90,
		RequiresLowPower: false,
	})
	if err != nil {
		logger.Error("AI cluster bootstrap decision failed", "error", err)
		os.Exit(1)
	}

	logger.Info(
		"AI cluster ready",
		"service", cfg.ServiceName,
		"environment", cfg.Environment,
		"port", cfg.Port,
		"command", decision.Command,
		"confidence", decision.Confidence,
	)
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
