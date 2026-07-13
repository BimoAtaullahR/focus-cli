# Tasks: Google Calendar Integration

## Phase 1: Foundation (OAuth2 & Local Storage)
- [x] Task 1: Setup storage for OAuth2 credentials and token
  - Acceptance: `model.Config` updated with GCal settings; `Store` has functions to read/write credentials and token.
  - Verify: Run storage tests and ensure model changes do not break loading config.
  - Files:
    - [model.go](file:///home/bimoar/Documents/personal-projects/focus-cli/internal/model/model.go)
    - [storage.go](file:///home/bimoar/Documents/personal-projects/focus-cli/internal/storage/storage.go)
    - [storage_test.go](file:///home/bimoar/Documents/personal-projects/focus-cli/internal/storage/storage_test.go)
- [x] Task 2: Implement OAuth2 login with local redirect server
  - Acceptance: `gcal.NewClient` initializes connection; `gcal.Login(ctx)` launches local server on port 8080/callback and fetches token.
  - Verify: Launch a mock/test function verifying browser opening and loopback listener.
  - Files:
    - [client.go](file:///home/bimoar/Documents/personal-projects/focus-cli/internal/gcal/client.go) [NEW]
- [ ] Task 3: CLI interface for gcal status, login, logout
  - Acceptance: CLI supports `focus gcal login`, `focus gcal logout`, `focus gcal status`.
  - Verify: Test commands in terminal and verify token is created/deleted.
  - Files:
    - [cli.go](file:///home/bimoar/Documents/personal-projects/focus-cli/internal/cli/cli.go)

### Checkpoint 1: Foundation Complete
- [ ] Verification: `focus gcal login` logs in successfully, `focus gcal status` shows correct connected state.

---

## Phase 2: Export Flow (Syncing Focus Sessions to GCal)
- [ ] Task 4: Implement GCal Event Export Service
  - Acceptance: `SyncSessionEvent(title, startTime, endTime)` creates Google Calendar event under "Focus Sessions" calendar.
  - Verify: Run mock unit tests checking GCal payload structure.
  - Files:
    - [sync.go](file:///home/bimoar/Documents/personal-projects/focus-cli/internal/gcal/sync.go) [NEW]
- [ ] Task 5: Auto-export pomodoro sessions on completion
  - Acceptance: Finished focus sessions in CLI and TUI trigger background GCal sync asynchronously.
  - Verify: Start and complete a short pomodoro session, confirm event shows on Google Calendar.
  - Files:
    - [cli.go](file:///home/bimoar/Documents/personal-projects/focus-cli/internal/cli/cli.go)
    - [tui.go](file:///home/bimoar/Documents/personal-projects/focus-cli/internal/tui/tui.go)

### Checkpoint 2: Export Complete
- [ ] Verification: Focus sessions successfully logged to user's Google Calendar from both CLI and TUI.

---

## Phase 3: Import Flow (Syncing Tasks from GCal to Focus-cli)
- [ ] Task 6: Implement Task Import from GCal
  - Acceptance: `ImportTasks()` pulls today's events from the target calendar and maps them to `model.Task`.
  - Verify: Mock calendar events parse correctly into Tasks.
  - Files:
    - [sync.go](file:///home/bimoar/Documents/personal-projects/focus-cli/internal/gcal/sync.go)
- [ ] Task 7: Integrate GCal Sync inside CLI and TUI Refresh
  - Acceptance: `focus gcal sync` command runs; pressing `r` in TUI triggers asynchronous GCal sync.
  - Verify: GCal events are listed as tasks in the CLI/TUI after refresh.
  - Files:
    - [cli.go](file:///home/bimoar/Documents/personal-projects/focus-cli/internal/cli/cli.go)
    - [tui.go](file:///home/bimoar/Documents/personal-projects/focus-cli/internal/tui/tui.go)

### Checkpoint 3: Integration Complete
- [ ] Verification: Comprehensive two-way sync works flawlessly, and dashboard renders sync state without blocking.
