// Package lock implements the 'ghtkn agent lock' command: it connects to a running
// agent over its Unix domain socket and asks it to discard its in-memory data key,
// returning to the locked state without stopping the process. The agent server lives
// in pkg/controller/agent.
package lock

import "os"

// Controller backs the 'ghtkn agent lock' command. It is a client: it only talks to the
// agent over the socket and never touches the token store.
type Controller struct {
	// getEnv reads an environment variable when resolving the socket path. It is a field
	// so tests can inject it without t.Setenv, which would forbid t.Parallel.
	getEnv func(string) string
}

// New creates a new lock Controller that reads the real environment.
func New() *Controller {
	return NewWithEnv(os.Getenv)
}

// NewWithEnv creates a lock Controller that resolves the socket path through getEnv.
func NewWithEnv(getEnv func(string) string) *Controller {
	return &Controller{getEnv: getEnv}
}
