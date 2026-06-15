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
