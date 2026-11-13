package runtime

import (
	"errors"
	"fmt"
	"github.com/project-horizon/horizon-core/services/ai/knowledge"
	"github.com/project-horizon/horizon-core/services/ai/learning"
)

var ErrCommandAdmission = errors.New("command conflicts with canonical native transaction")

func (r *BrainRuntime) transactionFaultLocked(boundary string) {
	if r.transactionFault != nil {
		r.transactionFault(boundary)
	}
}

// Production adapters may carry explicit neural addresses. A syntactically
// valid address is not permission to teach through an internal policy cell.
// This pre-append fence also covers inline experience inside observations.
func (r *BrainRuntime) admitJournalReferencesLocked(c LoggedEvent) error {
	if c.Type == EventTypeNativeControl {
		return nil
	}
	var groups [][]knowledge.NodeID
	addExperience := func(e *learning.Experience) {
		if e != nil {
			groups = append(groups, e.NeuralSequence...)
			groups = append(groups, e.NeuralContext...)
			groups = append(groups, e.NeuralData...)
		}
	}
	addExperience(c.Experience)
	if c.ObservationCommand != nil {
		addExperience(c.ObservationCommand.Experience)
	}
	if c.InquiryOutcome != nil {
		addExperience(c.InquiryOutcome.Observation.Experience)
	}
	if c.SpellingExposure != nil {
		groups = append(groups, c.SpellingExposure.Population)
	}
	if c.Consequence != nil {
		groups = append(groups, c.Consequence.TargetNodeIDs, c.Consequence.OutcomeNodeIDs)
	}
	return r.admitWorldReferencesLocked(groups...)
}

func (r *BrainRuntime) admitWorldReferencesLocked(groups ...[]knowledge.NodeID) error {
	if len(groups) == 0 {
		return nil
	}
	r.brain.RLock()
	defer r.brain.RUnlock()
	reserved := r.brain.NativeOwnershipLocked()
	if !reserved.Enabled() {
		return nil
	}
	for _, ids := range groups {
		for _, id := range ids {
			if reserved.Has(id) {
				return fmt.Errorf("%w: production learning references native policy unit %d", ErrCommandAdmission, id)
			}
		}
	}
	return nil
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
