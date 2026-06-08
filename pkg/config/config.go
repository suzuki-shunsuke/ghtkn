// Package config provides helpers for resolving ghtkn's configuration file path.
package config

import (
	"fmt"

	"github.com/suzuki-shunsuke/ghtkn-go-sdk/ghtkn"
)

// ResolvePath returns the configuration file path to use.
// When p is non-empty (i.e. the -c flag was given), it is returned unchanged.
// Otherwise the default path is resolved via the ghtkn SDK, which consults
// GHTKN_CONFIG, then XDG_CONFIG_HOME, then HOME.
func ResolvePath(p string) (string, error) {
	if p != "" {
		return p, nil
	}
	p, err := ghtkn.GetConfigPath()
	if err != nil {
		return "", fmt.Errorf("get the config path: %w", err)
	}
	return p, nil
}
