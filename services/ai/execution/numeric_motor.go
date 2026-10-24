package execution

import (
	"context"
	"errors"

	"github.com/project-horizon/horizon-core/services/ai/runtime"
)

// NumericMotorActuator is a physical I/O adapter. It must honor deadline and
// requestID; it performs no lexical or semantic interpretation.
type NumericMotorActuator interface {
	ExecuteMotor(context.Context, string, []float64) error
}

func (g *ExecutionGateway) RegisterMotorActuator(id string, actuator NumericMotorActuator) error {
	if g == nil || id == "" || actuator == nil {
		return errors.New("numeric actuator ID and adapter are required")
	}
	g.mu.Lock()
	defer g.mu.Unlock()
	if g.motorActuators == nil {
		g.motorActuators = map[string]NumericMotorActuator{}
	}
	if _, exists := g.motorActuators[id]; exists {
		return errors.New("numeric actuator is already registered")
	}
	g.motorActuators[id] = actuator
	return nil
}

func (g *ExecutionGateway) executeNumericMotorLocked(request runtime.ExecutionRequest) error {
	if err := request.Motor.Validate(); err != nil {
		return err
	}
	actuator := g.motorActuators[request.Motor.ActuatorID]
	if actuator == nil {
		return errors.New("numeric actuator is not registered")
	}
	ctx, cancel := context.WithDeadline(context.Background(), request.Expiry)
	defer cancel()
	// Reserve before dispatch. An error may mean the physical action happened
	// but acknowledgment was lost. Never automatically repeat that request ID.
	g.executedIDs[request.IdempotencyKey] = struct{}{}
	values := append([]float64(nil), request.Motor.Values...)
	return actuator.ExecuteMotor(ctx, request.RequestID, values)
}
