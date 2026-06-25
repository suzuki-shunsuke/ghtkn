// Package revoke implements the 'ghtkn revoke' command.
// It revokes GitHub App User Access Tokens via GitHub's credential revocation API
// and removes the revoked tokens from the backend. Tokens to revoke come from the
// app's stored token (selected by the app-name argument) and/or raw tokens passed
// with --token. This is useful when a token has been leaked and must be revoked
// quickly.
//
// Raw --token values are revoked directly. When neither --token nor an app name is
// given, it falls back to the app selected by GHTKN_APP (or the default app). When
// only --token is given, GHTKN_APP and the default app are NOT used, so revoking a
// raw token never revokes an unrelated app's stored token.
package revoke

import (
	"context"
	"fmt"

	"github.com/suzuki-shunsuke/ghtkn/pkg/cli/flag"
	"github.com/suzuki-shunsuke/ghtkn/pkg/config"
	"github.com/suzuki-shunsuke/ghtkn/pkg/controller/revoke"
	"github.com/suzuki-shunsuke/slog-util/slogutil"
	"github.com/urfave/cli/v3"
)

// Args holds the flag and argument values for the revoke command.
type Args struct {
	*flag.GlobalFlags

	Tokens  []string // --token / -t (repeatable)
	AppName string   // positional argument
}

// New creates a new revoke command instance with the provided logger.
// It returns a CLI command that can be registered with the main CLI application.
func New(logger *slogutil.Logger, gFlags *flag.GlobalFlags) *cli.Command {
	args := &Args{
		GlobalFlags: gFlags,
	}
	return &cli.Command{
		Name:      "revoke",
		Usage:     "Revoke GitHub App User Access Tokens",
		ArgsUsage: "[<app name>]",
		Description: `Revoke GitHub App User Access Tokens via GitHub's credential revocation API and remove them from the backend.

The tokens to revoke are the union of the token stored for the given app and any tokens passed with --token.
When neither --token nor an app name is given, the token stored for GHTKN_APP (or the default app) is revoked.`,
		Action: func(ctx context.Context, _ *cli.Command) error {
			return action(ctx, logger, args)
		},
		Flags: []cli.Flag{
			flag.LogLevel(&args.LogLevel),
			flag.Config(&args.Config),
			&cli.StringSliceFlag{
				Name:        "token",
				Aliases:     []string{"t"},
				Usage:       "an access token to revoke (can be specified multiple times)",
				Destination: &args.Tokens,
			},
		},
		Arguments: []cli.Argument{
			&cli.StringArg{
				Name:        "app-name",
				Destination: &args.AppName,
			},
		},
	}
}

// action revokes the requested tokens.
// Raw --token values are revoked directly by the controller. The app name is
// passed to the SDK to revoke the app's stored token. When neither --token nor an
// app name is given, the SDK falls back to GHTKN_APP / the default app; when only
// --token is given, the SDK is not called so a raw token never touches an
// unrelated app's stored token.
func action(ctx context.Context, logger *slogutil.Logger, args *Args) error {
	if err := logger.SetLevel(args.LogLevel); err != nil {
		return fmt.Errorf("set log level: %w", err)
	}

	input, err := revoke.NewInput()
	if err != nil {
		return fmt.Errorf("create the controller input: %w", err)
	}
	p, err := config.ResolvePath(args.Config)
	if err != nil {
		return err //nolint:wrapcheck
	}
	return revoke.New(input).Run(ctx, logger.Logger, &revoke.InputRevoke{ //nolint:wrapcheck
		Tokens:         args.Tokens,
		AppName:        args.AppName,
		ConfigFilePath: p,
	})
}
