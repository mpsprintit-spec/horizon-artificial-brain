package bridge

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/project-horizon/horizon-core/services/ai/runtime"
)

func TestBridgeReadEndpointsAreLive(t *testing.T) {
	server, err := New(runtime.NewBrainRuntime(nil), Config{RuntimeCommit: "integration"})
	if err != nil { t.Fatal(err) }
	for _, path := range []string{"/health", "/v1/brain/handshake", "/v1/brain/snapshot", "/v1/brain/events?after=0"} {
		req := httptest.NewRequest(http.MethodGet, path, nil)
		res := httptest.NewRecorder()
		server.Handler().ServeHTTP(res, req)
		if res.Code != http.StatusOK { t.Fatalf("%s status=%d body=%s", path, res.Code, res.Body.String()) }
		if !strings.Contains(res.Header().Get("Content-Type"), "application/json") { t.Fatalf("%s did not return JSON", path) }
	}
}

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

func TestBridgeObservationPersistsAndReplaysFromRuntime(t *testing.T) {
	dir := t.TempDir()
	memoryPath := filepath.Join(dir, "brain_memory.json")
	logPath := filepath.Join(dir, "events.jsonl")
	rt := runtime.NewBrainRuntime(nil)
	log, err := runtime.OpenEventLog(logPath)
	if err != nil { t.Fatal(err) }
	defer log.Close()
	rt.SetEventLog(log)
	server, err := New(rt, Config{EventLogPath: logPath, BrainMemoryPath: memoryPath, RuntimeCommit: "integration"})
	if err != nil { t.Fatal(err) }

	before, err := rt.MonitorSnapshot()
	if err != nil { t.Fatal(err) }
	if before.Counts.NeuralUnits == 0 { t.Fatal("runtime snapshot has no neural units") }
	if len(before.NeuralUnits) != before.Counts.NeuralUnits { t.Fatalf("neural unit count mismatch: %d != %d", len(before.NeuralUnits), before.Counts.NeuralUnits) }
	if len(before.Synapses) != before.Counts.Synapses { t.Fatalf("synapse count mismatch: %d != %d", len(before.Synapses), before.Counts.Synapses) }

	body := strings.NewReader(`{"schema_version":1,"brain_identity":"horizon-primary-brain","event_id":"browser-vision-1","timestamp":"2026-10-07T14:00:00Z","modality":"vision","source":"github-pages-camera","provenance":{"source":"github-pages-camera","modality":"vision","adapter":"github-pages-browser","synthetic":false},"payload":{"encoding":"application/octet-stream","data":"AQID"}}`)
	req := httptest.NewRequest(http.MethodPost, "/v1/observations", body)
	req.Header.Set("Content-Type", "application/json")
	res := httptest.NewRecorder()
	server.Handler().ServeHTTP(res, req)
	if res.Code != http.StatusAccepted { t.Fatalf("observation status=%d body=%s", res.Code, res.Body.String()) }
	if _, err := os.Stat(memoryPath); err != nil { t.Fatalf("brain memory missing: %v", err) }
	memory, err := os.ReadFile(memoryPath)
	if err != nil { t.Fatal(err) }
	for _, marker := range []string{`"neural_units"`, `"synapses"`, `"populations"`} {
		if !strings.Contains(string(memory), marker) { t.Fatalf("brain memory missing canonical field %s", marker) }
	}
	after, err := rt.MonitorSnapshot()
	if err != nil { t.Fatal(err) }
	if after.StateRevision <= before.StateRevision { t.Fatalf("observation did not advance canonical revision: before=%d after=%d", before.StateRevision, after.StateRevision) }
	if after.CanonicalStateHash == before.CanonicalStateHash { t.Fatal("observation did not change canonical brain hash") }
	if len(after.NeuralUnits) != after.Counts.NeuralUnits || len(after.Synapses) != after.Counts.Synapses { t.Fatal("snapshot counts do not describe actual canonical graph") }
	if after.Counts.NeuralUnits == 0 { t.Fatal("canonical graph has no neural units after observation") }

	logged, err := runtime.ReadEventLog(logPath)
	if err != nil { t.Fatal(err) }
	if len(logged) == 0 || logged[0].Event == nil || logged[0].Event.ID != "browser-vision-1" { t.Fatalf("observation not present in event log: %+v", logged) }

	req = httptest.NewRequest(http.MethodGet, "/v1/brain/events?after=0", nil)
	res = httptest.NewRecorder()
	server.Handler().ServeHTTP(res, req)
	if res.Code != http.StatusOK { t.Fatalf("events status=%d body=%s", res.Code, res.Body.String()) }
	var payload struct{ Events []map[string]any `json:"events"` }
	if err := json.Unmarshal(res.Body.Bytes(), &payload); err != nil { t.Fatal(err) }
	if len(payload.Events) == 0 { t.Fatal("bridge replay returned no events") }
}
