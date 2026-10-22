package bridge

import (
	"bufio"
	"encoding/json"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/project-horizon/horizon-core/services/ai/runtime"
)

func TestBridgeNegotiatesRevisionStreamAndPreservesDefaultHashes(t *testing.T) {
	for _, mode := range []string{"", "canonical", "revisions"} {
		t.Run("mode="+mode, func(t *testing.T) {
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
			_, err = conn.Write([]byte("GET /v1/brain/stream?telemetry=" + mode + " HTTP/1.1\r\nHost: " + address + "\r\nConnection: Upgrade\r\nUpgrade: websocket\r\nSec-WebSocket-Version: 13\r\nSec-WebSocket-Key: dGhlIHNhbXBsZSBub25jZQ==\r\n\r\n"))
			if err != nil {
				t.Fatal(err)
			}
			reader := bufio.NewReader(conn)
			response, err := http.ReadResponse(reader, &http.Request{Method: http.MethodGet})
			if err != nil {
				t.Fatal(err)
			}
			if response.StatusCode != 101 {
				t.Fatalf("upgrade: %d", response.StatusCode)
			}
			_, frame, err := readWebSocketFrame(reader)
			if err != nil {
				t.Fatal(err)
			}
			var handshake runtime.MonitorHandshake
			if err := json.Unmarshal(frame, &handshake); err != nil {
				t.Fatal(err)
			}
			if handshake.TelemetryMode != mode || handshake.CanonicalStateHash == "" {
				t.Fatal("missing negotiated mode or snapshot integrity")
			}
			if _, err := r.CognitiveProcess(runtime.Event{ID: "live", Stimulus: []string{"naku"}, Cycles: 1}); err != nil {
				t.Fatal(err)
			}
			_, frame, err = readWebSocketFrame(reader)
			if err != nil {
				t.Fatal(err)
			}
			var event runtime.TelemetryEvent
			if err := json.Unmarshal(frame, &event); err != nil {
				t.Fatal(err)
			}
			if event.StateRevision != 1 || event.EventHash == "" || event.StateDelta == nil {
				t.Fatal("missing live revision/event/delta")
			}
			snapshot, err := r.MonitorSnapshot()
			if err != nil {
				t.Fatal(err)
			}
			if mode == "revisions" {
				if event.SchemaVersion != 2 || event.CanonicalStateHash != "" {
					t.Fatal("revision stream sent legacy hash")
				}
			} else if event.CanonicalStateHash != snapshot.CanonicalStateHash {
				t.Fatal("legacy hash differs from snapshot")
			}
		})
	}
}

func TestBridgeRejectsUnknownTelemetryMode(t *testing.T) {
	server, _ := New(runtime.NewBrainRuntime(nil), Config{})
	response := httptest.NewRecorder()
	server.Handler().ServeHTTP(response, httptest.NewRequest("GET", "/v1/brain/stream?telemetry=invalid", nil))
	if response.Code != http.StatusBadRequest {
		t.Fatalf("status %d", response.Code)
	}
}
