// Package agent provides the controller for the 'ghtkn agent' command.
// It implements the agent server: a long-running process that caches GitHub App
// access tokens and serves them to clients over a Unix domain socket.
//
// The agent starts locked: it listens without a passphrase so it can run as a
// background service (e.g. systemd) that needs no terminal. A separate
// 'ghtkn agent unlock' command prompts for the passphrase on a terminal and sends
// it over the socket; only then is the data key loaded and tokens become readable.
//
// Tokens are encrypted at rest with AES-256-GCM under a data key that is itself
// wrapped with a passphrase-derived (Argon2id) key-encryption key. The derived
// keys live only in memory. The socket protocol (in ghtkn-go-sdk/ghtkn/backend/agent)
// is the contract between the agent and its clients.
package agent

import (
	"context"
	"log/slog"
	"sync"
)

// Controller runs the ghtkn agent server and also backs the agent client commands
// (stop, status, unlock).
type Controller struct {
	// mu guards store, which is swapped from nil (locked) to a disk store on unlock.
	mu    sync.RWMutex
	store *store // nil while locked

	// shutdown cancels the serve loop. It is set while the server is running
	// (see Start) and invoked when a STOP command is received.
	shutdown context.CancelFunc
	// logger is the server logger, set in Start so socket handlers can log.
	logger *slog.Logger
	// keyFile and tokenDir are the server's on-disk locations, set in Start.
	keyFile  string
	tokenDir string

	// readPassphrase reads a passphrase from the terminal. It is a field so tests
	// can inject a stub instead of driving a real TTY.
	readPassphrase func(prompt string) ([]byte, error)
}

// New creates a new agent Controller. The server starts locked (no token store);
// it is unlocked later via the UNLOCK command. The client commands (stop, status,
// unlock) reuse the same type but never touch the store.
func New() *Controller {
	return &Controller{
		readPassphrase: readPassphrase,
	}
}
