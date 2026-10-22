package bridge

import (
	"bufio"
	"encoding/json"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/project-horizon/horizon-core/services/ai/runtime"
)

func TestInspectionHTTPIsBoundedAndRevisionFenced(t *testing.T) {
	r := runtime.NewBrainRuntime(nil)
	server, _ := New(r, Config{Token: "test-token", RuntimeCommit: "inspection-test"})
	handler := server.Handler()
	read := func(path string, token bool) *httptest.ResponseRecorder {
		req := httptest.NewRequest("GET", path, nil)
		if token {
			req.Header.Set("Authorization", "Bearer test-token")
		}
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, req)
		return response
	}
	if response := read("/v1/brain/inspection", false); response.Code != 401 {
		t.Fatal("inspection bypassed auth")
	}
	response := read("/v1/brain/inspection?limit=1", true)
	if response.Code != 200 {
		t.Fatal(response.Code, response.Body.String())
	}
	if response.Body.Len() > 512*1024 {
		t.Fatal("inspection response exceeds limit")
	}
	var page runtime.MonitorInspectionPage
	if err := json.Unmarshal(response.Body.Bytes(), &page); err != nil {
		t.Fatal(err)
	}
	if page.RuntimeCommit != "inspection-test" || page.PageHash == "" || page.HashScope != "inspection_page" || strings.Contains(response.Body.String(), "canonical_state_hash") {
		t.Fatal("inspection claims a full-state digest")
	}
	if _, err := r.CognitiveThink(1); err != nil {
		t.Fatal(err)
	}
	if response := read("/v1/brain/inspection?limit=1&cursor="+url.QueryEscape(page.NextCursor), true); response.Code != 409 {
		t.Fatal("stale cursor status", response.Code)
	}
	for _, query := range []string{"limit=0", "limit=513", "limit=abc", "section=unknown", "cursor=broken", "order=invalid"} {
		if response := read("/v1/brain/inspection?"+query, true); response.Code != 400 {
			t.Fatal(query, response.Code)
		}
	}
}

func TestBoundedInspectionWebSocketHasExplicitPageIntegrity(t *testing.T) {
	r := runtime.NewBrainRuntime(nil)
	server, _ := New(r, Config{})
	ts := httptest.NewServer(server.Handler())
	defer ts.Close()
	address := strings.TrimPrefix(ts.URL, "http://")
	conn, err := net.Dial("tcp", address)
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	_ = conn.SetDeadline(time.Now().Add(5 * time.Second))
	_, err = conn.Write([]byte("GET /v1/brain/stream?telemetry=revisions&inspection=bounded HTTP/1.1\r\nHost: " + address + "\r\nConnection: Upgrade\r\nUpgrade: websocket\r\nSec-WebSocket-Version: 13\r\nSec-WebSocket-Key: dGhlIHNhbXBsZSBub25jZQ==\r\n\r\n"))
	if err != nil {
		t.Fatal(err)
	}
	reader := bufio.NewReader(conn)
	response, err := http.ReadResponse(reader, &http.Request{Method: "GET"})
	if err != nil || response.StatusCode != 101 {
		t.Fatal("upgrade", err)
	}
	_, wire, err := readWebSocketFrame(reader)
	if err != nil {
		t.Fatal(err)
	}
	var hello map[string]any
	_ = json.Unmarshal(wire, &hello)
	if hello["hash_scope"] != "inspection_page" || hello["inspection_epoch"] == "" {
		t.Fatal("bounded stream missing read scope")
	}
	if _, exists := hello["canonical_state_hash"]; exists {
		t.Fatal("bounded stream serialized a full Brain hash")
	}
	if _, err := r.CognitiveThink(1); err != nil {
		t.Fatal(err)
	}
	_, wire, err = readWebSocketFrame(reader)
	if err != nil {
		t.Fatal(err)
	}
	var event runtime.TelemetryEvent
	_ = json.Unmarshal(wire, &event)
	if event.SchemaVersion != 2 || event.CanonicalStateHash != "" {
		t.Fatal("bounded stream upgraded to full-hash events")
	}
}
