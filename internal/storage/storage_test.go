package storage

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"focus-cli/internal/model"
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

func TestStoreGCal(t *testing.T) {
	cfgHome := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", cfgHome)

	s, err := NewStore()
	if err != nil {
		t.Fatalf("NewStore() error = %v", err)
	}

	// 1. Test GCal config defaults in LoadConfig
	cfg, err := s.LoadConfig()
	if err != nil {
		t.Fatalf("LoadConfig() error = %v", err)
	}
	if cfg.GCalEnabled {
		t.Errorf("expected GCalEnabled to be false by default")
	}
	if cfg.GCalCalendarName != "Focus Sessions" {
		t.Errorf("expected GCalCalendarName to be 'Focus Sessions', got '%s'", cfg.GCalCalendarName)
	}

	// 2. Test saving and loading custom GCal config
	cfg.GCalEnabled = true
	cfg.GCalCalendarID = "calendar-123"
	err = s.SaveConfig(cfg)
	if err != nil {
		t.Fatalf("SaveConfig() error = %v", err)
	}

	cfg2, err := s.LoadConfig()
	if err != nil {
		t.Fatalf("LoadConfig() error = %v", err)
	}
	if !cfg2.GCalEnabled {
		t.Errorf("expected GCalEnabled to be true after save")
	}
	if cfg2.GCalCalendarID != "calendar-123" {
		t.Errorf("expected GCalCalendarID to be 'calendar-123', got '%s'", cfg2.GCalCalendarID)
	}

	// 3. Test OAuth token read/write
	tokenJSON := `{"access_token":"token-123","token_type":"Bearer","refresh_token":"refresh-123","expiry":"2026-07-13T21:18:14Z"}`
	err = s.SaveGCalToken([]byte(tokenJSON))
	if err != nil {
		t.Fatalf("SaveGCalToken() error = %v", err)
	}

	loadedToken, err := s.LoadGCalToken()
	if err != nil {
		t.Fatalf("LoadGCalToken() error = %v", err)
	}
	if string(loadedToken) != tokenJSON {
		t.Errorf("expected token '%s', got '%s'", tokenJSON, string(loadedToken))
	}

	// 4. Test deleting token
	err = s.DeleteGCalToken()
	if err != nil {
		t.Fatalf("DeleteGCalToken() error = %v", err)
	}
	_, err = s.LoadGCalToken()
	if err == nil {
		t.Errorf("expected error loading token after deletion")
	}

	// 5. Test reading GCal credentials file
	credentialsJSON := `{"installed":{"client_id":"client-id","client_secret":"client-secret"}}`
	// Write directly to file system to mock user placing the credentials
	credsPath := filepath.Join(cfgHome, "focus-cli", "gcal_credentials.json")
	err = os.WriteFile(credsPath, []byte(credentialsJSON), 0644)
	if err != nil {
		t.Fatalf("failed to write mock credentials: %v", err)
	}

	loadedCreds, err := s.ReadGCalCredentials()
	if err != nil {
		t.Fatalf("ReadGCalCredentials() error = %v", err)
	}
	if string(loadedCreds) != credentialsJSON {
		t.Errorf("expected credentials '%s', got '%s'", credentialsJSON, string(loadedCreds))
	}
}

func TestTaskStoreRoundTrip(t *testing.T) {
	cfgHome := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", cfgHome)

	s, err := NewStore()
	if err != nil {
		t.Fatalf("NewStore() error = %v", err)
	}

	importTime := time.Now().Round(time.Second)

	// Save task store with custom fields
	ts := model.TaskStore{
		NextID: 2,
		Tasks: []model.Task{
			{
				ID:             1,
				Title:          "Custom Task",
				FocusDuration:  45,
				BreakDuration:  10,
				GCalEventID:    "event-123",
				TargetSessions: 2,
				CreatedAt:      importTime,
				UpdatedAt:      importTime,
			},
		},
		DeletedGCalEventIDs: []string{"deleted-event-1", "deleted-event-2"},
	}

	err = s.SaveTasks(ts)
	if err != nil {
		t.Fatalf("SaveTasks() error = %v", err)
	}

	loadedTS, err := s.LoadTasks()
	if err != nil {
		t.Fatalf("LoadTasks() error = %v", err)
	}

	if loadedTS.NextID != 2 {
		t.Errorf("expected NextID 2, got %d", loadedTS.NextID)
	}

	if len(loadedTS.Tasks) != 1 {
		t.Fatalf("expected 1 task, got %d", len(loadedTS.Tasks))
	}

	task := loadedTS.Tasks[0]
	if task.Title != "Custom Task" || task.FocusDuration != 45 || task.BreakDuration != 10 || task.GCalEventID != "event-123" {
		t.Errorf("unexpected task loaded: %+v", task)
	}

	if len(loadedTS.DeletedGCalEventIDs) != 2 || loadedTS.DeletedGCalEventIDs[0] != "deleted-event-1" || loadedTS.DeletedGCalEventIDs[1] != "deleted-event-2" {
		t.Errorf("unexpected DeletedGCalEventIDs loaded: %v", loadedTS.DeletedGCalEventIDs)
	}
}

