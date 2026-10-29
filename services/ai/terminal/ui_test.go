package terminal

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Reproduce the first Termux conversation reported by a user. A vocabulary or
// question-form rejection must not be presented as a missing-object question.
func TestTerminalReportedUnsupportedConversation(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "chat")
	s, err := Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	before := map[string][]byte{}
	for _, name := range []string{"language-brain.json", "events.jsonl", "codec.json"} {
		before[name], err = os.ReadFile(filepath.Join(dir, name))
		if err != nil {
			t.Fatal(err)
		}
	}
	for _, tc := range []struct {
		text, reason string
		focus        float64
	}{
		{"hallo", "untrained-token", 0},
		{"apa itu lampu?", "untrained-token", 0},
		{"lampu?", "unsupported-composition", .25},
	} {
		out, err := s.Ask(tc.text)
		if err != nil || out.MeaningValid || out.Reason != tc.reason || out.Focus != tc.focus {
			t.Fatalf("%q: reason=%q focus=%g valid=%t error=%v", tc.text, out.Reason, out.Focus, out.MeaningValid, err)
		}
		t.Logf("%q: native=%q reason=%s focus=%g", tc.text, out.Text, out.Reason, out.Focus)
	}
	if err := s.NewConversation(); err != nil {
		t.Fatal(err)
	}
	input := "\n\nhallo\n/baru\nhallo\n/bantuan\napa itu lampu?\nlampu?\nBagaimana keadaan lampu?\n/keluar\n"
	var output bytes.Buffer
	if err := Run(s, strings.NewReader(input), &output, Options{EchoInput: true}); err != nil {
		t.Fatal(err)
	}
	text := output.String()
	if strings.Contains(text, "Objek yang mana?") {
		t.Error("unsupported words/question forms were misreported as a missing object")
	}
	if got := strings.Count(text, "Kosakata pesan belum seluruhnya dipelajari."); got != 3 {
		t.Errorf("got %d vocabulary diagnostics, want 3", got)
	}
	if !strings.Contains(text, "Gabungan kata ini belum membentuk pertanyaan yang didukung.") {
		t.Error("missing question-form diagnostic for a recognized noun")
	}
	if !strings.Contains(text, "Lampu itu sekarang padam.") || strings.Count(text, "Horizon ·") != 1 {
		t.Error("supported question did not remain a native Horizon answer")
	}
	for _, turn := range s.state.History {
		if strings.TrimSpace(turn) == "" {
			t.Error("empty input was treated as a conversation turn")
		}
	}
	for name, want := range before {
		got, err := os.ReadFile(filepath.Join(dir, name))
		if err != nil || !bytes.Equal(got, want) {
			t.Errorf("diagnostic handling changed canonical file %s", name)
		}
	}
}
