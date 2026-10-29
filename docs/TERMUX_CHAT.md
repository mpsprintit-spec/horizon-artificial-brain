# Konsol komunikasi Horizon di Termux

Konsol ini adalah tempat mengirim pesan ke Brain Horizon yang sedang dimuat.
Tampilan awal berisi percakapan; rincian kemampuan, diagnosis dan sensor simulasi
dibuka saat diperlukan. Anda boleh mengirim pesan umum. Kemampuan menjawab
mengikuti pengetahuan yang telah dilatih **dan** kemampuan jalur bahasa aktif.
Tampilan komunikasi umum belum berarti Horizon menguasai percakapan umum.

Program berjalan lokal melalui engine dan supervisor kanonis. Jawaban dihasilkan
oleh komposisi numerik serta ejaan sinaptik; tidak memerlukan LLM, API key, Python,
server web atau layanan berbayar. Keterangan bertanda `Sistem` adalah laporan
konsol, terpisah dari teks yang dihasilkan Horizon.

## Pasang dan jalankan

Di Termux, pasang Git dan Go menggunakan pengelola paket resminya:

```bash
pkg update
pkg install git golang
git clone https://github.com/mpsprintit-spec/horizon-artificial-brain.git
cd horizon-artificial-brain
bash scripts/horizon-chat.sh
```

Jika kode sudah ada, jalankan `git switch main` dan `git pull --ff-only` di folder
repository sebelum membuka konsol. Simpan repository dalam direktori pribadi
Termux agar Android mengizinkan eksekusi program. Internet diperlukan untuk
pemasangan dan pembaruan; percakapan berjalan lokal setelah program dibangun.

Warna dapat dimatikan dengan `bash scripts/horizon-chat.sh -no-color` atau
`NO_COLOR`. Tidak perlu browser, root, mouse atau mode terminal khusus.

| Perintah | Fungsi |
| --- | --- |
| `/brain` | Profil yang dimuat, revisi, sidik berkas dan sidik keadaan neural |
| `/kemampuan` | Cakupan pengujian dan batas paket hasil latihan yang cocok |
| `/diagnosis` | Status, alasan, pembacaan N dan teks native untuk pesan terakhir |
| `/riwayat` | Sampai 16 pesan pengguna yang menjadi konteks |
| `/baru` | Kosongkan konteks percakapan dan diagnosis terakhir |
| `/lab` | Buka sensor simulasi dan contohnya bila diperlukan |
| `/bantuan` | Daftar perintah |
| `/keluar` | Tutup konsol dan lepaskan profil |

Pesan yang belum dipahami mendapat pemberitahuan singkat. Konsol tidak terus
mengarahkan percakapan kembali ke latihan lampu. `/diagnosis` tetap memperlihatkan
alasan dan teks native mentah, termasuk klarifikasi yang belum sesuai konteks.
Tidak ada sapaan atau jawaban pengganti yang dibuat atas nama Horizon.

## Memeriksa dan memilih hasil latihan

`/brain` menghitung identitas sumber dari SHA-256 checkpoint, jurnal dan codec
yang dimuat. Sidik keadaan neural dan revisinya dibaca dari Brain aktif. Profil
dicocokkan dengan berkas paket, revisi dan versi adapter; sidik berkas sendiri
tidak membuktikan kemampuan atau kualitas pemahaman.

`/kemampuan` membaca deskripsi pengujian dari manifest paket hanya jika identitas
hasil latihan cocok. Profil berbeda tidak diberi klaim kemampuan paket bawaan.
Deskripsi kemampuan ini tidak memilih jawaban dan tidak berisi parameter belajar.

Profil default tetap `~/.local/share/horizon/chat-v1`. **Mengambil kode terbaru
tidak menimpa profil yang sudah ada.** Untuk memilih hasil latihan yang disertakan
versi kode saat ini, tutup konsol lalu jalankan:

```bash
git pull --ff-only
bash scripts/horizon-chat.sh -latest
```

