package terminal

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/project-horizon/horizon-core/services/ai/runtime"
)

func TestNativeChatContextRoomRestartAndReadOnlyBrain(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "chat")
	s, err := Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	closed := false
	defer func() {
		if !closed {
			_ = s.Close()
		}
	}()
	files := map[string][]byte{}
	for _, name := range []string{"language-brain.json", "events.jsonl", "codec.json"} {
		data, err := os.ReadFile(filepath.Join(dir, name))
		if err != nil {
			t.Fatal(err)
		}
		files[name] = data
	}
	if second, err := Open(dir); err == nil {
		_ = second.Close()
		t.Fatal("two active writers accepted the same profile")
	}
	answer := func(text, want string) {
		t.Helper()
		out, err := s.Ask(text)
		if err != nil || out.Text != want {
			t.Fatalf("%q => %q (%s), want %q; error %v", text, out.Text, out.Reason, want, err)
		}
	}
	if err := s.SetRoom("lampu", "nyala"); err != nil {
		t.Fatal(err)
	}
	answer("Bagaimana keadaan lampu?", "Lampu itu sekarang menyala.")
	if err := s.SetRoom("lampu", "mati"); err != nil {
		t.Fatal(err)
	}
	answer("Bagaimana keadaannya?", "Lampu itu sekarang padam.")
	answer("Bagaimana keadaannya tadi?", "Lampu itu tadi menyala.")
	if err := s.SetRoom("lampu", "buka"); err == nil {
		t.Fatal("invalid sensor state accepted")
	}
	// A device clock rollback cannot turn an observed room into future evidence.
	s.now = func() time.Time { return s.state.Room.PreviousAt.Add(-time.Hour) }
	answer("Bagaimana keadaan lampu?", "Lampu itu sekarang padam.")
	if err := s.Close(); err != nil {
		t.Fatal(err)
	}
	closed = true
	s, err = Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	closed = false
	answer("Bagaimana keadaannya?", "Lampu itu sekarang padam.")
	if err := s.NewConversation(); err != nil {
		t.Fatal(err)
	}
	answer("Bagaimana keadaannya?", "Objek yang mana?")
	for n := 0; n < 18; n++ {
		answer("Bagaimana keadaan lampu?", "Lampu itu sekarang padam.")
	}
	if len(s.state.History) != historyLimit {
		t.Fatal("chat history was not bounded")
	}
	if err := s.engine.Supervisor().With(func(rt *runtime.BrainRuntime) error {
		if rt.LastSequence() != 126 {
			t.Fatal("chat mutated the canonical Brain sequence")
		}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	if err := s.Close(); err != nil {
		t.Fatal(err)
	}
	closed = true
	for name, want := range files {
		got, err := os.ReadFile(filepath.Join(dir, name))
		if err != nil || !bytes.Equal(want, got) {
			t.Fatalf("chat changed canonical profile file %s", name)
		}
	}
	// A malformed saved session must fail without replacing it or the Brain.
	if err := os.WriteFile(filepath.Join(dir, "chat-session.json"), []byte("{}"), 0600); err != nil {
		t.Fatal(err)
	}
	if bad, err := Open(dir); err == nil {
		_ = bad.Close()
		t.Fatal("malformed context was silently adopted")
	}
}

func TestTerminalTranscriptCommandsAndPlainOutput(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "chat")
	s, err := Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	input := strings.Join([]string{"/bantuan", "/set lampu nyala", "Bagaimana keadaan lampu?", "/set lampu mati", "Bagaimana keadaannya?", "Bagaimana keadaannya tadi?", "/riwayat", "/baru", "Bagaimana keadaannya?", "/set pintu buka", "Bagaimana keadaan pintu?", strings.Repeat("a", 1025), "/ruang", "/tidak-ada", "\x1b[31m", "/keluar"}, "\n") + "\n"
	var output bytes.Buffer
	if err := Run(s, strings.NewReader(input), &output, Options{EchoInput: true, Width: 40}); err != nil {
		t.Fatal(err)
	}
	text := output.String()
	for _, want := range []string{"HORIZON", "Belum percakapan umum", "Lampu itu sekarang menyala.", "Lampu itu sekarang padam.", "Lampu itu tadi menyala.", "Objek yang mana?", "Pintu itu sekarang terbuka.", "1024 byte", "Perintah tidak dikenal", "Sesi ditutup", " ms"} {
		if !strings.Contains(text, want) {
			t.Errorf("transcript missing %q", want)
		}
	}
	if strings.Contains(text, "\x1b") {
		t.Fatal("plain UI emitted a terminal control sequence")
	}
}

func TestSessionPersistenceWithFullyEscapedHistory(t *testing.T) {
	at := time.Date(2026, 10, 29, 0, 0, 0, 0, time.UTC)
	state := sessionState{Version: 1, Room: runtime.DialogueRoom{Current: []float64{-1, -1, -1}, Previous: []float64{-1, -1, -1}, Kinds: []float64{1, 1, -1}, ObservedAt: at, PreviousAt: at.Add(-time.Second)}}
	for n := 0; n < historyLimit; n++ {
		state.History = append(state.History, strings.Repeat("<", 1024))
	}
	s := &Session{path: filepath.Join(t.TempDir(), "chat-session.json")}
	if err := s.save(state); err != nil {
		t.Fatal(err)
	}
	var restored sessionState
	if err := readJSON(s.path, &restored); err != nil {
		t.Fatalf("a valid full history cannot be resumed: %v", err)
	}
	if err := restored.validate(); err != nil {
		t.Fatal(err)
	}
	if strings.Join(restored.History, "") != strings.Join(state.History, "") {
		t.Fatal("valid escaped text changed during persistence")
	}
}
