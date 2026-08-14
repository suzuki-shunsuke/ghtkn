// Package auth implements the 'ghtkn auth' command.
// It authenticates to GitHub and caches a GitHub App User Access Token without
// printing it to stdout. It regenerates the token regardless of any cached token,
// so that running it proactively refreshes the cached token before it expires.
// Regeneration normally runs the OAuth device flow; with the agent backend and
// refresh enabled it is a silent refresh using the stored refresh token when one
// is available, and the device flow runs only when no usable refresh token exists.
// It is the only command that runs the device flow, so no other command can start
// one on the user's behalf; that is why it is the only interactive one.
// Unlike 'ghtkn get', it does not accept the --min-expiration flag nor read
// GHTKN_MIN_EXPIRATION; that knob is reserved for 'ghtkn get'.
package auth

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"
	"github.com/suzuki-shunsuke/ghtkn-go-sdk/ghtkn"
	"github.com/suzuki-shunsuke/ghtkn/pkg/cli/completion"
	"github.com/suzuki-shunsuke/ghtkn/pkg/cli/flag"
	"github.com/suzuki-shunsuke/ghtkn/pkg/clipboard"
	"github.com/suzuki-shunsuke/ghtkn/pkg/config"
	"github.com/suzuki-shunsuke/ghtkn/pkg/controller/auth"
	"github.com/suzuki-shunsuke/slog-util/slogutil"
)

// Args holds the flag and argument values for the auth command.
type Args struct {
	*flag.GlobalFlags

	AppName   string // positional argument
	Clipboard bool
}

// New creates a new auth command instance with the provided logger.
// It returns a CLI command that can be registered with the main CLI application.
func New(logger *slogutil.Logger, gFlags *flag.GlobalFlags) *cobra.Command {
	args := &Args{
		GlobalFlags: gFlags,
	}
	cmd := &cobra.Command{
		Use:   "auth [<app-name>]",
		Short: "Authenticate to GitHub and cache an access token without outputting it",
		Long: `Authenticate to GitHub and cache an access token without printing it.

Unlike 'ghtkn get', this regenerates the token regardless of any cached token, so
running it proactively refreshes the cached token before it expires. Regeneration
normally runs the OAuth device flow; with the agent backend and refresh enabled, a
valid stored refresh token is used to refresh silently and the device flow runs
only when no usable refresh token exists. This is the only command that runs the
device flow, so it is only ever started by you, never on your behalf by 'ghtkn get',
'ghtkn exec', the Git credential helper, or a tool built on the ghtkn SDK. Run it in
your own interactive terminal: it waits for you to enter a one-time code.
It does not accept --min-expiration. Use --clipboard to copy the one-time code to the
clipboard. If an app name is omitted, GHTKN_APP or the default app is used.

$ ghtkn auth
$ ghtkn auth my-app
$ ghtkn auth -p`,
		Args:              cobra.MaximumNArgs(1),
		ValidArgsFunction: completion.AppName(&args.Config),
		RunE: func(cmd *cobra.Command, positional []string) error {
			if len(positional) > 0 {
				args.AppName = positional[0]
			}
			return action(cmd.Context(), cmd, logger, args)
		},
	}
	flag.Clipboard(cmd.Flags(), &args.Clipboard)
	return cmd
}

// action authenticates to GitHub and caches an access token without printing it.
// The SDK's Auth is the only entry point that runs the device flow, and it always
// regenerates the token, so neither of those is a knob this command has to set.
func action(ctx context.Context, cmd *cobra.Command, logger *slogutil.Logger, args *Args) error {
	if err := logger.SetLevel(args.LogLevel); err != nil {
		return fmt.Errorf("set log level: %w", err)
	}
	inputAuth := &ghtkn.InputAuth{
		ConfigFilePath: args.Config,
	}
	if args.AppName != "" {
		inputAuth.AppName = args.AppName
	}

	input, err := auth.NewInput()
	if err != nil {
		return fmt.Errorf("create the controller input: %w", err)
	}
	// Always provide the clipboard implementation; whether to actually copy is
	// resolved by the SDK from the flag override below, GHTKN_CLIPBOARD, and the
	// config's clipboard.enable.
	input.Client.SetCopyOnetimeCodeToClipboard(clipboard.New())
	// Pass the clipboard override only when --clipboard is explicitly set (including
	// --clipboard=false) so it takes precedence over the env var and config.
	if cmd.Flags().Changed("clipboard") {
		inputAuth.Clipboard = &args.Clipboard
	}
	p, err := config.ResolvePath(inputAuth.ConfigFilePath)
	if err != nil {
		return err //nolint:wrapcheck
	}
	inputAuth.ConfigFilePath = p
	return auth.New(input).Run(ctx, logger.Logger, inputAuth) //nolint:wrapcheck
}
