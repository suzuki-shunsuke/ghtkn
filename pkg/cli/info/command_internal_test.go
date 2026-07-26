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

func TestAgentStatusFromResponse(t *testing.T) { //nolint:funlen
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
			// A locked agent still reports which binary it runs, so the version fields
			// are set even though the unlocked-state fields are not.
			name:    "running and locked omits refresh_token",
			running: true,
			resp:    &agentapi.Response{Locked: true, Version: "v0.3.1", ProtocolVersion: 1, MinProtocolVersion: 0},
			want: &info.AgentStatus{
				Running: true, Version: "v0.3.1", ProtocolVersion: new(1), MinProtocolVersion: new(0),
				Locked: new(true),
			},
		},
		{
			name:    "unlocked with a TTL",
			running: true,
			resp: &agentapi.Response{
				Version: "v0.3.4", ProtocolVersion: 1, MinProtocolVersion: 1,
				RefreshTokenEnabled: true, RefreshTokenTTL: 3 * 24 * time.Hour,
			},
			want: &info.AgentStatus{
				Running: true, Version: "v0.3.4", ProtocolVersion: new(1), MinProtocolVersion: new(1),
				Locked: new(false), RefreshToken: &info.AgentRefreshToken{Enabled: true, TTL: "3d"},
			},
		},
		{
			name:    "unlocked without a TTL (older agent) omits ttl",
			running: true,
			resp:    &agentapi.Response{RefreshTokenEnabled: false, Version: "v0.3.4", ProtocolVersion: 1},
			want: &info.AgentStatus{
				Running: true, Version: "v0.3.4", ProtocolVersion: new(1), MinProtocolVersion: new(0),
				Locked: new(false), RefreshToken: &info.AgentRefreshToken{Enabled: false},
			},
		},
		{
			// An agent too old to report its version: the version is left empty so it is
			// dropped from the output, but the protocol versions it decoded to (0, i.e.
			// pre-versioning) are true of such an agent and are reported.
			name:    "running agent that does not report its version",
			running: true,
			resp:    &agentapi.Response{RefreshTokenEnabled: true},
			want: &info.AgentStatus{
				Running: true, ProtocolVersion: new(0), MinProtocolVersion: new(0),
				Locked: new(false), RefreshToken: &info.AgentRefreshToken{Enabled: true},
			},
		},
		{
			// An agent that knows its protocol but not which binary it was built from.
			name:    "running agent with an unknown version",
			running: true,
			resp:    &agentapi.Response{Locked: true, Version: "unknown", ProtocolVersion: 1},
			want: &info.AgentStatus{
				Running: true, Version: "unknown", ProtocolVersion: new(1), MinProtocolVersion: new(0),
				Locked: new(true),
			},
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
