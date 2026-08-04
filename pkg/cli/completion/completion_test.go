package completion_test

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/suzuki-shunsuke/ghtkn/pkg/cli/completion"
	"github.com/urfave/cli/v3"
)

// completionFlag is the flag the completion scripts append to the command line. It is
// unexported in urfave/cli, so it is repeated here.
const completionFlag = "--generate-shell-completion"

const configContent = `apps:
  - name: first
    client_id: Iv1.first
  - name: second
    client_id: Iv1.second
`

// writeConfig writes a config file with two apps and returns its path.
func writeConfig(t *testing.T) string {
	t.Helper()
	p := filepath.Join(t.TempDir(), "ghtkn.yaml")
	if err := os.WriteFile(p, []byte(configContent), 0o600); err != nil {
		t.Fatalf("write a config file: %v", err)
	}
	return p
}

// complete runs the completion the way a shell does: it appends the completion flag to
// the arguments and returns what the command wrote as candidates.
//
// The command tree mirrors the real one closely enough for the completion path:
// completion is enabled on the root (urfave-cli-v3-util does that for ghtkn), and the
// subcommand carries the -c flag whose destination the ShellCompleteFunc reads.
func complete(t *testing.T, shellComplete func(*string) cli.ShellCompleteFunc, args ...string) string {
	t.Helper()
	configFilePath := ""
	buf := &bytes.Buffer{}
	cmd := &cli.Command{
		Name:                  "ghtkn",
		EnableShellCompletion: true,
		Writer:                buf,
		Commands: []*cli.Command{
			{
				Name: "target",
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:        "config",
						Aliases:     []string{"c"},
						Usage:       "configuration file path",
						Destination: &configFilePath,
					},
				},
				Arguments: []cli.Argument{
					&cli.StringArgs{Name: "app-name", Max: -1},
				},
				ShellComplete: shellComplete(&configFilePath),
			},
		},
	}
	osArgs := append([]string{"ghtkn", "target"}, args...)
	if err := cmd.Run(t.Context(), append(osArgs, completionFlag)); err != nil {
		t.Fatalf("run the command: %v", err)
	}
	return buf.String()
}

// lines splits completion output into candidates. The empty output is no candidate,
// which strings.Split would report as one empty candidate.
func lines(s string) []string {
	s = strings.TrimSuffix(s, "\n")
	if s == "" {
		return nil
	}
	return strings.Split(s, "\n")
}

func TestAppName(t *testing.T) {
	t.Parallel()
	configFilePath := writeConfig(t)

	tests := []struct {
		name string
		args []string
		want []string
	}{
		{
			name: "every app is a candidate",
			args: []string{"-c", configFilePath},
			want: []string{"first", "second"},
		},
		{
			// The command takes one app name, so nothing is left to complete once it is
			// given. The shell drops the word being typed, so an argument here is a
			// complete one rather than a prefix.
			name: "no candidate once the app name is given",
			args: []string{"-c", configFilePath, "first"},
			want: nil,
		},
		{
			// Defining ShellComplete replaces urfave/cli's own flag completion, so the
			// command must hand that case back to it.
			name: "flags are still completed",
			args: []string{"-c", configFilePath, "-"},
			want: []string{"--config:configuration file path", "--help:show help"},
		},
		{
			// '-c <TAB>' and a half-typed '-c' arrive identically: the completion scripts
			// append the word under the cursor only when it starts with '-'. So a flag
			// value position can only be completed as a flag name, which is what
			// urfave/cli does for every command that defines no ShellComplete.
			name: "a flag value position is completed as a flag",
			args: []string{"-c"},
			want: []string{"--config:configuration file path"},
		},
		{
			name: "a missing config file yields no candidate",
			args: []string{"-c", filepath.Join(t.TempDir(), "absent.yaml")},
			want: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := lines(complete(t, completion.AppName, tt.args...))
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
		name string
		args []string
		want []string
	}{
		{
			name: "every app is a candidate",
			args: []string{"-c", configFilePath},
			want: []string{"first", "second"},
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
		{
			name: "flags are still completed",
			args: []string{"-c", configFilePath, "-"},
			want: []string{"--config:configuration file path", "--help:show help"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := lines(complete(t, completion.AppNames, tt.args...))
			if diff := cmp.Diff(tt.want, got); diff != "" {
				t.Errorf("candidates mismatch (-want +got):\n%s", diff)
			}
		})
	}
}
