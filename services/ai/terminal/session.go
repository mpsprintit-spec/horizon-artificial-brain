// Package terminal adapts observed room state and raw chat history to the one
// active Horizon Brain. It contains no language teacher or answer templates.
package terminal

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"
	"unicode"

	profile "github.com/project-horizon/horizon-core/models/grounded-dialogue-v1"
	"github.com/project-horizon/horizon-core/services/ai/core"
	"github.com/project-horizon/horizon-core/services/ai/runtime"
)

const historyLimit = 16

type sessionState struct {
	Version int                  `json:"version"`
	Room    runtime.DialogueRoom `json:"room"`
	History []string             `json:"history"`
}

type Session struct {
	engine  *core.HorizonEngine
	journal *runtime.EventLog
	codec   runtime.DialogueCodec
	state   sessionState
	path    string
	now     func() time.Time
}

// Open selects a dedicated profile. Only a missing directory is initialized;
// existing Brain files must pass the normal authenticated recovery path.
func Open(directory string) (*Session, error) {
	if _, err := os.Stat(directory); errors.Is(err, os.ErrNotExist) {
		if err := os.MkdirAll(filepath.Dir(directory), 0700); err != nil {
			return nil, err
		}
		if err := profile.Prepare(directory); err != nil {
			return nil, err
		}
	} else if err != nil {
		return nil, err
	}
	for _, name := range []string{"codec.json", "language-brain.json", "events.jsonl"} {
		info, err := os.Lstat(filepath.Join(directory, name))
		if err != nil {
			return nil, fmt.Errorf("profil chat tidak lengkap; gunakan direktori profil baru: %w", err)
		}
		if !info.Mode().IsRegular() {
			return nil, fmt.Errorf("file profil harus berupa file biasa: %s", name)
		}
	}
	var codec runtime.DialogueCodec
	if err := readJSON(filepath.Join(directory, "codec.json"), &codec); err != nil {
		return nil, fmt.Errorf("baca codec profil: %w", err)
	}
	if err := codec.Validate(); err != nil {
		return nil, err
	}
	journal, err := runtime.OpenEventLog(filepath.Join(directory, "events.jsonl"))
	if err != nil {
		return nil, fmt.Errorf("buka jurnal profil (tutup sesi lain jika profil sedang dipakai): %w", err)
	}
	engine := core.NewHorizonEngine()
	if err := engine.Supervisor().RecoverFromCheckpoint(journal, filepath.Join(directory, "language-brain.json")); err != nil {
		_ = engine.Supervisor().Close()
		_ = journal.Close()
		return nil, err
	}
	at := time.Now().UTC()
	s := &Session{engine: engine, journal: journal, codec: codec, path: filepath.Join(directory, "chat-session.json"), now: time.Now,
		state: sessionState{Version: 1, Room: runtime.DialogueRoom{Current: []float64{-1, -1, -1}, Previous: []float64{-1, -1, -1}, Kinds: []float64{1, 1, -1}, ObservedAt: at, PreviousAt: at.Add(-time.Second)}}}
	var saved sessionState
	if err := readJSON(s.path, &saved); err == nil {
		s.state = saved
	} else if !errors.Is(err, os.ErrNotExist) {
		_ = s.Close()
		return nil, fmt.Errorf("konteks sesi tidak valid: %w", err)
	}
	if err := s.state.validate(); err != nil {
		_ = s.Close()
		return nil, err
	}
	return s, nil
}

func readJSON(path string, target any) error {
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()
	const budget = 64 << 10
	data, err := io.ReadAll(io.LimitReader(f, budget+1))
	if err != nil {
		return err
	}
	if len(data) > budget {
		return fmt.Errorf("file konteks/codec terlalu besar")
	}
	return json.Unmarshal(data, target)
}