`-latest` memilih paket dalam program yang baru dibangun, bukan mengunduh model.
Direktori profilnya memakai sidik lengkap tiga berkas hasil latihan di
`~/.local/share/horizon/trained/<sidik>`. Paket dengan berkas berbeda mendapat
direktori berbeda. Profil yang sudah berubah tidak ditimpa atau dilabeli ulang.
Pilihan ini memakai konteks milik profil tersebut; bukan penggabungan pengetahuan
atau konteks semua profil.

Untuk memuat hasil latihan pilihan sendiri yang kompatibel dengan adapter ini:

```bash
bash scripts/horizon-chat.sh -state "$HOME/profil-horizon-saya"
```

Profil harus berisi `language-brain.json`, `events.jsonl` dan `codec.json` yang
cocok. Checkpoint dan jurnal dipulihkan melalui supervisor aktif. `-state` dan
`-latest` tidak boleh digunakan bersamaan. Hasil pelatihan di mesin lain baru
terpakai setelah berkasnya diekspor, tersedia di HP dan dipilih. Hasil eksperimen
yang tidak diekspor atau belum dihubungkan ke jalur bahasa tidak otomatis dapat
ditanyakan melalui chat.

## Kemampuan yang tersedia saat ini

Paket yang diterbitkan sekarang berasal dari Brain baru untuk latihan bahasa
ruangan, bukan gabungan seluruh eksperimen Horizon sebelumnya. Jalur bahasa
yang terhubung saat ini masih `grounded-dialogue-io-v1`: lampu, kipas, pintu,
keadaan sekarang/sebelumnya, rujukan dan satu negasi. Sapaan, definisi benda,
kosakata terbuka dan percakapan umum belum dikuasai. Mengganti tampilan konsol
tidak memperluas keterampilan tersebut.

Untuk memeriksa salah satu hasil latihan dengan pengamatan yang berubah, buka
`/lab`, lalu ketik:

```text
/set lampu nyala
Bagaimana keadaan lampu?
/set lampu mati
Bagaimana keadaannya?
Bagaimana keadaannya tadi?
/diagnosis
```

Brain menghasilkan `Lampu itu sekarang menyala.`, kemudian `Lampu itu sekarang
padam.` dan `Lampu itu tadi menyala.` dari komposisi aktif. `/set` hanya memasok
pengamatan sensor simulasi. Jawaban keadaan diberi penanda `simulasi` di sebelah
waktu respons agar sumber pengamatan tetap terlihat saat panel lab ditutup.
`/ruang` tetap tersedia; pintu diatur dengan
`/set pintu buka|tutup`, kipas dengan `/set kipas nyala|mati`. “Tadi” mengacu pada
pengamatan lengkap sebelum perubahan sensor terakhir. Tidak ada sensor kamera,
penggerakan perangkat fisik atau eksekusi trading pada konsol ini.

## Penyimpanan dan verifikasi

Satu proses memegang satu Brain aktif. Writer kedua pada profil yang sama ditolak.
Profil baru diperiksa ukuran dan SHA-256 sebelum diekstrak; profil lama tidak
diisi ulang diam-diam. Konteks adapter disimpan secara atomik di `chat-session.json`
dan dipulihkan saat restart. Mengobrol tidak melatih sinaps; konteks ini bukan
pengetahuan neural baru. Konteks yang rusak ditolak tanpa reset diam-diam.

```bash
go test ./models/grounded-dialogue-v1 ./services/ai/terminal ./cmd/horizon-chat
go test -race ./services/ai/terminal -run '^Test(Terminal|Communication)'
```

Tes mencakup konsol tanpa layar latihan awal, asal profil, pilihan paket terpisah,
revisi yang dipakai jawaban, diagnosis, konteks, restart dan berkas kanonis yang
tetap identik. CI juga menjalankan evaluasi bahasa serta ablasi terpisah pada
salinan Brain: menghapus kontribusi rujukan merusak pemahaman objek sambil
mempertahankan ejaan; menghapus sinaps ejaan menghilangkan teks jawaban. Ini bukti
ketergantungan pada sinaps untuk tugas terbatas, bukan bukti pemahaman umum.

Pengujian Linux dan build Android arm64 tidak menggantikan pengujian pada
perangkat Android/Termux fisik.

Referensi pengelola paket: [dokumentasi resmi Termux](https://github.com/termux/termux-tools/blob/master/doc/termux.1.md.in).
