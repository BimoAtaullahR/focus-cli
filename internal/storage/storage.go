package storage

import (
	"embed"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"focus-cli/internal/model"
)

//go:embed assets/sounds/*.wav
var soundFiles embed.FS

type Store struct {
	baseDir string
}

func NewStore() (*Store, error) {
	cfgDir, err := os.UserConfigDir()
	if err != nil {
		return nil, fmt.Errorf("get user config dir: %w", err)
	}
	baseDir := filepath.Join(cfgDir, "focus-cli")
	if err := os.MkdirAll(baseDir, 0o755); err != nil {
		return nil, fmt.Errorf("create data dir: %w", err)
	}

	// Ensure sound files are available
	if err := ensureSoundFiles(baseDir); err != nil {
		// Log but don't fail if sound setup fails
		fmt.Fprintf(os.Stderr, "Warning: could not setup sound files: %v\n", err)
	}

	return &Store{baseDir: baseDir}, nil
}

func (s *Store) tasksPath() string {
	return filepath.Join(s.baseDir, "tasks.json")
}

func (s *Store) configPath() string {
	return filepath.Join(s.baseDir, "config.json")
}

func (s *Store) historyPath() string {
	return filepath.Join(s.baseDir, "history.json")
}

func (s *Store) gcalCredentialsPath() string {
	return filepath.Join(s.baseDir, "gcal_credentials.json")
}

func (s *Store) gcalTokenPath() string {
	return filepath.Join(s.baseDir, "gcal_token.json")
}

func (s *Store) ReadGCalCredentials() ([]byte, error) {
	return os.ReadFile(s.gcalCredentialsPath())
}

func (s *Store) SaveGCalToken(data []byte) error {
	return os.WriteFile(s.gcalTokenPath(), data, 0o600)
}

func (s *Store) LoadGCalToken() ([]byte, error) {
	return os.ReadFile(s.gcalTokenPath())
}

func (s *Store) DeleteGCalToken() error {
	err := os.Remove(s.gcalTokenPath())
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	return err
}

func (s *Store) LoadTasks() (model.TaskStore, error) {
	var out model.TaskStore
	b, err := os.ReadFile(s.tasksPath())
	if errors.Is(err, os.ErrNotExist) {
		out.NextID = 1
		return out, nil
	}
	if err != nil {
		return out, fmt.Errorf("read tasks: %w", err)
	}
	if err := json.Unmarshal(b, &out); err != nil {
		return out, fmt.Errorf("parse tasks: %w", err)
	}
	if out.NextID < 1 {
		out.NextID = 1
	}
	return out, nil
}

func (s *Store) SaveTasks(ts model.TaskStore) error {
	return s.writeJSON(s.tasksPath(), ts)
}

func (s *Store) LoadConfig() (model.Config, error) {
	var out model.Config
	b, err := os.ReadFile(s.configPath())
	if errors.Is(err, os.ErrNotExist) {
		cfg := model.DefaultConfig()
		// Set sound file path to bundled notification sound
		if cfg.Notifications != nil && cfg.Notifications.Sound != nil {
			cfg.Notifications.Sound.SoundFile = s.GetSoundFilePath()
		}
		return cfg, nil
	}
	if err != nil {
		return out, fmt.Errorf("read config: %w", err)
	}
	if err := json.Unmarshal(b, &out); err != nil {
		return out, fmt.Errorf("parse config: %w", err)
	}
	out = model.NormalizeConfig(out)
	// Auto-set sound file if empty
	if out.Notifications != nil && out.Notifications.Sound != nil && out.Notifications.Sound.SoundFile == "" {
		out.Notifications.Sound.SoundFile = s.GetSoundFilePath()
	}
	return out, nil
}

func (s *Store) SaveConfig(cfg model.Config) error {
	return s.writeJSON(s.configPath(), cfg)
}

func (s *Store) LoadHistory() ([]model.SessionHistory, error) {
	var out []model.SessionHistory
	b, err := os.ReadFile(s.historyPath())
	if errors.Is(err, os.ErrNotExist) {
		return out, nil
	}
	if err != nil {
		return nil, fmt.Errorf("read history: %w", err)
	}
	if err := json.Unmarshal(b, &out); err != nil {
		return nil, fmt.Errorf("parse history: %w", err)
	}
	return out, nil
}

func (s *Store) SaveHistory(h []model.SessionHistory) error {
	return s.writeJSON(s.historyPath(), h)
}

func (s *Store) writeJSON(path string, v any) error {
	b, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal json: %w", err)
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, b, 0o644); err != nil {
		return fmt.Errorf("write temp file: %w", err)
	}
	if err := os.Rename(tmp, path); err != nil {
		return fmt.Errorf("atomic replace: %w", err)
	}
	return nil
}

// ensureSoundFiles copies embedded sound files to config directory on first run
func ensureSoundFiles(baseDir string) error {
	soundPath := filepath.Join(baseDir, "notification.wav")

	// If sound file already exists, skip
	if _, err := os.Stat(soundPath); err == nil {
		return nil
	}

	// Read embedded notification.wav
	data, err := soundFiles.ReadFile("assets/sounds/notification.wav")
	if err != nil {
		return fmt.Errorf("read embedded sound file: %w", err)
	}

	// Write to config directory
	if err := os.WriteFile(soundPath, data, 0o644); err != nil {
		return fmt.Errorf("write sound file: %w", err)
	}

	return nil
}

// GetSoundFilePath returns the path to the default notification sound file
func (s *Store) GetSoundFilePath() string {
	return filepath.Join(s.baseDir, "notification.wav")
}
