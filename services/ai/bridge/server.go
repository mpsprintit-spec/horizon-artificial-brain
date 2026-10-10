package bridge

import (
	"bufio"
	"crypto/sha1"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"net"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/project-horizon/horizon-core/services/ai/runtime"
)

type Config struct {
	AllowedOrigins []string
	Token string
	RuntimeCommit string
	EventLogPath string
	BrainMemoryPath string
	MaxPayloadBytes int64
	RateLimitPerMinute int
}

type Server struct {
	runtime *runtime.BrainRuntime
	cfg Config
	rateMu sync.Mutex
	rate map[string]*rateBucket
}

type rateBucket struct { started time.Time; count int }

func New(brainRuntime *runtime.BrainRuntime, cfg Config) (*Server, error) {
	if brainRuntime == nil { return nil, errors.New("brain runtime is required") }
	if cfg.MaxPayloadBytes <= 0 { cfg.MaxPayloadBytes = 4 << 20 }
	if cfg.RateLimitPerMinute <= 0 { cfg.RateLimitPerMinute = 120 }
	return &Server{runtime: brainRuntime, cfg: cfg, rate: make(map[string]*rateBucket)}, nil
}

func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/health", s.health)
	mux.HandleFunc("/v1/brain/handshake", s.handshake)
	mux.HandleFunc("/v1/brain/snapshot", s.snapshot)
	mux.HandleFunc("/v1/brain/events", s.events)
	mux.HandleFunc("/v1/brain/stream", s.stream)
	mux.HandleFunc("/v1/observations", s.observations)
	mux.HandleFunc("/v1/outcomes", s.outcomes)
	return s.withCORS(s.withAuth(s.withRateLimit(mux)))
}

func (s *Server) health(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet { writeError(w,http.StatusMethodNotAllowed,"method not allowed"); return }
	snapshot, err := s.runtime.MonitorSnapshot()
	if err != nil { writeError(w,http.StatusServiceUnavailable,err.Error()); return }
	writeJSON(w,http.StatusOK,map[string]any{"status":"ok","brain_identity":runtime.BrainIdentity,"state_revision":snapshot.StateRevision,"canonical_state_hash":snapshot.CanonicalStateHash})
}

func (s *Server) handshake(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet { writeError(w,http.StatusMethodNotAllowed,"method not allowed"); return }
	handshake, err := s.runtime.MonitorHandshake(s.cfg.RuntimeCommit)
	if err != nil { writeError(w,http.StatusServiceUnavailable,err.Error()); return }
	writeJSON(w,http.StatusOK,handshake)
}

func (s *Server) snapshot(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet { writeError(w,http.StatusMethodNotAllowed,"method not allowed"); return }
	snapshot, err := s.runtime.MonitorSnapshot()
	if err != nil { writeError(w,http.StatusServiceUnavailable,err.Error()); return }
	writeJSON(w,http.StatusOK,snapshot)
}

func (s *Server) events(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet { writeError(w,http.StatusMethodNotAllowed,"method not allowed"); return }
	events, err := s.readEvents(afterRevision(r))
	if err != nil { writeError(w,http.StatusServiceUnavailable,err.Error()); return }
	writeJSON(w,http.StatusOK,map[string]any{"brain_identity":runtime.BrainIdentity,"events":events})
}

type replayEvent struct {
	Type string `json:"type"`
	BrainIdentity string `json:"brain_identity"`
	StateRevision uint64 `json:"state_revision"`
	Timestamp time.Time `json:"timestamp"`
	EventHash string `json:"event_hash"`
	Event *runtime.Event `json:"event,omitempty"`
	Experience any `json:"experience,omitempty"`
	Outcome *runtime.OutcomeEvent `json:"outcome,omitempty"`
}

func (s *Server) readEvents(after uint64) ([]replayEvent,error) {
	if strings.TrimSpace(s.cfg.EventLogPath)=="" { return []replayEvent{},nil }
	logged, err := runtime.ReadEventLog(s.cfg.EventLogPath)
	if err != nil { return nil,err }
	out := make([]replayEvent,0)
	for _, e := range logged {
		if e.Sequence <= after { continue }
		payload,_ := json.Marshal(e)
		hash:=sha256.Sum256(payload)
		item:=replayEvent{Type:e.Type,BrainIdentity:e.BrainIdentity,StateRevision:e.Sequence,Timestamp:e.Timestamp,EventHash:"sha256:"+hex.EncodeToString(hash[:]),Event:e.Event,Outcome:e.Outcome}
		out=append(out,item)
	}
	return out,nil
}

