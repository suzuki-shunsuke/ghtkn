package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	sdkenv "github.com/suzuki-shunsuke/ghtkn-go-sdk/ghtkn/env"
	"github.com/suzuki-shunsuke/ghtkn/pkg/cli/docs"
	"github.com/suzuki-shunsuke/ghtkn/pkg/cli/flag"
	"github.com/suzuki-shunsuke/slog-util/slogutil"
	"github.com/suzuki-shunsuke/urfave-cli-v3-util/urfave"
	"github.com/urfave/cli/v3"
)

const program = "ghtkn"

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
