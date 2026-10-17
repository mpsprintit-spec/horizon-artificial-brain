package bridge

import (
	"bufio"
	"errors"
	"net"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/project-horizon/horizon-core/services/ai/runtime"
)

func managedBridgeFixture(t *testing.T) (*runtime.RuntimeSupervisor, *runtime.BrainRuntime, *runtime.EventLog, string) {
	t.Helper()
	log, err := runtime.OpenEventLog(filepath.Join(t.TempDir(), "events.jsonl"))
	if err != nil { t.Fatal(err) }
	t.Cleanup(func(){ _ = log.Close() })
	original := runtime.NewBrainRuntime(nil)
	original.SetEventLog(log)
	at := time.Date(2026, 10, 17, 12, 0, 0, 0, time.UTC)
	if _, _, err := original.Process(runtime.Event{ID:"bridge-initial", Stimulus:[]string{"saya"}, Cycles:1, Timestamp:at}); err != nil { t.Fatal(err) }
	checkpointPath := filepath.Join(t.TempDir(), "checkpoint.json")
	if err := original.Checkpoint(checkpointPath); err != nil { t.Fatal(err) }
	supervisor, err := runtime.NewRuntimeSupervisor(original)
	if err != nil { t.Fatal(err) }
	return supervisor, original, log, checkpointPath
}

func TestManagedBridgeUsesNewRuntimeAfterCheckpointHandover(t *testing.T) {
	supervisor, original, log, checkpointPath := managedBridgeFixture(t)
	server, err := NewManaged(supervisor, Config{})
	if err != nil { t.Fatal(err) }
	handler := server.Handler()
	checkStatus := func(method, path, payload string, expected int) {
		t.Helper()
		resp := httptest.NewRecorder()
		handler.ServeHTTP(resp, httptest.NewRequest(method, path, strings.NewReader(payload)))
		if resp.Code != expected { t.Fatalf("%s %s got %d expected %d, body=%s", method,path,resp.Code,expected,resp.Body.String()) }
	}
	checkStatus(http.MethodGet,"/health","",http.StatusOK)
	checkStatus(http.MethodGet,"/v1/brain/handshake","",http.StatusOK)
	checkStatus(http.MethodGet,"/v1/brain/snapshot","",http.StatusOK)
	checkStatus(http.MethodGet,"/v1/brain/events","",http.StatusOK)

	if _, _, err := original.Process(runtime.Event{ID:"bridge-before-swap",Cycles:1,Timestamp:time.Date(2026,10,17,12,0,1,0,time.UTC)}); err != nil { t.Fatal(err) }
	if err := supervisor.RecoverFromCheckpoint(log, checkpointPath); err != nil { t.Fatalf("handover: %v", err) }
	if !original.RecoveryRequired() { t.Fatal("old bridge runtime remained writable") }
	checkStatus(http.MethodGet,"/health","",http.StatusOK)
	checkStatus(http.MethodGet,"/v1/brain/snapshot","",http.StatusOK)
	checkStatus(http.MethodGet,"/v1/brain/events?after=1","",http.StatusOK)

	payload := `{"schema_version":1,"brain_identity":"horizon-primary-brain","event_id":"managed-observation-1","timestamp":"2026-10-17T12:00:02Z","modality":"text","source":"bridge-managed-test","tokens":["saya"]}`
	checkStatus(http.MethodPost,"/v1/observations",payload,http.StatusAccepted)
	if original.LastSequence()!=2 { t.Fatalf("bridge mutated retired runtime, seq=%d",original.LastSequence()) }
	if err := supervisor.With(func(active *runtime.BrainRuntime) error {
		if active == original { t.Fatal("bridge retained old runtime") }
		if active.LastSequence()<=2 { t.Fatalf("observation not sent to new runtime: seq=%d",active.LastSequence()) }
		return nil
	}); err != nil { t.Fatal(err) }

	if err := supervisor.Close(); err != nil { t.Fatal(err) }
	checkStatus(http.MethodGet,"/health","",http.StatusServiceUnavailable)
	checkStatus(http.MethodGet,"/v1/brain/snapshot","",http.StatusServiceUnavailable)
	checkStatus(http.MethodGet,"/v1/brain/events","",http.StatusServiceUnavailable)
	checkStatus(http.MethodPost,"/v1/observations",payload,http.StatusServiceUnavailable)
	checkStatus(http.MethodPost,"/v1/outcomes",`{"RequestID":"closed"}`,http.StatusServiceUnavailable)
}

func TestManagedBridgeWebSocketTerminatesWhenRuntimeChanges(t *testing.T) {
	supervisor, _, log, checkpointPath := managedBridgeFixture(t)
	server, err := NewManaged(supervisor, Config{})
	if err != nil { t.Fatal(err) }
	ts := httptest.NewServer(server.Handler())
	defer ts.Close()
	address := strings.TrimPrefix(ts.URL,"http://")
	conn, err := net.Dial("tcp",address)
	if err != nil { t.Fatal(err) }
	defer conn.Close()
	_, err = conn.Write([]byte("GET /v1/brain/stream?after=1 HTTP/1.1\r\nHost: "+address+"\r\nConnection: Upgrade\r\nUpgrade: websocket\r\nSec-WebSocket-Version: 13\r\nSec-WebSocket-Key: dGhlIHNhbXBsZSBub25jZQ==\r\n\r\n"))
	if err != nil { t.Fatal(err) }
	reader := bufio.NewReader(conn)
	response, err := http.ReadResponse(reader, &http.Request{Method:http.MethodGet})
	if err != nil { t.Fatal(err) }
	if response.StatusCode != http.StatusSwitchingProtocols {
		t.Fatalf("websocket handshake returned %d",response.StatusCode)
	}
	if err := conn.SetReadDeadline(time.Now().Add(3*time.Second)); err != nil { t.Fatal(err) }
	opcode, frame, err := readWebSocketFrame(reader)
	if err != nil { t.Fatalf("read bridge telemetry handshake: %v",err) }
	if opcode != 1 || !strings.Contains(string(frame),"brain.handshake") { t.Fatalf("unexpected handshake frame: %q",frame) }

	if err := supervisor.RecoverFromCheckpoint(log,checkpointPath); err != nil { t.Fatal(err) }
	// Handovers terminate the previous WebSocket subscription rather than
	// silently continuing to stream telemetry from the retired neural state.
	if _,_,err := readWebSocketFrame(reader); err == nil {
		t.Fatal("old telemetry WebSocket stayed alive after runtime replacement")
	} else if errors.Is(err, net.ErrClosed) {
		// The connection has been closed as expected.
	}
}

func TestManagedBridgeRejectsNilSupervisor(t *testing.T) {
	if _,err:=NewManaged(nil,Config{});err==nil { t.Fatal("nil supervisor accepted") }
}
