package config

import (
	"path/filepath"
)

// GetPath returns the default configuration file path for ghtkn.
// It combines the XDG_CONFIG_HOME directory with the ghtkn configuration filename.
// The typical path is $XDG_CONFIG_HOME/ghtkn/ghtkn.yaml.
func GetPath(env *Env) string {
	return filepath.Join(env.XDGConfigHome, "ghtkn", "ghtkn.yaml")
}
