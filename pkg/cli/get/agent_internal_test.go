package get

import (
	"testing"

	"github.com/suzuki-shunsuke/ghtkn/pkg/agent/server"
)

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
