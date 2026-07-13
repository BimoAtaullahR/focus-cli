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
8. Tinjau ringkasan dan pastikan status Publishing status adalah **Testing**.

---

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
