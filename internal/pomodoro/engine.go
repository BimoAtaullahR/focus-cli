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

type EngineConfig struct {
	FocusDuration      time.Duration
	ShortBreakDuration time.Duration
	LongBreakDuration  time.Duration
	LongBreakEvery     int
	TargetSessions     int
	WarningDuration    time.Duration
	TickInterval       time.Duration
}

type SessionEngine struct {
	config EngineConfig

	OnTick          func(state EngineState)
	OnPhaseStart    func(state EngineState)
	OnPhaseComplete func(phase Phase, sessionCount int, startedAt, endedAt time.Time, completed bool)
	OnSessionWarn   func(state EngineState)
	OnComplete      func()

	state          EngineState
	cancel         context.CancelFunc
	phaseStartedAt time.Time
	warnTriggered  bool
}

func NewSessionEngine(config EngineConfig) *SessionEngine {
	if config.TickInterval <= 0 {
		config.TickInterval = time.Second
	}
	return &SessionEngine{
		config: config,
		state: EngineState{
			Phase:         PhaseFocus,
			SessionCount:  1,
			TotalSessions: config.TargetSessions,
			Remaining:     config.FocusDuration,
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
		ticker := time.NewTicker(e.config.TickInterval)
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
				e.state.Remaining -= e.config.TickInterval
				
				if !e.warnTriggered && e.config.WarningDuration > 0 && e.state.Remaining <= e.config.WarningDuration {
					e.warnTriggered = true
					if e.OnSessionWarn != nil {
						e.OnSessionWarn(e.state)
					}
				}

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

// AdvancePhase manually completes the current phase.
// It sets remaining to 0 which will be picked up by the tick loop to transition to next phase.
func (e *SessionEngine) AdvancePhase() {
	e.state.Remaining = 0
}

func (e *SessionEngine) nextPhase() {
	e.warnTriggered = false
	if e.state.Phase == PhaseFocus {
		if e.state.SessionCount >= e.config.TargetSessions {
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

		if e.state.SessionCount%e.config.LongBreakEvery == 0 {
			e.state.Phase = PhaseLongBreak
			e.state.Remaining = e.config.LongBreakDuration
		} else {
			e.state.Phase = PhaseShortBreak
			e.state.Remaining = e.config.ShortBreakDuration
		}
	} else {
		// From break to focus
		e.state.SessionCount++
		e.state.Phase = PhaseFocus
		e.state.Remaining = e.config.FocusDuration
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
