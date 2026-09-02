package domain

import "testing"

func TestContractNamesAreCanonical(t *testing.T) {
	tests := map[string]string{
		"vision module":        string(ModuleVision),
		"sensor fusion module": string(ModuleSensorFusion),
		"decision module":      string(ModuleDecision),
		"learning module":      string(ModuleLearning),
		"frame captured event": string(EventFrameCaptured),
		"decision event":       string(EventDecisionCreated),
		"learning event":       string(EventLearningUpdated),
	}

	for name, value := range tests {
		if value == "" {
			t.Fatalf("%s contract name must be canonical", name)
		}
	}
}
