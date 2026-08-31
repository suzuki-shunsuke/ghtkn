package exec

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/spf13/cobra"
	"github.com/suzuki-shunsuke/ghtkn/pkg/cli/flag"
	"github.com/suzuki-shunsuke/slog-util/slogutil"
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
			args: []string{"exec"},
		},
		{
			name: "no separator",
			args: []string{"exec", "gh"},
		},
		{
			name: "an invalid -e",
			args: []string{"exec", "-e", "GH_TOKEN=my-app", "--", "gh"},
		},
		{
			name: "the same environment variable twice",
			args: []string{"exec", "-e", "A", "-e", "A:app", "--", "gh"},
		},
		{
			name: "an invalid min expiration",
			args: []string{"exec", "-m", "invalid", "--", "gh"},
		},
		{
			name: "an invalid log level",
			args: []string{"exec", "--log-level", "invalid", "--", "gh"},
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
			gFlags := &flag.GlobalFlags{}
			cmd := &cobra.Command{
				Use:           "ghtkn",
				SilenceErrors: true,
				SilenceUsage:  true,
			}
			// The real root registers these as persistent flags, and 'exec' reads
			// --log-level through them.
			flag.LogLevel(cmd.PersistentFlags(), &gFlags.LogLevel)
			flag.Config(cmd.PersistentFlags(), &gFlags.Config)
			cmd.AddCommand(New(logger, gFlags))
			cmd.SetOut(&bytes.Buffer{})
			cmd.SetErr(&bytes.Buffer{})
			cmd.SetArgs(tt.args)
			if err := cmd.ExecuteContext(t.Context()); err == nil {
				t.Fatal("an error must be returned")
			}
		})
	}
}
