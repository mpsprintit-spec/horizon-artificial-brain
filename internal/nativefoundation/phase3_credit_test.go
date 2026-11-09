package nativefoundation

import (
	k "github.com/project-horizon/horizon-core/services/ai/knowledge"
	r "github.com/project-horizon/horizon-core/services/ai/runtime"
	"path/filepath"
	"testing"
)

func TestPhase3CreditBothEncodersAndCheckpoint(t *testing.T) {
	for _, encoder := range []string{"identity", "relational"} {
		t.Run(encoder, func(t *testing.T) {
			dir := filepath.Join(t.TempDir(), "run")
			result, err := Phase3CreditExperiment(dir, 5003, "action", 2, 4, 16100000, true, encoder, 2)
			if err != nil {
				t.Fatal(err)
			}
			if result.ParameterChanges == 0 || result.Episodes != 8 {
				t.Fatal("missing canonical learning", result)
			}
			rt := r.NewBrainRuntime(k.NewBrain())
			if err = rt.RestoreCheckpoint(filepath.Join(dir, "trained-checkpoint.json")); err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestPhase3RecoveryAuditPreservesPendingWAL(t *testing.T) {
	if err := Phase3RecoveryAudit(filepath.Join(t.TempDir(), "audit")); err != nil {
		t.Fatal(err)
	}
}
