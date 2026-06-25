// Package status implements the 'ghtkn agent status' command: it connects to the
// agent over its Unix domain socket and reports whether the agent is running,
// whether it is locked, and how many tokens it caches. The agent server lives in
// pkg/controller/agent.
package status

// Controller backs the 'ghtkn agent status' command. It is a client: it only queries
// the agent over the socket and never touches the token store.
type Controller struct{}

// New creates a new status Controller.
func New() *Controller {
	return &Controller{}
}
