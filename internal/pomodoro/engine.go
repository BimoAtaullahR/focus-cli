package pomodoro

import (
	"context"
	"sync"
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
	mu     sync.Mutex
	config EngineConfig

	OnTick          func(state EngineState)
	OnPhaseStart    func(state EngineState)
	OnPhaseComplete func(phase Phase, sessionCount int, startedAt, endedAt time.Time, completed bool)
	OnSessionWarn   func(state EngineState)
	OnComplete      func()

	state          EngineState
	cancel         context.CancelFunc
	runCtx         context.Context
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

func (e *SessionEngine) Resume(phase Phase, sessionCount int, remaining time.Duration) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.state.Phase = phase
	e.state.SessionCount = sessionCount
	e.state.Remaining = remaining
}

func (e *SessionEngine) Start(ctx context.Context) {
	e.mu.Lock()
	if e.state.IsRunning {
		e.mu.Unlock()
		return
	}
	e.state.IsRunning = true
	e.phaseStartedAt = time.Now()

	runCtx, cancel := context.WithCancel(ctx)
	e.cancel = cancel
	e.runCtx = runCtx
	state := e.state
	e.mu.Unlock()

	if e.OnPhaseStart != nil {
		e.OnPhaseStart(state)
	}

	if e.OnTick != nil {
		e.OnTick(state)
	}

	go func() {
		ticker := time.NewTicker(e.config.TickInterval)
		defer ticker.Stop()

		for {
			select {
			case <-runCtx.Done():
				e.mu.Lock()
				if e.runCtx == runCtx {
					e.state.IsRunning = false
					e.cancel = nil
					e.runCtx = nil
				}
				phase := e.state.Phase
				sessionCount := e.state.SessionCount
				phaseStartedAt := e.phaseStartedAt
				e.mu.Unlock()

				if e.OnPhaseComplete != nil {
					e.OnPhaseComplete(phase, sessionCount, phaseStartedAt, time.Now(), false)
				}
				return
			case <-ticker.C:
				e.mu.Lock()
				if !e.state.IsRunning || e.runCtx != runCtx {
					e.mu.Unlock()
					return
				}

				e.state.Remaining -= e.config.TickInterval
				if e.state.Remaining < 0 {
					e.state.Remaining = 0
				}

				warn := false
				if !e.warnTriggered && e.config.WarningDuration > 0 && e.state.Remaining <= e.config.WarningDuration {
					e.warnTriggered = true
					warn = true
				}
				state := e.state
				e.mu.Unlock()

				if warn && e.OnSessionWarn != nil {
					e.OnSessionWarn(state)
				}

				e.mu.Lock()
				// Double-check active run status
				if !e.state.IsRunning || e.runCtx != runCtx {
					e.mu.Unlock()
					return
				}

				if e.state.Remaining <= 0 {
					state = e.state
					endedAt := time.Now()
					completedPhase := e.state.Phase
					completedSession := e.state.SessionCount
					phaseStartedAt := e.phaseStartedAt
					e.mu.Unlock()

					if e.OnTick != nil {
						e.OnTick(state)
					}

					if e.OnPhaseComplete != nil {
						e.OnPhaseComplete(completedPhase, completedSession, phaseStartedAt, endedAt, true)
					}

					e.mu.Lock()
					completed := e.nextPhase()
					if completed {
						e.mu.Unlock()
						if e.OnComplete != nil {
							e.OnComplete()
						}
						return
					}
					state = e.state
					e.mu.Unlock()

					if e.OnPhaseStart != nil {
						e.OnPhaseStart(state)
					}
					if e.OnTick != nil {
						e.OnTick(state)
					}
				} else {
					e.mu.Unlock()
					if e.OnTick != nil {
						e.OnTick(state)
					}
				}
			}
		}
	}()
}

func (e *SessionEngine) Pause() {
	e.mu.Lock()
	cancel := e.cancel
	e.cancel = nil
	e.runCtx = nil
	e.state.IsRunning = false
	e.mu.Unlock()

	if cancel != nil {
		cancel()
	}
}

func (e *SessionEngine) Stop() {
	e.mu.Lock()
	cancel := e.cancel
	e.cancel = nil
	e.runCtx = nil
	e.state.IsRunning = false
	phase := e.state.Phase
	sessionCount := e.state.SessionCount
	phaseStartedAt := e.phaseStartedAt
	e.mu.Unlock()

	if cancel != nil {
		cancel()
	} else {
		if e.OnPhaseComplete != nil {
			e.OnPhaseComplete(phase, sessionCount, phaseStartedAt, time.Now(), false)
		}
	}
}

func (e *SessionEngine) AdvancePhase() {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.state.Remaining = 0
}

func (e *SessionEngine) nextPhase() bool {
	e.warnTriggered = false
	if e.state.Phase == PhaseFocus {
		if e.state.SessionCount >= e.config.TargetSessions {
			e.state.IsRunning = false
			e.runCtx = nil
			cancel := e.cancel
			e.cancel = nil
			e.mu.Unlock()
			if cancel != nil {
				cancel()
			}
			e.mu.Lock()
			return true
		}

		if e.state.SessionCount%e.config.LongBreakEvery == 0 {
			e.state.Phase = PhaseLongBreak
			e.state.Remaining = e.config.LongBreakDuration
		} else {
			e.state.Phase = PhaseShortBreak
			e.state.Remaining = e.config.ShortBreakDuration
		}
	} else {
		e.state.SessionCount++
		e.state.Phase = PhaseFocus
		e.state.Remaining = e.config.FocusDuration
	}

	e.phaseStartedAt = time.Now()
	return false
}

func (e *SessionEngine) State() EngineState {
	e.mu.Lock()
	defer e.mu.Unlock()
	return e.state
}
