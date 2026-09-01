package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/spf13/cobra"
	"github.com/suzuki-shunsuke/cobra-util/cobrautil"
	"github.com/suzuki-shunsuke/ghtkn/pkg/cli/flag"
	"github.com/suzuki-shunsuke/slog-util/slogutil"
)

// TestCompleteAppNames checks that the commands taking an app name argument are wired
// to the app name completion, which the tests of pkg/cli/completion can't see: they
// drive the completion functions through a command tree of their own, so they still
// pass if a command drops its ValidArgsFunction.
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
			// name to complete.
			name: "git-credential offers no app",
			args: []string{"git-credential", "-c", configFilePath},
			want: nil,
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

// completeCommandLine runs the real command tree the way a shell does, by calling
// cobra's hidden __complete command, and returns the candidates it wrote. The empty
// last argument is the word under the cursor, which the shell passes even when the
// cursor sits on a fresh word.
func completeCommandLine(t *testing.T, args ...string) []string {
	t.Helper()
	env := &cobrautil.Env{
		Program: program,
		Version: "v1.0.0",
		Stdin:   os.Stdin,
		Getenv:  os.Getenv,
		Args:    append(append([]string{program, cobra.ShellCompRequestCmd}, args...), ""),
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
	cmd.SetOut(stdout)
	cmd.SetErr(&bytes.Buffer{})

	if err := cmd.ExecuteContext(t.Context()); err != nil {
		t.Fatalf("run the completion: %v", err)
	}

	var got []string
	for line := range strings.SplitSeq(strings.TrimSuffix(stdout.String(), "\n"), "\n") {
		// The last line carries the ShellCompDirective rather than a candidate.
		if line == "" || strings.HasPrefix(line, ":") {
			continue
		}
		got = append(got, line)
	}
	return got
}
