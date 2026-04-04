package notify

import (
	"context"
	"sync"

	"focus-cli/internal/model"
)

// Manager aggregates multiple notifiers and handles notification delivery
type Manager struct {
	notifiers []Notifier
	mu        sync.RWMutex
}

// NewManager creates a new notification manager
func NewManager() *Manager {
	return &Manager{
		notifiers: make([]Notifier, 0),
	}
}

// NewManagerFromConfig creates a notification manager from config
func NewManagerFromConfig(cfg *model.NotificationConfig) *Manager {
	manager := NewManager()

	if cfg == nil || !cfg.Enabled {
		return manager
	}

	// Add desktop notifier
	if cfg.Desktop != nil && cfg.Desktop.Enabled {
		manager.AddNotifier(GetDesktopNotifierForSystem(cfg.Desktop))
	}

	// Add sound notifier
	if cfg.Sound != nil && cfg.Sound.Enabled {
		manager.AddNotifier(NewSoundNotifier(true, cfg.Sound.SoundFile))
	}

	// Add file notifier
	if cfg.LogFile != nil && cfg.LogFile.Enabled && cfg.LogFile.Path != "" {
		manager.AddNotifier(NewFileNotifier(true, cfg.LogFile.Path))
	}

	return manager
}

// AddNotifier adds a notifier to the manager
func (m *Manager) AddNotifier(n Notifier) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.notifiers = append(m.notifiers, n)
}

// SendNotification sends a notification through all configured notifiers
func (m *Manager) SendNotification(ctx context.Context, event model.NotificationEvent) error {
	m.mu.RLock()
	notifiers := m.notifiers
	m.mu.RUnlock()

	// Send to all notifiers in parallel, collect errors but don't fail
	var wg sync.WaitGroup
	errChan := make(chan error, len(notifiers))

	for _, n := range notifiers {
		wg.Add(1)
		go func(notifier Notifier) {
			defer wg.Done()
			if err := notifier.Notify(ctx, event); err != nil {
				errChan <- err
			}
		}(n)
	}

	wg.Wait()
	close(errChan)

	// Log errors but don't return them (non-critical)
	// In a real implementation, you might want to log these
	for range errChan {
		// Silently ignore errors from notifiers
	}

	return nil
}

// Close closes all notifiers
func (m *Manager) Close() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	for _, n := range m.notifiers {
		n.Close()
	}

	return nil
}

// IsEnabled checks if notifications are enabled
func (m *Manager) IsEnabled() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return len(m.notifiers) > 0
}

// GetNotifierCount returns the number of active notifiers
func (m *Manager) GetNotifierCount() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return len(m.notifiers)
}
