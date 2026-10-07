package bridge

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/project-horizon/horizon-core/services/ai/runtime"
)

func TestSnapshotBridgeServesCanonicalRuntime(t *testing.T) {
	runtime := runtime.NewBrainRuntime(nil)
	server, err := New(runtime, Config{AllowedOrigins: []string{"https://example.github.io"}, RuntimeCommit: "test"})
	if err != nil {
		t.Fatal(err)
	}
	req := httptest.NewRequest(http.MethodGet, "/v1/brain/handshake", nil)
	req.Header.Set("Origin", "https://example.github.io")
	res := httptest.NewRecorder()
	server.Handler().ServeHTTP(res, req)
	if res.Code != http.StatusOK {
		t.Fatalf("handshake status=%d body=%s", res.Code, res.Body.String())
	}
	if got := res.Header().Get("Access-Control-Allow-Origin"); got != "https://example.github.io" {
		t.Fatalf("unexpected CORS origin: %q", got)
	}
}

func TestSnapshotBridgeRejectsUnknownOrigin(t *testing.T) {
	server, _ := New(runtime.NewBrainRuntime(nil), Config{AllowedOrigins: []string{"https://example.github.io"}})
	req := httptest.NewRequest(http.MethodGet, "/v1/brain/snapshot", nil)
	req.Header.Set("Origin", "https://attacker.example")
	res := httptest.NewRecorder()
	server.Handler().ServeHTTP(res, req)
	if res.Code != http.StatusForbidden {
		t.Fatalf("status=%d body=%s", res.Code, res.Body.String())
	}
}

func TestSnapshotBridgeRequiresTokenWhenConfigured(t *testing.T) {
	server, _ := New(runtime.NewBrainRuntime(nil), Config{Token: "secret"})
	req := httptest.NewRequest(http.MethodGet, "/v1/brain/snapshot", nil)
	res := httptest.NewRecorder()
	server.Handler().ServeHTTP(res, req)
	if res.Code != http.StatusUnauthorized {
		t.Fatalf("status=%d body=%s", res.Code, res.Body.String())
	}
}
