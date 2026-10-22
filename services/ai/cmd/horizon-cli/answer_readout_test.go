package main

import (
	"bytes"
	"context"
	"io"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/project-horizon/horizon-core/services/ai/learning"
	"github.com/project-horizon/horizon-core/services/ai/perception"
	"github.com/project-horizon/horizon-core/services/ai/runtime"
)

type readoutPerception struct{ at time.Time }

func (p readoutPerception) Perceive(text string) ([]perception.PerceptionSignal, error) {
	return []perception.PerceptionSignal{{Source: "readout-test", Modality: "text", Tokens: strings.Fields(text), ObservedAt: p.at}}, nil
}

func TestCLIAnswerDoesNotDecodeActivityLabels(t *testing.T) {
	for _, trained := range []bool{false, true} {
		t.Run(map[bool]string{false: "untrained", true: "trained-forecast"}[trained], func(t *testing.T) {
			app := newCLI(strings.NewReader(""), io.Discard, filepath.Join(t.TempDir(), "brain.json"))
			defer app.horizon.Supervisor().Close()
			at := time.Date(2026, 10, 21, 5, 0, 0, 0, time.UTC)
			app.horizon.Perception = readoutPerception{at: at.Add(time.Second)}
			if trained {
				if err := app.horizon.Supervisor().WithMutation(func(r *runtime.BrainRuntime) error {
					for i := 0; i < 4; i++ {
						if _, err := r.LearnExperience(learning.Experience{Sequence: []string{"naku", "melo", "tiva"}, Weight: .8, Confidence: .8}, at); err != nil {
							return err
						}
					}
					return nil
				}); err != nil {
					t.Fatal(err)
				}
			}
			result := app.process(context.Background(), "naku melo")
			if result.Answer != "Jawaban bahasa belum tersedia." || result.Understanding != "" || result.Reasoning != "" {
				t.Fatalf("activity was presented as language or understanding: %+v", result)
			}
			if len(result.Activation) == 0 {
				t.Fatal("diagnostic activity was suppressed")
			}
			if trained {
				if result.AnswerStatus != runtime.AnswerNeuralForecast || len(result.Forecast) != 1 || result.Confidence <= 0 {
					t.Fatalf("trained forecast lost from debug result: %+v", result)
				}
			} else if result.AnswerStatus != runtime.AnswerUninterpreted || len(result.Forecast) != 0 {
				t.Fatalf("untrained input invented a forecast: %+v", result)
			}
			var debug bytes.Buffer
			app.out = &debug
			app.printDebug(result)
			for _, marker := range []string{"[debug] Activation:", "[debug] Forecast:", "[debug] Answer status:"} {
				if !strings.Contains(debug.String(), marker) {
					t.Fatalf("missing separate diagnostic %q", marker)
				}
			}
		})
	}
}
