package exec

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/suzuki-shunsuke/ghtkn/pkg/cli/flag"
	"github.com/suzuki-shunsuke/slog-util/slogutil"
	"github.com/suzuki-shunsuke/urfave-cli-v3-util/urfave"
	"github.com/urfave/cli/v3"
)

// TestNew_invalid checks the command line validation, which runs before any
// dependency is created. Only failing cases are covered: running a command
// successfully would need a backend and a GitHub App.
func TestNew_invalid(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		args []string
	}{
		{
			name: "no command",
			args: []string{"ghtkn", "exec"},
		},
		{
			name: "no separator",
			args: []string{"ghtkn", "exec", "gh"},
		},
		{
			name: "an invalid -e",
			args: []string{"ghtkn", "exec", "-e", "GH_TOKEN=my-app", "--", "gh"},
		},
		{
			name: "the same environment variable twice",
			args: []string{"ghtkn", "exec", "-e", "A", "-e", "A:app", "--", "gh"},
		},
		{
			name: "an invalid min expiration",
			args: []string{"ghtkn", "exec", "-m", "invalid", "--", "gh"},
		},
		{
			name: "an invalid log level",
			args: []string{"ghtkn", "exec", "--log-level", "invalid", "--", "gh"},
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
			logger := slogutil.New(&slogutil.InputNew{Name: "ghtkn", Out: logFile})
			env := &urfave.Env{
				Program: "ghtkn",
				Args:    tt.args,
				Getenv:  func(string) string { return "" },
			}
			cmd := &cli.Command{
				Name:     "ghtkn",
				Writer:   &bytes.Buffer{},
				Commands: []*cli.Command{New(logger, env, &flag.GlobalFlags{})},
			}
			if err := cmd.Run(t.Context(), tt.args); err == nil {
				t.Fatal("an error must be returned")
			}
		})
	}
}
