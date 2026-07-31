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
	sdkenv "github.com/suzuki-shunsuke/ghtkn-go-sdk/ghtkn/env"
	"github.com/suzuki-shunsuke/ghtkn/pkg/cli/docs"
	"github.com/suzuki-shunsuke/ghtkn/pkg/cli/flag"
	"github.com/suzuki-shunsuke/go-error-with-exit-code/ecerror"
	"github.com/suzuki-shunsuke/slog-error/slogerr"
	"github.com/suzuki-shunsuke/slog-util/slogutil"
	"github.com/suzuki-shunsuke/urfave-cli-v3-util/urfave"
	"github.com/urfave/cli/v3"
)

const program = "ghtkn"

// TestWithHelp checks that the exit code of an error survives the hint added to it.
// The root command disables cli.HandleExitCoder, so this is the only thing applying
// the exit codes of 'ghtkn exec' and of urfave/cli itself.
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
			err:        ecerror.Wrap(urfave.ErrSilent, 3),
			wantCode:   3,
			wantSilent: true,
		},
		{
			// cli.Exit is what an unknown command fails with.
			name:     "an exit code raised by urfave/cli is kept",
			err:      cli.Exit("no such command", 3),
			wantCode: 3,
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

type hintDocsOnVersionCase struct {
	name     string
	args     []string
	getenv   func(string) string
	stdout   string
	wantHint bool
}

// TestHintDocsOnVersion checks that the version output keeps stdout to the version
// alone, logs the docs hint to stderr, and is silenced by the log level.
// It doesn't run in parallel because it replaces the global cli.VersionPrinter.
func TestHintDocsOnVersion(t *testing.T) {
	tests := []*hintDocsOnVersionCase{
		{
			name:     "version flag",
			args:     []string{program, "-v"},
			stdout:   "ghtkn version v1.0.0\n",
			wantHint: true,
		},
		{
			name:     "long version flag",
			args:     []string{program, "--version"},
			stdout:   "ghtkn version v1.0.0\n",
			wantHint: true,
		},
		{
			name:     "version command",
			args:     []string{program, "version"},
			stdout:   "v1.0.0\n",
			wantHint: true,
		},
		{
			name:   "the log level flag silences the hint",
			args:   []string{program, "--log-level", "warn", "-v"},
			stdout: "ghtkn version v1.0.0\n",
		},
		{
			name:   "the log level environment variable silences the hint",
			args:   []string{program, "-v"},
			getenv: func(string) string { return "warn" },
			stdout: "ghtkn version v1.0.0\n",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// The log level flag reads this too, so clear it to keep the cases
			// independent of the developer's environment.
			t.Setenv(sdkenv.LogLevel, "")
			testHintDocsOnVersion(t, tt)
		})
	}
}

func testHintDocsOnVersion(t *testing.T, tt *hintDocsOnVersionCase) {
	t.Helper()
	// hintDocsOnVersion replaces the process global cli.VersionPrinter with a
	// closure over this case's logger, whose output file is gone once the case
	// ends. Restore it so that it doesn't leak into the other cases or tests.
	printer := cli.VersionPrinter
	t.Cleanup(func() {
		cli.VersionPrinter = printer
	})
	getenv := tt.getenv
	if getenv == nil {
		getenv = func(string) string { return "" }
	}
	logFile, err := os.Create(filepath.Join(t.TempDir(), "stderr"))
	if err != nil {
		t.Fatal(err)
	}
	defer logFile.Close()

	stdout := &bytes.Buffer{}
	env := &urfave.Env{Program: program, Version: "v1.0.0", Getenv: getenv}
	gFlags := &flag.GlobalFlags{}
	cmd := urfave.Command(env, &cli.Command{
		Name:   program,
		Writer: stdout,
		Flags:  []cli.Flag{flag.LogLevel(&gFlags.LogLevel)},
	})
	logger := slogutil.New(&slogutil.InputNew{Name: program, Version: env.Version, Out: logFile})
	hintDocsOnVersion(cmd, logger, env, gFlags)

	if err := cmd.Run(t.Context(), tt.args); err != nil {
		t.Fatal(err)
	}
	if diff := cmp.Diff(tt.stdout, stdout.String()); diff != "" {
		t.Errorf("stdout is unexpected (-want +got):\n%s", diff)
	}
	logs, err := os.ReadFile(logFile.Name())
	if err != nil {
		t.Fatal(err)
	}
	if gotHint := strings.Contains(string(logs), docs.Hint); gotHint != tt.wantHint {
		t.Errorf("the docs hint is logged = %v, want %v: %s", gotHint, tt.wantHint, logs)
	}
}
