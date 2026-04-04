package notify

import (
	"context"

	"focus-cli/internal/model"
)

// Notifier interface defines how notifications are sent
type Notifier interface {
	// Notify sends a notification event
	Notify(ctx context.Context, event model.NotificationEvent) error
	// Close performs cleanup if needed
	Close() error
}

// NotifierFunc is a function adapter for Notifier interface
type NotifierFunc func(ctx context.Context, event model.NotificationEvent) error

func (f NotifierFunc) Notify(ctx context.Context, event model.NotificationEvent) error {
	return f(ctx, event)
}

func (f NotifierFunc) Close() error {
	return nil
}

// NoopNotifier is a notifier that does nothing
type NoopNotifier struct{}

func (n *NoopNotifier) Notify(ctx context.Context, event model.NotificationEvent) error {
	return nil
}

func (n *NoopNotifier) Close() error {
	return nil
}
