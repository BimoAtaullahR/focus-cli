package model

import "time"

type Task struct {
	ID                 int       `json:"id"`
	Title              string    `json:"title"`
	Description        string    `json:"description,omitempty"`
	Done               bool      `json:"done"`
	TargetSessions     int       `json:"target_sessions"`
	CompletedPomodoros int       `json:"completed_pomodoros"`
	TimerPhase         string    `json:"timer_phase,omitempty"`
	TimerRemainingSec  int       `json:"timer_remaining_sec,omitempty"`
	TimerSessionIndex  int       `json:"timer_session_index,omitempty"`
	TimerTotalSessions int       `json:"timer_total_sessions,omitempty"`
	CreatedAt          time.Time `json:"created_at"`
	UpdatedAt          time.Time `json:"updated_at"`
	GCalEventID        string    `json:"gcal_event_id,omitempty"`
	FocusDuration      int       `json:"focus_duration,omitempty"`
	BreakDuration      int       `json:"break_duration,omitempty"`
}

type TaskStore struct {
	NextID              int      `json:"next_id"`
	Tasks               []Task   `json:"tasks"`
	DeletedGCalEventIDs []string `json:"deleted_gcal_event_ids,omitempty"`
}

type Config struct {
	FocusMinutes      int                 `json:"focus_minutes"`
	ShortBreakMinutes int                 `json:"short_break_minutes"`
	LongBreakMinutes  int                 `json:"long_break_minutes"`
	LongBreakEvery    int                 `json:"long_break_every"`
	Theme             string              `json:"theme"`
	Keys              Keys                `json:"keys"`
	Notifications     *NotificationConfig `json:"notifications,omitempty"`
	GCalEnabled       bool                `json:"gcal_enabled"`
	GCalCalendarName  string              `json:"gcal_calendar_name"`
	GCalCalendarID    string              `json:"gcal_calendar_id"`
}

type NotificationConfig struct {
	Enabled              bool                `json:"enabled"`
	WarningMinutesBefore int                 `json:"warning_minutes_before"`
	Desktop              *DesktopNotifConfig `json:"desktop,omitempty"`
	Sound                *SoundNotifConfig   `json:"sound,omitempty"`
	LogFile              *LogFileNotifConfig `json:"log_file,omitempty"`
}

type DesktopNotifConfig struct {
	Enabled    bool `json:"enabled"`
	UseTimeout bool `json:"use_timeout"`
	TimeoutMS  int  `json:"timeout_ms"`
}

type SoundNotifConfig struct {
	Enabled   bool   `json:"enabled"`
	SoundFile string `json:"sound_file,omitempty"`
}

type LogFileNotifConfig struct {
	Enabled bool   `json:"enabled"`
	Path    string `json:"path,omitempty"`
}

type NotificationEvent struct {
	Type       string    `json:"type"`
	Timestamp  time.Time `json:"timestamp"`
	SessionNum int       `json:"session_num,omitempty"`
	PhaseType  string    `json:"phase_type,omitempty"`
	TaskID     int       `json:"task_id,omitempty"`
	Message    string    `json:"message,omitempty"`
}

const (
	NotificationFocusComplete = "focus_complete"
	NotificationBreakComplete = "break_complete"
	NotificationTaskComplete  = "task_complete"
	NotificationSessionWarn   = "session_warning"
)

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
		Notifications:     DefaultNotificationConfig(),
		GCalEnabled:       false,
		GCalCalendarName:  "Focus Sessions",
		GCalCalendarID:    "",
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

func DefaultNotificationConfig() *NotificationConfig {
	return &NotificationConfig{
		Enabled:              true,
		WarningMinutesBefore: 5,
		Desktop:              NewDesktopNotifConfig(),
		Sound:                NewSoundNotifConfig(),
		LogFile:              NewLogFileNotifConfig(),
	}
}

func NewDesktopNotifConfig() *DesktopNotifConfig {
	return &DesktopNotifConfig{
		Enabled:    true,
		UseTimeout: true,
		TimeoutMS:  5000,
	}
}

func NewSoundNotifConfig() *SoundNotifConfig {
	return &SoundNotifConfig{
		Enabled:   true,
		SoundFile: "",
	}
}

func NewLogFileNotifConfig() *LogFileNotifConfig {
	return &LogFileNotifConfig{
		Enabled: false,
		Path:    "",
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
	if cfg.Notifications == nil {
		cfg.Notifications = def.Notifications
	} else {
		// Normalize notification sub-configs
		if cfg.Notifications.WarningMinutesBefore <= 0 {
			cfg.Notifications.WarningMinutesBefore = def.Notifications.WarningMinutesBefore
		}
		if cfg.Notifications.Desktop == nil {
			cfg.Notifications.Desktop = def.Notifications.Desktop
		}
		if cfg.Notifications.Sound == nil {
			cfg.Notifications.Sound = def.Notifications.Sound
		}
		if cfg.Notifications.LogFile == nil {
			cfg.Notifications.LogFile = def.Notifications.LogFile
		}
	}
	if cfg.GCalCalendarName == "" {
		cfg.GCalCalendarName = def.GCalCalendarName
	}
	return cfg
}
