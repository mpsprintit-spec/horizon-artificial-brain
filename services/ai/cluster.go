package main

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"time"
)

// Module describes an HC-002 AI cluster capability.
type Module interface {
	Name() string
	Evaluate(context.Context, []Signal, ContextInformation) (float64, error)
}

// Cluster orchestrates Vision AI, Sensor Fusion AI, Decision AI, Drone AI, HUD AI, and Learning Engine.
type Cluster struct {
	modules []Module
	clock   func() time.Time
}

// NewCluster creates the production AI cluster composition without external dependencies.
func NewCluster() *Cluster {
	return &Cluster{
		modules: []Module{
			weightedModule{name: "vision_ai", weights: map[SignalType]float64{SignalVision: 0.70, SignalSensor: 0.10, SignalLocation: 0.10}},
			weightedModule{name: "sensor_fusion_ai", weights: map[SignalType]float64{SignalTelemetry: 0.35, SignalSensor: 0.35, SignalLocation: 0.20}},
			weightedModule{name: "decision_ai", weights: map[SignalType]float64{SignalVision: 0.20, SignalAudio: 0.10, SignalTelemetry: 0.25, SignalSensor: 0.20, SignalLocation: 0.15, SignalUser: 0.10}},
			weightedModule{name: "drone_ai", weights: map[SignalType]float64{SignalTelemetry: 0.35, SignalLocation: 0.30, SignalVision: 0.20, SignalSensor: 0.10}},
			weightedModule{name: "hud_ai", weights: map[SignalType]float64{SignalVision: 0.25, SignalAudio: 0.15, SignalUser: 0.25, SignalSensor: 0.10}},
			weightedModule{name: "learning_engine", weights: map[SignalType]float64{SignalUser: 0.30, SignalTelemetry: 0.20, SignalSensor: 0.20, SignalVision: 0.10}},
		},
		clock: time.Now,
	}
}

// Decide validates and processes real-time signals into an actionable command.
func (c *Cluster) Decide(ctx context.Context, signals []Signal, info ContextInformation) (Decision, error) {
	if len(signals) == 0 {
		return Decision{}, errors.New("at least one signal is required")
	}
	for _, signal := range signals {
		if err := signal.Validate(); err != nil {
			return Decision{}, err
		}
	}

	scores := make(map[string]float64, len(c.modules))
	var aggregate float64
	for _, module := range c.modules {
		score, err := module.Evaluate(ctx, signals, info)
		if err != nil {
			return Decision{}, fmt.Errorf("%s evaluation failed: %w", module.Name(), err)
		}
		scores[module.Name()] = score
		aggregate += score
	}
	aggregate = clamp01(aggregate / float64(len(c.modules)))

	return Decision{
		Command:       selectCommand(aggregate, info),
		Priority:      selectPriority(aggregate, info),
		Confidence:    aggregate,
		Explanation:   explainDecision(aggregate, info),
		ModuleScores:  scores,
		GeneratedAt:   c.clock().UTC(),
		RecommendedTo: []string{"drone_control", "smart_glasses_hud", "horizon_hub", "cloud_services"},
	}, nil
}

type weightedModule struct {
	name    string
	weights map[SignalType]float64
}

func (m weightedModule) Name() string { return m.name }

func (m weightedModule) Evaluate(ctx context.Context, signals []Signal, info ContextInformation) (float64, error) {
	select {
	case <-ctx.Done():
		return 0, ctx.Err()
	default:
	}

	var total, observed float64
	for _, signal := range signals {
		weight := m.weights[signal.Type]
		if weight == 0 {
			continue
		}
		total += weight
		observed += weight * signal.Confidence
	}
	if total == 0 {
		return 0, nil
	}

	contextBoost := (clamp01(info.EnvironmentRisk)*0.35 + (1-clamp01(info.BatteryLevel))*0.15 + clamp01(info.Connectivity)*0.10)
	if info.RequiresLowPower {
		contextBoost -= 0.10
	}
	return clamp01((observed / total) + contextBoost), nil
}

func selectCommand(confidence float64, info ContextInformation) string {
	if info.RequiresLowPower || info.BatteryLevel < 0.15 {
		return "optimize_low_power_operations"
	}
	if info.EnvironmentRisk >= 0.75 && confidence >= 0.65 {
		return "execute_autonomous_protective_action"
	}
	if confidence >= 0.80 {
		return "execute_best_action"
	}
	if confidence >= 0.55 {
		return "request_operator_confirmation"
	}
	return "collect_more_context"
}

func selectPriority(confidence float64, info ContextInformation) int {
	priority := int(confidence*6) + int(info.EnvironmentRisk*4)
	if info.RequiresLowPower {
		priority--
	}
	if priority < 1 {
		return 1
	}
	if priority > 10 {
		return 10
	}
	return priority
}

func explainDecision(confidence float64, info ContextInformation) string {
	factors := []string{"multi-modal perception", "sensor fusion", "mission context"}
	if info.RequiresLowPower {
		factors = append(factors, "low-power constraint")
	}
	if info.EnvironmentRisk >= 0.75 {
		factors = append(factors, "elevated environmental risk")
	}
	sort.Strings(factors)
	return fmt.Sprintf("confidence %.2f derived from %v", confidence, factors)
}
