// Package agent provides the controller for the 'ghtkn agent' command.
// It implements the agent server: a long-running process that caches GitHub App
// access tokens and serves them to clients over a Unix domain socket.
//
// Tokens are encrypted at rest with AES-256-GCM under a data key that is itself
// wrapped with a passphrase-derived (Argon2id) key-encryption key. The passphrase
// is entered interactively at start and the derived keys live only in memory.
// The socket protocol defined in this package is the contract between the agent
// and its clients.
package agent

import "context"

// Controller runs the ghtkn agent server.
type Controller struct {
	store *store
	// shutdown cancels the serve loop. It is set while the server is running
	// (see Start) and invoked when a STOP command is received.
	shutdown context.CancelFunc
	// readPassphrase reads a passphrase from the terminal. It is a field so tests
	// can inject a stub instead of driving a real TTY.
	readPassphrase func(prompt string) ([]byte, error)
}

// New creates a new agent Controller. The token store starts in memory-only mode;
// Start replaces it with a disk-backed, encrypted store once the passphrase is
// entered and the data key is loaded.
func New() *Controller {
	return &Controller{
		store:          newStore(),
		readPassphrase: readPassphrase,
	}
}
