package config

import (
	"path/filepath"
)

func GetPath(env *Env) string {
	return filepath.Join(env.XDGConfigHome, "ghtkn", "ghtkn.yaml")
}
