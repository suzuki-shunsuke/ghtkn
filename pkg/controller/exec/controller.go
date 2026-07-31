// Package exec implements the 'ghtkn exec' command, which runs an external command
// with GitHub App User Access Tokens in its environment.
//
// The token is never written to stdout, which is the point of the command: it can't
// end up in a terminal transcript, a log or a chat message the way the output of
// 'ghtkn get' can. An access token is created or read once per app, even when several
// environment variables are bound to the same app.
package exec

import (
	"context"
	"fmt"
	"log/slog"
	"os"

	"github.com/suzuki-shunsuke/ghtkn-go-sdk/ghtkn"
	"github.com/suzuki-shunsuke/ghtkn/pkg/proc"
)

// Client is the subset of the ghtkn SDK client used to get access tokens.
type Client interface {
	Get(ctx context.Context, logger *slog.Logger, input *ghtkn.InputGet) (*ghtkn.AccessToken, *ghtkn.AppConfig, error)
}

// Runner runs the external command and returns its exit code.
type Runner interface {
	Run(logger *slog.Logger, env []string, name string, args ...string) (int, error)
}

// Input contains the dependencies needed by the Controller.
type Input struct {
	Client Client
	Runner Runner
	// Environ returns ghtkn's own environment, which the command inherits. It is a
	// field rather than a call to os.Environ so that tests don't need t.Setenv, which
	// would forbid t.Parallel.
	Environ func() []string
}

// NewInput creates a new Input with default production dependencies.
func NewInput() (*Input, error) {
	client, err := ghtkn.New()
	if err != nil {
		return nil, fmt.Errorf("create a ghtkn client: %w", err)
	}
	return &Input{
		Client:  client,
		Runner:  proc.New(os.Stdin, os.Stdout, os.Stderr),
		Environ: os.Environ,
	}, nil
}

// EnvVar maps an environment variable to the app whose access token it receives.
type EnvVar struct {
	// Name is the name of the environment variable set in the command's environment.
	Name string
	// AppName is the app the access token is issued for. An empty value means the app
	// ghtkn selects automatically, the same way 'ghtkn get' does.
	AppName string
}

// InputRun holds the values needed to run the command.
type InputRun struct {
	// InputGet is the template passed to the SDK. Its AppName is set per EnvVar on a
	// copy, so the value it holds is ignored.
	InputGet *ghtkn.InputGet
	// EnvVars are the environment variables receiving an access token, in the order
	// they were given.
	EnvVars []*EnvVar
	// Command is the command to run and its arguments.
	Command []string
	// ContinueOnError runs the command even when an access token can't be created or
	// read.
	ContinueOnError bool
}

// Controller runs an external command with access tokens in its environment.
type Controller struct {
	input *Input
}

// New creates a new Controller with the provided input.
func New(input *Input) *Controller {
	return &Controller{
		input: input,
	}
}
