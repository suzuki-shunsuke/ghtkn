// Package stop implements the 'ghtkn agent stop' command: it connects to a running
// agent over its Unix domain socket and asks it to shut down. The agent server lives
// in pkg/controller/agent.
package stop

import "os"

// Controller backs the 'ghtkn agent stop' command. It is a client: it only talks to
// the agent over the socket and never touches the token store.
type Controller struct {
	// getEnv reads an environment variable when resolving the socket path. It is a field
	// so tests (and reset, which stops the agent) can inject it without t.Setenv, which
	// would forbid t.Parallel.
	getEnv func(string) string
}

// New creates a new stop Controller that reads the real environment.
func New() *Controller {
	return NewWithEnv(os.Getenv)
}

// NewWithEnv creates a stop Controller that resolves the socket path through getEnv. It
// lets reset stop the agent using the same injected environment as the rest of its run.
func NewWithEnv(getEnv func(string) string) *Controller {
	return &Controller{getEnv: getEnv}
}
