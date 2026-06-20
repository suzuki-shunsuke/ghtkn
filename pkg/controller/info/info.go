package info

import (
	"encoding/json"
	"fmt"
	"runtime"
)

// Output is the troubleshooting information printed by the info command as JSON.
type Output struct {
	OS         string            `json:"os"`
	Arch       string            `json:"arch"`
	Version    string            `json:"version"`
	Envs       map[string]string `json:"envs"`
	Backend    string            `json:"backend"`
	App        string            `json:"app"`
	ConfigPath string            `json:"config_path"`
}

// Info writes the environment information to the controller's stdout as indented JSON.
// The caller resolves and passes configPath, so the controller reads only the
// environment variables it reports, via the injected getEnv.
// Token-bearing variables (GH_TOKEN, GITHUB_TOKEN, GHTKN_GITHUB_TOKEN) are
// redacted, and empty variables are omitted. Backend defaults to "keyring" when
// GHTKN_BACKEND is unset. appName, when non-empty, overrides GHTKN_APP.
func (c *Controller) Info(configPath, appName, version string) error {
	output := &Output{
		OS:         runtime.GOOS,
		Arch:       runtime.GOARCH,
		Version:    version,
		ConfigPath: configPath,
	}

	envNames := []string{
		"GHTKN_CONFIG",
		"GHTKN_APP",
		"GHTKN_MIN_EXPIRATION",
		"GHTKN_LOG_LEVEL",
		"GHTKN_ENABLE_DEVICE_FLOW",
		"GHTKN_SKIP_ACCOUNT_PICKER",
		"GHTKN_BACKEND",
		"GHTKN_AGENT_SOCKET",
		"GHTKN_AGENT_KEY",
		"GHTKN_ENABLE",
		"GHTKN_TEXT_BACKEND_DIR",
		"XDG_CACHE_HOME",
		"XDG_RUNTIME_DIR",
		"XDG_CONFIG_HOME",
	}

	// read envs
	envs := make(map[string]string, len(envNames))
	for _, name := range envNames {
		value := c.getEnv(name)
		if value != "" {
			envs[name] = value
		}
	}

	tokenEnvNames := []string{
		"GH_TOKEN",
		"GITHUB_TOKEN",
		"GHTKN_GITHUB_TOKEN",
	}
	for _, name := range tokenEnvNames {
		value := c.getEnv(name)
		if value != "" {
			envs[name] = "<REDACTED>"
		}
	}
	output.Envs = envs

	backend := c.getEnv("GHTKN_BACKEND")
	if backend == "" {
		backend = "keyring"
	}
	output.Backend = backend

	output.App = c.getEnv("GHTKN_APP")
	if appName != "" {
		output.App = appName
	}

	return c.output(output)
}

func (c *Controller) output(output *Output) error {
	encoder := json.NewEncoder(c.stdout)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(output); err != nil {
		return fmt.Errorf("encode info as JSON: %w", err)
	}
	return nil
}
