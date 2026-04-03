package tui

import (
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"focus-cli/internal/model"
	"focus-cli/internal/storage"
)

func newTestModel(t *testing.T) *Model {
	t.Helper()
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())

	s, err := storage.NewStore()
	if err != nil {
		t.Fatalf("NewStore() error = %v", err)
	}

	cfg := model.DefaultConfig()
	cfg.FocusMinutes = 25
	cfg.ShortBreakMinutes = 5
	cfg.LongBreakMinutes = 15
	cfg.LongBreakEvery = 2

	now := time.Now()
	ts := model.TaskStore{
		NextID: 4,
		Tasks: []model.Task{
			{ID: 1, Title: "Task A", TargetSessions: 2, CreatedAt: now, UpdatedAt: now},
			{ID: 2, Title: "Task B", TargetSessions: 2, CreatedAt: now, UpdatedAt: now},
			{ID: 3, Title: "Task C", TargetSessions: 1, CreatedAt: now, UpdatedAt: now},
		},
	}
	if err := s.SaveTasks(ts); err != nil {
		t.Fatalf("SaveTasks() error = %v", err)
	}
	if err := s.SaveConfig(cfg); err != nil {
		t.Fatalf("SaveConfig() error = %v", err)
	}

	return &Model{
		store:   s,
		tasks:   ts,
		config:  cfg,
		history: nil,
		cursor:  1,
		mode:    modeDashboard,
	}
}

func TestMoveTaskPersistsOrder(t *testing.T) {
	m := newTestModel(t)

	m.moveTask(-1)

	if got, want := m.cursor, 0; got != want {
		t.Fatalf("cursor = %d, want %d", got, want)
	}
	if got, want := m.tasks.Tasks[0].ID, 2; got != want {
		t.Fatalf("tasks[0].ID = %d, want %d", got, want)
	}
	if got, want := m.tasks.Tasks[1].ID, 1; got != want {
		t.Fatalf("tasks[1].ID = %d, want %d", got, want)
	}

	reloaded, err := m.store.LoadTasks()
	if err != nil {
		t.Fatalf("LoadTasks() error = %v", err)
	}
	if got, want := reloaded.Tasks[0].ID, 2; got != want {
		t.Fatalf("persisted tasks[0].ID = %d, want %d", got, want)
	}
	if got, want := reloaded.Tasks[1].ID, 1; got != want {
		t.Fatalf("persisted tasks[1].ID = %d, want %d", got, want)
	}
}

func TestPomodoroCycleFlow(t *testing.T) {
	m := newTestModel(t)
	m.cursor = 0

	if _, _ = m.startSelectedCycle(); m.run == nil {
		t.Fatalf("run should be initialized")
	}
	if got, want := m.run.phase, runPhaseFocus; got != want {
		t.Fatalf("phase = %v, want %v", got, want)
	}
	if got, want := m.run.totalSessions, 2; got != want {
		t.Fatalf("totalSessions = %d, want %d", got, want)
	}

	if _, _ = m.finishFocusSession(false); m.run == nil {
		t.Fatalf("run should continue into break")
	}
	if got, want := m.run.phase, runPhaseBreak; got != want {
		t.Fatalf("phase after focus = %v, want %v", got, want)
	}

	if _, _ = m.finishBreakSession(false); m.run == nil {
		t.Fatalf("run should continue into next focus")
	}
	if got, want := m.run.phase, runPhaseFocus; got != want {
		t.Fatalf("phase after break = %v, want %v", got, want)
	}
	if got, want := m.run.sessionIndex, 2; got != want {
		t.Fatalf("sessionIndex = %d, want %d", got, want)
	}

	_, _ = m.finishFocusSession(false)
	if m.run != nil {
		t.Fatalf("run should end after final focus session")
	}

	if got, want := m.tasks.Tasks[0].CompletedPomodoros, 2; got != want {
		t.Fatalf("completed pomodoros = %d, want %d", got, want)
	}
	if got, want := m.tasks.Tasks[0].Done, true; got != want {
		t.Fatalf("task done = %v, want %v", got, want)
	}
	if got, want := len(m.history), 3; got != want {
		t.Fatalf("history length = %d, want %d", got, want)
	}
	if got, want := m.history[0].Type, "focus"; got != want {
		t.Fatalf("history[0].Type = %s, want %s", got, want)
	}
	if got, want := m.history[1].Type, "short_break"; got != want {
		t.Fatalf("history[1].Type = %s, want %s", got, want)
	}
	if got, want := m.history[2].Type, "focus"; got != want {
		t.Fatalf("history[2].Type = %s, want %s", got, want)
	}
}

func TestStartCycleBlockedForDoneTask(t *testing.T) {
	m := newTestModel(t)
	m.cursor = 0
	m.tasks.Tasks[0].Done = true

	_, cmd := m.startSelectedCycle()
	if m.run != nil {
		t.Fatalf("run should not start for completed task")
	}
	if cmd != nil {
		t.Fatalf("cmd should be nil when cycle is blocked")
	}
	if got := m.status; got == "" {
		t.Fatalf("status should contain completion warning")
	}
}

func TestNormalizeKeySpace(t *testing.T) {
	if got, want := normalizeKey(" "), "space"; got != want {
		t.Fatalf("normalizeKey(space) = %q, want %q", got, want)
	}
}

func TestKeyIsMatchesSpaceBinding(t *testing.T) {
	msg := tea.KeyMsg{Type: tea.KeySpace, Runes: []rune{' '}}
	if !keyIs(msg, "space") {
		t.Fatalf("keyIs should match space binding")
	}
}
