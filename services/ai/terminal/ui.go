package terminal

import (
	"bufio"
	"fmt"
	"io"
	"strings"
	"time"
)

type Options struct {
	Color     bool
	EchoInput bool // Piped transcripts have no terminal echo.
	Width     int
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
		fmt.Fprintln(output, "  /ruang                   Lihat pengamatan ruangan")
		fmt.Fprintln(output, "  /set lampu nyala         Lampu: nyala atau mati")
		fmt.Fprintln(output, "  /set kipas mati          Kipas: nyala atau mati")
		fmt.Fprintln(output, "  /set pintu buka          Pintu: buka atau tutup")
		fmt.Fprintln(output, "  /riwayat                 Lihat 16 pesan konteks terakhir")
		fmt.Fprintln(output, "  /baru                    Mulai konteks percakapan baru")
		fmt.Fprintln(output, "  /keluar                  Tutup Horizon")
		fmt.Fprintln(output, "\nContoh pertanyaan:")
		fmt.Fprintln(output, "  Bagaimana keadaan lampu?")
		fmt.Fprintln(output, "  Bagaimana keadaannya?")
		fmt.Fprintln(output, "  Apakah lampu masih nyala sekarang?")
		fmt.Fprintln(output, "  Bagaimana keadaan pintu tadi?")
		fmt.Fprintln(output)
	}
	fmt.Fprintf(output, "\n%s%sHORIZON%s\n%sPercakapan neural · lokal · tanpa LLM%s\n%s\n", bold, cyan, reset, dim, reset, line)
	system("Bahasa terlatih: keadaan lampu, kipas dan pintu. Belum percakapan umum.")
	system("Konteks sesi tersimpan lokal; mengobrol tidak melatih ulang Brain.")
	if len(session.state.History) > 0 {
		system(fmt.Sprintf("Melanjutkan %d pesan konteks. /baru untuk memulai lagi.", len(session.state.History)))
	}
	room()
	fmt.Fprintln(output, "/bantuan · /ruang · /baru · /keluar")
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
			case "/ruang":
				room()
			case "/baru":
				if err := session.NewConversation(); err != nil {
					system(err.Error())
				} else {
					system("Konteks percakapan dikosongkan; keadaan ruangan tetap.")
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
		if reply.Text == "" {
			system("Horizon belum menghasilkan jawaban untuk pesan ini. Coba /bantuan.")
			continue
		}
		fmt.Fprintf(output, "\n%s%sHorizon%s %s· %.1f ms%s\n  %s\n", bold, cyan, reset, dim, elapsed, reset, cleanText(reply.Text))
		if !reply.MeaningValid && reply.Reason != "" {
			system("Jawaban belum menyatakan keadaan objek. Gunakan nama objek dan kosakata terlatih.")
		}
	}
}
