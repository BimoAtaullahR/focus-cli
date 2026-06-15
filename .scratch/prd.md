## Problem Statement

Aplikasi `focus-cli` saat ini mengalami gesekan arsitektural (architectural friction) akibat logika domain yang dangkal (*shallow*) dan tersebar di lapisan presentasi (CLI dan TUI). Masalah utama meliputi:
1. **Duplikasi Mesin Status Pomodoro**: Logika pergantian sesi (fokus, short break, long break) dan perhitungan waktu ditulis dua kali—sebagai loop *blocking* di CLI dan sebagai loop *tick asinkron* di TUI.
2. **Kebocoran Invarian Task**: TUI dan CLI memanipulasi data *Task* secara langsung (membaca dari storage, memodifikasi angka pomodoro, mengecek penyelesaian, lalu menyimpan kembali).
3. **Pemicu Notifikasi Manual**: Lapisan UI harus menghitung sendiri kapan waktu *warning* tiba dan merakit event notifikasi secara manual.
Kondisi ini menyebabkan kode sulit diuji (sulit membuat *unit test*), rawan bug inkonsistensi antara TUI/CLI, dan sulit dimodifikasi.

## Solution

Melakukan refaktor arsitektur dengan menerapkan tiga modul mendalam (*deep modules*):
1. **Session Engine**: Sebuah modul terpusat yang mengatur status timer pomodoro secara asinkron dan memancarkan *callback/event*.
2. **Task Manager**: Layanan domain yang merangkum semua logika mutasi dan aturan (invariants) tugas.
3. **Event-Driven Notifier**: Integrasi di mana manajer notifikasi bereaksi terhadap kejadian (event) dari *Session Engine* dan *Task Manager* alih-alih dipanggil manual oleh UI.

## User Stories

1. As a CLI user, I want the timer to run consistently, so that I can rely on accurate focus and break periods.
2. As a TUI user, I want the interactive timer to use the same core logic as the CLI, so that behavior is identical and bug-free across both modes.
3. As a user, I want notifications to be triggered reliably when a phase ends or a warning is reached, so that I don't miss transitions.
4. As a user, I want my task progress to be saved accurately when a session completes, so that my productivity stats are always correct.
5. As a maintainer, I want the Pomodoro cycle logic isolated from UI controllers, so that I can easily unit test timer transitions without waiting real time.
6. As a maintainer, I want task state transitions to be encapsulated, so that I don't accidentally corrupt data in the presentation layer.

## Implementation Decisions

- **`SessionEngine` Module**: Akan dibangun di `internal/pomodoro`. Menyediakan *interface* sederhana (`Start()`, `Pause()`, `Stop()`) dan mengekspos *callback* (`OnTick`, `OnPhaseComplete`, `OnSessionWarn`). TUI dan CLI hanya akan memanggil engine ini dan merender tampilannya berdasarkan *callback* yang diterima.
- **`TaskManager` Module**: Akan diimplementasikan untuk membungkus `internal/storage`. Mengekspos fungsi presisi tinggi seperti `CompletePomodoro(taskID int)` yang secara internal memuat data, menambah sesi, mengecek apakah target sesi tercapai (mengubah `Done = true`), menyimpan data, dan mengembalikan status penyelesaian.
- **Event-driven Notifier Hook**: TUI dan CLI tidak akan lagi merakit `NotificationEvent`. `SessionEngine` akan langsung terhubung ke Notifier untuk mengirim peringatan dan info pergantian fase secara otomatis.

## Testing Decisions

- Uji coba (test) akan difokuskan pada pengujian perilaku eksternal (*external behavior*), bukan detail implementasi.
- **SessionEngine**: Akan dibuatkan *unit test* menggunakan manipulasi waktu virtual (*mock clock/ticker*) untuk memvalidasi bahwa transisi dari *Focus* -> *Short Break* -> *Focus* -> *Long Break* berjalan akurat di interval sesi yang benar.
- **TaskManager**: Akan diuji dengan *mock storage* untuk memverifikasi invarian (misalnya: task otomatis menjadi `Done` ketika jumlah pomodoro mencapai target).

## Out of Scope

- Penambahan fitur TUI baru atau perubahan pada antarmuka pengguna secara visual.
- Penambahan penyimpanan database eksternal (tetap menggunakan file JSON lokal yang sudah ada).

## Further Notes

Arsitektur baru ini akan membuat repositori jauh lebih bersahabat dengan AI (*AI-navigable*) karena semua logika kompleks tersembunyi di balik *interface* yang dalam (*deep seams*).
