package notify

import (
	"context"
	"fmt"
	"os/exec"

	"focus-cli/internal/model"
)

// SoundNotifier sends notification through system sound
type SoundNotifier struct {
	enabled   bool
	soundFile string
}

// NewSoundNotifier creates a new sound notifier
func NewSoundNotifier(enabled bool, soundFile string) *SoundNotifier {
	return &SoundNotifier{
		enabled:   enabled,
		soundFile: soundFile,
	}
}

// Notify plays a system sound or bell
func (s *SoundNotifier) Notify(ctx context.Context, event model.NotificationEvent) error {
	if !s.enabled {
		return nil
	}

	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	// If custom sound file is configured, try to play it
	if s.soundFile != "" {
		return s.playAudioFile(ctx)
	}

	// Fall back to system bell
	fmt.Print("\a")
	return nil
}

// Close closes the notifier
func (s *SoundNotifier) Close() error {
	return nil
}

// playAudioFile attempts to play an audio file using available tools
func (s *SoundNotifier) playAudioFile(ctx context.Context) error {
	// Try different audio player commands in order
	players := []string{"ffplay", "paplay", "aplay", "play"}

	for _, player := range players {
		cmd := exec.CommandContext(ctx, player, "-nodisp", "-autoexit", s.soundFile)
		if err := cmd.Run(); err == nil {
			return nil
		}
		// Continue to next player if this one fails
	}

	// If no player worked, fall back to bell
	fmt.Print("\a")
	return nil
}
