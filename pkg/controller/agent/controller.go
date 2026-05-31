// Package agent provides the controller for the 'ghtkn agent' command.
// It implements the agent server: a long-running process that caches GitHub App
// access tokens in memory and serves them to clients over a Unix domain socket.
//
// On-disk encryption (passphrase-derived key, AES-256-GCM) and persistence are
// intentionally out of scope here and are planned for a later change. The socket
// protocol defined in this package is the contract between the agent and its clients.
package agent

import "context"

// Controller runs the ghtkn agent server.
type Controller struct {
	store *store
	// shutdown cancels the serve loop. It is set while the server is running
	// (see Start) and invoked when a STOP command is received.
	shutdown context.CancelFunc
}

// New creates a new agent Controller with an empty in-memory token store.
func New() *Controller {
	return &Controller{
		store: newStore(),
	}
}
