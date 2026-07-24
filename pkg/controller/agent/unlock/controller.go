// Package unlock implements the 'ghtkn agent unlock' command: the client half of the
// locked-start workflow. It prompts for the agent passphrase on the terminal and
// sends it to a running agent over the Unix domain socket so the agent can load its
// data key and make cached tokens readable. The agent server lives in
// pkg/agent/server.
package unlock

import (
	"os"

	"github.com/suzuki-shunsuke/ghtkn/pkg/agent/tty"
)

// Controller backs the 'ghtkn agent unlock' command. It is a client: it never
// touches the token store, only the socket and the terminal.
type Controller struct {
	// readPassphrase reads a passphrase from the terminal. It is a field so tests
	// can inject a stub instead of driving a real TTY.
	readPassphrase func(prompt string) ([]byte, error)
	// confirm asks the user a yes/no question on the terminal. It is a field so tests
	// can inject a stub instead of driving a real TTY. It is used to confirm dropping
	// stored refresh tokens when unlocking without --enable-refresh.
	confirm func(prompt string) (bool, error)
	// getEnv reads an environment variable. It is a field so tests can inject the socket
	// path without t.Setenv (which would forbid t.Parallel).
	getEnv func(string) string
}

// New creates a new unlock Controller using the real terminal helpers.
func New() *Controller {
	return &Controller{
		readPassphrase: tty.ReadPassphrase,
		confirm:        tty.Confirm,
		getEnv:         os.Getenv,
	}
}
