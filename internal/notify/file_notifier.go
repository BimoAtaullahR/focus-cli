package notify

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"focus-cli/internal/model"
)

// FileNotifier logs notifications to a file
type FileNotifier struct {
	enabled bool
	path    string
}

// NewFileNotifier creates a new file notifier
func NewFileNotifier(enabled bool, path string) *FileNotifier {
	return &FileNotifier{
		enabled: enabled,
		path:    path,
	}
}

// Notify writes a notification event to a file in JSON format
func (f *FileNotifier) Notify(ctx context.Context, event model.NotificationEvent) error {
	if !f.enabled || f.path == "" {
		return nil
	}

	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	// Create directory if it doesn't exist
	dir := filepath.Dir(f.path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("create log directory: %w", err)
	}

	// Marshal event to JSON
	data, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("marshal notification event: %w", err)
	}

	// Append to file with newline
	file, err := os.OpenFile(f.path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return fmt.Errorf("open log file: %w", err)
	}
	defer file.Close()

	if _, err := file.Write(append(data, '\n')); err != nil {
		return fmt.Errorf("write to log file: %w", err)
	}

	return nil
}

// Close closes the notifier
func (f *FileNotifier) Close() error {
	return nil
}
