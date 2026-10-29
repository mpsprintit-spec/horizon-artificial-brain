package terminal

import (
	"bufio"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/project-horizon/horizon-core/services/ai/runtime"
)

type Options struct {
	Color     bool
	EchoInput bool // Piped transcripts have no terminal echo.
	Width     int
}

// The finite training curriculum uses the same native clarification for many
// rejected meanings. Report the actual rejection as UI feedback; do not present
// that generic text as a relevant answer or invent a replacement Brain reply.
func rejectionMessage(reply runtime.GroundedDialogueReadout) string {
	if reply.MeaningValid {
		return ""
	}
	switch reply.Reason {
	case "untrained-token":
		return "Pesan belum dipahami oleh Brain. /diagnosis untuk rinciannya."
	case "unsupported-composition":
		return "Pesan belum dipahami oleh Brain. /diagnosis untuk rinciannya."
	case "utterance-bound":
		return "Pesan melampaui bentuk masukan yang didukung Brain saat ini. /diagnosis untuk rinciannya."
	case "unresolved-meaning":
		if reply.Focus == 0 && reply.Text != "" {
			// No single referent was resolved by the learned reference head:
			// preserve the native question for genuine reference ambiguity.
			return ""
		}
		return "Horizon belum dapat menyimpulkan jawaban. /diagnosis untuk rinciannya."
	default:
		return "Pembacaan Brain belum menghasilkan jawaban yang dapat digunakan. /diagnosis untuk rinciannya."
	}
}

