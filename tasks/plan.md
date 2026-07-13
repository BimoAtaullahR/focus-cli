# Implementation Plan: Google Calendar (GCal) Integration

## Overview
Integrasi `focus-cli` dengan Google Calendar untuk pencatatan otomatis sesi fokus pomodoro ke Google Calendar (Time Tracking) dan pengimporan daftar tugas (Task Import) dari Google Calendar ke daftar tugas lokal.

## Architecture Decisions
1. **Penyimpanan Kredensial**:
   - Pengguna mendaftarkan aplikasi mereka sendiri di Google Cloud Console dan menaruh file `gcal_credentials.json` di direktori konfigurasi `~/.config/focus-cli/`.
   - File token OAuth2 disimpan secara terpisah di `~/.config/focus-cli/gcal_token.json`.
2. **Koneksi Asinkron (Bubble Tea TUI)**:
   - Bubble Tea bersifat *single-threaded events loop*. Semua request ke Google Calendar API (yang melibatkan I/O jaringan lambat) harus dijalankan sebagai `tea.Cmd` asinkron agar UI tidak membeku (*freeze*).
3. **Pencatatan Kalender Khusus**:
   - Jika kalender khusus (misal "Focus Sessions") belum ada, sistem akan membuatnya secara otomatis di Google Calendar pengguna, lalu menyimpan Calendar ID-nya ke dalam konfigurasi lokal agar tidak perlu dicari berulang kali.

---

## Task List

### Phase 1: Foundation (OAuth2 & Local Storage)

#### Task 1: Setup storage for OAuth2 credentials and token
- **Description**: Menambahkan kolom konfigurasi GCal di struct `Config` dan mengimplementasikan fungsi utility di `storage` untuk membaca `gcal_credentials.json` serta menulis/membaca `gcal_token.json`.
- **Acceptance criteria**:
  - Struct `Config` memiliki field `GCalEnabled` (bool), `GCalCalendarName` (string), dan `GCalCalendarID` (string).
  - Class `Store` di `storage.go` memiliki method `ReadGCalCredentials()`, `SaveGCalToken()`, `LoadGCalToken()`, dan `DeleteGCalToken()`.
- **Verification**:
  - Unit test penyimpanan berhasil membaca/menulis config yang dimodifikasi.
