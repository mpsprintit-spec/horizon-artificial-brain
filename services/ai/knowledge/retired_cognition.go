package knowledge

import (
	"bytes"
	"encoding/json"
	"errors"
)

// Retired controller bytes are recognized only to reject them. There is no
// mapping from V1–V5 actor/optimizer state to native operation-policy state.
// Keep the input intact and use the pre-cleanup Git revision for inspection.
func rejectRetiredCognition(schema int, top, nested json.RawMessage) error {
	if schema >= 4 && schema <= 6 || nonNullRetiredPayload(top) || nonNullRetiredPayload(nested) {
		return errors.New("retired cognitive V1–V5 state: restore refused; inspect with Git revision 6b6db6c9ec0f373fed51f38a415d0a36baf7e812; no automatic native migration exists")
	}
	return nil
}

func nonNullRetiredPayload(raw json.RawMessage) bool {
	raw = bytes.TrimSpace(raw)
	return len(raw) != 0 && !bytes.Equal(raw, []byte("null"))
}

func rejectRetiredPopulations(populations []ProjectionPopulation) error {
	for _, population := range populations {
		if population.Scope != nil && population.Scope.Modality == "internal-numeric" && population.Scope.Field == "internal-shared-gru-muse-bmps-v1" {
			return errors.New("retired cognitive population: restore refused; retain the original file and inspect with the pre-cleanup Git revision")
		}
	}
	return nil
}
