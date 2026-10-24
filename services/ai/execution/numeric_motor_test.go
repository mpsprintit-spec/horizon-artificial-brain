package execution

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/project-horizon/horizon-core/services/ai/runtime"
)

type uncertainMotor struct{ calls int }

func (m *uncertainMotor) ExecuteMotor(ctx context.Context, id string, values []float64) error {
	m.calls++
	if _, ok := ctx.Deadline(); !ok || id == "" || len(values) != 4 {
		panic("missing bounded numeric actuator contract")
	}
	return errors.New("physical action may have completed; acknowledgment lost")
}

func TestNumericMotorGatewayDoesNotRepeatUncertainPhysicalAction(t *testing.T) {
	g := NewExecutionGateway(nil)
	motor := &uncertainMotor{}
	if err := g.RegisterMotorActuator("contact", motor); err != nil {
		t.Fatal(err)
	}
	safety := runtime.SafetyBoundary{}
	recommendation := runtime.Recommendation{RequestID: "motor-once", BrainIdentity: runtime.BrainIdentity, Intent: "numeric-motor", RiskLevel: runtime.RiskMedium, Motor: &runtime.NumericMotorCommand{ActuatorID: "contact", Values: []float64{0, 0, 1, 1}}}
	assessment, err := safety.Assess(recommendation)
	if err != nil {
		t.Fatal(err)
	}
	authorization, err := safety.Authorize(recommendation, assessment, "test-operator")
	if err != nil {
		t.Fatal(err)
	}
	request, err := safety.BuildExecutionRequest(recommendation, assessment, authorization, time.Now().Add(time.Second))
	if err != nil {
		t.Fatal(err)
	}
	if err := g.Execute(request); err == nil {
		t.Fatal("uncertain device response was hidden")
	}
	if err := g.Execute(request); err == nil || motor.calls != 1 {
		t.Fatal("uncertain physical action repeated")
	}
}
