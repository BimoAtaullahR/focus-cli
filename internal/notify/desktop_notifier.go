package notify

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strconv"

	"focus-cli/internal/model"
)

// DesktopNotifier sends notifications via the system desktop notification service
type DesktopNotifier struct {
	enabled    bool
	useTimeout bool
	timeoutMS  int
}

// NewDesktopNotifier creates a new desktop notifier
func NewDesktopNotifier(enabled, useTimeout bool, timeoutMS int) *DesktopNotifier {
	return &DesktopNotifier{
		enabled:    enabled,
		useTimeout: useTimeout,
		timeoutMS:  timeoutMS,
	}
}

// Notify sends a desktop notification
func (d *DesktopNotifier) Notify(ctx context.Context, event model.NotificationEvent) error {
	if !d.enabled {
		return nil
	}

	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	title, body := d.formatMessage(event)

	// Try different notification methods
	if err := d.notifyViaDbus(ctx, title, body); err == nil {
		return nil
	}
	if err := d.notifyViaNotifySend(ctx, title, body); err == nil {
		return nil
	}
	if err := d.notifyViaZenity(ctx, title, body); err == nil {
		return nil
	}

	// If all methods fail, return nil (don't treat as a critical error)
	return nil
}

// Close closes the notifier
func (d *DesktopNotifier) Close() error {
	return nil
}

// formatMessage formats a notification event into title and body
func (d *DesktopNotifier) formatMessage(event model.NotificationEvent) (string, string) {
	title := "Focus Timer"
	body := event.Message

	// Provide localized defaults if Message is empty
	if body == "" {
		switch event.Type {
		case "focus_complete":
			title = "Sesi Focus Selesai"
			body = fmt.Sprintf("Sesi focus #%d berhasil diselesaikan. Waktu untuk istirahat!", event.SessionNum)
		case "session_warning":
			title = "Peringatan Waktu"
			body = fmt.Sprintf("Sesi akan berakhir dalam %d menit", 5)
		case "break_complete":
			title = "Break Selesai"
			body = "Saatnya kembali bekerja!"
		case "task_complete":
			title = "Task Selesai"
			body = "Semua sesi untuk task ini sudah diselesaikan!"
		}
	}

	return title, body
}

// notifyViaDbus sends notification via D-Bus (native Linux)
func (d *DesktopNotifier) notifyViaDbus(ctx context.Context, title, body string) error {
	// This is a simplified approach using org.freedesktop.Notifications
	// via dbus-send command
	cmd := exec.CommandContext(ctx, "dbus-send",
		"--print-reply",
		"--dest=org.freedesktop.Notifications",
		"/org/freedesktop/Notifications",
		"org.freedesktop.Notifications.Notify",
		"string:focus-cli",
		"uint32:0",
		"string:",
		"string:"+title,
		"string:"+body,
		"array:string:",
		"dict:string:string:",
		"int32:"+strconv.Itoa(d.timeoutMS),
	)

	return cmd.Run()
}

// notifyViaNotifySend sends notification via notify-send (common on Linux)
func (d *DesktopNotifier) notifyViaNotifySend(ctx context.Context, title, body string) error {
	args := []string{title, body}

	if d.useTimeout {
		args = append(args, "--expire-time="+strconv.Itoa(d.timeoutMS))
	}

	cmd := exec.CommandContext(ctx, "notify-send", args...)
	return cmd.Run()
}

// notifyViaZenity sends notification via zenity (common on some DEs)
func (d *DesktopNotifier) notifyViaZenity(ctx context.Context, title, body string) error {
	args := []string{
		"--notification",
		"--text=" + title + "\n" + body,
	}

	if d.useTimeout {
		timeout := d.timeoutMS / 1000 // Convert to seconds
		args = append(args, "--timeout="+strconv.Itoa(timeout))
	}

	cmd := exec.CommandContext(ctx, "zenity", args...)
	// Set displaycommand to not capture stdout
	cmd.Stdout = nil
	cmd.Stderr = nil
	return cmd.Run()
}

// isDesktopEnvironment checks if we're in a desktop environment
func isDesktopEnvironment() bool {
	desktopEnv := os.Getenv("DESKTOP_SESSION")
	if desktopEnv != "" {
		return true
	}

	// Check for wayland or x11
	if os.Getenv("WAYLAND_DISPLAY") != "" || os.Getenv("DISPLAY") != "" {
		return true
	}

	// Try to detect via systemd user environment
	if os.Getenv("SYSTEMD_EXEC_PID") != "" {
		return true
	}

	return false
}

// GetDesktopNotifierForSystem creates a notifier suitable for the current desktop environment
func GetDesktopNotifierForSystem(config *model.DesktopNotifConfig) *DesktopNotifier {
	if config == nil {
		config = model.NewDesktopNotifConfig()
	}

	// Disable if not in desktop environment
	enabled := config.Enabled && isDesktopEnvironment()

	return NewDesktopNotifier(enabled, config.UseTimeout, config.TimeoutMS)
}
