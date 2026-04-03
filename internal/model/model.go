package model

import "time"

type Task struct {
	ID                 int       `json:"id"`
	Title              string    `json:"title"`
	Description        string    `json:"description,omitempty"`
	Done               bool      `json:"done"`
	TargetSessions     int       `json:"target_sessions"`
	CompletedPomodoros int       `json:"completed_pomodoros"`
	CreatedAt          time.Time `json:"created_at"`
	UpdatedAt          time.Time `json:"updated_at"`
}

type TaskStore struct {
	NextID int    `json:"next_id"`
	Tasks  []Task `json:"tasks"`
}

type Config struct {
	FocusMinutes      int `json:"focus_minutes"`
	ShortBreakMinutes int `json:"short_break_minutes"`
	LongBreakMinutes  int `json:"long_break_minutes"`
	LongBreakEvery    int `json:"long_break_every"`
	Theme             string `json:"theme"`
	Keys              Keys   `json:"keys"`
}

type Keys struct {
	NavUp       string `json:"nav_up"`
	NavDown     string `json:"nav_down"`
	AltNavUp    string `json:"alt_nav_up"`
	AltNavDown  string `json:"alt_nav_down"`
	ReorderUp   string `json:"reorder_up"`
	ReorderDown string `json:"reorder_down"`
	AddTask     string `json:"add_task"`
	EditTask    string `json:"edit_task"`
	DeleteTask  string `json:"delete_task"`
	ToggleDone  string `json:"toggle_done"`
	StartCycle  string `json:"start_cycle"`
	Settings    string `json:"settings"`
	Refresh     string `json:"refresh"`
	Quit        string `json:"quit"`
	Pause       string `json:"pause"`
	EndPhase    string `json:"end_phase"`
	NextPhase   string `json:"next_phase"`
}

type SessionHistory struct {
	StartedAt time.Time `json:"started_at"`
	EndedAt   time.Time `json:"ended_at"`
	TaskID    int       `json:"task_id,omitempty"`
	Type      string    `json:"type"`
	Completed bool      `json:"completed"`
}

func DefaultConfig() Config {
	return Config{
		FocusMinutes:      25,
		ShortBreakMinutes: 5,
		LongBreakMinutes:  15,
		LongBreakEvery:    4,
		Theme:             "sunrise",
		Keys:              DefaultKeys(),
	}
}

func DefaultKeys() Keys {
	return Keys{
		NavUp:       "up",
		NavDown:     "down",
		AltNavUp:    "k",
		AltNavDown:  "j",
		ReorderUp:   "ctrl+k",
		ReorderDown: "ctrl+j",
		AddTask:     "a",
		EditTask:    "e",
		DeleteTask:  "d",
		ToggleDone:  "space",
		StartCycle:  "enter",
		Settings:    "s",
		Refresh:     "r",
		Quit:        "q",
		Pause:       "p",
		EndPhase:    "x",
		NextPhase:   "n",
	}
}

func NormalizeConfig(cfg Config) Config {
	def := DefaultConfig()
	if cfg.FocusMinutes <= 0 {
		cfg.FocusMinutes = def.FocusMinutes
	}
	if cfg.ShortBreakMinutes <= 0 {
		cfg.ShortBreakMinutes = def.ShortBreakMinutes
	}
	if cfg.LongBreakMinutes <= 0 {
		cfg.LongBreakMinutes = def.LongBreakMinutes
	}
	if cfg.LongBreakEvery <= 0 {
		cfg.LongBreakEvery = def.LongBreakEvery
	}
	if cfg.Theme == "" {
		cfg.Theme = def.Theme
	}
	if cfg.Keys.NavUp == "" {
		cfg.Keys.NavUp = def.Keys.NavUp
	}
	if cfg.Keys.NavDown == "" {
		cfg.Keys.NavDown = def.Keys.NavDown
	}
	if cfg.Keys.AltNavUp == "" {
		cfg.Keys.AltNavUp = def.Keys.AltNavUp
	}
	if cfg.Keys.AltNavDown == "" {
		cfg.Keys.AltNavDown = def.Keys.AltNavDown
	}
	if cfg.Keys.ReorderUp == "" {
		cfg.Keys.ReorderUp = def.Keys.ReorderUp
	}
	if cfg.Keys.ReorderDown == "" {
		cfg.Keys.ReorderDown = def.Keys.ReorderDown
	}
	if cfg.Keys.AddTask == "" {
		cfg.Keys.AddTask = def.Keys.AddTask
	}
	if cfg.Keys.EditTask == "" {
		cfg.Keys.EditTask = def.Keys.EditTask
	}
	if cfg.Keys.DeleteTask == "" {
		cfg.Keys.DeleteTask = def.Keys.DeleteTask
	}
	if cfg.Keys.ToggleDone == "" {
		cfg.Keys.ToggleDone = def.Keys.ToggleDone
	}
	if cfg.Keys.StartCycle == "" {
		cfg.Keys.StartCycle = def.Keys.StartCycle
	}
	if cfg.Keys.Settings == "" {
		cfg.Keys.Settings = def.Keys.Settings
	}
	if cfg.Keys.Refresh == "" {
		cfg.Keys.Refresh = def.Keys.Refresh
	}
	if cfg.Keys.Quit == "" {
		cfg.Keys.Quit = def.Keys.Quit
	}
	if cfg.Keys.Pause == "" {
		cfg.Keys.Pause = def.Keys.Pause
	}
	if cfg.Keys.EndPhase == "" {
		cfg.Keys.EndPhase = def.Keys.EndPhase
	}
	if cfg.Keys.NextPhase == "" {
		cfg.Keys.NextPhase = def.Keys.NextPhase
	}
	return cfg
}
