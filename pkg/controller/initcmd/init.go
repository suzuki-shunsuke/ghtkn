package initcmd

import (
	_ "embed"
	"errors"
	"fmt"
	"io/fs"
	"log/slog"
	"os"
	"path/filepath"
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
	switch _, err := os.Stat(configFilePath); {
	case err == nil:
		logger.Warn("The configuration file already exists", "path", configFilePath)
		return nil
	case !errors.Is(err, fs.ErrNotExist):
		// Anything other than "it isn't there" means the answer is unknown, so the file
		// must not be written over whatever is actually at that path.
		return fmt.Errorf("check if a configuration file exists: %w", err)
	}
	if err := os.MkdirAll(filepath.Dir(configFilePath), dirPermission); err != nil {
		return fmt.Errorf("create config dir: %w", err)
	}
	if err := os.WriteFile(configFilePath, defaultConfig, filePermission); err != nil {
		return fmt.Errorf("create a configuration file: %w", err)
	}
	logger.Info("The configuration file has been created", slog.String("path", configFilePath))
	return nil
}
