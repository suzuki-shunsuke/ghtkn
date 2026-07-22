// Package initcmd implements the business logic for the 'ghtkn init' command.
// It handles the creation of ghtkn configuration files with default templates.
package initcmd

// Controller manages the initialization of ghtkn configuration.
// It provides methods to create configuration files with appropriate permissions.
type Controller struct{}

// New creates a new Controller instance.
func New() *Controller {
	return &Controller{}
}
