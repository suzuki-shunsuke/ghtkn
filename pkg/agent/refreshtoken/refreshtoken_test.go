package refreshtoken_test

import (
	"testing"

	"github.com/suzuki-shunsuke/ghtkn/pkg/agent/refreshtoken"
)

// TestSupported verifies the refresh-token feature is gated off on Windows and allowed
// elsewhere.
func TestSupported(t *testing.T) {
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
			if got := refreshtoken.Supported(d.goos); got != d.want {
				t.Fatalf("Supported(%q) = %v, want %v", d.goos, got, d.want)
			}
		})
	}
}
