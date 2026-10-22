package bridge

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/project-horizon/horizon-core/services/ai/knowledge"
	"github.com/project-horizon/horizon-core/services/ai/runtime"
)

func (s *Server) inspection(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, 405, "method not allowed")
		return
	}
	query := r.URL.Query()
	limit := 128
	if text := query.Get("limit"); text != "" {
		var err error
		limit, err = strconv.Atoi(text)
		if err != nil {
			writeError(w, 400, "invalid inspection limit")
			return
		}
	}
	if limit < 1 || limit > knowledge.MaxInspectionLimit {
		writeError(w, 400, "inspection limit is out of range")
		return
	}
	request := runtime.InspectionPageRequest{Section: query.Get("section"), Order: query.Get("order"), Limit: limit, Cursor: query.Get("cursor")}
	var page runtime.MonitorInspectionPage
	err := s.withRuntime(func(rt *runtime.BrainRuntime) error {
		var err error
		page, err = rt.MonitorInspectionPage(request, s.cfg.RuntimeCommit)
		return err
	})
	if err != nil {
		status := 400
		if errors.Is(err, runtime.ErrInspectionChanged) {
			status = 409
		} else if errors.Is(err, knowledge.ErrInspectionTooLarge) {
			status = 413
		} else if errors.Is(err, runtime.ErrRuntimeRecoveryRequired) {
			status = 503
		}
		writeError(w, status, err.Error())
		return
	}
	w.Header().Set("Cache-Control", "no-store")
	writeJSON(w, 200, page)
}
