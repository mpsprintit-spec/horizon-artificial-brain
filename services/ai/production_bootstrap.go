package main

import (
    "errors"
    "fmt"
    "time"

    "github.com/project-horizon/horizon-core/services/ai/core"
    "github.com/project-horizon/horizon-core/services/ai/runtime"
)

// finishProductionStartup respects the recovery decision from
// initializeProductionHorizon. Existing committed neural state must never
// receive another event with the fixed bootstrap ID.
func finishProductionStartup(horizon *core.HorizonEngine, needsBootstrap bool, memoryPath, checkpointPath string) (runtime.CognitiveOutput, int, error) {
    var output runtime.CognitiveOutput
    if horizon == nil || horizon.Supervisor() == nil {
        return output, 0, errors.New("production Horizon supervisor is unavailable")
    }
    if !needsBootstrap {
        err := horizon.Supervisor().With(func(active *runtime.BrainRuntime) error {
            output = runtime.CognitiveOutput{
                BrainIdentity: runtime.BrainIdentity,
                Sequence: active.LastSequence(),
                Timestamp: time.Now().UTC(),
            }
            return nil
        })
        return output, 0, err
    }
    signals := bootstrapSignals()
    for _, signal := range signals {
        if err := signal.Validate(); err != nil {
            return output, 0, fmt.Errorf("invalid bootstrap signal: %w", err)
        }
    }
    stimulus := signalStimulus(signals)
    err := horizon.Supervisor().WithMutation(func(active *runtime.BrainRuntime) error {
        var processErr error
        output, processErr = active.CognitiveProcess(runtime.Event{
            ID: "bootstrap",
            Stimulus: stimulus,
            Cycles: 8,
            Timestamp: time.Now().UTC(),
        })
        if processErr != nil {
            return processErr
        }
        if err := active.SaveBrain(memoryPath); err != nil {
            return err
        }
        return active.Checkpoint(checkpointPath)
    })
    return output, len(stimulus), err
}
