package bridge

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestLegacyBridgeRefusesRetiredRuntimeReadEndpoints(t *testing.T) {
	supervisor, original, journal, checkpoint := managedBridgeFixture(t)
	defer supervisor.Close()
	legacy, err := New(original, Config{})
	if err != nil {
		t.Fatal(err)
	}
	managed, err := NewManaged(supervisor, Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := supervisor.RecoverFromCheckpoint(journal, checkpoint); err != nil {
		t.Fatal(err)
	}
	for _, path := range []string{"/health", "/v1/brain/handshake", "/v1/brain/snapshot", "/v1/brain/events"} {
		t.Run(path, func(t *testing.T) {
			for _, bridge := range []struct {
				server *Server
				status int
			}{{legacy, http.StatusServiceUnavailable}, {managed, http.StatusOK}} {
				response := httptest.NewRecorder()
				bridge.server.Handler().ServeHTTP(response, httptest.NewRequest(http.MethodGet, path, nil))
				if response.Code != bridge.status {
					t.Fatalf("status=%d want=%d: %.200s", response.Code, bridge.status, response.Body.String())
				}
			}
		})
	}
	request := httptest.NewRequest(http.MethodGet, "/v1/brain/stream", nil)
	request.Header.Set("Connection", "Upgrade")
	request.Header.Set("Upgrade", "websocket")
	request.Header.Set("Sec-WebSocket-Key", "dGhlIHNhbXBsZSBub25jZQ==")
	response := httptest.NewRecorder()
	legacy.Handler().ServeHTTP(response, request)
	if response.Code != http.StatusServiceUnavailable {
		t.Fatalf("retired WebSocket status=%d: %.200s", response.Code, response.Body.String())
	}
}
