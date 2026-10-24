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
	"github.com/project-horizon/horizon-core/services/ai/knowledge"
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
	brainMemoryPath := strings.TrimSpace(os.Getenv("HORIZON_BRAIN_MEMORY_PATH"))
	if brainMemoryPath == "" { brainMemoryPath = "brain_memory.json" }
	eventLogPath := strings.TrimSpace(os.Getenv("HORIZON_EVENT_LOG_PATH"))
	if eventLogPath == "" { eventLogPath = "horizon_events.jsonl" }
	checkpointPath := strings.TrimSpace(os.Getenv("HORIZON_CHECKPOINT_PATH"))
	if checkpointPath == "" { checkpointPath = brainMemoryPath + ".checkpoint.json" }

	eventLog, err := runtime.OpenEventLog(eventLogPath)
	if err != nil {
		logger.Error("failed to lock canonical brain event journal", "path", eventLogPath, "error", err)
		os.Exit(1)
	}
	defer eventLog.Close()

	// The production brain is reconstructed and anchored BEFORE any
	// asynchronous service starts accepting observations or cognitive work.
	horizon, loadedBrain, err := initializeProductionHorizon(eventLog, eventLogPath, brainMemoryPath, checkpointPath)
	if err != nil {
		logger.Error("canonical brain startup refused", "error", err)
		os.Exit(1)
	}
	var output runtime.CognitiveOutput
	stimulusCount := 0
	if !loadedBrain {
		signals := bootstrapSignals()
		for _, signal := range signals {
			if err := signal.Validate(); err != nil {
				logger.Error("invalid bootstrap signal", "error", err)
				os.Exit(1)
			}
		}
		stimulus := signalStimulus(signals)
		stimulusCount = len(stimulus)
		err = horizon.Supervisor().WithMutation(func(active *runtime.BrainRuntime) error {
			var processErr error
			output, processErr = active.CognitiveProcess(runtime.Event{
				ID: "bootstrap", Stimulus: stimulus, Cycles: 8, Timestamp: time.Now().UTC(),
			})
			if processErr != nil { return processErr }
			if processErr = active.SaveBrain(brainMemoryPath); processErr != nil { return processErr }
			return active.Checkpoint(checkpointPath)
		})
	} else {
		err = horizon.Supervisor().With(func(active *runtime.BrainRuntime) error {
			output = runtime.CognitiveOutput{
				BrainIdentity: runtime.BrainIdentity,
				Sequence: active.LastSequence(),
				Timestamp: time.Now().UTC(),
			}
			return nil
		})
	}
	if err != nil {
		logger.Error("Horizon brain initialization failed", "error", err)
		os.Exit(1)
	}

	// Explicit startup administration only. The enabled contract is persisted;
	// ordinary startup/merging code never silently trains or migrates a Brain.
	if backupPath:=strings.TrimSpace(os.Getenv("HORIZON_COMPOSITION_INIT_BACKUP"));backupPath!="" {
		state,stateErr:=horizon.CompositionStatus()
		if stateErr!=nil {logger.Error("composition status refused","error",stateErr);os.Exit(1)}
		if state==nil {
			if err:=horizon.InitializeNeuralComposition(knowledge.DefaultCompositionConfig(),backupPath,time.Now().UTC());err!=nil {logger.Error("explicit composition initialization refused","error",err);os.Exit(1)}
			if err:=horizon.Supervisor().WithMutation(func(rt *runtime.BrainRuntime)error{return rt.Checkpoint(checkpointPath)});err!=nil {logger.Error("composition checkpoint failed","error",err);os.Exit(1)}
		}
	}

	service, err := runtime.NewManagedContinuousService(horizon.Supervisor(), cfg.CognitiveInterval, 1)
	if err != nil {
		logger.Error("failed to initialize managed cognitive service", "error", err)
		os.Exit(1)
	}

	bridgeAddress := strings.TrimSpace(os.Getenv("HORIZON_BRIDGE_ADDRESS"))
	if bridgeAddress == "" {
		bridgeAddress = "127.0.0.1:8765"
	}
	bridgeOrigins := splitMonitorOrigins(os.Getenv("HORIZON_MONITOR_ORIGINS"))
	if len(bridgeOrigins) == 0 {
		bridgeOrigins = []string{"https://mpsprintit-spec.github.io", "http://localhost:8080"}
	}
	monitorBridge, err := bridge.NewManaged(horizon.Supervisor(), bridge.Config{
		AllowCompositionTraining: strings.EqualFold(strings.TrimSpace(os.Getenv("HORIZON_COMPOSITION_TRAINING")),"true"),
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
	bridgeTLSCert := strings.TrimSpace(os.Getenv("HORIZON_BRIDGE_TLS_CERT"))
	bridgeTLSKey := strings.TrimSpace(os.Getenv("HORIZON_BRIDGE_TLS_KEY"))
	if (bridgeTLSCert == "") != (bridgeTLSKey == "") {
		logger.Error("Horizon bridge TLS requires both HORIZON_BRIDGE_TLS_CERT and HORIZON_BRIDGE_TLS_KEY")
		os.Exit(1)
	}
	bridgeErrors := make(chan error, 1)
	go func() {
		logger.Info("Horizon monitor bridge listening", "address", bridgeAddress, "origins", bridgeOrigins, "tls", bridgeTLSCert != "")
		var err error
		if bridgeTLSCert != "" {
			err = bridgeServer.ListenAndServeTLS(bridgeTLSCert, bridgeTLSKey)
		} else {
			err = bridgeServer.ListenAndServe()
		}
		if err != nil && err != http.ErrServerClosed {
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
		"stimulus_count", stimulusCount,
		"cognitive_interval", cfg.CognitiveInterval,
	)

	if err := service.Run(ctx, func(output runtime.CognitiveOutput, err error) {
		if err != nil {
			logger.Error("continuous cognition cycle failed", "error", err)
			return
		}
		// The callback is outside the service's lease. Acquire a fresh
		// exclusive lease so a checkpoint and memory file always refer to
		// the current canonical runtime, even after an atomic handover.
		if saveErr := horizon.Supervisor().WithMutation(func(active *runtime.BrainRuntime) error {
			if err := active.SaveBrain(brainMemoryPath); err != nil { return err }
			return active.Checkpoint(checkpointPath)
		}); saveErr != nil {
			logger.Error("canonical brain persistence failed", "error", saveErr)
		}
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
