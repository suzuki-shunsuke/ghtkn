// Package exec implements the 'ghtkn exec' command.
// It runs an external command with GitHub App User Access Tokens in its environment,
// so that the token is never written to stdout and can't leak into a terminal
// transcript, a log or a chat message the way the output of 'ghtkn get' can.
// The '--' in front of the command is required, which is what keeps the command's own
// flags from being parsed by ghtkn.
// Where the platform allows it, ghtkn then replaces its own process with the command
// rather than wrapping it, so the command's exit code, signals and terminal are the
// ones it would have if it had been run directly.
package exec

import (
	"context"
	"fmt"

	"github.com/suzuki-shunsuke/ghtkn-go-sdk/ghtkn"
	"github.com/suzuki-shunsuke/ghtkn/pkg/cli/flag"
	"github.com/suzuki-shunsuke/ghtkn/pkg/config"
	"github.com/suzuki-shunsuke/ghtkn/pkg/controller/exec"
	"github.com/suzuki-shunsuke/slog-util/slogutil"
	"github.com/suzuki-shunsuke/urfave-cli-v3-util/urfave"
	"github.com/urfave/cli/v3"
)

const execDescription = `Run a command with GitHub App User Access Tokens in its environment.

Unlike 'ghtkn get', the token is never written to stdout, so it can't end up in a
terminal transcript, a log, or a chat message. If you are a coding agent, prefer this
command over 'ghtkn get': tokens leaked by agents printing what 'ghtkn get' returned
are the reason it exists. The token is still an environment variable of the command,
though, so a command that prints its environment still exposes it.

The '--' in front of the command is required. Everything after it is the command and
its arguments, so the command's own flags are passed through instead of being parsed
by ghtkn.

By default the token is set to GITHUB_TOKEN. -e replaces that default entirely, so
GITHUB_TOKEN is not set implicitly once -e is given. '-e <env name>' uses the app
ghtkn selects automatically, the same way 'ghtkn get' does, and
'-e <env name>:<app name>' uses the given app. -e can be repeated to give several
tools their own token. A token is created or read once per app, and an environment
variable inherited from ghtkn's environment is replaced rather than duplicated.
Everything else is inherited unchanged: the rest of the environment, the working
directory, and stdin, stdout and stderr. Nothing ghtkn writes goes to stdout, so
the command's stdout carries the command's output alone.

The device flow behaves as in 'ghtkn get': it is disabled by default, so an app with
no valid cached token fails instead of prompting. Enable it with -device-flow or
GHTKN_ENABLE_DEVICE_FLOW=true, or run 'ghtkn auth' beforehand.

On Linux and macOS, ghtkn doesn't stay around while the command runs: it replaces its
own process with the command, which therefore keeps ghtkn's process id, process group
and terminal. Signals, job control and 'wait' behave as if the command had been run
directly, because it is the process you started. Windows has no such call, so there
the command runs as a child, the signals ghtkn receives are forwarded to it, and the
same signal twice kills it (except Ctrl-C, so that a second one doesn't kill an
interactive command).

The exit code is the command's exit code, or 128 plus the signal number when it is
killed by a signal. 127 means the command was not found and 126 that it could not be
executed. If an access token can't be created or read, the command is not run and
ghtkn exits 111; with -continue-on-error it is run anyway, the environment variables
whose tokens were acquired are still set, and the others keep the values inherited
from ghtkn's environment.

$ ghtkn exec -- gh pr view
$ ghtkn exec -e GH_TOKEN -- gh pr view
$ ghtkn exec -e PINACT_GITHUB_TOKEN:suzuki-shunsuke/read -e GH_TOKEN:suzuki-shunsuke/write -- bash foo.sh
$ ghtkn exec -m 1h -- terraform plan`

// commandName is the name of the subcommand.
const commandName = "exec"

