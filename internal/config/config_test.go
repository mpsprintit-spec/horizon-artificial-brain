package config

import "testing"

func TestLoadDefaults(t *testing.T) {
	cfg := Load()
	if err := cfg.Validate(); err != nil {
		t.Fatal(err)
	}
	if cfg.App.Name == "" || cfg.HTTP.Address == "" || cfg.GRPC.Address == "" {
		t.Fatalf("expected defaults: %+v", cfg)
	}
}
func TestLoadEnvironmentOverride(t *testing.T) {
	t.Setenv("HORIZON_APP_NAME", "test-core")
	cfg := Load()
	if cfg.App.Name != "test-core" {
		t.Fatalf("expected env override, got %q", cfg.App.Name)
	}
}
