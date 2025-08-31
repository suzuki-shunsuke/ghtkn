package config

import (
	"errors"
	"path/filepath"
)

// GetPath returns the default configuration file path for ghtkn.
// It combines the XDG_CONFIG_HOME directory with the ghtkn configuration filename.
// The typical path is $XDG_CONFIG_HOME/ghtkn/ghtkn.yaml.
func GetPath(env *Env) (string, error) {
	if env.GOOS == "windows" {
		if env.AppData != "" {
			return filepath.Join(env.AppData, "ghtkn", "ghtkn.yaml"), nil
		}
		return "", errors.New("APPDATA is required on Windows")
	}
	if env.XDGConfigHome != "" {
		return filepath.Join(env.XDGConfigHome, "ghtkn", "ghtkn.yaml"), nil
	}
	if env.Home != "" {
		return filepath.Join(env.Home, ".config", "ghtkn", "ghtkn.yaml"), nil
	}
	return "", errors.New("XDG_CONFIG_HOME or HOME is required on Linux and macOS")
}
