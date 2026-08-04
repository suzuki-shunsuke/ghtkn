package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/suzuki-shunsuke/ghtkn/pkg/cli/flag"
	"github.com/suzuki-shunsuke/slog-util/slogutil"
	"github.com/suzuki-shunsuke/urfave-cli-v3-util/urfave"
)

// completionFlag is the flag the shell completion scripts append to the command line.
// It is unexported in urfave/cli, so it is repeated here.
const completionFlag = "--generate-shell-completion"

// TestCompleteAppNames checks that the commands taking an app name argument are wired
// to the app name completion, which the tests of pkg/cli/completion can't see: they
// drive the completion functions through a command tree of their own, so they still
// pass if a command drops its ShellComplete.
func TestCompleteAppNames(t *testing.T) {
	configFilePath := filepath.Join(t.TempDir(), "ghtkn.yaml")
	if err := os.WriteFile(configFilePath, []byte(`apps:
  - name: first
    client_id: Iv1.first
  - name: second
    client_id: Iv1.second
`), 0o600); err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name string
		args []string
		want []string
	}{
		{
			name: "get",
			args: []string{"get", "-c", configFilePath},
			want: []string{"first", "second"},
		},
		{
			name: "auth",
			args: []string{"auth", "-c", configFilePath},
			want: []string{"first", "second"},
		},
		{
			name: "revoke",
			args: []string{"revoke", "-c", configFilePath},
			want: []string{"first", "second"},
		},
		{
			// revoke takes any number of app names, so it goes on completing them.
			name: "revoke drops an app already given",
			args: []string{"revoke", "-c", configFilePath, "first"},
			want: []string{"second"},
		},
		{
			// --all revokes every app and ignores the app name arguments, so offering
			// them would only make them look like they still select what is revoked.
			name: "revoke --all offers no app",
			args: []string{"revoke", "--all", "-c", configFilePath},
			want: nil,
		},
		{
			// git-credential selects its app by repository owner, so it takes no app
			// name to complete. urfave/cli's own completion answers instead.
			name: "git-credential offers no app",
			args: []string{"git-credential", "-c", configFilePath},
			want: []string{"help:Shows a list of commands or help for one command"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// The commands read GHTKN_CONFIG when -c is absent, and the completion is
			// meant to be independent of the developer's environment either way.
			t.Setenv("GHTKN_CONFIG", "")
			got := completeCommandLine(t, tt.args...)
			if diff := cmp.Diff(tt.want, got); diff != "" {
				t.Errorf("candidates mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

// completeCommandLine runs the real command tree the way a shell does, by appending
// the completion flag to the command line, and returns the candidates it wrote.
func completeCommandLine(t *testing.T, args ...string) []string {
	t.Helper()
	env := &urfave.Env{
		Program: program,
		Version: "v1.0.0",
		Stdin:   os.Stdin,
		Getenv:  os.Getenv,
	}
	// The logger writes to a file that goes away with the test: the completion must
	// log nothing, but a stray log must not reach the terminal either.
	logFile, err := os.Create(filepath.Join(t.TempDir(), "stderr"))
	if err != nil {
		t.Fatal(err)
	}
	defer logFile.Close()
	logger := slogutil.New(&slogutil.InputNew{Name: program, Version: env.Version, Out: logFile})
	cmd := newCommand(logger, env, &flag.GlobalFlags{})
	stdout := &bytes.Buffer{}
	cmd.Writer = stdout

	osArgs := append([]string{program}, args...)
	if err := cmd.Run(t.Context(), append(osArgs, completionFlag)); err != nil {
		t.Fatalf("run the command: %v", err)
	}

	out := strings.TrimSuffix(stdout.String(), "\n")
	if out == "" {
		return nil
	}
	// The completion command is unhidden by urfave-cli-v3-util, so it shows up
	// wherever subcommands are suggested. It says nothing about the app names.
	return slices.DeleteFunc(strings.Split(out, "\n"), func(line string) bool {
		return strings.HasPrefix(line, "completion:")
	})
}
