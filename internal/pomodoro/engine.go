package pomodoro

import (
	"context"
	"time"
)

type Phase string

const (
	PhaseFocus      Phase = "focus"
	PhaseShortBreak Phase = "short_break"
	PhaseLongBreak  Phase = "long_break"
)

type EngineState struct {
	Phase         Phase
	SessionCount  int
	TotalSessions int
	Remaining     time.Duration
	IsRunning     bool
}

type SessionEngine struct {
	FocusMinutes      int
	ShortBreakMinutes int
	LongBreakMinutes  int
	LongBreakEvery    int
	TargetSessions    int

	OnTick          func(state EngineState)
	OnPhaseStart    func(state EngineState)
	OnPhaseComplete func(phase Phase, sessionCount int, startedAt, endedAt time.Time, completed bool)
	OnComplete      func()

	state          EngineState
	cancel         context.CancelFunc
	phaseStartedAt time.Time
}

func NewSessionEngine(focus, short, long, longEvery, targetSessions int) *SessionEngine {
	return &SessionEngine{
		FocusMinutes:      focus,
		ShortBreakMinutes: short,
		LongBreakMinutes:  long,
		LongBreakEvery:    longEvery,
		TargetSessions:    targetSessions,
		state: EngineState{
			Phase:         PhaseFocus,
			SessionCount:  1,
			TotalSessions: targetSessions,
			Remaining:     time.Duration(focus) * time.Minute,
			IsRunning:     false,
		},
	}
}

func (e *SessionEngine) Start(ctx context.Context) {
	if e.state.IsRunning {
		return
	}
	e.state.IsRunning = true
	e.phaseStartedAt = time.Now()

	ctx, cancel := context.WithCancel(ctx)
	e.cancel = cancel

	if e.OnPhaseStart != nil {
		e.OnPhaseStart(e.state)
	}

	if e.OnTick != nil {
		e.OnTick(e.state)
	}

	go func() {
		ticker := time.NewTicker(1 * time.Second)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				e.state.IsRunning = false
				if e.OnPhaseComplete != nil {
					e.OnPhaseComplete(e.state.Phase, e.state.SessionCount, e.phaseStartedAt, time.Now(), false)
				}
				return
			case <-ticker.C:
				e.state.Remaining -= time.Second
				if e.state.Remaining <= 0 {
					e.state.Remaining = 0
					if e.OnTick != nil {
						e.OnTick(e.state)
					}
					endedAt := time.Now()
					completedPhase := e.state.Phase
					completedSession := e.state.SessionCount

					if e.OnPhaseComplete != nil {
						e.OnPhaseComplete(completedPhase, completedSession, e.phaseStartedAt, endedAt, true)
					}
					e.nextPhase()
					if !e.state.IsRunning {
						return // Completed
					}
				} else {
					if e.OnTick != nil {
						e.OnTick(e.state)
					}
				}
			}
		}
	}()
}

func (e *SessionEngine) Pause() {
	if e.cancel != nil {
		e.cancel()
		e.cancel = nil
	}
	e.state.IsRunning = false
}

func (e *SessionEngine) Stop() {
	if e.cancel != nil {
		e.cancel()
		e.cancel = nil
	}
	e.state.IsRunning = false
	if e.OnPhaseComplete != nil {
		e.OnPhaseComplete(e.state.Phase, e.state.SessionCount, e.phaseStartedAt, time.Now(), false)
	}
}

func (e *SessionEngine) nextPhase() {
	if e.state.Phase == PhaseFocus {
		if e.state.SessionCount >= e.TargetSessions {
			e.state.IsRunning = false
			if e.cancel != nil {
				e.cancel()
				e.cancel = nil
			}
			if e.OnComplete != nil {
				e.OnComplete()
			}
			return
		}

		if e.state.SessionCount%e.LongBreakEvery == 0 {
			e.state.Phase = PhaseLongBreak
			e.state.Remaining = time.Duration(e.LongBreakMinutes) * time.Minute
		} else {
			e.state.Phase = PhaseShortBreak
			e.state.Remaining = time.Duration(e.ShortBreakMinutes) * time.Minute
		}
	} else {
		// From break to focus
		e.state.SessionCount++
		e.state.Phase = PhaseFocus
		e.state.Remaining = time.Duration(e.FocusMinutes) * time.Minute
	}

	e.phaseStartedAt = time.Now()

	if e.OnPhaseStart != nil {
		e.OnPhaseStart(e.state)
	}
	if e.OnTick != nil {
		e.OnTick(e.state)
	}
}

func (e *SessionEngine) State() EngineState {
	return e.state
}
