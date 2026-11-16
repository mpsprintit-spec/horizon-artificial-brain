package main

import (
	"context"
	"io"
	"testing"
)

// Gate 4 integration boundary: the CLI must reach the same neural runtime
// path as the core entrypoint when legacy cognition is disabled. A Pulse now
// records both the process and the learning transition, so one interaction
// advances the runtime sequence twice.
func TestCLIProcessUsesNeuralRuntimeByDefault(t *testing.T) {
	app := newCLI(nil, io.Discard, "")
	result := app.process(context.Background(), "uji jalur neural dari cli")

	if result.Path != "neural_runtime" {
		t.Fatalf("CLI process path = %q, want neural_runtime", result.Path)
	}
	if runtimeForCLITest(t, app.horizon).LastSequence() != 1 {
		t.Fatalf("CLI process expected one canonical process transition: sequence=%d", runtimeForCLITest(t, app.horizon).LastSequence())
	}
}

func TestCLIProcessCannotFallBackToLegacyControlWhenDisabled(t *testing.T) {
	app := newCLI(nil, io.Discard, "")
	result := app.process(context.Background(), "iya")

	if result.Path != "neural_runtime" {
		t.Fatalf("CLI control-like input entered path %q while legacy cognition was disabled", result.Path)
	}
	if result.Confidence >= 1 {
		t.Fatalf("CLI neural path returned hard-coded certainty: %.3f", result.Confidence)
	}
}
