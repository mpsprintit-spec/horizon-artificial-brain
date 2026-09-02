package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/project-horizon/horizon-core/internal/utils"
)

func TestRequestIDMiddleware(t *testing.T) {
	h := RequestID(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if utils.RequestIDFromContext(r.Context()) == "" {
			t.Fatal("missing request id")
		}
		w.WriteHeader(http.StatusNoContent)
	}))
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "/", nil))
	if rr.Header().Get("X-Request-ID") == "" {
		t.Fatal("missing response request id")
	}
}
func TestSecurityHeaders(t *testing.T) {
	h := SecurityHeaders(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusNoContent) }))
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "/", nil))
	if rr.Header().Get("X-Frame-Options") != "DENY" {
		t.Fatal("missing security header")
	}
}
