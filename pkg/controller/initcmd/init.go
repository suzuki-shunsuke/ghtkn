package initcmd

import (
	_ "embed"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"

	"github.com/spf13/afero"
)

// defaultConfig provides a default configuration template for ghtkn.
// This template can be used to create an initial configuration file.
//
//go:embed init.yaml
var defaultConfig []byte

// File and directory permissions for created configuration files
const (
	filePermission os.FileMode = 0o644 // Standard file permissions (rw-r--r--)
	dirPermission  os.FileMode = 0o755 // Standard directory permissions (rwxr-xr-x)
)

// Init creates a new ghtkn configuration file if it doesn't exist.
// It checks if the configuration file already exists and creates it with
// a template configuration if it doesn't exist.
// Returns an error if file operations fail, nil if successful or file already exists.
func (c *Controller) Init(logger *slog.Logger, configFilePath string) error {
	f, err := afero.Exists(c.fs, configFilePath)
	if err != nil {
		return fmt.Errorf("check if a configuration file exists: %w", err)
	}
	if f {
		logger.Warn("The configuration file already exists", "path", configFilePath)
		return nil
	}
	if err := c.fs.MkdirAll(filepath.Dir(configFilePath), dirPermission); err != nil {
		return fmt.Errorf("create config dir: %w", err)
	}
	if err := afero.WriteFile(c.fs, configFilePath, defaultConfig, filePermission); err != nil {
		return fmt.Errorf("create a configuration file: %w", err)
	}
	logger.Info("The configuration file has been created", slog.String("path", configFilePath))
	return nil
}