// Args holds the flag and argument values for the exec command.
type Args struct {
	*flag.GlobalFlags

	MinExpiration string
	// Envs are the raw -e values, each an environment variable name optionally
	// followed by ':' and an app name.
	Envs []string
	// Command is the command to run and its arguments, taken from the positional
	// arguments after '--'.
	Command         []string
	DeviceFlow      bool
	ContinueOnError bool
}

// New creates a new exec command instance with the provided logger.
// It returns a CLI command that can be registered with the main CLI application.
func New(logger *slogutil.Logger, env *urfave.Env, gFlags *flag.GlobalFlags) *cli.Command {
	args := &Args{
		GlobalFlags: gFlags,
	}
	r := &runner{
		rawArgs: env.Args,
	}
	return r.Command(logger, args)
}

// runner holds the state the exec command needs beyond its flags.
type runner struct {
	// rawArgs is the command line as given to ghtkn. urfave/cli removes the '--' from
	// the parsed arguments, so it can only be found here.
	rawArgs []string
}

// Command returns the CLI command definition for the exec subcommand.
func (r *runner) Command(logger *slogutil.Logger, args *Args) *cli.Command {
	return &cli.Command{
		Name:        commandName,
		Usage:       "Run a command with access tokens in environment variables",
		ArgsUsage:   "-- <command> [<args>...]",
		Description: execDescription,
		Action: func(ctx context.Context, cmd *cli.Command) error {
			return r.action(ctx, cmd, logger, args)
		},
		Flags: []cli.Flag{
			flag.LogLevel(&args.LogLevel),
			flag.Config(&args.Config),
			flag.MinExpiration(&args.MinExpiration),
			flag.DeviceFlow(&args.DeviceFlow),
			&cli.StringSliceFlag{
				Name:        "env",
				Aliases:     []string{"e"},
				Usage:       "environment variable set to an access token: <env name>[:<app name>]. Repeatable",
				Destination: &args.Envs,
			},
			&cli.BoolFlag{
				Name:        "continue-on-error",
				Usage:       "Run the command even if an access token can't be created or read",
				Destination: &args.ContinueOnError,
			},
		},
		Arguments: []cli.Argument{
			// Min is 0 so that a missing command is reported by requireSeparator,
			// which explains the '--' too.
			&cli.StringArgs{
				Name:        "command",
				Min:         0,
				Max:         -1,
				Destination: &args.Command,
			},
		},
	}
}

// action implements the main logic for the exec command.
// The command line is validated before anything else so that a mistake in it doesn't
// unlock the backend or start a device flow.
func (r *runner) action(ctx context.Context, cmd *cli.Command, logger *slogutil.Logger, args *Args) error {
	if err := logger.SetLevel(args.LogLevel); err != nil {
		return fmt.Errorf("set log level: %w", err)
	}
	if err := requireSeparator(r.rawArgs, args.Command); err != nil {
		return err
	}
	envVars, err := parseEnvs(args.Envs)
	if err != nil {
		return err
	}

	inputGet := &ghtkn.InputGet{
		ConfigFilePath: args.Config,
	}
	if err := flag.SetMinExpiration(inputGet, args.MinExpiration); err != nil {
		return err //nolint:wrapcheck
	}
	// Pass the device-flow override only when the flag is explicitly set so it takes
	// precedence over GHTKN_ENABLE_DEVICE_FLOW and the config; otherwise leave it nil
	// so the SDK resolves them itself.
	if cmd.IsSet("device-flow") {
		inputGet.EnableDeviceFlow = &args.DeviceFlow
	}

	input, err := exec.NewInput()
	if err != nil {
		return fmt.Errorf("create the controller input: %w", err)
	}
	p, err := config.ResolvePath(inputGet.ConfigFilePath)
	if err != nil {
		return err //nolint:wrapcheck
	}
	inputGet.ConfigFilePath = p
	return exec.New(input).Run(ctx, logger.Logger, &exec.InputRun{ //nolint:wrapcheck
		InputGet:        inputGet,
		EnvVars:         envVars,
		Command:         args.Command,
		ContinueOnError: args.ContinueOnError,
	})
}