func (s *Server) stream(w http.ResponseWriter, r *http.Request) {
	if !websocketRequested(r) { writeError(w,http.StatusBadRequest,"websocket upgrade required"); return }
	if !originAllowed(r.Header.Get("Origin"),s.cfg.AllowedOrigins) { writeError(w,http.StatusForbidden,"origin is not allowed"); return }
	h, ok := w.(http.Hijacker); if !ok { writeError(w,http.StatusInternalServerError,"websocket hijacking is unavailable"); return }
	conn, rw, err := h.Hijack(); if err != nil { return }
	defer conn.Close()
	key:=r.Header.Get("Sec-WebSocket-Key")
	accept:=websocketAccept(key)
	_,_ = rw.WriteString("HTTP/1.1 101 Switching Protocols\r\nUpgrade: websocket\r\nConnection: Upgrade\r\nSec-WebSocket-Accept: "+accept+"\r\n\r\n")
	_ = rw.Flush()

	sub:=s.runtime.Telemetry().Subscribe()
	if sub==nil { return }
	defer s.runtime.Telemetry().Unsubscribe(sub)
	handshake, err := s.runtime.MonitorHandshake(s.cfg.RuntimeCommit)
	if err == nil { _ = writeWebSocketJSON(rw.Writer,handshake) }

	after:=afterRevision(r)
	if events, err := s.readEvents(after); err == nil {
		for _, event := range events { if err:=writeWebSocketJSON(rw.Writer,event); err!=nil{return} }
	}
	var writeMu sync.Mutex
	writeJSONFrame := func(value any) error {
		writeMu.Lock()
		defer writeMu.Unlock()
		return writeWebSocketJSON(rw.Writer, value)
	}
	readDone := make(chan struct{})
	go func() {
		defer close(readDone)
		for {
			opcode, payload, err := readWebSocketFrame(rw.Reader)
			if err != nil { return }
			switch opcode {
			case 0x8: // close
				writeMu.Lock()
				_ = writeWebSocketFrame(rw.Writer, 0x8, payload)
				_ = rw.Flush()
				writeMu.Unlock()
				return
			case 0x9: // ping
				writeMu.Lock()
				_ = writeWebSocketFrame(rw.Writer, 0xA, payload)
				_ = rw.Flush()
				writeMu.Unlock()
			}
		}
	}()
	for {
		select {
		case <-readDone:
			return
		case event, ok := <-sub.C():
			if !ok { return }
			if event.StateRevision <= after { continue }
			if err:=writeJSONFrame(event); err!=nil{return}
			after=event.StateRevision
		}
	}
}

func (s *Server) observations(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost { writeError(w,http.StatusMethodNotAllowed,"method not allowed"); return }
	defer r.Body.Close()
	r.Body=http.MaxBytesReader(w,r.Body,s.cfg.MaxPayloadBytes)
	var envelope runtime.ObservationEnvelope
	if err:=json.NewDecoder(r.Body).Decode(&envelope); err!=nil { writeError(w,http.StatusBadRequest,"invalid observation envelope: "+err.Error()); return }
	if envelope.BrainIdentity!="" && envelope.BrainIdentity!=runtime.BrainIdentity { writeError(w,http.StatusBadRequest,"invalid brain_identity"); return }
	interpretation, err:=s.runtime.ProcessObservationEnvelope(envelope)
	if err!=nil { writeError(w,http.StatusUnprocessableEntity,err.Error()); return }
	if s.cfg.BrainMemoryPath!="" { if err:=s.runtime.SaveBrain(s.cfg.BrainMemoryPath); err!=nil { writeError(w,http.StatusInternalServerError,"brain persistence failed: "+err.Error()); return } }
	writeJSON(w,http.StatusAccepted,map[string]any{"brain_identity":runtime.BrainIdentity,"event_id":envelope.EventID,"state_revision":interpretation.State.Sequence,"prediction_error":interpretation.State.PredictionError,"active_node_ids":interpretation.State.ActiveNodeIDs})
}

func (s *Server) outcomes(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost { writeError(w,http.StatusMethodNotAllowed,"method not allowed"); return }
	defer r.Body.Close(); r.Body=http.MaxBytesReader(w,r.Body,s.cfg.MaxPayloadBytes)
	var outcome runtime.OutcomeEvent
	if err:=json.NewDecoder(r.Body).Decode(&outcome); err!=nil { writeError(w,http.StatusBadRequest,"invalid outcome: "+err.Error()); return }
	seq,err:=s.runtime.ObserveOutcome(outcome)
	if err!=nil { writeError(w,http.StatusUnprocessableEntity,err.Error()); return }
	if s.cfg.BrainMemoryPath!="" { if err:=s.runtime.SaveBrain(s.cfg.BrainMemoryPath); err!=nil { writeError(w,http.StatusInternalServerError,"brain persistence failed: "+err.Error()); return } }
	writeJSON(w,http.StatusAccepted,map[string]any{"brain_identity":runtime.BrainIdentity,"request_id":outcome.RequestID,"state_revision":seq})
}

