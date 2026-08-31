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

	"github.com/spf13/cobra"
	"github.com/suzuki-shunsuke/ghtkn/pkg/cli/completion"
	"github.com/suzuki-shunsuke/ghtkn/pkg/cli/flag"
	"github.com/suzuki-shunsuke/ghtkn/pkg/config"
	"github.com/suzuki-shunsuke/ghtkn/pkg/controller/revoke"
	"github.com/suzuki-shunsuke/slog-util/slogutil"
)

// Args holds the flag and argument values for the revoke command.
type Args struct {
	*flag.GlobalFlags

	Args []string // positional arguments (raw tokens and/or app names)
	All  bool     // --all: revoke the stored tokens of every app in the config
}

// New creates a new revoke command instance with the provided logger.
// It returns a CLI command that can be registered with the main CLI application.
func New(logger *slogutil.Logger, gFlags *flag.GlobalFlags) *cobra.Command {
	args := &Args{
		GlobalFlags: gFlags,
	}
	cmd := &cobra.Command{
		Use:   "revoke [<access token | app name>...]",
		Short: "Revoke GitHub App User Access Tokens",
		Long: `Revoke GitHub App User Access Tokens via GitHub's credential revocation API and remove them from the backend.

Each argument is classified by its prefix: arguments starting with a GitHub token prefix (ghp_, github_pat_, gho_, ghu_, ghr_) are revoked directly as raw access tokens, and all other arguments are treated as app names whose stored tokens are revoked and removed from the backend.
When no argument is given, the token stored for GHTKN_APP (or the default app) is revoked.

With --all, the stored tokens of every app in the config are revoked. This is meant for incident response: when the environment running ghtkn is compromised, all stored tokens can be revoked at once. App name arguments are ignored when --all is set, but raw access tokens are still revoked as usual.`,
		Args: cobra.ArbitraryArgs,
		// Raw access tokens are arguments too, but only app names can be completed.
		ValidArgsFunction: completion.AppNames(&args.Config, ignoresAppNames),
		RunE: func(cmd *cobra.Command, positional []string) error {
			args.Args = positional
			return action(cmd.Context(), logger, args)
		},
	}
	cmd.Flags().BoolVar(&args.All, "all", false, "Revoke the stored tokens of every app in the config")
	return cmd
}

// ignoresAppNames reports whether the command line being completed makes revoke drop
// its app name arguments, which --all does (see action). Completing them there would
// only make them look like they still select what is revoked.
func ignoresAppNames(cmd *cobra.Command) bool {
	all, err := cmd.Flags().GetBool("all")
	// The flag is registered right above, so the error can only mean the command being
	// completed is not this one; offering no app name is the safe answer either way.
	return err == nil && all
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
