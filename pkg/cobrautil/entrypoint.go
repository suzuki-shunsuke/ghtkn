// Package cobrautil provides the pieces every CLI of ours needs on top of
// spf13/cobra: the process entry point, a version command, a command dumping the
// help of the whole tree, and environment variable sources for flags.
//
// It is the cobra counterpart of suzuki-shunsuke/urfave-cli-v3-util. It lives inside
// ghtkn while the migration from urfave/cli is being validated, and is meant to be
// extracted into a repository of its own once its API has settled; nothing in it
// depends on ghtkn.
package cobrautil

import (
	"context"
	"errors"
	"os"
	"os/signal"
	"syscall"

	"github.com/suzuki-shunsuke/go-error-with-exit-code/ecerror"
	"github.com/suzuki-shunsuke/slog-error/slogerr"
	"github.com/suzuki-shunsuke/slog-util/slogutil"
)

// Env is the process environment handed to Run, so that commands read it from here
// rather than reaching for the globals in os and can be driven by a test.
type Env struct {
	Program string
	Version string
	Stdin   *os.File
	Stdout  *os.File
	Stderr  *os.File
	Getenv  func(string) string
	Args    []string
}

// Run is the entry point of the CLI, called by Main with the process environment.
type Run func(ctx context.Context, logger *slogutil.Logger, env *Env) error

// ErrSilent is an error with an empty message, which Main logs nothing for. Wrap it
// with ecerror.Wrap to exit with a code without reporting a failure of the CLI's own,
// as when a command propagates the exit code of a command it ran.
var ErrSilent = errors.New("")

// Main runs the CLI and exits the process with the exit code of the error it returns.
// args are extra attributes for the logger.
func Main(name, version string, run Run, args ...any) {
	if code := core(name, version, run, args...); code != 0 {
		os.Exit(code)
	}
}

// core is Main without the os.Exit, so that a test can assert on the exit code.
func core(name, version string, run Run, args ...any) int {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	logger := slogutil.New(&slogutil.InputNew{
		Name:    name,
		Version: version,
		Out:     os.Stderr,
		Attrs:   args,
	})
	env := &Env{
		Program: name,
		Version: version,
		Stdin:   os.Stdin,
		Stdout:  os.Stdout,
		Stderr:  os.Stderr,
		Getenv:  os.Getenv,
		Args:    os.Args,
	}
	if err := run(ctx, logger, env); err != nil {
		if err.Error() != "" {
			slogerr.WithError(logger.Logger, err).Error(name + " failed")
		}
		return ecerror.GetExitCode(err)
	}
	return 0
}
