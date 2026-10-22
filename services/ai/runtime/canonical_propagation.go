package runtime

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"os"

	"github.com/project-horizon/horizon-core/services/ai/knowledge"
)

// NewCanonicalPropagationRuntime starts an empty canonical Brain with the
// explicit canonical-propagation-v1 law. It cannot adopt an existing Brain.
// NewBrainRuntime and historical journal/checkpoint readers keep their defaults.
func NewCanonicalPropagationRuntime() *BrainRuntime {
	r := NewBrainRuntime(knowledge.NewBrain())
	r.dynamicsEncoding = knowledge.CanonicalPropagationDynamics
	return r
}

// Genesis recovery must reproduce the empty foundation of the opt-in writer;
// changing an existing foundation would be an unrecorded dynamics migration.
func validateCanonicalPropagationFoundation(brain *knowledge.Brain) error {
	actual, err := brain.CanonicalJSON()
	if err != nil {
		return err
	}
	empty, err := knowledge.NewBrain().CanonicalJSON()
	if err != nil {
		return err
	}
	if !bytes.Equal(actual, empty) {
		return errors.New("canonical propagation genesis requires the original empty Brain foundation")
	}
	return nil
}

// Historical journal law transitions stay supported. Crossing into or out of
// canonical propagation requires a separately designed migration, not replay.
func validateCanonicalPropagationTimeline(events []LoggedEvent, canonical bool) error {
	for _, event := range events {
		if event.DynamicsEncoding.HasCanonicalPropagation() != canonical {
			return errors.New("canonical propagation journal requires a matching runtime/checkpoint law; migration is not supported")
		}
	}
	return nil
}

func validateCheckpointPropagation(snapshot BrainSnapshot) error {
	if snapshot.SchemaVersion == CanonicalPropagationSnapshotSchemaVersion {
		if snapshot.DynamicsEncoding != knowledge.CanonicalPropagationDynamics {
			return errors.New("checkpoint schema 14 requires canonical-propagation-v1")
		}
	} else if snapshot.DynamicsEncoding != knowledge.LegacyDynamics {
		return errors.New("historical checkpoint cannot carry an unauthenticated dynamics encoding")
	}
	return nil
}

// Only the first bounded record is needed: this distinguishes the new law
// without changing historical append/repair semantics. Full replay validates
// every record and refuses any mixed timeline before its first mutation.
func journalPropagationLaw(path string) (canonical bool, hasRecord bool, err error) {
	f, err := os.Open(path)
	if err != nil {
		return false, false, err
	}
	defer f.Close()
	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 64*1024), 16<<20)
	if !scanner.Scan() {
		return false, false, scanner.Err()
	}
	var header struct {
		DynamicsEncoding knowledge.DynamicsEncoding `json:"dynamics_encoding"`
	}
	if err := json.Unmarshal(scanner.Bytes(), &header); err != nil {
		return false, false, err
	}
	if err := header.DynamicsEncoding.Validate(); err != nil {
		return false, false, err
	}
	return header.DynamicsEncoding.HasCanonicalPropagation(), true, nil
}
