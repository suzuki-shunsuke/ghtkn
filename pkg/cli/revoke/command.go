// Package revoke implements the 'ghtkn revoke' command.
// It revokes GitHub App User Access Tokens via GitHub's credential revocation API
// and removes the revoked tokens from the backend. This is useful when a token has
// been leaked and must be revoked quickly.
//
// Each positional argument is classified by its prefix: arguments that start with a
// GitHub token prefix (ghp_, github_pat_, gho_, ghu_, ghr_) are treated as raw
// access tokens and revoked directly; all other arguments are treated as app names,
// whose stored tokens are revoked and removed from the backend. When no argument is
// given, the token stored for GHTKN_APP (or the default app) is revoked. When only
// raw tokens are given, GHTKN_APP and the default app are NOT used, so revoking a
// raw token never revokes an unrelated app's stored token.
//
// The --all flag revokes the stored tokens of every app in the config at once,
// for incident response when the environment running ghtkn is compromised. With
// --all, app name arguments are ignored, but raw access tokens are still revoked.
package revoke

import (
	"context"
	"fmt"
	"strings"

	"github.com/suzuki-shunsuke/ghtkn/pkg/cli/flag"
	"github.com/suzuki-shunsuke/ghtkn/pkg/config"
	"github.com/suzuki-shunsuke/ghtkn/pkg/controller/revoke"
	"github.com/suzuki-shunsuke/slog-util/slogutil"
	"github.com/urfave/cli/v3"
)

// Args holds the flag and argument values for the revoke command.
type Args struct {
	*flag.GlobalFlags

	Args []string // positional arguments (raw tokens and/or app names)
	All  bool     // --all: revoke the stored tokens of every app in the config
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
		ArgsUsage: "[<access token | app name>...]",
		Description: `Revoke GitHub App User Access Tokens via GitHub's credential revocation API and remove them from the backend.

Each argument is classified by its prefix: arguments starting with a GitHub token prefix (ghp_, github_pat_, gho_, ghu_, ghr_) are revoked directly as raw access tokens, and all other arguments are treated as app names whose stored tokens are revoked and removed from the backend.
When no argument is given, the token stored for GHTKN_APP (or the default app) is revoked.

With --all, the stored tokens of every app in the config are revoked. This is meant for incident response: when the environment running ghtkn is compromised, all stored tokens can be revoked at once. App name arguments are ignored when --all is set, but raw access tokens are still revoked as usual.`,
		Action: func(ctx context.Context, _ *cli.Command) error {
			return action(ctx, logger, args)
		},
		Flags: []cli.Flag{
			flag.LogLevel(&args.LogLevel),
			flag.Config(&args.Config),
			&cli.BoolFlag{
				Name:        "all",
				Usage:       "Revoke the stored tokens of every app in the config",
				Destination: &args.All,
			},
		},
		Arguments: []cli.Argument{
			&cli.StringArgs{
				Name:        "token-or-app",
				Min:         0,
				Max:         -1,
				Destination: &args.Args,
			},
		},
	}
}

// classify splits positional arguments into raw access tokens and app names by
// their prefix (see isToken). An argument that looks like a GitHub credential is a
// token; any other argument is an app name.
func classify(positional []string) (tokens, appNames []string) {
	for _, a := range positional {
		if isToken(a) {
			tokens = append(tokens, a)
		} else {
			appNames = append(appNames, a)
		}
	}
	return tokens, appNames
}

// isToken reports whether s looks like a GitHub credential based on its prefix.
// A positional argument that starts with one of these prefixes is treated as a raw
// access token rather than an app name.
func isToken(s string) bool {
	tokenPrefixes := []string{
		"ghp_",        // Personal access tokens (classic)
		"github_pat_", // Fine-grained personal access tokens
		"gho_",        // OAuth app access tokens
		"ghu_",        // User-to-server tokens from GitHub Apps
		"ghr_",        // Refresh tokens from GitHub Apps
	}
	for _, p := range tokenPrefixes {
		if strings.HasPrefix(s, p) {
			return true
		}
	}
	return false
}

// action revokes the requested tokens.
// Positional arguments are classified into raw tokens (revoked directly) and app
// names (whose stored tokens are revoked via the SDK). When no argument is given,
// the SDK falls back to GHTKN_APP / the default app; when only raw tokens are
// given, the SDK is not called so a raw token never touches an unrelated app's
// stored token.
func action(ctx context.Context, logger *slogutil.Logger, args *Args) error {
	if err := logger.SetLevel(args.LogLevel); err != nil {
		return fmt.Errorf("set log level: %w", err)
	}

	tokens, appNames := classify(args.Args)
	if args.All {
		// --all revokes every app's stored token, so explicit app names are ignored.
		// Raw tokens are still revoked.
		appNames = nil
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
		Tokens:         tokens,
		AppNames:       appNames,
		ConfigFilePath: p,
		All:            args.All,
	})
}
