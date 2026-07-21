# Implementation Plan - GCal Token Refresh Auto-Save, Friendly Error Handling & GCP App Publishing Docs (Issue #12)

## Overview
Meningkatkan penanganan token Google Calendar di `focus-cli` dengan:
1. **Penyimpanan Otomatis Token Refresh (`persistingTokenSource`)**: Me-refresh dan menyimpan token baru secara otomatis ke disk (`gcal_token.json`) saat `golang.org/x/oauth2` melakukan perpanjangan token.
2. **Penanganan Error Intuitif (`invalid_grant`)**: Mendeteksi error token kadaluwarsa/dicabut (`invalid_grant` / `Token has been expired or revoked`) dan mengonversinya menjadi pesan error yang ramah pengguna yang menyarankan pengguna untuk me-login kembali (`focus gcal login`).
3. **Pembaruan Dokumentasi**: Memperbarui `docs/gcal-setup.md` dan `README.md` untuk menegaskan pentingnya mengubah GCP OAuth Consent Screen Publishing Status dari "Testing" menjadi "In Production" agar Refresh Token berlaku permanen (tidak hangus setelah 7 hari).

## Architecture Decisions
1. **Custom `oauth2.TokenSource` Wrapper**: Dibuat struct `persistingTokenSource` di `internal/gcal/client.go` yang mengemas `oauth2.TokenSource`. Jika token baru yang valid didapatkan, token tersebut langsung disimpan ke `storage.Store`.
2. **Graceful Error Formatting**: Ditambahkan utilitas `FormatGCalError(err)` untuk membungkus error API GCal sehingga pesan error teknis dari Google diubah menjadi petunjuk aksi yang jelas untuk pengguna CLI dan TUI.

---

## Task List

### Phase 1: Documentation Update
#### Task 1: Update GCP Setup Guide & README
- **Description**: Memperbarui `docs/gcal-setup.md` (Langkah 3 & Troubleshooting) dan `README.md` terkait GCP Publishing Status "In Production" dan penanganan error `invalid_grant`.
- **Acceptance criteria**:
  - `docs/gcal-setup.md` menjelaskan bahwa status *Testing* membatasi umurnya menjadi 7 hari, dan menyarankan klik "Publish App" untuk mengubah ke *In Production*.
  - `docs/gcal-setup.md` menambahkan poin troubleshooting khusus untuk error `invalid_grant`.
  - `README.md` memiliki catatan singkat mengenai GCP status *In Production*.
- **Verification**:
  - Tinjau tampilan markdown file docs.
- **Dependencies**: None
- **Files likely touched**:
  - [gcal-setup.md](file:///home/bimoar/Documents/personal-projects/focus-cli/docs/gcal-setup.md)
  - [README.md](file:///home/bimoar/Documents/personal-projects/focus-cli/README.md)

---

### Phase 2: GCal Package Enhancement (`internal/gcal`)
#### Task 2: Implement Auto-Saving Token Source (`persistingTokenSource`)
- **Description**: Menambahkan struct `persistingTokenSource` di `client.go` untuk menyimpan token baru yang berhasil di-refresh ke disk via `store.SaveGCalToken()`.
- **Acceptance criteria**:
  - Method `GetHTTPClient` di `client.go` menggunakan `persistingTokenSource`.
- **Verification**:
  - Unit test `client_test.go` memverifikasi token baru tersimpan ke store saat `Token()` dipanggil.
- **Dependencies**: None
- **Files likely touched**:
  - [client.go](file:///home/bimoar/Documents/personal-projects/focus-cli/internal/gcal/client.go)
  - [client_test.go](file:///home/bimoar/Documents/personal-projects/focus-cli/internal/gcal/client_test.go)

#### Task 3: Friendly Error Detection & Formatting for `invalid_grant`
- **Description**: Menambahkan helper `IsInvalidGrantError` dan `FormatGCalError` di `client.go`, serta menerapkan pembungkusan error di `sync.go`.
- **Acceptance criteria**:
  - `FormatGCalError` mengonversi error `invalid_grant` menjadi pesan:
    > `"Token Google Calendar telah kadaluwarsa atau dicabut. Silakan jalankan 'focus gcal login' untuk otentikasi ulang akun Anda."`
  - Semua pemanggilan API GCal di `sync.go` menggunakan `FormatGCalError`.
- **Verification**:
  - Unit test `sync_test.go` memverifikasi pembungkusan error `invalid_grant`.
- **Dependencies**: None
- **Files likely touched**:
  - [client.go](file:///home/bimoar/Documents/personal-projects/focus-cli/internal/gcal/client.go)
  - [sync.go](file:///home/bimoar/Documents/personal-projects/focus-cli/internal/gcal/sync.go)
  - [sync_test.go](file:///home/bimoar/Documents/personal-projects/focus-cli/internal/gcal/sync_test.go)

---

### Phase 3: CLI Error Formatting (`internal/cli`)
#### Task 4: Integrate Friendly Error Messages in `focus gcal` CLI
- **Description**: Memastikan perintah `focus gcal status` dan `focus gcal sync` di `cli.go` menampilkan pesan error hasil format `FormatGCalError`.
- **Acceptance criteria**:
  - Perintah CLI `focus gcal sync` dan `status` menampilkan pesan ramah jika token kadaluwarsa.
- **Verification**:
  - Unit test / CLI test di `internal/cli`.
- **Dependencies**: Task 3
- **Files likely touched**:
  - [cli.go](file:///home/bimoar/Documents/personal-projects/focus-cli/internal/cli/cli.go)

---

## Verification Plan

### Automated Tests
- Menjalankan `go test ./...` untuk memastikan seluruh package (`internal/gcal`, `internal/cli`, `internal/storage`, dll.) lulus tanpa error.

### Manual Verification
- Menjalankan `go test -v ./internal/gcal/...` untuk memastikan fungsionalitas auto-save token dan formatting error `invalid_grant`.
