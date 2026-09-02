package cli

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/suzuki-shunsuke/cobra-util/cobrautil"
	sdkenv "github.com/suzuki-shunsuke/ghtkn-go-sdk/ghtkn/env"
	"github.com/suzuki-shunsuke/ghtkn/pkg/cli/docs"
	"github.com/suzuki-shunsuke/ghtkn/pkg/cli/flag"
	"github.com/suzuki-shunsuke/go-error-with-exit-code/ecerror"
	"github.com/suzuki-shunsuke/slog-error/slogerr"
	"github.com/suzuki-shunsuke/slog-util/slogutil"
)

const program = "ghtkn"

// TestWithHelp checks that the exit code of an error survives the hint added to it.
func TestWithHelp(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name       string
		err        error
		wantCode   int
		wantSilent bool
	}{
		{
			name:     "an error without an exit code exits 1",
			err:      errors.New("something failed"),
			wantCode: 1,
		},
		{
			name:     "an exit code is kept",
			err:      ecerror.Wrap(errors.New("something failed"), 111),
			wantCode: 111,
		},
		{
			// slogerr.With collapses the chain to the innermost error holding
			// attributes, which drops the wrapper holding the exit code. The code has
			// to be read before that happens.
			name:     "an exit code survives an error holding attributes",
			err:      ecerror.Wrap(fmt.Errorf("get an access token: %w", slogerr.With(errors.New("no app"), "app_name", "foo")), 111),
			wantCode: 111,
		},
		{
			// 'ghtkn exec' propagates the exit code of the command it ran without
			// logging a failure of ghtkn's own.
			name:       "a silent error stays silent",
			err:        ecerror.Wrap(cobrautil.ErrSilent, 3),
			wantCode:   3,
			wantSilent: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			err := withHelp(tt.err)
			if code := ecerror.GetExitCode(err); code != tt.wantCode {
				t.Errorf("the exit code is %d, want %d", code, tt.wantCode)
			}
			if silent := err.Error() == ""; silent != tt.wantSilent {
				t.Errorf("the error is silent = %v, want %v: %v", silent, tt.wantSilent, err)
			}
		})
	}
}

// TestHintDocsOnVersion checks that the version output keeps stdout to the version
// alone, logs the docs hint to stderr, and is silenced by the log level. It runs the
// real command tree, because the wiring of the hint into --version, -v, and the
// version command is the thing worth checking.
func TestHintDocsOnVersion(t *testing.T) { //nolint:funlen // The length is the table of cases.
	tests := []struct {
		name     string
		args     []string
		logLevel string
		stdout   string
		wantHint bool
	}{
		{
			name:     "version flag",
			args:     []string{"-v"},
			stdout:   "ghtkn version v1.0.0\n",
			wantHint: true,
		},
		{
			name:     "long version flag",
			args:     []string{"--version"},
			stdout:   "ghtkn version v1.0.0\n",
			wantHint: true,
		},
		{
			name:     "version command",
			args:     []string{"version"},
			stdout:   "v1.0.0\n",
			wantHint: true,
		},
		{
			name:   "the log level flag silences the hint",
			args:   []string{"--log-level", "warn", "-v"},
			stdout: "ghtkn version v1.0.0\n",
		},
		{
			// The version is printed by the root command's action, which runs after the
			// environment variables have been applied to the flags, so GHTKN_LOG_LEVEL
			// reaches the hint the same way the flag does.
			name:     "the log level environment variable silences the hint",
			args:     []string{"-v"},
			logLevel: "warn",
			stdout:   "ghtkn version v1.0.0\n",
		},
		{
			name:     "the log level environment variable silences the hint of the version command",
			args:     []string{"version"},
			logLevel: "warn",
			stdout:   "v1.0.0\n",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// The log level flag reads this too, so set it explicitly to keep the cases
			// independent of the developer's environment.
			t.Setenv(sdkenv.LogLevel, tt.logLevel)

			logFile, err := os.Create(filepath.Join(t.TempDir(), "stderr"))
			if err != nil {
				t.Fatal(err)
			}
			defer logFile.Close()

			env := &cobrautil.Env{
				Program: program,
				Version: "v1.0.0",
				Getenv:  os.Getenv,
				Args:    append([]string{program}, tt.args...),
			}
			logger := slogutil.New(&slogutil.InputNew{Name: program, Version: env.Version, Out: logFile})
			cmd := newCommand(logger, env, &flag.GlobalFlags{})
			stdout := &bytes.Buffer{}
			cmd.SetOut(stdout)
			cmd.SetErr(&bytes.Buffer{})

			if err := cmd.ExecuteContext(t.Context()); err != nil {
				t.Fatal(err)
			}
			if diff := cmp.Diff(tt.stdout, stdout.String()); diff != "" {
				t.Errorf("stdout is unexpected (-want +got):\n%s", diff)
			}
			logs, err := os.ReadFile(logFile.Name())
			if err != nil {
				t.Fatal(err)
			}
			if gotHint := strings.Contains(string(logs), docs.Hint()); gotHint != tt.wantHint {
				t.Errorf("the docs hint is logged = %v, want %v: %s", gotHint, tt.wantHint, logs)
			}
		})
	}
}
