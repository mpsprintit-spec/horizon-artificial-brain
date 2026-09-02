package config

import (
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

const (
	defaultServiceName = "horizon-ai-cluster"
	defaultEnvironment = "development"
	defaultPort        = 8080
	defaultTimeout     = 1500 * time.Millisecond
)

// Config contains runtime settings for the Horizon AI Cluster service.
type Config struct {
	ServiceName     string
	Environment     string
	Port            int
	DecisionTimeout time.Duration
}

// Load reads configuration from environment variables and applies secure defaults.
func Load() (Config, error) {
	cfg := Config{
		ServiceName:     valueOrDefault("HORIZON_AI_SERVICE_NAME", defaultServiceName),
		Environment:     valueOrDefault("HORIZON_ENV", defaultEnvironment),
		Port:            defaultPort,
		DecisionTimeout: defaultTimeout,
	}

	if raw := strings.TrimSpace(os.Getenv("HORIZON_AI_PORT")); raw != "" {
		port, err := strconv.Atoi(raw)
		if err != nil || port < 1 || port > 65535 {
			return Config{}, fmt.Errorf("invalid HORIZON_AI_PORT: %q", raw)
		}
		cfg.Port = port
	}

	if raw := strings.TrimSpace(os.Getenv("HORIZON_AI_DECISION_TIMEOUT")); raw != "" {
		timeout, err := time.ParseDuration(raw)
		if err != nil || timeout <= 0 {
			return Config{}, fmt.Errorf("invalid HORIZON_AI_DECISION_TIMEOUT: %q", raw)
		}
		cfg.DecisionTimeout = timeout
	}

	if err := cfg.Validate(); err != nil {
		return Config{}, err
	}
	return cfg, nil
}

// Validate verifies that all configuration values are safe to use.
func (c Config) Validate() error {
	if strings.TrimSpace(c.ServiceName) == "" {
		return errors.New("service name is required")
	}
	if strings.TrimSpace(c.Environment) == "" {
		return errors.New("environment is required")
	}
	if c.Port < 1 || c.Port > 65535 {
		return fmt.Errorf("port must be between 1 and 65535: %d", c.Port)
	}
	if c.DecisionTimeout <= 0 {
		return errors.New("decision timeout must be positive")
	}
	return nil
}

func valueOrDefault(key, fallback string) string {
	if value := strings.TrimSpace(os.Getenv(key)); value != "" {
		return value
	}
	return fallback
}
