package main

import (
	"context"
	"io"
	"testing"

	"github.com/project-horizon/horizon-core/services/ai/core"
)

// Gate 4 integration boundary: the CLI must reach the same neural runtime
// path as the core entrypoint when legacy cognition is disabled. A Pulse now
// records both the process and the learning transition, so one interaction
// advances the runtime sequence twice.
func TestCLIProcessUsesNeuralRuntimeByDefault(t *testing.T) {
	old := core.LegacyCognitionEnabled
	core.LegacyCognitionEnabled = false
	defer func() { core.LegacyCognitionEnabled = old }()

	app := newCLI(nil, io.Discard, "")
	result := app.process(context.Background(), "uji jalur neural dari cli")

	if result.Path != "neural_runtime" {
		t.Fatalf("CLI process path = %q, want neural_runtime", result.Path)
	}
	if app.horizon.Runtime.LastSequence() != 1 {
		t.Fatalf("CLI process expected one canonical process transition: sequence=%d", app.horizon.Runtime.LastSequence())
	}
}

func TestCLIProcessCannotFallBackToLegacyControlWhenDisabled(t *testing.T) {
	old := core.LegacyCognitionEnabled
	core.LegacyCognitionEnabled = false
	defer func() { core.LegacyCognitionEnabled = old }()

	app := newCLI(nil, io.Discard, "")
	result := app.process(context.Background(), "iya")

	if result.Path != "neural_runtime" {
		t.Fatalf("CLI control-like input entered path %q while legacy cognition was disabled", result.Path)
	}
	if result.Confidence >= 1 {
		t.Fatalf("CLI neural path returned hard-coded certainty: %.3f", result.Confidence)
	}
}
