package terminal

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	profile "github.com/project-horizon/horizon-core/models/grounded-dialogue-v1"
	"github.com/project-horizon/horizon-core/services/ai/runtime"
)

func TestCommunicationConsoleAndTrainingProvenance(t *testing.T) {
	dir, err := BundledProfileDirectory(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	s, err := Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	release, err := profile.BundledRelease()
	if err != nil {
		t.Fatal(err)
	}
	info := s.BrainInfo()
	if !info.MatchesBundle || info.SourceFingerprint != release.Files.Fingerprint() || info.Sequence != release.Sequence || info.CanonicalStateHash == "" {
		t.Fatalf("the selected training artifacts were not identified: %+v", info)
	}
	var startup bytes.Buffer
	if err := Run(s, strings.NewReader("/keluar\n"), &startup, Options{}); err != nil {
		t.Fatal(err)
	}
	for _, forbidden := range []string{"RUANG SIMULASI", "lampu", "kipas", "pintu", "Contoh pertanyaan", "Sapaan dan definisi"} {
		if strings.Contains(startup.String(), forbidden) {
			t.Errorf("default console still presents a training exercise: %q", forbidden)
		}
	}
	var output bytes.Buffer
	input := "/brain\n/kemampuan\nBagaimana keadaan lampu?\n/diagnosis\n/baru\n/diagnosis\n/lab\n/keluar\n"
	if err := Run(s, strings.NewReader(input), &output, Options{EchoInput: true}); err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{info.SourceFingerprint, info.CanonicalStateHash, "Berkas cocok dengan paket hasil latihan", "Percakapan umum, sapaan dan definisi belum dikuasai", "Lampu itu sekarang padam.", " ms · simulasi", "Makna diterima: true", "Pembacaan N diterima:", "Belum ada pesan yang dapat diperiksa", "RUANG SIMULASI"} {
		if !strings.Contains(output.String(), want) {
			t.Errorf("console did not report %q", want)
		}
	}
	if err := s.engine.Supervisor().With(func(rt *runtime.BrainRuntime) error {
		state, err := rt.MonitorSnapshot()
		if err == nil && (state.CanonicalStateHash != info.CanonicalStateHash || state.StateRevision != info.Sequence) {
			t.Error("console inspection changed the trained Brain")
		}
		return err
	}); err != nil {
		t.Fatal(err)
	}
}

func TestCommunicationDistinguishesSelectedAndBundledProfiles(t *testing.T) {
	root := t.TempDir()
	selected := filepath.Join(root, "my-brain")
	if err := profile.PrepareServingProfile(selected); err != nil {
		t.Fatal(err)
	}
	// Semantically identical codec with different source bytes: conservative
	// provenance matching must not silently label arbitrary files as published.
	codec := filepath.Join(selected, "codec.json")
	data, err := os.ReadFile(codec)
	if err != nil {
		t.Fatal(err)
	}
	data = append(data, '\n')
	if err := os.WriteFile(codec, data, 0600); err != nil {
		t.Fatal(err)
	}
	s, err := Open(selected)
	if err != nil {
		t.Fatal(err)
	}
	if s.BrainInfo().MatchesBundle {
		t.Error("different profile was labeled as the bundled training result")
	}
	var output bytes.Buffer
	if err := Run(s, strings.NewReader("/brain\n/kemampuan\n/keluar\n"), &output, Options{}); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(output.String(), "Profil aktif berbeda") || !strings.Contains(output.String(), "-latest") || !strings.Contains(output.String(), "Belum ada daftar kemampuan") {
		t.Error("selected profile provenance/limits were not exposed")
	}
	if err := s.Close(); err != nil {
		t.Fatal(err)
	}
	latest, err := BundledProfileDirectory(root)
	if err != nil || latest == selected {
		t.Fatal("published training did not select a separate profile", err)
	}
	s, err = Open(latest)
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	if !s.BrainInfo().MatchesBundle {
		t.Error("published training was not loaded")
	}
	out, err := s.Ask("Bagaimana keadaan lampu?")
	if err != nil || !out.MeaningValid || out.Text != "Lampu itu sekarang padam." {
		t.Fatalf("published Brain did not supply the answer: %q; %v", out.Text, err)
	}
	for _, trace := range out.Trace {
		if trace.Readout.Sequence != s.BrainInfo().Sequence || trace.Readout.BrainIdentity != s.BrainInfo().Identity {
			t.Error("answer used a different canonical Brain revision")
		}
	}
	unchanged, err := os.ReadFile(codec)
	if err != nil || !bytes.Equal(unchanged, data) {
		t.Error("selecting published training overwrote the previous profile")
	}
}
