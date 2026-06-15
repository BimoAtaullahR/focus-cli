package tui

import (
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"focus-cli/internal/model"
	"focus-cli/internal/pomodoro"
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

func TestEditTaskUpdatesCompletedAndDone(t *testing.T) {
	m := newTestModel(t)
	err := m.editTask(1, "Task A Updated", "desc", 3, 3)
	if err != nil {
		t.Fatalf("editTask() error = %v", err)
	}

	if got, want := m.tasks.Tasks[0].CompletedPomodoros, 3; got != want {
		t.Fatalf("completed = %d, want %d", got, want)
	}
	if got, want := m.tasks.Tasks[0].Done, true; got != want {
		t.Fatalf("done = %v, want %v", got, want)
	}

	err = m.editTask(1, "Task A Updated", "desc", 5, 2)
	if err != nil {
		t.Fatalf("editTask() error = %v", err)
	}
	if got, want := m.tasks.Tasks[0].Done, false; got != want {
		t.Fatalf("done after lowering completed = %v, want %v", got, want)
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

func TestTUI_EngineTickUpdatesView(t *testing.T) {
	cfg := model.DefaultConfig()
	cfg.Keys = model.DefaultKeys()
	m := &Model{
		config: cfg,
		run: &runState{
			phase:         runPhaseFocus,
			label:         "FOCUS",
			remaining:     60 * time.Second,
			phaseDuration: 60 * time.Second,
			taskID:        1,
			sessionIndex:  1,
			totalSessions: 1,
		},
		mode:  modeRunning,
		ready: true,
	}

	// We'll define engineTickMsg as pomodoro.EngineState internally.
	// But let's create it.
	state := pomodoro.EngineState{
		Phase:         pomodoro.PhaseFocus,
		Remaining:     59 * time.Second,
		SessionCount:  1,
		TotalSessions: 1,
		IsRunning:     true,
	}
	msg := engineTickMsg(state)
	
	newModel, _ := m.Update(msg)
	
	viewStr := newModel.View()
	
	if !strings.Contains(viewStr, "00:59") {
		t.Errorf("expected view to contain 00:59, got:\n%s", viewStr)
	}
}

func TestTUI_EnginePhaseCompleteUpdatesTaskProgress(t *testing.T) {
	m := newTestModel(t)
	m.run = &runState{
		taskID: 1, // Task A
	}

	msg := enginePhaseCompleteMsg{
		Phase:        pomodoro.PhaseFocus,
		SessionCount: 1,
		StartedAt:    time.Now().Add(-25 * time.Minute),
		EndedAt:      time.Now(),
		Completed:    true,
	}

	_, _ = m.Update(msg)

	// Verify task progress is updated
	if m.tasks.Tasks[0].CompletedPomodoros != 1 {
		t.Errorf("expected CompletedPomodoros=1, got %d", m.tasks.Tasks[0].CompletedPomodoros)
	}
	// Verify history is saved
	if len(m.history) != 1 {
		t.Errorf("expected history length=1, got %d", len(m.history))
	}
	// Verify it was saved to storage
	reloaded, _ := m.store.LoadTasks()
	if reloaded.Tasks[0].CompletedPomodoros != 1 {
		t.Errorf("expected persisted CompletedPomodoros=1, got %d", reloaded.Tasks[0].CompletedPomodoros)
	}
}

func TestTUI_BeginFocusCycleStartsEngine(t *testing.T) {
	m := newTestModel(t)
	m.cursor = 0

	_, cmd := m.beginFocusCycle(1, 2)
	if m.engine == nil {
		t.Fatal("engine should be initialized")
	}
	if m.engineChan == nil {
		t.Fatal("engineChan should be initialized")
	}
	if cmd == nil {
		t.Fatal("expected waitForEngineMsg cmd to be returned")
	}
	m.engineCancel() // cleanup
}

func TestSaveCurrentRunProgress(t *testing.T) {
	m := newTestModel(t)
	m.cursor = 0

	_, _ = m.startSelectedCycle()
	if m.run == nil {
		t.Fatalf("run should be initialized")
	}
	m.run.remaining = 13*time.Minute + 12*time.Second
	m.saveCurrentRunProgress()

	reloaded, err := m.store.LoadTasks()
	if err != nil {
		t.Fatalf("LoadTasks() error = %v", err)
	}
	if got, want := reloaded.Tasks[0].TimerPhase, "focus"; got != want {
		t.Fatalf("timer phase = %s, want %s", got, want)
	}
	if got, want := reloaded.Tasks[0].TimerRemainingSec, 13*60+12; got != want {
		t.Fatalf("timer remaining sec = %d, want %d", got, want)
	}
}

func TestStartSelectedCycleResumesFromSavedProgress(t *testing.T) {
	m := newTestModel(t)
	m.cursor = 0
	m.tasks.Tasks[0].TimerPhase = "focus"
	m.tasks.Tasks[0].TimerRemainingSec = 8*60 + 30
	m.tasks.Tasks[0].TimerSessionIndex = 1
	m.tasks.Tasks[0].TimerTotalSessions = 2

	_, _ = m.startSelectedCycle()
	if m.run == nil {
		t.Fatalf("run should be initialized from saved progress")
	}
	if m.engine == nil {
		t.Fatalf("engine should be initialized when resuming from saved progress")
	}
	if got, want := int(m.run.remaining/time.Second), 8*60+30; got != want {
		t.Fatalf("remaining sec = %d, want %d", got, want)
	}
	if got, want := m.run.sessionIndex, 1; got != want {
		t.Fatalf("session index = %d, want %d", got, want)
	}
	if got, want := m.run.totalSessions, 2; got != want {
		t.Fatalf("total sessions = %d, want %d", got, want)
	}
}

func TestResumedCycleCanBePausedAndResumed(t *testing.T) {
	m := newTestModel(t)
	m.cursor = 0
	m.tasks.Tasks[0].TimerPhase = "focus"
	m.tasks.Tasks[0].TimerRemainingSec = 8*60 + 30
	m.tasks.Tasks[0].TimerSessionIndex = 1
	m.tasks.Tasks[0].TimerTotalSessions = 2

	_, _ = m.startSelectedCycle()
	if m.run == nil {
		t.Fatalf("run should be initialized")
	}

	// Default pause key is "p".
	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'p'}}
	model, _ := m.Update(msg)
	updatedModel := model.(*Model)

	if !updatedModel.run.paused {
		t.Fatalf("expected cycle to be paused")
	}

	// Press pause key again to resume
	model, _ = updatedModel.Update(msg)
	updatedModel = model.(*Model)

	if updatedModel.run.paused {
		t.Fatalf("expected cycle to be resumed")
	}
}

func TestTaskResumeLabel(t *testing.T) {
	task := model.Task{TimerPhase: "focus", TimerRemainingSec: 510}
	if got, want := taskResumeLabel(task), "resume 08:30 focus"; got != want {
		t.Fatalf("taskResumeLabel() = %q, want %q", got, want)
	}
	if got := taskResumeLabel(model.Task{}); got != "" {
		t.Fatalf("taskResumeLabel(empty) = %q, want empty", got)
	}
}
