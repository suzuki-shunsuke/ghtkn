// Package stop implements the 'ghtkn agent stop' command: it connects to a running
// agent over its Unix domain socket and asks it to shut down. The agent server lives
// in pkg/controller/agent.
package stop

// Controller backs the 'ghtkn agent stop' command. It is a client: it only talks to
// the agent over the socket and never touches the token store.
type Controller struct{}

// New creates a new stop Controller.
func New() *Controller {
	return &Controller{}
}