- **Dependencies**: None
- **Files likely touched**:
  - [model.go](file:///home/bimoar/Documents/personal-projects/focus-cli/internal/model/model.go)
  - [storage.go](file:///home/bimoar/Documents/personal-projects/focus-cli/internal/storage/storage.go)
  - [storage_test.go](file:///home/bimoar/Documents/personal-projects/focus-cli/internal/storage/storage_test.go)
- **Estimated scope**: Small (1-2 files)

#### Task 2: Implement OAuth2 login with local redirect server
- **Description**: Membuat modul `gcal` baru dengan mekanisme login OAuth2. Menjalankan local HTTP server sementara pada port `http://localhost:8080/callback` untuk menangkap authorization code dari Google Consent page.
- **Acceptance criteria**:
  - `gcal.NewClient(...)` mengembalikan OAuth2 client yang terhubung dengan Google API.
  - `gcal.Login(ctx)` membuka browser, memutar local server, memproses callback Google, menyimpan token, lalu mematikan server.
- **Verification**:
  - Pemanggilan modul otentikasi berhasil mendeteksi kredensial dan melakukan penukaran token (uji manual/dummy server).
- **Dependencies**: Task 1
- **Files likely touched**:
  - [client.go](file:///home/bimoar/Documents/personal-projects/focus-cli/internal/gcal/client.go) [NEW]
- **Estimated scope**: Medium (2-3 files)

#### Task 3: CLI interface for gcal status, login, logout
- **Description**: Menambahkan subcommand `gcal` ke parser CLI utama (`cli.go`) dengan aksi `login`, `logout`, dan `status`.
- **Acceptance criteria**:
  - `focus gcal login` menjalankan alur otentikasi.
  - `focus gcal logout` menghapus file token secara aman.
  - `focus gcal status` menampilkan apakah token ada, valid, dan mendeteksi ketersediaan file `gcal_credentials.json`.
- **Verification**:
  - CLI berhasil mengenali perintah baru dan menampilkan status/pesan error yang sesuai secara graceful jika kredensial tidak ditemukan.
- **Dependencies**: Task 2
- **Files likely touched**:
  - [cli.go](file:///home/bimoar/Documents/personal-projects/focus-cli/internal/cli/cli.go)
- **Estimated scope**: Small (1-2 files)

### Checkpoint: Foundation
- [ ] Subcommand `focus gcal` terdaftar dan berjalan.
- [ ] Pengguna bisa login dan token tersimpan di `~/.config/focus-cli/gcal_token.json`.
- [ ] Pengguna bisa logout dan status terupdate.

---

### Phase 2: Export Flow (Syncing Focus Sessions to GCal)

#### Task 4: Implement GCal Event Export Service
- **Description**: Implementasi fungsi untuk mengunggah sesi fokus yang selesai ke Google Calendar. Jika kalender khusus belum ada, buat kalender baru di akun pengguna Google.
- **Acceptance criteria**:
  - Fungsi `SyncSessionEvent(title, startTime, endTime)` dapat membuat Google Calendar event bertajuk `Focus: [Task Title]` dengan waktu mulai dan selesai yang presisi.
  - Membuat kalender otomatis jika nama kalender yang dikonfigurasi belum ada di Google Calendar pengguna.
- **Verification**:
  - Unit test/mock memverifikasi objek event yang dikirimkan ke GCal API valid.
- **Dependencies**: Task 2
- **Files likely touched**:
  - [sync.go](file:///home/bimoar/Documents/personal-projects/focus-cli/internal/gcal/sync.go) [NEW]
- **Estimated scope**: Small (1-2 files)

#### Task 5: Auto-export pomodoro sessions on completion
- **Description**: Mengintegrasikan modul `gcal` ke alur engine pomodoro di CLI (`runPomodoro`) dan TUI dashboard agar setelah sesi fokus selesai, event langsung ter-sync asinkron.
- **Acceptance criteria**:
  - Event dikirimkan ke GCal asinkron tanpa memblokir siklus/timer pomodoro.
  - Berhasil terintegrasi baik di mode CLI `focus run` maupun dashboard interaktif TUI.
- **Verification**:
  - Selesaikan sesi fokus pomodoro dan periksa kalender Google (atau mock output log) untuk melihat event baru terbuat.
- **Dependencies**: Task 4
- **Files likely touched**:
  - [cli.go](file:///home/bimoar/Documents/personal-projects/focus-cli/internal/cli/cli.go)
  - [tui.go](file:///home/bimoar/Documents/personal-projects/focus-cli/internal/tui/tui.go)
- **Estimated scope**: Medium (2-3 files)

### Checkpoint: Export Flow
- [ ] Sesi fokus yang diselesaikan via CLI `focus run` otomatis masuk ke Google Calendar.
- [ ] Sesi fokus yang diselesaikan via TUI otomatis masuk ke Google Calendar.
- [ ] Apabila koneksi internet mati/terputus, aplikasi tidak freeze dan tidak crash.

---

### Phase 3: Import Flow (Syncing Tasks from GCal to Focus-cli)

#### Task 6: Implement Task Import from GCal
- **Description**: Mengambil event dari kalender GCal khusus (atau event kalender utama dengan filter kata tertentu, misal prefix `[Focus]`) dan mengubahnya menjadi task lokal `focus-cli`.
- **Acceptance criteria**:
  - Fungsi `ImportTasks()` membaca event kalender (misal: yang aktif untuk hari ini) dan menyimpannya sebagai Task baru jika belum ada di `tasks.json`.
- **Verification**:
  - Unit test memverifikasi event GCal ter-mapping menjadi model Task `focus-cli` dengan benar.
- **Dependencies**: Task 4
- **Files likely touched**:
  - [sync.go](file:///home/bimoar/Documents/personal-projects/focus-cli/internal/gcal/sync.go)
- **Estimated scope**: Small (1-2 files)

#### Task 7: Integrate GCal Sync inside CLI and TUI Refresh
- **Description**: Mengintegrasikan sinkronisasi tugas ke CLI command `focus gcal sync` dan ke tombol refresh `r` pada TUI dashboard secara asinkron.
- **Acceptance criteria**:
  - Perintah `focus gcal sync` berhasil melakukan import task.
  - Menekan tombol `r` di TUI memicu pemanggilan asinkron (`tea.Cmd`) untuk mengimpor tugas dari GCal dan memperbarui tampilan daftar tugas.
- **Verification**:
  - Buat event baru di Google Calendar, tekan `r` di dashboard `focus-cli` TUI, pastikan tugas baru muncul di daftar tugas.
- **Dependencies**: Task 6
- **Files likely touched**:
  - [cli.go](file:///home/bimoar/Documents/personal-projects/focus-cli/internal/cli/cli.go)
  - [tui.go](file:///home/bimoar/Documents/personal-projects/focus-cli/internal/tui/tui.go)
- **Estimated scope**: Medium (2-3 files)

### Checkpoint: Integration Complete
- [ ] Sinkronisasi dua arah (export sesi & import tugas) berjalan lancar.
- [ ] Tampilan status integrasi GCal di TUI dashboard berjalan responsif dan non-blocking.

---

## Risks and Mitigations
| Risk | Impact | Mitigation |
|------|--------|------------|
| Google API Quota limits / client credentials compromised | High | Pengguna membuat kredensial Google API miliknya sendiri melalui panduan di dokumentasi. |
| Koneksi lambat/hang saat sync di TUI | Medium | Menjalankan API request secara asinkron menggunakan goroutine dan `tea.Cmd`. |
| Duplikasi tugas saat import berkali-kali | Medium | Memanfaatkan event ID GCal unik (`GCalEventID`) untuk mencocokkan tugas yang sudah pernah diimpor sebelumnya. |
