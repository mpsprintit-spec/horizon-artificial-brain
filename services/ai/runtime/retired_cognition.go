package runtime

import (
	"bytes"
	"errors"
)

// WAL5–13 belong to retired research controllers. This boundary recognizes
// their envelope without decoding, executing, or converting their commands.
func rejectRetiredJournal(event LoggedEvent) error {
	payload := bytes.TrimSpace(event.RetiredCognition)
	if event.SchemaVersion >= 5 && event.SchemaVersion <= 13 || event.Type == "cognitive_model_command" || len(payload) != 0 && !bytes.Equal(payload, []byte("null")) {
		return errors.New("retired cognitive V1–V5 journal: append/replay refused; retain the journal and inspect with Git revision 6b6db6c9ec0f373fed51f38a415d0a36baf7e812; no automatic native migration exists")
	}
	return nil
}
