// Package info implements the controller for the 'ghtkn info' command, which
// renders environment information useful for troubleshooting as JSON.
package info

import "io"

// Controller renders the troubleshooting information.
// stdout is where the JSON is written, and getEnv looks up environment
// variables (injected so the command is easy to test).
type Controller struct {
	getEnv func(string) string
	stdout io.Writer
}

// New creates a Controller that writes to stdout and reads environment
// variables via getEnv.
func New(stdout io.Writer, getEnv func(string) string) *Controller {
	return &Controller{
		stdout: stdout,
		getEnv: getEnv,
	}
}
