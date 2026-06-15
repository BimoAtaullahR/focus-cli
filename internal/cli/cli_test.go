package cli

import (
	"testing"
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


