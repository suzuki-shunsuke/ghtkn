package completion_test

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/spf13/cobra"
	"github.com/suzuki-shunsuke/ghtkn/pkg/cli/completion"
)

const configContent = `apps:
  - name: first
    client_id: Iv1.first
  - name: second
    client_id: Iv1.second
`

// writeConfig writes a config file with two apps and returns its path.
func writeConfig(t *testing.T) string {
	t.Helper()
	return writeConfigContent(t, configContent)
}

// writeConfigContent writes content as a config file and returns its path.
func writeConfigContent(t *testing.T, content string) string {
	t.Helper()
	p := filepath.Join(t.TempDir(), "ghtkn.yaml")
	if err := os.WriteFile(p, []byte(content), 0o600); err != nil {
		t.Fatalf("write a config file: %v", err)
	}
	return p
}

// never is a predicate for a command line that ignores nothing.
func never(*cobra.Command) bool { return false }

// complete runs the completion the way a shell does, by calling cobra's hidden
// __complete command, and returns the candidates it wrote.
//
// The command tree mirrors the real one closely enough for the completion path: the
// subcommand carries the -c flag whose destination the completion function reads.
// toComplete is the word under the cursor, which the shell passes as the last
// argument and which is empty when the cursor sits on a fresh word.
func complete(t *testing.T, fn func(*string) completion.Func, toComplete string, args ...string) []string {
	t.Helper()
	configFilePath := ""
	target := &cobra.Command{
		Use:               "target",
		Args:              cobra.ArbitraryArgs,
		ValidArgsFunction: fn(&configFilePath),
		RunE:              func(*cobra.Command, []string) error { return nil },
	}
	target.Flags().StringVarP(&configFilePath, "config", "c", "", "configuration file path")
	root := &cobra.Command{Use: "ghtkn", SilenceErrors: true, SilenceUsage: true}
	root.AddCommand(target)

	stdout := &bytes.Buffer{}
	root.SetOut(stdout)
	root.SetErr(&bytes.Buffer{})
	root.SetArgs(append(append([]string{cobra.ShellCompRequestCmd, "target"}, args...), toComplete))
	if err := root.ExecuteContext(t.Context()); err != nil {
		t.Fatalf("run the completion: %v", err)
	}
	return candidates(stdout.String())
}

// candidates splits the output of __complete into the candidate list, dropping the
// trailing line that carries the ShellCompDirective rather than a candidate.
func candidates(out string) []string {
	lines := strings.Split(strings.TrimSuffix(out, "\n"), "\n")
	var got []string
	for _, line := range lines {
		if line == "" || strings.HasPrefix(line, ":") {
			continue
		}
		got = append(got, line)
	}
	return got
}

func TestAppName(t *testing.T) { //nolint:funlen // The length is the table of cases.
	t.Parallel()
	configFilePath := writeConfig(t)

	tests := []struct {
		name       string
		args       []string
		toComplete string
		want       []string
	}{
		{
			name: "every app is a candidate",
			args: []string{"-c", configFilePath},
			want: []string{"first", "second"},
		},
		{
			// The shell filters the candidates too, but a completion function is
			// expected to narrow them itself, and cobra offers what it is given.
			name:       "the candidates are narrowed to the typed prefix",
			args:       []string{"-c", configFilePath},
			toComplete: "f",
			want:       []string{"first"},
		},
		{
			// The command takes one app name, so nothing is left to complete once it is
			// given. The word under the cursor is passed separately, so an argument here
			// is a complete one rather than a prefix.
			name: "no candidate once the app name is given",
			args: []string{"-c", configFilePath, "first"},
			want: nil,
		},
		{
			// cobra completes the flags itself, before it asks for argument candidates,
			// so nothing in this package has to hand that case back the way it did with
			// urfave/cli.
			name:       "flags are completed by cobra",
			toComplete: "-",
			want: []string{
				"--config\tconfiguration file path", "-c\tconfiguration file path",
				"--help\thelp for target", "-h\thelp for target",
			},
		},
		{
			name: "a missing config file yields no candidate",
			args: []string{"-c", filepath.Join(t.TempDir(), "absent.yaml")},
			want: nil,
		},
		{
			// A config file that can't be read leaves the shell with nothing rather than
			// with an error message offered as a candidate.
			name: "an unreadable config file yields no candidate",
			args: []string{"-c", t.TempDir()},
			want: nil,
		},
		{
			// Config.Validate rejects a duplicate name, but nothing validates the config
			// here, and the same candidate twice is noise rather than a report of it.
			name: "a duplicate app name is offered once",
			args: []string{"-c", writeConfigContent(t, `apps:
  - name: dup
    client_id: Iv1.one
  - name: dup
    client_id: Iv1.two
`)},
			want: []string{"dup"},
		},
		{
			// An app without a name is no candidate: ghtkn can't be asked for it.
			name: "an app without a name is skipped",
			args: []string{"-c", writeConfigContent(t, `apps:
  - name: ""
    client_id: Iv1.nameless
  - name: named
    client_id: Iv1.named
`)},
			want: []string{"named"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := complete(t, completion.AppName, tt.toComplete, tt.args...)
			if diff := cmp.Diff(tt.want, got); diff != "" {
				t.Errorf("candidates mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestAppNames(t *testing.T) {
	t.Parallel()
	configFilePath := writeConfig(t)

	tests := []struct {
		name       string
		args       []string
		toComplete string
		ignored    bool
		want       []string
	}{
		{
			name: "every app is a candidate",
			args: []string{"-c", configFilePath},
			want: []string{"first", "second"},
		},
		{
			// 'ghtkn revoke --all' revokes every app and ignores its app name arguments,
			// so completing them would only make them look like they still matter.
			name:    "no candidate where the argument is ignored",
			args:    []string{"-c", configFilePath},
			ignored: true,
			want:    nil,
		},
		{
			// revoke takes any number of app names, so the completion goes on after the
			// first one; offering it again would only produce a duplicate argument.
			name: "an app already given is dropped",
			args: []string{"-c", configFilePath, "first"},
			want: []string{"second"},
		},
		{
			// revoke also takes raw access tokens, which are no app name and so exclude
			// nothing.
			name: "a raw token excludes no app",
			args: []string{"-c", configFilePath, "ghu_xxx"},
			want: []string{"first", "second"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			ignored := never
			if tt.ignored {
				ignored = func(*cobra.Command) bool { return true }
			}
			fn := func(configFilePath *string) completion.Func {
				return completion.AppNames(configFilePath, ignored)
			}
			got := complete(t, fn, tt.toComplete, tt.args...)
			if diff := cmp.Diff(tt.want, got); diff != "" {
				t.Errorf("candidates mismatch (-want +got):\n%s", diff)
			}
		})
	}
}
