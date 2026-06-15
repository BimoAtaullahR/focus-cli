## Parent

#1

## What to build

Menambahkan dukungan *domain events/callbacks* di `SessionEngine` (seperti `OnPhaseComplete` dan `OnSessionWarn`).
Memindahkan logika *update* task (seperti menambahkan `CompletedPomodoros`) dan pengiriman notifikasi dari prosedur sinkron CLI (`cli.go`) ke dalam fungsi pendengar (listener/callback) yang merespons event dari `SessionEngine`.

## Acceptance criteria

- [ ] `SessionEngine` menyediakan callback `OnPhaseComplete` dan `OnSessionWarn`.
- [ ] Saat durasi warning tercapai, engine memicu callback `OnSessionWarn`.
- [ ] CLI (`cli.go`) mengirimkan notifikasi peringatan dan notifikasi selesai melalui *Notification Manager* menggunakan callback engine.
- [ ] Task progress (`CompletedPomodoros`) diperbarui saat `OnPhaseComplete` dipanggil, dan status `Done` otomatis bernilai true jika target tercapai.
- [ ] Riwayat sesi (history) dicatat setiap fase selesai.

## Blocked by

- #2
