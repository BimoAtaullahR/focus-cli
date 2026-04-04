package cli

import (
	"context"
	"testing"

	"focus-cli/internal/model"
	"focus-cli/internal/notify"
)

func TestParseOnOff(t *testing.T) {
	t.Parallel()

	cases := []struct {
		in      string
		want    bool
		wantErr bool
	}{
		{in: "on", want: true},
		{in: "OFF", want: false},
		{in: "yes", want: true},
		{in: "0", want: false},
		{in: "maybe", wantErr: true},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.in, func(t *testing.T) {
			t.Parallel()
			got, err := parseOnOff(tc.in)
			if tc.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("parseOnOff() error = %v", err)
			}
			if got != tc.want {
				t.Fatalf("parseOnOff() = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestScheduleWarningNotificationThreshold(t *testing.T) {
	t.Parallel()

	n := notify.NewManager()
	cfg := &model.NotificationConfig{Enabled: true, WarningMinutesBefore: 3}

	// warning >= phase => no timer
	tmr := scheduleWarningNotification(context.Background(), n, cfg, 1, 1, "focus", 3)
	if tmr != nil {
		t.Fatal("expected nil timer when warning threshold >= phase minutes")
	}

	// warning < phase => timer scheduled
	tmr = scheduleWarningNotification(context.Background(), n, cfg, 1, 1, "focus", 5)
	if tmr == nil {
		t.Fatal("expected non-nil timer when warning threshold is valid")
	}
	tmr.Stop()
}
