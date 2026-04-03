package pomodoro

import (
	"context"
	"fmt"
	"time"
)

func Countdown(ctx context.Context, label string, minutes int) (startedAt time.Time, endedAt time.Time, completed bool, err error) {
	if minutes <= 0 {
		return time.Time{}, time.Time{}, false, fmt.Errorf("minutes must be > 0")
	}

	startedAt = time.Now()
	remaining := time.Duration(minutes) * time.Minute
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	printRemaining(label, remaining)
	for remaining > 0 {
		select {
		case <-ctx.Done():
			fmt.Print("\n")
			endedAt = time.Now()
			return startedAt, endedAt, false, ctx.Err()
		case <-ticker.C:
			remaining -= time.Second
			if remaining < 0 {
				remaining = 0
			}
			printRemaining(label, remaining)
		}
	}

	fmt.Print("\a\n")
	endedAt = time.Now()
	return startedAt, endedAt, true, nil
}

func printRemaining(label string, d time.Duration) {
	mins := int(d / time.Minute)
	secs := int((d % time.Minute) / time.Second)
	fmt.Printf("\r%s %02d:%02d", label, mins, secs)
}
