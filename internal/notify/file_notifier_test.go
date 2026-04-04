package notify

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"focus-cli/internal/model"
)

func TestFileNotifierWritesJSONLine(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	logPath := filepath.Join(dir, "notifications.log")
	n := NewFileNotifier(true, logPath)

	event := model.NotificationEvent{
		Type:      model.NotificationFocusComplete,
		Timestamp: time.Now(),
		Message:   "ok",
	}
	if err := n.Notify(context.Background(), event); err != nil {
		t.Fatalf("Notify() error = %v", err)
	}

	b, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}
	line := strings.TrimSpace(string(b))
	if line == "" {
		t.Fatal("expected one JSON line, got empty")
	}

	var decoded model.NotificationEvent
	if err := json.Unmarshal([]byte(line), &decoded); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}
	if decoded.Type != model.NotificationFocusComplete {
		t.Fatalf("decoded.Type = %q, want %q", decoded.Type, model.NotificationFocusComplete)
	}
}

func TestFileNotifierNoopWhenDisabledOrPathEmpty(t *testing.T) {
	t.Parallel()

	n := NewFileNotifier(false, "")
	if err := n.Notify(context.Background(), model.NotificationEvent{Type: model.NotificationTaskComplete}); err != nil {
		t.Fatalf("Notify() error = %v", err)
	}
}
