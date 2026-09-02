package config

import (
	"testing"
	"time"
)

func TestLoadAppliesDefaults(t *testing.T) {
	t.Setenv("HORIZON_AI_SERVICE_NAME", "")
	t.Setenv("HORIZON_ENV", "")
	t.Setenv("HORIZON_AI_PORT", "")
	t.Setenv("HORIZON_AI_DECISION_TIMEOUT", "")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}
	if cfg.ServiceName != defaultServiceName || cfg.Environment != defaultEnvironment || cfg.Port != defaultPort || cfg.DecisionTimeout != defaultTimeout {
		t.Fatalf("unexpected defaults: %+v", cfg)
	}
}

func TestLoadRejectsInvalidPort(t *testing.T) {
	t.Setenv("HORIZON_AI_PORT", "70000")
	if _, err := Load(); err == nil {
		t.Fatal("expected invalid port error")
	}
}

func TestLoadAcceptsTimeout(t *testing.T) {
	t.Setenv("HORIZON_AI_DECISION_TIMEOUT", "2s")
	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}
	if cfg.DecisionTimeout != 2*time.Second {
		t.Fatalf("unexpected timeout: %s", cfg.DecisionTimeout)
	}
}
