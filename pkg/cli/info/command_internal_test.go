package info

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	agentapi "github.com/suzuki-shunsuke/ghtkn-go-sdk/ghtkn/backend/agent"
	"github.com/suzuki-shunsuke/ghtkn/pkg/agent/server"
	"github.com/suzuki-shunsuke/ghtkn/pkg/cli/docs"
	"github.com/suzuki-shunsuke/ghtkn/pkg/controller/info"
	"github.com/suzuki-shunsuke/slog-util/slogutil"
)

// TestHintDocs checks that a successful info run points coding agents at
// `ghtkn docs`, that the log level silences the hint, and that a failed run leaves
// it to the error's own hint instead of saying it twice.
func TestHintDocs(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		logLevel string
		err      error
		wantHint bool
	}{
		{
			name:     "successful run",
			wantHint: true,
		},
		{
			name:     "the log level silences the hint",
			logLevel: "warn",
		},
		{
			name: "failed run",
			err:  errors.New("output the information"),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			logFile, err := os.Create(filepath.Join(t.TempDir(), "stderr"))
			if err != nil {
				t.Fatal(err)
			}
			defer logFile.Close()

			logger := slogutil.New(&slogutil.InputNew{Name: "ghtkn", Version: "v1.0.0", Out: logFile})
			if err := logger.SetLevel(tt.logLevel); err != nil {
				t.Fatal(err)
			}
			hintDocs(logger, tt.err)

			logs, err := os.ReadFile(logFile.Name())
			if err != nil {
				t.Fatal(err)
			}
			if gotHint := strings.Contains(string(logs), docs.Hint); gotHint != tt.wantHint {
				t.Errorf("the docs hint is logged = %v, want %v: %s", gotHint, tt.wantHint, logs)
			}
		})
	}
}

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

func TestStaleAgent(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name         string
		agentVersion string
		ghtknVersion string
		want         bool
	}{
		{
			name:         "same version",
			agentVersion: "v0.3.4",
			ghtknVersion: "v0.3.4",
		},
		{
			name:         "older agent",
			agentVersion: "v0.3.1",
			ghtknVersion: "v0.3.4",
			want:         true,
		},
		{
			// Not an ordering: a newer agent is just as much "not this ghtkn's code".
			name:         "newer agent",
			agentVersion: "v0.4.0",
			ghtknVersion: "v0.3.4",
			want:         true,
		},
		{
			name:         "agent too old to report a version",
			agentVersion: "",
			ghtknVersion: "v0.3.4",
		},
		{
			name:         "agent built without version information",
			agentVersion: server.UnknownVersion,
			ghtknVersion: "v0.3.4",
		},
		{
			name:         "ghtkn built without version information",
			agentVersion: "v0.3.4",
			ghtknVersion: server.UnknownVersion,
		},
		{
			name:         "both built without version information",
			agentVersion: server.UnknownVersion,
			ghtknVersion: server.UnknownVersion,
		},
		{
			name:         "no ghtkn version",
			agentVersion: "v0.3.4",
			ghtknVersion: "",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := staleAgent(tt.agentVersion, tt.ghtknVersion); got != tt.want {
				t.Errorf("staleAgent(%q, %q) = %v, want %v", tt.agentVersion, tt.ghtknVersion, got, tt.want)
			}
		})
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
