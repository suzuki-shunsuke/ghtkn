// Package completion implements shell completion of the app name arguments that
// 'ghtkn get', 'ghtkn auth', and 'ghtkn revoke' take. The names come from the
// configuration file, so the shell offers exactly the apps that ghtkn can issue a
// token for.
//
// Shell completion itself is enabled by urfave-cli-v3-util on the root command, and
// the completion scripts come from urfave/cli's hidden 'completion' command; this
// package only supplies the candidates.
package completion

import (
	"context"
	"fmt"
	"slices"
	"strings"

	"github.com/suzuki-shunsuke/ghtkn-go-sdk/ghtkn"
	"github.com/urfave/cli/v3"
)

// AppName returns a ShellCompleteFunc for a command that takes a single app name,
// such as 'ghtkn get'. configFilePath points at the -c flag's destination, which is
// read when the completion runs rather than when the command is built.
func AppName(configFilePath *string) cli.ShellCompleteFunc {
	return func(ctx context.Context, cmd *cli.Command) {
		if rest := cmd.Args().Slice(); len(rest) > 0 {
			// The app name is already given, so there is nothing left to complete. The
			// exception is a flag being completed, which urfave/cli would handle itself
			// if this command defined no ShellComplete, so hand it back.
			completeFlags(ctx, cmd, rest)
			return
		}
		writeAppNames(cmd, *configFilePath, nil)
	}
}

// AppNames is AppName for a command that takes any number of app names, namely
// 'ghtkn revoke'. Arguments already on the command line are dropped from the
// candidates, so an app is not offered twice.
func AppNames(configFilePath *string) cli.ShellCompleteFunc {
	return func(ctx context.Context, cmd *cli.Command) {
		rest := cmd.Args().Slice()
		if completeFlags(ctx, cmd, rest) {
			return
		}
		writeAppNames(cmd, *configFilePath, rest)
	}
}

// completeFlags hands the completion back to urfave/cli when a flag is being
// completed, and reports whether it did. Defining ShellComplete replaces
// DefaultCompleteWithFlags, which is what completes '-' and '--conf' otherwise, so
// every ShellComplete here has to delegate that case explicitly.
//
// The last argument is the one being completed: the completion scripts append the
// word under the cursor only when it starts with '-', and pass the words before the
// cursor as is otherwise.
func completeFlags(ctx context.Context, cmd *cli.Command, args []string) bool {
	if len(args) == 0 || !strings.HasPrefix(args[len(args)-1], "-") {
		return false
	}
	cli.DefaultCompleteWithFlags(ctx, cmd)
	return true
}

// writeAppNames writes the name of every configured app, except those in exclude, as
// completion candidates. The shell filters them by the prefix the user has typed.
//
// Errors are dropped: the output is the candidate list itself, so a message about a
// broken config file would be offered as a candidate. Nothing here reaches the
// keyring, the agent, or GitHub either, because this runs on every press of the tab
// key.
func writeAppNames(cmd *cli.Command, configFilePath string, exclude []string) {
	// LoadConfig resolves GHTKN_CONFIG itself when the path is empty. The -c flag's
	// own environment variable source can't be relied on here: urfave/cli applies it
	// after the point where it runs the completion. A missing config file is not an
	// error, and yields no candidate.
	cfg, err := ghtkn.LoadConfig(&ghtkn.InputLoadConfig{ConfigFilePath: configFilePath})
	if err != nil {
		return
	}
	w := cmd.Root().Writer
	for _, app := range cfg.Apps {
		if app == nil || app.Name == "" || slices.Contains(exclude, app.Name) {
			continue
		}
		fmt.Fprintln(w, app.Name)
	}
}
