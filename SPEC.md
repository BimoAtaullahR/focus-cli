# Spec: Google Calendar (GCal) Integration

## Objective
Mengintegrasikan `focus-cli` dengan Google Calendar untuk mempermudah manajemen waktu dan tracking aktivitas fokus pengguna. Fitur ini ditargetkan bagi developer atau power user yang menggunakan Google Calendar untuk time-blocking harian.

### User Stories / Fitur Utama:
1. **Otentikasi OAuth2**: Pengguna dapat masuk (`login`) dan keluar (`logout`) dari Google Calendar langsung dari CLI.
2. **Auto-Sync Completed Sessions (Time Tracking)**: Setiap kali sesi fokus pomodoro selesai, aplikasi secara otomatis mencatatnya sebagai event di Google Calendar (misal: `Focus: [Judul Task]`).
3. **Import Tasks dari Google Calendar**: Pengguna dapat mengambil daftar event dari kalender tertentu (misal: kalender "Focus Tasks" atau event dengan prefix `[Focus]`) dan mengubahnya menjadi task lokal di `focus-cli`.
4. **Offline Resilience**: Jika tidak ada koneksi internet, aplikasi tidak boleh crash dan harus terus berfungsi secara lokal seperti biasa, dengan opsi sync ulang saat online.

---

## Tech Stack
- **Language**: Go (v1.23)
- **Key Dependencies**:
  - `golang.org/x/oauth2` (versi terbaru) untuk menangani alur OAuth2.
  - `google.golang.org/api/calendar/v3` (versi terbaru) untuk berinteraksi dengan Google Calendar API.
  - `google.golang.org/api/option` untuk konfigurasi klien.

---

## Commands
Akan ditambahkan sub-command baru `gcal`:

```bash
# Melakukan otentikasi Google OAuth2 via local redirect server
focus gcal login

# Menghapus token otentikasi lokal
focus gcal logout

# Menampilkan status koneksi dan informasi akun GCal yang terhubung
focus gcal status

# Melakukan sinkronisasi manual (import task dari GCal dan upload history jika ada antrean)
focus gcal sync [--import-only | --export-only]
```

Dan penambahan flags pada command `config`:
```bash
# Mengatur konfigurasi integrasi GCal
focus config set --gcal-enabled on|off --gcal-calendar-name "Focus Sessions"
```

---

## Project Structure
Modifikasi struktur direktori untuk menambahkan modul baru `gcal`:

```
internal/
├── gcal/
│   ├── client.go       # Logika klien GCal, otentikasi OAuth2, dan token storage
│   ├── sync.go         # Logika sinkronisasi task (import) & sesi fokus (export)
│   └── client_test.go  # Unit testing untuk modul gcal
```

- Token OAuth2 yang berhasil didapatkan akan disimpan di `~/.config/focus-cli/gcal_token.json`.
- Pengguna dapat meletakkan file kredensial Google App mereka sendiri di `~/.config/focus-cli/gcal_credentials.json` jika ingin menggunakan Client ID milik mereka sendiri.

---

## Code Style
Kami mengikuti standar penulisan kode Go idiomatik yang sudah ada di proyek ini. Contoh struktur untuk modul asinkron GCal yang digunakan di Bubble Tea:

```go
package gcal

import (
	"context"
	"time"
	"google.golang.org/api/calendar/v3"
)

type GCalClient struct {
	service *calendar.Service
}

// SyncSession mencatat sesi fokus ke Google Calendar
func (c *GCalClient) SyncSession(ctx context.Context, title string, startTime, endTime time.Time, calendarName string) error {
	event := &calendar.Event{
		Summary:     "Focus: " + title,
		Description: "Sesi fokus pomodoro yang diselesaikan menggunakan focus-cli",
		Start: &calendar.EventDateTime{
			DateTime: startTime.Format(time.RFC3339),
			TimeZone: "Local",
		},
		End: &calendar.EventDateTime{
			DateTime: endTime.Format(time.RFC3339),
			TimeZone: "Local",
		},
	}
	
	// Logika memasukkan event ke kalender tertentu
	// ...
	return nil
}
```

---

## Testing Strategy
- **Unit Testing**:
  - Mocking Google Calendar API service untuk memverifikasi logika import/export di `internal/gcal/client_test.go`.
  - Pengetesan parsing format tanggal dan sinkronisasi task lokal.
- **Manual Verification**:
  - Menjalankan flow `focus gcal login` di environment lokal dan memverifikasi token berhasil disimpan.
  - Menyelesaikan satu sesi fokus pomodoro dan memverifikasi event tercatat di Google Calendar target.

---

## Boundaries
- **Always**:
  - Lakukan operasi jaringan (network call) Google API secara asinkron (`tea.Cmd`) di dalam TUI agar antarmuka tidak membeku.
  - Tangani token kedaluwarsa secara otomatis menggunakan `oauth2.TokenSource`.
- **Ask first**:
  - Menambahkan dependency eksternal baru (`google-api-go-client`).
- **Never**:
  - Menyimpan Client Secret Google bawaan di tempat publik tanpa enkripsi (atau sebaiknya membiarkan pengguna menyediakan `gcal_credentials.json` mereka sendiri).

---

## Success Criteria
- [ ] Pengguna bisa login menggunakan browser via local callback server `http://localhost:8080/callback`.
- [ ] Setiap kali sesi fokus selesai, event tercatat di Google Calendar dalam waktu kurang dari 5 detik (apabila koneksi lancar).
- [ ] Task yang diimport dari Google Calendar masuk ke daftar task lokal di `focus-cli`.
- [ ] Aplikasi tetap berjalan lancar tanpa internet (tidak crash, melainkan menampilkan warning status offline di TUI).

---

## Open Questions

> [!IMPORTANT]
> **Keputusan Desain yang Perlu Disepakati:**
> 1. **Client Credentials**: Apakah kita akan menyertakan Client ID & Secret bawaan di dalam aplikasi `focus-cli` (default) dengan risiko quota limits dan security audit Google, ATAU mewajibkan pengguna membuat Client ID mereka sendiri di Google Cloud Console dan memasukkannya lewat `gcal_credentials.json`?
>    *Rekomendasi:* Sediakan instruksi bagi pengguna untuk membuat `gcal_credentials.json` mereka sendiri, namun kita bisa menyediakan kredensial default bawaan opsional jika ingin mempermudah onboarding pengguna pertama kali.
>
> 2. **Calendar Selection**: Apakah kita akan mencatat sesi fokus di kalender utama (Primary Calendar) pengguna, atau membuat kalender khusus secara otomatis bernama "Focus Sessions"?
>    *Rekomendasi:* Buat kalender baru bernama "Focus Sessions" secara otomatis agar tidak mengotori kalender utama pengguna, namun berikan opsi konfigurasi bagi pengguna untuk memilih kalender tujuan.
>
> 3. **Sync Strategy (Dua Arah vs Satu Arah)**:
>    *   **Focus -> GCal**: Export otomatis sesi fokus yang selesai (selalu sinkron).
>    *   **GCal -> Focus**: Apakah import task dari GCal bersifat otomatis secara berkala (background polling) atau hanya ketika pengguna menjalankan command/menekan tombol refresh `r` di TUI?
>    *   *Rekomendasi:* Import dilakukan secara manual saat menekan refresh `r` atau menjalankan command `focus gcal sync`, agar performa TUI tetap optimal dan tidak menghabiskan kuota API secara sia-sia.
