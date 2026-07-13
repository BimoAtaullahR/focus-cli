package pomodoro_test

import (
	"context"
	"testing"
	"time"

	"focus-cli/internal/pomodoro"
)

func TestSessionEngine_TriggersWarning(t *testing.T) {
	cfg := pomodoro.EngineConfig{
		FocusDuration:      10 * time.Millisecond,
		ShortBreakDuration: 5 * time.Millisecond,
		LongBreakDuration:  15 * time.Millisecond,
		LongBreakEvery:     4,
		TargetSessions:     1,
		WarningDuration:    5 * time.Millisecond,
		TickInterval:       1 * time.Millisecond,
	}

	engine := pomodoro.NewSessionEngine(cfg)

	warnCalled := false
	engine.OnSessionWarn = func(state pomodoro.EngineState) {
		warnCalled = true
	}

	complete := make(chan struct{})
	engine.OnComplete = func() {
		close(complete)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	engine.Start(ctx)

	select {
	case <-complete:
	case <-time.After(100 * time.Millisecond):
		t.Fatal("engine did not complete in time")
	}

	if !warnCalled {
		t.Errorf("OnSessionWarn was not called")
	}
}

func TestSessionEngine_PauseResume(t *testing.T) {
	cfg := pomodoro.EngineConfig{
		FocusDuration:      100 * time.Millisecond,
		ShortBreakDuration: 50 * time.Millisecond,
		LongBreakDuration:  150 * time.Millisecond,
		LongBreakEvery:     4,
		TargetSessions:     1,
		TickInterval:       10 * time.Millisecond,
	}

	engine := pomodoro.NewSessionEngine(cfg)

	var tickTimes []time.Time
	engine.OnTick = func(state pomodoro.EngineState) {
		tickTimes = append(tickTimes, time.Now())
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Start the engine
	engine.Start(ctx)
	time.Sleep(25 * time.Millisecond) // Let it tick a few times

	// Pause the engine
	engine.Pause()
	if engine.State().IsRunning {
		t.Fatal("expected engine to not be running after pause")
	}

	// Capture the tick count when paused
	ticksAtPause := len(tickTimes)

	// Wait for a while to ensure no ticks happen during pause
	time.Sleep(30 * time.Millisecond)
	if len(tickTimes) != ticksAtPause {
		t.Fatalf("expected no ticks during pause, got %d more ticks", len(tickTimes)-ticksAtPause)
	}

	// Resume the engine
	engine.Start(ctx)
	if !engine.State().IsRunning {
		t.Fatal("expected engine to be running after resume")
	}

	time.Sleep(25 * time.Millisecond) // Let it tick again

	// Verify ticks resumed
	if len(tickTimes) <= ticksAtPause {
		t.Fatal("expected ticks to resume after start, but no new ticks recorded")
	}
}
