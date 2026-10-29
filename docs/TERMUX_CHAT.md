# Mengobrol dengan Horizon di Termux

UI ini berjalan langsung di terminal Android. Brain hasil pelatihan V6 dibawa
di dalam program, dimuat sekali dan digunakan untuk setiap pesan. Tidak ada
LLM, kunci API, server web, Python, atau layanan berbayar yang diperlukan.
Internet diperlukan saat mengambil kode dan memasang Go; percakapan berjalan
lokal setelah program selesai dibangun.

## Pasang dan jalankan

Di Termux, pasang Git dan Go dengan pengelola paket resminya:

```bash
pkg update
pkg install git golang
git clone https://github.com/mpsprintit-spec/horizon-artificial-brain.git
cd horizon-artificial-brain
bash scripts/horizon-chat.sh
```

Jika repository sudah ada, masuk ke foldernya, lalu jalankan `git pull --ff-only`
dan `bash scripts/horizon-chat.sh`. Launcher membangun versi kode yang ada;
kompilasi pertama dapat lebih lama di HP. Simpan repository di direktori pribadi
Termux, bukan penyimpanan bersama Android yang dapat melarang eksekusi program.

Layar menampilkan nama Horizon, cakupan kemampuan, keadaan ruangan simulasi,
prompt `Kamu ›`, jawaban native, dan waktu respons dalam milidetik. Semua warna
bisa dimatikan dengan `bash scripts/horizon-chat.sh -no-color` atau `NO_COLOR`.
Tidak perlu browser, mouse, root, atau pengubahan terminal ke raw mode.

## Coba percakapan pertama

Ketik satu baris setiap kali. Perintah yang diawali `/` mengatur UI dan sensor
simulasi; pertanyaan biasa dikirim ke Brain Horizon.

```text
/set lampu nyala
Bagaimana keadaan lampu?
/set lampu mati
Bagaimana keadaannya?
Bagaimana keadaannya tadi?
```

Untuk contoh ini, Brain terlatih menghasilkan:

```text
Lampu itu sekarang menyala.
Lampu itu sekarang padam.
Lampu itu tadi menyala.
```

Jawaban tersebut dihitung oleh komposisi dan ejaan sinaptik aktif. UI tidak
memilih jawaban dari tabel contoh. Pengaturan ruangan hanya memasok pengamatan
yang diperlukan agar ucapan mempunyai konteks.

| Perintah | Fungsi |
| --- | --- |
| `/bantuan` | Perintah dan contoh pertanyaan |
| `/ruang` | Keadaan sekarang dan pengamatan sebelumnya |
| `/set lampu nyala` atau `/set lampu mati` | Ubah lampu simulasi |
| `/set kipas nyala` atau `/set kipas mati` | Ubah kipas simulasi |
| `/set pintu buka` atau `/set pintu tutup` | Ubah pintu simulasi |
| `/riwayat` | Tampilkan sampai 16 pesan pengguna terakhir |
| `/baru` | Kosongkan konteks percakapan, pertahankan ruangan dan Brain |
| `/keluar` | Tutup sesi dan lepaskan profil |

## Kemampuan saat ini

Bahasa terlatih masih terbatas pada lampu, kipas, pintu, rujukan percakapan,
negasi, serta keadaan sekarang/sebelumnya. Ini belum chatbot percakapan umum.
Sapaan, definisi benda dan penyebutan nama benda saja belum dilatih. UI
membedakan kata yang belum dipelajari dari gabungan kata yang belum didukung:

| Pesan | Perilaku saat ini |
| --- | --- |
| `hallo` | Sistem menjelaskan bahwa kosakata pesan belum seluruhnya dipelajari |
| `apa itu lampu?` | Kata `apa` belum dipelajari; pertanyaan definisi belum didukung |
| `lampu?` | Nama lampu dikenal, tetapi pesan belum membentuk pertanyaan keadaan yang didukung |
| `Bagaimana keadaan lampu?` | Brain menjawab berdasarkan keadaan ruangan |
| `Bagaimana keadaannya?` tanpa rujukan | Brain meminta klarifikasi objek |

Diagnosis ditandai `Sistem` dan tetap menampilkan waktu pembacaan Brain.
Penjelasan ini berasal dari alasan penolakan native, bukan jawaban bahasa yang
dipelajari. Brain masih dapat menghasilkan klarifikasi umum `Objek yang mana?`
untuk kata asing melalui API langsung; UI tidak menampilkannya sebagai jawaban
atas sapaan atau pertanyaan definisi. Perbaikan tampilan ini tidak menambah
kosakata, kemampuan percakapan atau pengetahuan definisi pada sinaps. Tidak ada
jawaban pengganti atas nama Horizon.

Ruangan adalah simulasi yang diatur pengguna, bukan hasil kamera HP. “Tadi”
mengacu pada pengamatan lengkap sebelum perintah `/set` terakhir. Mengobrol
menyimpan konteks tetapi tidak melatih ulang sinaps atau menciptakan pengetahuan
baru secara otomatis. Tidak ada kendali perangkat fisik atau eksekusi trading.

## Profil dan penyimpanan

Profil khusus chat dibuat saat pertama dijalankan di
`~/.local/share/horizon/chat-v1`. Aset bawaan diperiksa ukuran dan SHA-256 sebelum
diekstrak. Profil yang sudah ada tidak ditimpa. Checkpoint dan jurnal dipulihkan
melalui supervisor kanonis yang sama dengan engine utama; satu proses memiliki
satu Brain aktif. Proses kedua dengan profil yang sama ditolak.

Konteks UI disimpan lokal di `chat-session.json` secara atomik, terpisah dari
checkpoint pengetahuan. Sesi memuat kembali keadaan ruangan dan 16 pesan terakhir
pada restart. Ini konteks adapter, bukan klaim memori bahasa jangka panjang.
File konteks yang rusak ditolak dengan pesan kesalahan, tanpa reset diam-diam.

Untuk memilih direktori profil baru:

```bash
bash scripts/horizon-chat.sh -state "$HOME/horizon-chat-kedua"
```

Gunakan direktori yang belum ada untuk profil baru. File Brain utama seperti
`brain_memory.json` tidak dipilih atau ditimpa otomatis. UI menggunakan pembacaan
native tanpa siklus pembelajaran latar belakang. Untuk aplikasi atau pelatihan
lanjutan, gunakan API engine dan prosedur jurnal, bukan mengedit file sinaps.

## Verifikasi pengembangan

```bash
go test ./models/grounded-dialogue-v1 ./services/ai/terminal ./cmd/horizon-chat
go test -race ./services/ai/terminal -run '^TestTerminal(TranscriptCommandsAndPlainOutput|ReportedUnsupportedConversation)$'
```

Tes memakai Brain asli yang disertakan: pertanyaan dan rujukan, perubahan
pengamatan sekarang/tadi, restart, batas riwayat, penolakan writer kedua,
penolakan konteks rusak, output terminal, serta checkpoint/jurnal yang tetap
identik setelah percakapan. Pengujian Linux dan kompilasi lintas target tidak
menggantikan pengujian pada perangkat Android/Termux fisik.

Tes regresi juga mengulang percakapan awal pengguna (`hallo`, `apa itu lampu?`,
`lampu?`) untuk memastikan penolakan kosakata/bentuk pertanyaan tidak ditampilkan
sebagai kehilangan rujukan objek. Pertanyaan keadaan terlatih tetap dijawab
oleh Brain, dan baris kosong diabaikan.

Referensi pengelola paket: [dokumentasi resmi Termux](https://github.com/termux/termux-tools/blob/master/doc/termux.1.md.in).
