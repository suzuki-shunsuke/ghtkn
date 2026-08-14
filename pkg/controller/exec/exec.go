package exec

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"github.com/suzuki-shunsuke/ghtkn-go-sdk/ghtkn"
	"github.com/suzuki-shunsuke/ghtkn/pkg/cobrautil"
	"github.com/suzuki-shunsuke/go-error-with-exit-code/ecerror"
	"github.com/suzuki-shunsuke/slog-error/slogerr"
)

const (
	// exitCodeTokenFailure is the exit code when an access token can't be created or
	// read and --continue-on-error isn't set. The command isn't run at all.
	exitCodeTokenFailure = 111
	// exitCodeInterrupted follows the shell convention for a process interrupted by
	// SIGINT: 128 plus the signal number.
	exitCodeInterrupted = 130
)

// Run gets an access token for each requested environment variable and runs the
// command with them.
//
// The exit code is the command's own, so it is carried by the returned error rather
// than reported separately. A failure of the command itself is silent because the
// command has already reported it; only failures of ghtkn are logged.
func (c *Controller) Run(ctx context.Context, logger *slog.Logger, input *InputRun) error {
	if len(input.Command) == 0 || input.Command[0] == "" {
		return errors.New("a command to run is required")
	}
	vars, err := c.tokens(ctx, logger, input)
	if err != nil {
		return err
	}
	code, err := c.input.Runner.Run(logger, buildEnv(c.input.Environ(), vars), input.Command[0], input.Command[1:]...)
	if err != nil {
		err = fmt.Errorf("run the command: %w", err)
		if code == 0 {
			return err
		}
		return ecerror.Wrap(err, code)
	}
	if code != 0 {
		// The command has already reported its own failure, so ghtkn adds nothing but
		// the exit code. cobrautil.ErrSilent has an empty message, which keeps ghtkn from
		// logging an error of its own.
		return ecerror.Wrap(cobrautil.ErrSilent, code)
	}
	return nil
}

// envVar is an environment variable and the access token it carries.
type envVar struct {
	name  string
	value string
}

// tokens gets an access token for each environment variable, in the order the
// variables were given.
//
// Tokens are acquired one at a time, never concurrently, so a failure is reported
// against the '-e' that caused it and in the order they were given.
func (c *Controller) tokens(ctx context.Context, logger *slog.Logger, input *InputRun) ([]*envVar, error) {
	// Tokens are keyed by the requested app name, an empty string meaning the app
	// ghtkn selects automatically, so an app is asked for only once however many
	// environment variables are bound to it. The SDK caches nothing between calls, so
	// asking twice would repeat the whole backend request for a token already in hand.
	tokens := make(map[string]string, len(input.EnvVars))
	failed := make(map[string]struct{}, len(input.EnvVars))
	vars := make([]*envVar, 0, len(input.EnvVars))
	for _, ev := range input.EnvVars {
		token, ok := tokens[ev.AppName]
		if !ok {
			if _, f := failed[ev.AppName]; f {
				// The app already failed and --continue-on-error is set, so don't ask
				// for it again.
				continue
			}
			t, err := c.token(ctx, logger, input.InputGet, ev)
			if err != nil {
				if err := handleTokenError(logger, input.ContinueOnError, err); err != nil {
					return nil, err
				}
				failed[ev.AppName] = struct{}{}
				continue
			}
			token = t
			tokens[ev.AppName] = token
		}
		vars = append(vars, &envVar{
			name:  ev.Name,
			value: token,
		})
	}
	return vars, nil
}

// handleTokenError decides what a failure to get an access token means. It returns
// the error to fail with, or nil to run the command without that environment
// variable.
func handleTokenError(logger *slog.Logger, continueOnError bool, err error) error {
	if errors.Is(err, context.Canceled) {
		// ghtkn was interrupted while getting the token, for example during the device
		// flow. The user knows why, so exit like a shell does without logging an error,
		// and without running the command even with --continue-on-error.
		return ecerror.Wrap(cobrautil.ErrSilent, exitCodeInterrupted)
	}
	if !continueOnError {
		return ecerror.Wrap(err, exitCodeTokenFailure)
	}
	// The error already carries the name of the environment variable and of the app as
	// attributes.
	slogerr.WithError(logger, err).Warn(
		"run the command without setting the environment variable. It keeps the value inherited from ghtkn's environment, if any, so the command may use a different credential",
	)
	return nil
}

// token gets an access token for the app bound to ev.
func (c *Controller) token(ctx context.Context, logger *slog.Logger, base *ghtkn.InputGet, ev *EnvVar) (string, error) {
	token, app, err := c.input.Client.Get(ctx, logger, inputGet(base, ev.AppName))
	if err != nil {
		return "", fmt.Errorf("get an access token: %w", slogerr.With(err, "env_name", ev.Name, "app_name", ev.AppName))
	}
	if app == nil && ev.AppName != "" {
		// GHTKN_GITHUB_TOKEN short circuits the SDK, which then returns that token for
		// every app. The requested app is ignored, so several environment variables can
		// silently share one token.
		logger.Warn("GHTKN_GITHUB_TOKEN is set, so the requested app is ignored and the token of GHTKN_GITHUB_TOKEN is used",
			"env_name", ev.Name, "app_name", ev.AppName)
	}
	return token.AccessToken, nil
}

// inputGet copies base and points the copy at appName, so that each app is requested
// without changing the caller's value.
func inputGet(base *ghtkn.InputGet, appName string) *ghtkn.InputGet {
	input := *base
	input.AppName = appName
	return &input
}
