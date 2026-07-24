package info

import (
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	agentapi "github.com/suzuki-shunsuke/ghtkn-go-sdk/ghtkn/backend/agent"
	"github.com/suzuki-shunsuke/ghtkn/pkg/controller/info"
)

func TestFormatTTLDays(t *testing.T) {
	t.Parallel()
	tests := []struct {
		d    time.Duration
		want string
	}{
		{3 * 24 * time.Hour, "3d"},
		{28 * 24 * time.Hour, "28d"},
		{36 * time.Hour, "1.5d"}, // 1.5 days, e.g. from a fractional week
	}
	for _, tt := range tests {
		if got := formatTTLDays(tt.d); got != tt.want {
			t.Errorf("formatTTLDays(%s) = %q, want %q", tt.d, got, tt.want)
		}
	}
}

func TestAgentStatusFromResponse(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		running bool
		resp    *agentapi.Response
		want    *info.AgentStatus
	}{
		{
			name:    "not running",
			running: false,
			resp:    nil,
			want:    &info.AgentStatus{Running: false},
		},
		{
			name:    "running and locked omits refresh_token",
			running: true,
			resp:    &agentapi.Response{Locked: true},
			want:    &info.AgentStatus{Running: true, Locked: new(true)},
		},
		{
			name:    "unlocked with a TTL",
			running: true,
			resp:    &agentapi.Response{RefreshTokenEnabled: true, RefreshTokenTTL: 3 * 24 * time.Hour},
			want:    &info.AgentStatus{Running: true, Locked: new(false), RefreshToken: &info.AgentRefreshToken{Enabled: true, TTL: "3d"}},
		},
		{
			name:    "unlocked without a TTL (older agent) omits ttl",
			running: true,
			resp:    &agentapi.Response{RefreshTokenEnabled: false},
			want:    &info.AgentStatus{Running: true, Locked: new(false), RefreshToken: &info.AgentRefreshToken{Enabled: false}},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if diff := cmp.Diff(tt.want, agentStatusFromResponse(tt.running, tt.resp)); diff != "" {
				t.Errorf("agentStatusFromResponse (-want +got):\n%s", diff)
			}
		})
	}
}
