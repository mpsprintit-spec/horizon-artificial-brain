package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"strings"
	"os/signal"
	"syscall"
	"time"

	"github.com/project-horizon/horizon-core/services/ai/bridge"
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
	brainMemoryPath := strings.TrimSpace(os.Getenv("HORIZON_BRAIN_MEMORY_PATH"))
	if brainMemoryPath == "" { brainMemoryPath = "brain_memory.json" }
	eventLogPath := strings.TrimSpace(os.Getenv("HORIZON_EVENT_LOG_PATH"))
	if eventLogPath == "" { eventLogPath = "horizon_events.jsonl" }
	if _, statErr := os.Stat(brainMemoryPath); statErr == nil {
		if err := horizon.Runtime.LoadBrain(brainMemoryPath); err != nil {
			logger.Error("failed to load canonical brain memory", "path", brainMemoryPath, "error", err)
			os.Exit(1)
		}
	}
	eventLog, err := runtime.OpenEventLog(eventLogPath)
	if err != nil { logger.Error("failed to open brain event log", "path", eventLogPath, "error", err); os.Exit(1) }
	defer eventLog.Close()
	horizon.Runtime.SetEventLog(eventLog)
	signals := bootstrapSignals()
	for _, signal := range signals {
		if err := signal.Validate(); err != nil {
			logger.Error("invalid bootstrap signal", "error", err)
			os.Exit(1)
		}
	}

	stimulus := signalStimulus(signals)
	output, err := horizon.Runtime.CognitiveProcess(runtime.Event{
		ID:        "bootstrap",
		Stimulus:  stimulus,
		Cycles:    8,
		Timestamp: time.Now().UTC(),
	})
	if err != nil {
		logger.Error("Horizon brain bootstrap failed", "error", err)
		os.Exit(1)
	}

	service := runtime.NewContinuousService(horizon.Runtime, cfg.CognitiveInterval, 1)

	bridgeAddress := strings.TrimSpace(os.Getenv("HORIZON_BRIDGE_ADDRESS"))
	if bridgeAddress == "" {
		bridgeAddress = "127.0.0.1:8765"
	}
	bridgeOrigins := splitMonitorOrigins(os.Getenv("HORIZON_MONITOR_ORIGINS"))
	if len(bridgeOrigins) == 0 {
		bridgeOrigins = []string{"http://localhost:8080"}
	}
	monitorBridge, err := bridge.New(horizon.Runtime, bridge.Config{
		AllowedOrigins: bridgeOrigins,
		Token: strings.TrimSpace(os.Getenv("HORIZON_BRIDGE_TOKEN")),
		RuntimeCommit: strings.TrimSpace(os.Getenv("HORIZON_RUNTIME_COMMIT")),
		EventLogPath: eventLogPath,
		BrainMemoryPath: brainMemoryPath,
	})
	if err != nil {
		logger.Error("failed to initialize monitor bridge", "error", err)
		os.Exit(1)
	}
	bridgeServer := &http.Server{Addr: bridgeAddress, Handler: monitorBridge.Handler()}
	bridgeErrors := make(chan error, 1)
	go func() {
		logger.Info("Horizon monitor bridge listening", "address", bridgeAddress, "origins", bridgeOrigins)
		if err := bridgeServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			bridgeErrors <- err
		}
	}()

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

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
		"cognitive_interval", cfg.CognitiveInterval,
	)

	if err := service.Run(ctx, func(output runtime.CognitiveOutput, err error) {
		if err != nil {
			logger.Error("continuous cognition cycle failed", "error", err)
			return
		}
		if saveErr := horizon.Runtime.SaveBrain(brainMemoryPath); saveErr != nil { logger.Error("brain persistence failed", "error", saveErr) }
		logger.Info(
			"Horizon cognitive cycle",
			"sequence", output.Sequence,
			"resonance", output.Resonance,
			"prediction_error", output.PredictionError,
			"active_nodes", len(output.RankedNodeIDs),
		)
	}); err != nil && err != context.Canceled {
		logger.Error("continuous brain service stopped unexpectedly", "error", err)
		os.Exit(1)
	}
	if err := bridgeServer.Shutdown(context.Background()); err != nil {
		logger.Error("monitor bridge shutdown failed", "error", err)
	}
	select {
	case err := <-bridgeErrors:
		logger.Error("monitor bridge stopped unexpectedly", "error", err)
	default:
	}
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

func splitMonitorOrigins(value string) []string {
	parts := strings.Split(value, ",")
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		if origin := strings.TrimSpace(part); origin != "" {
			out = append(out, origin)
		}
	}
	return out
}
