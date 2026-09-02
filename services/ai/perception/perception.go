package perception

import (
	"strings"
	"time"
)

type PerceptionKind string

const (
	PerceptionUserInput PerceptionKind = "user_input"
	PerceptionWebSearch PerceptionKind = "web_search"
)

// PerceptionSignal adalah bentuk SERAGAM untuk apa pun yang masuk ke Horizon
// -- teks user sekarang, nanti Vision/Speech/Sensor -- semuanya harus
// menghasilkan bentuk ini sebelum menyentuh Learning. RawText dipertahankan
// (bukan cuma Tokens) karena Learning butuh struktur kalimat/klausa penuh,
// bukan sekadar kumpulan kata lepas.
type PerceptionSignal struct {
	Kind       PerceptionKind
	Source     string
	RawText    string
	Tokens     []string
	Confidence float64
	ObservedAt time.Time
}

type PerceptionLayer interface {
	Perceive(input string) ([]PerceptionSignal, error)
}

type UserInputPerception struct{}

func (UserInputPerception) Perceive(input string) ([]PerceptionSignal, error) {
	text := strings.TrimSpace(input)
	return []PerceptionSignal{{
		Kind:       PerceptionUserInput,
		Source:     "user",
		RawText:    text,
		Tokens:     strings.Fields(strings.ToLower(text)),
		Confidence: 1,
		ObservedAt: time.Now().UTC(),
	}}, nil
}

// FromWebSearch membungkus hasil pencarian eksternal jadi PerceptionSignal
// yang seragam -- supaya WebSearch tidak lagi punya jalur pintas langsung
// ke Learning, melewati Perception seperti sebelumnya.
func FromWebSearch(source string, tokens []string, confidence float64) PerceptionSignal {
	return PerceptionSignal{
		Kind:       PerceptionWebSearch,
		Source:     source,
		RawText:    strings.Join(tokens, " "),
		Tokens:     tokens,
		Confidence: confidence,
		ObservedAt: time.Now().UTC(),
	}
}
