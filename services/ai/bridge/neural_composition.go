package bridge

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"time"

	"github.com/project-horizon/horizon-core/services/ai/knowledge"
	"github.com/project-horizon/horizon-core/services/ai/runtime"
)

// Training is a separately configured, authenticated teaching boundary. The
// ordinary observation endpoint cannot turn user words into supervised truth.
func (s *Server) compositionExposures(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	if !s.cfg.AllowCompositionTraining || s.cfg.Token == "" {
		writeError(w, http.StatusForbidden, "composition teaching is disabled or lacks configured authentication")
		return
	}
	defer r.Body.Close()
	r.Body = http.MaxBytesReader(w, r.Body, s.cfg.MaxPayloadBytes)
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	var exposure knowledge.CompositionExposure
	if err := decoder.Decode(&exposure); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	if err := decoder.Decode(new(any)); err != io.EOF {
		writeError(w, http.StatusBadRequest, "body must contain one composition exposure")
		return
	}
	var sequence uint64
	err := s.withMutatingRuntime(func(rt *runtime.BrainRuntime) error {
		var err error
		sequence, err = rt.LearnComposition(exposure, time.Time{})
		if err == nil && s.cfg.BrainMemoryPath != "" {
			err = rt.SaveBrain(s.cfg.BrainMemoryPath)
		}
		return err
	})
	if err != nil {
		status := http.StatusUnprocessableEntity
		if errors.Is(err, runtime.ErrRuntimeRecoveryRequired) || errors.Is(err, runtime.ErrRuntimeSupervisorClosed) {
			status = http.StatusServiceUnavailable
		}
		writeError(w, status, err.Error())
		return
	}
	writeJSON(w, http.StatusAccepted, map[string]any{"brain_identity": runtime.BrainIdentity, "state_revision": sequence, "event_id": exposure.ID})
}
