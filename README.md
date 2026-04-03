# focus-cli

Pomodoro app berbasis terminal Linux, ditulis dengan Go (hasil dari vibe coding dalam waktu semalam, showings its potential for personal projects).

![alt text](image.png)
![alt text](image-1.png)

## Fitur MVP

- Timer fokus dan break (short/long)
- Menjalankan beberapa sesi pomodoro sekaligus
- CRUD task (add, list, edit, delete)
- Menandai task selesai
- Menyimpan progress sesi pomodoro per task
- Konfigurasi durasi sesi
- Statistik sederhana
- Dashboard interaktif terminal
- Theme preset (sunrise, forest, mono)
- Keybinding customization via config

## Build

```bash
go build -o focus-cli .
```

## Installation

Setelah build, buat symlink ke /usr/local/bin untuk akses global:

```bash
sudo ln -s $(pwd)/focus-cli /usr/local/bin/focus
```

Setelah itu, kamu bisa pakai command `focus` dari mana saja tanpa perlu prefix path.

## Command

```bash
focus help
```

## Mode Interaktif

Jalankan saja:

```bash
focus
```

Ini akan membuka dashboard interaktif dengan:

- navigasi task pakai `↑` dan `↓` (atau `k` dan `j`)
- pindah urutan task pakai `ctrl+j` dan `ctrl+k`
- tambah task dengan `a`
- edit task dengan `e`
- hapus task dengan `d`
- toggle selesai dengan `space`
- mulai pomodoro cycle dengan `enter`
- ubah config dengan `s`
- refresh data dengan `r`
- keluar dengan `q`

Theme bisa diganti dari form config (`s`) atau command CLI.

Saat sesi berjalan:

- `p` atau `space` untuk pause/resume
- `x` untuk mengakhiri sesi sekarang lalu masuk break
- `n` untuk lanjut ke sesi berikutnya setelah break
- `q` untuk menghentikan cycle dan kembali ke dashboard

### Task

```bash
focus task add "Belajar Go" --target 4 --desc "module concurrency"
focus task list
focus task edit 1 --title "Belajar Go Lanjut" --target 6
focus task done 1 true
focus task delete 1
```

### Config

```bash
focus config show
focus config set --focus 25 --short 5 --long 15 --long-every 4 --theme forest
focus config key show
focus config key set nav_up w
focus config key set nav_down s
focus config key set start_cycle enter
```

### Timer

```bash
focus timer --minutes 25 --label "Deep Work"
```

### Jalankan Pomodoro

```bash
focus run --task 1 --sessions 4
```

### Shortcut Commands

```bash
focus focus
focus focus 4 --task 1
focus break
focus break long
focus t 15 Deep Work
focus a "Review PR"
focus ls
focus e 1 --title "Review PR Backend"
focus d 1
focus done 1
focus cfg
focus set focus 30
```

### Stats

```bash
focus stats
```

## Lokasi Data

Data disimpan di:

- `~/.config/focus-cli/tasks.json`
- `~/.config/focus-cli/config.json`
- `~/.config/focus-cli/history.json`

## Catatan

- Hentikan timer dengan `Ctrl+C`.
- Untuk MVP ini, fitur pause/resume belum interaktif penuh; yang ada saat ini adalah start dan interrupt.
