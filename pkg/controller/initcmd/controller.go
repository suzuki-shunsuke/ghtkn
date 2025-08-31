// Package initcmd implements the business logic for the 'ghtkn init' command.
// It handles the creation of ghtkn configuration files with default templates.
package initcmd

import (
	"github.com/spf13/afero"
	"github.com/suzuki-shunsuke/ghtkn/pkg/config"
)

// Controller manages the initialization of ghtkn configuration.
// It provides methods to create configuration files with appropriate permissions.
type Controller struct {
	fs  afero.Fs
	env *config.Env
}

// New creates a new Controller instance with the provided filesystem and environment.
// The filesystem is used for all file operations, allowing for easy testing with mock filesystems.
func New(fs afero.Fs, env *config.Env) *Controller {
	return &Controller{
		fs:  fs,
		env: env,
	}
}
