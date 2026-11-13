package runtime

import (
	"errors"
	"fmt"
	"github.com/project-horizon/horizon-core/services/ai/knowledge"
)

var ErrCommandAdmission = errors.New("command conflicts with canonical native transaction")

func (r *BrainRuntime) transactionFaultLocked(boundary string) {
	if r.transactionFault != nil {
		r.transactionFault(boundary)
	}
}

// admitCommandLocked guards the first mutation AND the first durable append.
// Read-only inspection and a lock-consistent checkpoint do not enter this
// boundary. Native commands retain their finer trajectory/provenance preflight.
// This is an ownership rule for transactions, not a cognitive action selector.
func (r *BrainRuntime) admitCommandLocked(kind string) error {
	if err := r.ensureWritableLocked(); err != nil {
		return err
	}
	state := r.brain.NativeAdmission()
	if state == knowledge.NativeUninitialized || kind == EventTypeNativeControl {
		return nil
	}
	if state != knowledge.NativeIdle {
		return fmt.Errorf("%w: %s rejected during %s; settle the native outcome or pending contrast first", ErrCommandAdmission, kind, state)
	}
	switch kind {
	case EventTypeProcess, EventTypeCognitiveProcess, EventTypeObservation,
		EventTypeThink, EventTypeCognitiveThink, EventTypeSensorIntent,
		EventTypeLearn, EventTypeSpellingExposure, EventTypeCompositionExposure,
		EventTypeCompositionPerception, EventTypeVectorGround, EventTypeKnowledgeImport,
		EventTypeInquiryOutcome, EventTypeOutcome, EventTypeConsequence, EventTypeActionBinding:
		return nil
	default:
		// Controller replacement and serving migrations are not ordinary
		// observations. Admit them only after an explicit compatibility proof.
		return fmt.Errorf("%w: %s has no native coexistence contract", ErrCommandAdmission, kind)
	}
}
