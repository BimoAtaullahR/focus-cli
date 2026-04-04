package notify

import (
	"testing"

	"focus-cli/internal/model"
)

func TestNewManagerFromConfigDisabled(t *testing.T) {
	t.Parallel()

	m := NewManagerFromConfig(&model.NotificationConfig{Enabled: false})
	if m.GetNotifierCount() != 0 {
		t.Fatalf("GetNotifierCount() = %d, want 0", m.GetNotifierCount())
	}
	if m.IsEnabled() {
		t.Fatal("IsEnabled() = true, want false")
	}
}

func TestNewManagerFromConfigBuildsExpectedNotifiers(t *testing.T) {
	t.Parallel()

	cfg := &model.NotificationConfig{
		Enabled: true,
		Desktop: &model.DesktopNotifConfig{Enabled: false},
		Sound:   &model.SoundNotifConfig{Enabled: true},
		LogFile: &model.LogFileNotifConfig{Enabled: true, Path: "./tmp-notif.log"},
	}
	m := NewManagerFromConfig(cfg)
	if m.GetNotifierCount() != 2 {
		t.Fatalf("GetNotifierCount() = %d, want 2", m.GetNotifierCount())
	}
	if !m.IsEnabled() {
		t.Fatal("IsEnabled() = false, want true")
	}
}
