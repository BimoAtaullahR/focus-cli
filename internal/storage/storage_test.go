package storage

import (
	"path/filepath"
	"testing"
)

func TestStoreRoundTrip(t *testing.T) {
	cfgHome := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", cfgHome)

	s, err := NewStore()
	if err != nil {
		t.Fatalf("NewStore() error = %v", err)
	}

	cfg, err := s.LoadConfig()
	if err != nil {
		t.Fatalf("LoadConfig() error = %v", err)
	}
	if cfg.FocusMinutes <= 0 {
		t.Fatalf("invalid default config: %+v", cfg)
	}

	ts, err := s.LoadTasks()
	if err != nil {
		t.Fatalf("LoadTasks() error = %v", err)
	}
	if ts.NextID != 1 {
		t.Fatalf("unexpected NextID: %d", ts.NextID)
	}

	if got := s.tasksPath(); got != filepath.Join(cfgHome, "focus-cli", "tasks.json") {
		t.Fatalf("unexpected tasks path: %s", got)
	}
}
