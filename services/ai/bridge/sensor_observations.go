package bridge

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"

	"github.com/project-horizon/horizon-core/services/ai/runtime"
)

// sensorObservations carries numerical sensor state without token/hash
// fallback. Transport and disk persistence use the same supervisor mutation
// lease as the complete multi-field journal transaction.
func (s *Server) sensorObservations(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	defer r.Body.Close()
	r.Body = http.MaxBytesReader(w, r.Body, s.cfg.MaxPayloadBytes)
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	var sensor runtime.SensorObservation
	if err := decoder.Decode(&sensor); err != nil {
		writeError(w, http.StatusBadRequest, "invalid sensor observation: "+err.Error())
		return
	}
	if err := decoder.Decode(new(any)); err != io.EOF {
		writeError(w, http.StatusBadRequest, "sensor body must contain one observation")
		return
	}
	var result runtime.CognitiveInterpretation
	var persistenceErr error
	err := s.withMutatingRuntime(func(rt *runtime.BrainRuntime) error {
		var err error
		result, err = rt.ProcessSensorObservation(sensor)
		if err != nil {
			return err
		}
		if s.cfg.BrainMemoryPath != "" {
			persistenceErr = rt.SaveBrain(s.cfg.BrainMemoryPath)
		}
		return persistenceErr
	})
	if err != nil {
		status := http.StatusUnprocessableEntity
		if errors.Is(err, runtime.ErrRuntimeRecoveryRequired) || errors.Is(err, runtime.ErrRuntimeSupervisorClosed) {
			status = http.StatusServiceUnavailable
		}
		if persistenceErr != nil {
			status = http.StatusInternalServerError
		}
		writeError(w, status, err.Error())
		return
	}
	writeJSON(w, http.StatusAccepted, map[string]any{
		"brain_identity": runtime.BrainIdentity, "event_id": sensor.SampleID,
		"state_revision": result.State.Sequence, "prediction_error": result.State.PredictionError,
		"active_node_ids": result.State.ActiveNodeIDs, "sensor_bindings": result.SensorBindings,
		"answer": readoutForAnswer(result.Answer),
	})
}
