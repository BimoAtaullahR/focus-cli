# Panduan Penyiapan Integrasi Google Calendar

Integrasi Google Calendar dengan `focus-cli` memungkinkan Anda untuk:
1. Mengekspor sesi fokus secara otomatis setelah selesai.
2. Mengimpor tugas/acara dari kalender langsung ke daftar tugas lokal CLI & TUI.

Agar `focus-cli` dapat berkomunikasi dengan Google Calendar API menggunakan akun Google Anda sendiri secara aman, Anda perlu membuat kredensial API OAuth2 mandiri di Google Cloud Console. Berikut adalah panduan langkah demi langkahnya:

---

## Langkah 1: Buat Project Baru di Google Cloud

1. Buka [Google Cloud Console](https://console.cloud.google.com/).
2. Masuk menggunakan akun Google Anda.
3. Di pojok kiri atas (sebelah logo Google Cloud), klik dropdown project, lalu pilih **New Project**.
4. Masukkan nama project (misalnya `Focus CLI Calendar`) dan klik **Create**.
5. Tunggu proses pembuatan selesai, lalu pilih project tersebut dari dropdown project.

---

## Langkah 2: Aktifkan Google Calendar API

1. Buka menu navigasi kiri, pilih **APIs & Services** > **Library**.
2. Di kolom pencarian, ketik **Google Calendar API**.
3. Pilih **Google Calendar API** dari hasil pencarian.
4. Klik tombol **Enable** dan tunggu hingga API berhasil diaktifkan.

---

## Langkah 3: Konfigurasi OAuth Consent Screen

Sebelum membuat kredensial, Google mengharuskan Anda menentukan layar persetujuan (Consent Screen) yang akan ditampilkan saat login:

1. Di menu navigasi kiri, pilih **APIs & Services** > **OAuth consent screen**.
2. Pilih User Type: **External** dan klik **Create**.
3. **App Information**:
   * **App name**: `Focus CLI`
   * **User support email**: Pilih email Google Anda.
4. **Developer contact information**:
   * **Email addresses**: Masukkan email Google Anda.
5. Klik **Save and Continue**.
6. **Scopes** (Penting):
   * Klik **Add or Remove Scopes**.
   * Di tabel, cari dan centang scope berikut:
     * `.../auth/calendar` (Melihat, mengedit, membagikan, dan menghapus kalender)
     * `.../auth/calendar.events` (Melihat dan mengedit event di kalender)
   * Klik **Update** di bagian bawah.
   * Klik **Save and Continue**.
7. **Test Users** (Sangat Penting):
   * Karena aplikasi ini berjalan di mode *Testing* (belum diverifikasi oleh Google secara publik), **hanya email yang didaftarkan di sini yang bisa melakukan login**.
   * Klik **Add Users**.
   * Masukkan alamat email Google yang akan Anda gunakan untuk mencatat sesi fokus.
   * Klik **Add** lalu klik **Save and Continue**.
54: 8. Tinjau ringkasan dan periksa status Publishing status.
55: 9. **Ubah Publishing Status ke In Production (Penting)**:
56:    * Secara bawaan, Publishing Status akan bernilai **Testing**. Pada mode *Testing*, Google secara otomatis membatasi umur Refresh Token hanya bertahan selama **7 hari** (setelah 7 hari token expired dan memicu error `invalid_grant`).
57:    * Untuk penggunaan personal agar login berlaku permanen tanpa perlu login ulang tiap minggu, klik tombol **Publish App** pada halaman *OAuth consent screen* dan konfirmasi pemublikasian ke status **In Production**. (Aplikasi personal untuk penggunaan pribadi tidak memerlukan proses verifikasi resmi dari Google).
58: 
59: ---

## Langkah 4: Buat Kredensial OAuth Client ID

1. Di menu navigasi kiri, klik **APIs & Services** > **Credentials**.
2. Di bagian atas, klik **Create Credentials** > **OAuth client ID**.
3. Pilih Application type: **Desktop app**.
4. Berikan nama (misalnya `Focus CLI Desktop Client`).
5. Klik **Create**.
6. Dialog baru akan muncul menampilkan Client ID dan Client Secret. Klik **OK**.
7. Pada daftar *OAuth 2.0 Client IDs*, cari kredensial yang baru saja Anda buat, lalu klik ikon **Download** (panah bawah) di sebelah kanan untuk mengunduh berkas `.json`.

---

## Langkah 5: Hubungkan dengan `focus-cli`

1. Cari file `.json` yang baru saja Anda unduh.
2. Ubah nama file tersebut menjadi `gcal_credentials.json`.
3. Pindahkan file tersebut ke direktori konfigurasi `focus-cli`:
   * **Linux & macOS**: `~/.config/focus-cli/gcal_credentials.json`
   * **Windows**: `%USERPROFILE%\.config\focus-cli\gcal_credentials.json`
4. Jalankan perintah otentikasi di terminal:
   ```bash
   focus gcal login
   ```
5. Browser Anda akan terbuka secara otomatis meminta izin akses. Berikan persetujuan akses.
6. Periksa status integrasi Anda dengan menjalankan:
   ```bash
   focus gcal status
   ```

Setelah langkah di atas selesai, Anda dapat mengaktifkan sinkronisasi otomatis menggunakan perintah:
```bash
focus config set --gcal-enabled on
```

---

## Penyelesaian Masalah (Troubleshooting)

Berikut adalah beberapa kendala umum yang sering terjadi saat melakukan setup integrasi Google Calendar dan cara mengatasinya:

### 1. Error `Access blocked: focus-cli has not completed the Google verification process` (Error 403: access_denied)
* **Penyebab**: Alamat email Google yang Anda gunakan untuk login belum didaftarkan sebagai *Test User*. Karena aplikasi berada dalam status *Testing*, Google membatasi akses login.
* **Solusi**:
  1. Buka [Google Cloud Console](https://console.cloud.google.com/).
  2. Buka menu **APIs & Services** > **OAuth consent screen**.
  3. Gulir ke bagian **Test users** dan klik **Add Users**.
  4. Tambahkan email Google Anda, simpan, lalu coba jalankan `focus gcal login` kembali.

### 2. Error `Google Calendar API has not been used in project [ID] before or it is disabled`
* **Penyebab**: Google Calendar API belum diaktifkan pada proyek GCP Anda.
* **Solusi**:
  1. Buka tautan aktivasi yang tertera di pesan error terminal Anda, atau buka menu **APIs & Services** > **Library** di Google Cloud Console.
  2. Cari **Google Calendar API** dan klik tombol **Enable** (Aktifkan).
  3. Tunggu 1–2 menit agar perubahan diterapkan oleh Google, lalu periksa kembali menggunakan `focus gcal status`.

### 3. Error Izin/Permission saat Sinkronisasi (Gagal membuat/menemukan kalender)
* **Penyebab**: Saat melakukan otentikasi pertama kali di browser, Anda tidak mencentang opsi izin akses penuh ke Google Calendar.
* **Solusi**:
  1. Jalankan kembali perintah `focus gcal login`.
  2. Saat halaman persetujuan Google muncul di browser, pastikan Anda **mencentang semua kotak centang izin** yang diminta (termasuk izin untuk melihat, mengedit, dan mengelola kalender Anda secara permanen). Izin ini aman dan hanya digunakan oleh `focus-cli` untuk membuat kalender khusus bernama `Focus Sessions`.

### 4. Error `oauth2: "invalid_grant" "Token has been expired or revoked."`
* **Penyebab**: Refresh Token Google sudah kadaluwarsa (misalnya karena OAuth consent screen masih di status *Testing* sehingga kedaluwarsa setelah 7 hari), atau akses aplikasi dicabut dari akun Google.
* **Solusi**:
  1. Jalankan perintah otentikasi ulang: `focus gcal login`.
  2. Pastikan status GCP OAuth Consent Screen telah diubah dari **Testing** ke **In Production** (klik **Publish App**) di Google Cloud Console seperti pada Langkah 3 agar token tidak expired lagi setelah 7 hari.