func (s sessionState) validate() error {
	r := s.Room
	if s.Version != 1 || len(s.History) > historyLimit || len(r.Current) != 3 || len(r.Previous) != 3 || len(r.Kinds) != 3 || r.ObservedAt.IsZero() || r.PreviousAt.IsZero() || r.PreviousAt.After(r.ObservedAt) {
		return fmt.Errorf("format konteks sesi tidak valid")
	}
	for _, values := range [][]float64{r.Current, r.Previous} {
		for _, v := range values {
			if v != 1 && v != -1 {
				return fmt.Errorf("keadaan ruangan bukan nilai biner")
			}
		}
	}
	if r.Kinds[0] != 1 || r.Kinds[1] != 1 || r.Kinds[2] != -1 {
		return fmt.Errorf("koordinat jenis ruangan tidak cocok dengan profil")
	}
	for _, text := range s.History {
		if len(text) > 1024 || text != cleanText(text) {
			return fmt.Errorf("riwayat sesi tidak valid")
		}
	}
	return nil
}

func cleanText(text string) string {
	return strings.Map(func(r rune) rune {
		if unicode.IsControl(r) || unicode.In(r, unicode.Cf) {
			return -1
		}
		return r
	}, text)
}

func (s *Session) save(next sessionState) error {
	if err := next.validate(); err != nil {
		return err
	}
	data, err := json.MarshalIndent(next, "", "  ")
	if err != nil {
		return err
	}
	f, err := os.CreateTemp(filepath.Dir(s.path), ".chat-session-*")
	if err != nil {
		return err
	}
	defer os.Remove(f.Name())
	if _, err = f.Write(append(data, '\n')); err != nil {
		f.Close()
		return err
	}
	if err = f.Sync(); err != nil {
		f.Close()
		return err
	}
	if err = f.Close(); err != nil {
		return err
	}
	if err = os.Rename(f.Name(), s.path); err != nil {
		return err
	}
	s.state = next
	return nil
}

func (s *Session) at() time.Time {
	at := s.now().UTC()
	if at.Before(s.state.Room.ObservedAt) {
		return s.state.Room.ObservedAt
	}
	return at
}

func (s *Session) Ask(text string) (runtime.GroundedDialogueReadout, error) {
	if text == "" || len(text) > 1024 || text != cleanText(text) {
		return runtime.GroundedDialogueReadout{}, fmt.Errorf("pesan harus bersih dan maksimal 1024 byte")
	}
	out, err := s.engine.ReadGroundedDialogue(s.codec, runtime.GroundedDialogueRequest{Text: text, History: append([]string(nil), s.state.History...), Room: s.state.Room, At: s.at()})
	if err != nil {
		return out, err
	}
	next := s.state
	next.History = append(append([]string(nil), next.History...), text)
	if len(next.History) > historyLimit {
		next.History = next.History[len(next.History)-historyLimit:]
	}
	if err := s.save(next); err != nil {
		return out, fmt.Errorf("konteks belum tersimpan: %w", err)
	}
	return out, nil
}

// SetRoom is an explicit simulated sensor control, never a training label or a
// command to a physical device. Previous keeps the last complete observation.
func (s *Session) SetRoom(entity, state string) error {
	index := -1
	switch entity {
	case "lampu":
		index = 0
	case "kipas":
		index = 1
	case "pintu":
		index = 2
	}
	value := 0.
	if index == 2 {
		switch state {
		case "buka":
			value = 1
		case "tutup":
			value = -1
		}
	} else if index >= 0 {
		switch state {
		case "nyala":
			value = 1
		case "mati":
			value = -1
		}
	}
	if index < 0 || value == 0 {
		return fmt.Errorf("gunakan /set lampu|kipas nyala|mati, atau /set pintu buka|tutup")
	}
	next := s.state
	r := next.Room
	r.Previous = append([]float64(nil), r.Current...)
	r.PreviousAt = r.ObservedAt
	r.Current = append([]float64(nil), r.Current...)
	r.Current[index] = value
	r.ObservedAt = s.at()
	next.Room = r
	return s.save(next)
}

func (s *Session) NewConversation() error { next := s.state; next.History = nil; return s.save(next) }
func (s *Session) Close() error           { return errors.Join(s.engine.Supervisor().Close(), s.journal.Close()) }