func (s *Server) withAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter,r *http.Request) {
		if s.cfg.Token=="" || r.URL.Path=="/health" { next.ServeHTTP(w,r); return }
	auth := r.Header.Get("Authorization")
		if auth == "" && r.URL.Path == "/v1/brain/stream" { if token := r.URL.Query().Get("token"); token != "" { auth = "Bearer " + token } }
		if auth!="Bearer "+s.cfg.Token { writeError(w,http.StatusUnauthorized,"missing or invalid bridge token"); return }
		next.ServeHTTP(w,r)
	})
}
func (s *Server) withRateLimit(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter,r *http.Request) {
		key:=r.RemoteAddr; if host,_,err:=net.SplitHostPort(r.RemoteAddr);err==nil{key=host}
		now:=time.Now()
		s.rateMu.Lock(); b:=s.rate[key]; if b==nil || now.Sub(b.started)>=time.Minute { b=&rateBucket{started:now}; s.rate[key]=b }; b.count++; count:=b.count; s.rateMu.Unlock()
		if count>s.cfg.RateLimitPerMinute { writeError(w,http.StatusTooManyRequests,"rate limit exceeded"); return }
		next.ServeHTTP(w,r)
	})
}
func (s *Server) withCORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter,r *http.Request) {
		origin:=strings.TrimSpace(r.Header.Get("Origin"))
		if origin!="" {
			if !originAllowed(origin,s.cfg.AllowedOrigins){writeError(w,http.StatusForbidden,"origin is not allowed");return}
			w.Header().Set("Access-Control-Allow-Origin",origin); w.Header().Set("Vary","Origin"); w.Header().Set("Access-Control-Allow-Headers","Authorization, Content-Type"); w.Header().Set("Access-Control-Allow-Methods","GET, POST, OPTIONS")
		}
		if r.Method==http.MethodOptions { w.WriteHeader(http.StatusNoContent);return }
		next.ServeHTTP(w,r)
	})
}
func originAllowed(origin string,allowed []string) bool { if origin=="" {return true}; for _,v:=range allowed{if strings.TrimSpace(v)==origin{return true}};return false }
func websocketRequested(r *http.Request) bool { return strings.EqualFold(r.Header.Get("Connection"),"Upgrade")&&strings.EqualFold(r.Header.Get("Upgrade"),"websocket")&&r.Header.Get("Sec-WebSocket-Key")!="" }
func websocketAccept(key string) string { sum:=sha1.Sum([]byte(key+"258EAFA5-E914-47DA-95CA-C5AB0DC85B11"));return base64.StdEncoding.EncodeToString(sum[:]) }
func readWebSocketFrame(r *bufio.Reader) (byte, []byte, error) {
	first, err := r.ReadByte()
	if err != nil { return 0, nil, err }
	second, err := r.ReadByte()
	if err != nil { return 0, nil, err }
	opcode := first & 0x0F
	masked := second&0x80 != 0
	length := int64(second & 0x7F)
	if length == 126 {
		var b [2]byte
		if _, err := io.ReadFull(r, b[:]); err != nil { return 0, nil, err }
		length = int64(b[0])<<8 | int64(b[1])
	} else if length == 127 {
		var b [8]byte
		if _, err := io.ReadFull(r, b[:]); err != nil { return 0, nil, err }
		for _, v := range b { if length > (1<<56)-1 { return 0, nil, errors.New("websocket frame too large") }; length = (length << 8) | int64(v) }
	}
	if length > 4<<20 { return 0, nil, errors.New("websocket frame too large") }
	var mask [4]byte
	if masked { if _, err := io.ReadFull(r, mask[:]); err != nil { return 0, nil, err } }
	payload := make([]byte, int(length))
	if _, err := io.ReadFull(r, payload); err != nil { return 0, nil, err }
	if masked { for i := range payload { payload[i] ^= mask[i%4] } }
	return opcode, payload, nil
}

func writeWebSocketFrame(w io.Writer, opcode byte, payload []byte) error {
	n := len(payload)
	frame := make([]byte, 0, n+10)
	frame = append(frame, 0x80|(opcode&0x0F))
	switch {
	case n < 126:
		frame = append(frame, byte(n))
	case n <= 65535:
		frame = append(frame, 126, byte(n>>8), byte(n))
	default:
		frame = append(frame, 127, 0, 0, 0, 0, byte(uint64(n)>>24), byte(uint64(n)>>16), byte(uint64(n)>>8), byte(uint64(n)))
	}
	frame = append(frame, payload...)
	_, err := w.Write(frame)
	return err
}

func writeWebSocketJSON(w io.Writer,v any) error { data,err:=json.Marshal(v);if err!=nil{return err}; n:=len(data); frame:=make([]byte,0,n+10);frame=append(frame,0x81);switch{case n<126:frame=append(frame,byte(n));case n<=65535:frame=append(frame,126,byte(n>>8),byte(n));default:frame=append(frame,127,0,0,0,0,byte(uint64(n)>>24),byte(uint64(n)>>16),byte(uint64(n)>>8),byte(uint64(n)))};frame=append(frame,data...);_,err=w.Write(frame);return err}
func afterRevision(r *http.Request) uint64 { value,_:=strconv.ParseUint(r.URL.Query().Get("after"),10,64);return value }

func writeJSON(w http.ResponseWriter,status int,value any){w.Header().Set("Content-Type","application/json");w.WriteHeader(status);_=json.NewEncoder(w).Encode(value)}
func writeError(w http.ResponseWriter,status int,message string){writeJSON(w,status,map[string]any{"error":message})}