## Parent

#1

## What to build

Me-*refactor* antarmuka terminal interaktif di `internal/tui/tui.go` untuk menghapus duplikasi logika *state machine* Bubble Tea (seperti `tea.Tick` dan pengecekan fase manual). TUI diubah agar menginstansiasi `SessionEngine` dan menerjemahkan *callback* dari engine menjadi `tea.Msg` untuk merender tampilan UI dan memicu notifikasi secara reaktif.

## Acceptance criteria

- [ ] `tui.go` menggunakan `SessionEngine` untuk seluruh manajemen waktu pomodoro.
- [ ] `runTickMsg` dan `tea.Tick` manual dihapus atau diganti dengan pengiriman pesan dari callback `SessionEngine`.
- [ ] UI terupdate secara reaktif sesuai status *remaining time* dan *phase* dari engine.
- [ ] Notifikasi dan pembaruan task di TUI berjalan sesuai dengan alur baru tanpa logika ganda.

## Blocked by

- #3
