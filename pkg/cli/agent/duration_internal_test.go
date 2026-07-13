package agent

import (
	"testing"
	"time"
)

func TestParseRefreshTokenTTL(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		in      string
		want    time.Duration
		wantErr bool
	}{
		{name: "days", in: "3d", want: 72 * time.Hour},
		{name: "weeks", in: "4w", want: 4 * 7 * 24 * time.Hour},
		{name: "week default", in: "1w", want: 7 * 24 * time.Hour},
		{name: "months", in: "2m", want: 2 * 30 * 24 * time.Hour},
		{name: "fractional week", in: "1.5w", want: time.Duration(1.5 * float64(7*24*time.Hour))},
		{name: "hours fallback", in: "168h", want: 168 * time.Hour},
		{name: "zero uses server default", in: "0w", want: 0},
		{name: "just under six months", in: "179d", want: 179 * 24 * time.Hour},
		{name: "exactly six months accepted", in: "6m", want: 6 * 30 * 24 * time.Hour},

		{name: "over six months", in: "7m", wantErr: true},
		{name: "over six months in days", in: "200d", wantErr: true},
		{name: "negative", in: "-1w", wantErr: true},
		{name: "empty", in: "", wantErr: true},
		{name: "no number", in: "w", wantErr: true},
		{name: "unknown unit", in: "4x", wantErr: true},
		{name: "not a duration", in: "abc", wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, err := parseRefreshTokenTTL(tt.in)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("parseRefreshTokenTTL(%q) = %v, want error", tt.in, got)
				}
				return
			}
			if err != nil {
				t.Fatalf("parseRefreshTokenTTL(%q) unexpected error: %v", tt.in, err)
			}
			if got != tt.want {
				t.Fatalf("parseRefreshTokenTTL(%q) = %v, want %v", tt.in, got, tt.want)
			}
		})
	}
}
