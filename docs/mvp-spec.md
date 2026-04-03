# MVP Spec - focus-cli

## Goal

Menyediakan aplikasi Pomodoro berbasis terminal Linux dengan bahasa Go untuk:

- Menjalankan timer fokus dan break
- Mengelola task (add/edit/delete/done)
- Mengatur target sesi pomodoro

## Command Surface

- `task add <title> [--target N] [--desc text]`
- `task list`
- `task edit <id> [--title text] [--target N] [--desc text]`
- `task delete <id>`
- `task done <id> [true|false]`
- `config show`
- `config set [--focus N] [--short N] [--long N] [--long-every N]`
- `run [--task ID] [--sessions N]`
- `timer [--minutes N] [--label text]`
- `stats`

Short aliases:

- `a`, `ls`, `e`, `d`, `done`
- `focus`, `break`, `t`
- `cfg`, `set`

## Data Model

Task:

- id
- title
- description
- done
- target_sessions
- completed_pomodoros
- created_at
- updated_at

Config:

- focus_minutes
- short_break_minutes
- long_break_minutes
- long_break_every

History:

- started_at
- ended_at
- task_id
- type (focus / short_break / long_break)
- completed

## Persistence

Data disimpan sebagai JSON di `~/.config/focus-cli/`:

- tasks.json
- config.json
- history.json

## MVP Scope Notes

- Timer mendukung start dan stop melalui interrupt (`Ctrl+C`).
- Pause/resume interaktif belum masuk pada MVP ini.
- UI saat ini command-based CLI, belum TUI.
