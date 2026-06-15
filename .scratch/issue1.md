## Parent

#1

## What to build

Membuat modul `SessionEngine` di `internal/pomodoro` yang mengelola mesin status siklus pomodoro (timer, transisi fase, perhitungan total sesi) secara asinkron. Refactor `internal/cli/cli.go` (khususnya command `run`) agar menggunakan `SessionEngine` menggantikan `pomodoro.Countdown`. CLI akan merender countdown ke stdout dengan mendengarkan callback `OnTick` dari engine. 
Engine ini belum perlu mengelola notifikasi atau progress penyimpanan task, cukup berfokus pada perpindahan siklus waktu.

## Acceptance criteria

- [ ] Struct `SessionEngine` dibuat di `internal/pomodoro/engine.go` (atau file serupa) yang mendukung method Start, Pause, dan Stop.
- [ ] Engine memiliki callback `OnTick` untuk memancarkan sisa waktu setiap detiknya.
- [ ] Command `run` di CLI menggunakan `SessionEngine` dan tidak lagi memanggil `pomodoro.Countdown`.
- [ ] Timer CLI berjalan dengan akurat, mencetak sisa waktu ke stdout.
- [ ] Siklus berpindah dengan benar (Focus -> Short Break -> Focus -> Long Break).

## Blocked by

None - can start immediately