// Run is a line-oriented terminal UI: no raw-mode changes, mouse, screen
// replacement, Python dependency, network listener or remote chat service.
func Run(session *Session, input io.Reader, output io.Writer, options Options) error {
	cyan, bold, dim, reset := "", "", "", ""
	if options.Color {
		cyan = "\x1b[36m"
		bold = "\x1b[1m"
		dim = "\x1b[2m"
		reset = "\x1b[0m"
	}
	width := options.Width
	if width < 28 {
		width = 40
	}
	if width > 72 {
		width = 72
	}
	line := strings.Repeat("─", width)
	system := func(text string) { fmt.Fprintf(output, "%sSistem%s  %s\n", dim, reset, cleanText(text)) }
	room := func() {
		state := func(values []float64) string {
			lamp, fan, door := "mati", "mati", "tertutup"
			if values[0] > 0 {
				lamp = "nyala"
			}
			if values[1] > 0 {
				fan = "nyala"
			}
			if values[2] > 0 {
				door = "terbuka"
			}
			return fmt.Sprintf("Lampu %s · Kipas %s · Pintu %s", lamp, fan, door)
		}
		fmt.Fprintf(output, "\n%sRUANG SIMULASI%s\n  Sekarang: %s\n  Tadi:     %s\n\n", bold, reset, state(session.state.Room.Current), state(session.state.Room.Previous))
	}
	help := func() {
		fmt.Fprintln(output, "\nPerintah:")
		fmt.Fprintln(output, "  /brain                   Periksa Brain dan sumber hasil latihan")
		fmt.Fprintln(output, "  /kemampuan               Lihat kemampuan yang sudah diverifikasi")
		fmt.Fprintln(output, "  /diagnosis               Periksa pembacaan untuk pesan terakhir")
		fmt.Fprintln(output, "  /riwayat                 Lihat 16 pesan konteks terakhir")
		fmt.Fprintln(output, "  /baru                    Mulai konteks percakapan baru")
		fmt.Fprintln(output, "  /lab                     Buka pengamatan simulasi bila diperlukan")
		fmt.Fprintln(output, "  /keluar                  Tutup Horizon")
		fmt.Fprintln(output)
	}
	brain := func() {
		info := session.BrainInfo()
		fmt.Fprintf(output, "\nBRAIN AKTIF\n  Identitas: %s\n  Revisi: %d\n  Profil: %s\n", cleanText(info.Identity), info.Sequence, cleanText(info.Directory))
		fmt.Fprintf(output, "  Sidik berkas dimuat: sha256:%s\n  Sidik keadaan neural: %s\n", info.SourceFingerprint, info.CanonicalStateHash)
		fmt.Fprintf(output, "  Komposisi: %s\n  Unit: %d · Bentuk masukan: %d · Bentuk keluaran: %d\n", cleanText(info.Composition), info.Units, info.InputForms, info.OutputForms)
		if info.MatchesBundle {
			system("Berkas cocok dengan paket hasil latihan " + info.BundleTrainedOn + " yang disertakan versi program ini.")
		} else {
			system("Profil aktif berbeda dari paket bawaan. Riwayat pelatihannya belum diverifikasi oleh konsol ini.")
			system("Untuk memilih paket yang disertakan kode ini: keluar, lalu jalankan bash scripts/horizon-chat.sh -latest. Profil lama tetap tersedia melalui -state.")
		}
		system("Mengobrol memakai pembacaan Brain aktif. Penyimpanan konteks percakapan tidak melatih sinaps.")
	}
	capabilities := func() {
		fmt.Fprintln(output, "\nKEMAMPUAN KOMUNIKASI")
		if session.BrainInfo().MatchesBundle {
			fmt.Fprintln(output, "  Cakupan yang diuji pada paket hasil latihan ini:")
			for _, skill := range session.release.Communication.VerifiedSkills {
				fmt.Fprintln(output, "  · "+cleanText(skill))
			}
			for _, example := range session.release.Communication.Examples {
				fmt.Fprintln(output, "  Contoh: "+cleanText(example))
			}
			for _, limit := range session.release.Communication.Limits {
				fmt.Fprintln(output, "  Batas: "+cleanText(limit))
			}
		} else {
			fmt.Fprintln(output, "  Belum ada daftar kemampuan yang diverifikasi untuk profil pilihan ini.")
		}
		system("Anda dapat mengirim pesan umum. Kemampuan menjawab bergantung pada hasil latihan yang dimuat dan jalur bahasa yang mendukungnya.")
	}
	diagnosis := func() {
		if session.last == nil {
			system("Belum ada pesan yang dapat diperiksa dalam sesi ini.")
			return
		}
		reply := session.last
		accepted := 0
		for _, trace := range reply.Trace {
			if trace.Readout.Accepted && trace.Readout.NResolution != nil {
				accepted++
			}
		}
		fmt.Fprintf(output, "\nDIAGNOSIS PESAN\n  Pesan: %s\n  Status native: %s\n  Alasan native: %s\n", cleanText(session.lastText), cleanText(reply.Status), cleanText(reply.Reason))
		fmt.Fprintf(output, "  Makna diterima: %t\n  Revisi Brain: %d\n  Pembacaan N diterima: %d/%d\n  Kata dari ejaan sinaptik: %d\n", reply.MeaningValid, session.BrainInfo().Sequence, accepted, len(reply.Trace), len(reply.Words))
		if reply.Text != "" {
			fmt.Fprintf(output, "  Teks native: %s\n", cleanText(reply.Text))
		}
		system("Teks native di sini dapat berupa klarifikasi umum yang belum sesuai pesan. Diagnosis ini adalah laporan konsol, bukan ucapan Horizon.")
	}
	lab := func() {
		room()
		fmt.Fprintln(output, "Sensor simulasi: /set lampu nyala|mati · /set kipas nyala|mati · /set pintu buka|tutup")
		fmt.Fprintln(output, "Contoh pesan: Bagaimana keadaan lampu? · Bagaimana keadaannya tadi?")
	}
	fmt.Fprintf(output, "\n%s%sHORIZON%s\n%sKomunikasi lokal · Brain aktif%s\n%s\n", bold, cyan, reset, dim, reset, line)
	system(fmt.Sprintf("Brain siap · revisi %d. /brain untuk memeriksa sumber hasil latihan.", session.BrainInfo().Sequence))
	if len(session.state.History) > 0 {
		system(fmt.Sprintf("Melanjutkan %d pesan konteks. /baru untuk memulai lagi.", len(session.state.History)))
	}
	fmt.Fprintln(output, "/bantuan · /brain · /kemampuan · /baru · /keluar")
	scanner := bufio.NewScanner(input)
	scanner.Buffer(make([]byte, 1024), 16<<10)
	for {
		fmt.Fprintf(output, "\n%sKamu ›%s ", cyan, reset)
		if !scanner.Scan() {
			if err := scanner.Err(); err != nil {
				return fmt.Errorf("input terlalu panjang atau tidak terbaca: %w", err)
			}
			fmt.Fprintln(output)
			return nil
		}
		text := strings.TrimSpace(cleanText(scanner.Text()))
		if options.EchoInput {
			fmt.Fprintln(output, text)
		}
		if text == "" {
			continue
		}
		if strings.HasPrefix(text, "/") {
			parts := strings.Fields(strings.ToLower(text))
			switch parts[0] {
			case "/keluar", "/exit", "/quit":
				system("Sesi ditutup. Sampai jumpa.")
				return nil
			case "/bantuan", "/help":
				help()
			case "/brain":
				brain()
			case "/kemampuan":
				capabilities()
			case "/diagnosis":
				diagnosis()
			case "/lab":
				lab()
			case "/ruang":
				room()
			case "/baru":
				if err := session.NewConversation(); err != nil {
					system(err.Error())
				} else {
					system("Konteks percakapan dikosongkan.")
				}
			case "/riwayat":
				if len(session.state.History) == 0 {
					system("Konteks percakapan masih kosong.")
				}
				for i, turn := range session.state.History {
					fmt.Fprintf(output, "  %d. %s\n", i+1, cleanText(turn))
				}
			case "/set":
				if len(parts) != 3 {
					system("Contoh: /set lampu nyala. Lihat /bantuan.")
					continue
				}
				if err := session.SetRoom(parts[1], parts[2]); err != nil {
					system(err.Error())
				} else {
					system("Pengamatan simulasi diperbarui.")
					room()
				}
			default:
				system("Perintah tidak dikenal. Ketik /bantuan.")
			}
			continue
		}
		start := time.Now()
		reply, err := session.Ask(text)
		elapsed := float64(time.Since(start).Microseconds()) / 1000
		if err != nil {
			system(err.Error())
			continue
		}
		if message := rejectionMessage(reply); message != "" {
			system(fmt.Sprintf("%s (%.1f ms)", message, elapsed))
			continue
		}
		if reply.Text == "" {
			system("Horizon belum menghasilkan jawaban untuk pesan ini. Coba /bantuan.")
			continue
		}
		context := ""
		if reply.MeaningValid {
			// This adapter supplies simulated observations, not phone sensors.
			// Keep the evidence source visible even when the lab panel is closed.
			context = " · simulasi"
		}
		fmt.Fprintf(output, "\n%s%sHorizon%s %s· %.1f ms%s%s\n  %s\n", bold, cyan, reset, dim, elapsed, context, reset, cleanText(reply.Text))
	}
}
