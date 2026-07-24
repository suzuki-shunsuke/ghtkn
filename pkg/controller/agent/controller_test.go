package agent_test

import (
	"testing"

	"github.com/suzuki-shunsuke/ghtkn/pkg/controller/agent"
)

// TestRefreshTokenSupported verifies the refresh-token feature is gated off on Windows
// and allowed elsewhere.
func TestRefreshTokenSupported(t *testing.T) {
	t.Parallel()
	for _, d := range []struct {
		goos string
		want bool
	}{
		{goos: "linux", want: true},
		{goos: "darwin", want: true},
		{goos: "windows", want: false},
	} {
		t.Run(d.goos, func(t *testing.T) {
			t.Parallel()
			if got := agent.RefreshTokenSupported(d.goos); got != d.want {
				t.Fatalf("RefreshTokenSupported(%q) = %v, want %v", d.goos, got, d.want)
			}
		})
	}
}
