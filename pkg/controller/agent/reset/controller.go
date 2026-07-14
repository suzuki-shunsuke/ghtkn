// Package reset implements the 'ghtkn agent reset' command: it recovers from a
// forgotten passphrase by stopping the agent, deleting the key file and cached
// tokens, and recreating the key from a freshly entered passphrase. Unlike the other
// client commands it manipulates the key file and token store on disk directly (see
// pkg/controller/agent/keyfile and pkg/controller/agent/tokenstore) rather than
// talking to the agent over the socket.
package reset

import (
	"os"

	"github.com/suzuki-shunsuke/ghtkn/pkg/controller/agent/tty"
)

// Controller backs the 'ghtkn agent reset' command.
type Controller struct {
	// readPassphrase reads a passphrase from the terminal. It is a field so tests
	// can inject a stub instead of driving a real TTY.
	readPassphrase func(prompt string) ([]byte, error)
	// confirm asks the user a yes/no question on the terminal. It is a field so tests
	// can inject a stub instead of driving a real TTY.
	confirm func(prompt string) (bool, error)
	// getEnv reads an environment variable when resolving the key/token/socket paths. It
	// is a field so tests can inject it without t.Setenv, which would forbid t.Parallel.
	getEnv func(string) string
}

// New creates a new reset Controller using the real terminal helpers and environment.
func New() *Controller {
	return &Controller{
		readPassphrase: tty.ReadPassphrase,
		confirm:        tty.Confirm,
		getEnv:         os.Getenv,
	}
}
