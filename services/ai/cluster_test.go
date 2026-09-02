package main

import (
	"context"
	"testing"
	"time"
)

func TestClusterDecideProducesActionableDecision(t *testing.T) {
	cluster := NewCluster()
	cluster.clock = func() time.Time { return time.Date(2026, 7, 14, 0, 0, 0, 0, time.UTC) }

	decision, err := cluster.Decide(context.Background(), []Signal{
		{Type: SignalVision, Source: "camera", Confidence: 0.92, Timestamp: time.Now()},
		{Type: SignalTelemetry, Source: "drone", Confidence: 0.89, Timestamp: time.Now()},
		{Type: SignalSensor, Source: "imu", Confidence: 0.87, Timestamp: time.Now()},
		{Type: SignalLocation, Source: "gps", Confidence: 0.84, Timestamp: time.Now()},
		{Type: SignalUser, Source: "profile", Confidence: 0.76, Timestamp: time.Now()},
	}, ContextInformation{EnvironmentRisk: 0.80, BatteryLevel: 0.70, Connectivity: 0.95})
	if err != nil {
		t.Fatalf("Decide returned error: %v", err)
	}
	if decision.Command != "execute_autonomous_protective_action" {
		t.Fatalf("unexpected command: %s", decision.Command)
	}
	if decision.Confidence <= 0 || decision.Confidence > 1 {
		t.Fatalf("confidence out of range: %f", decision.Confidence)
	}
	if len(decision.ModuleScores) != 6 {
		t.Fatalf("expected scores for 6 modules, got %d", len(decision.ModuleScores))
	}
}

func TestClusterRejectsInvalidSignal(t *testing.T) {
	_, err := NewCluster().Decide(context.Background(), []Signal{{Type: SignalVision, Confidence: 1, Timestamp: time.Now()}}, ContextInformation{})
	if err == nil {
		t.Fatal("expected invalid signal error")
	}
}

func TestClusterRespectsContextCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := NewCluster().Decide(ctx, []Signal{{Type: SignalVision, Source: "camera", Confidence: 1, Timestamp: time.Now()}}, ContextInformation{})
	if err == nil {
		t.Fatal("expected context cancellation error")
	}
}
