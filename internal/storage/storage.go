package storage

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"focus-cli/internal/model"
)

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
		return model.DefaultConfig(), nil
	}
	if err != nil {
		return out, fmt.Errorf("read config: %w", err)
	}
	if err := json.Unmarshal(b, &out); err != nil {
		return out, fmt.Errorf("parse config: %w", err)
	}
	return model.NormalizeConfig(out), nil
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
