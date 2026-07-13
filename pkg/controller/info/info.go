package info

import (
	"encoding/json"
	"fmt"
	"runtime"

	"github.com/suzuki-shunsuke/ghtkn-go-sdk/ghtkn/env"
)

// redacted replaces the value of credential-bearing environment variables in the output.
const redacted = "<REDACTED>"

// Output is the troubleshooting information printed by the info command as JSON.
type Output struct {
	OS         string            `json:"os"`
	Arch       string            `json:"arch"`
	Version    string            `json:"version"`
	Envs       map[string]string `json:"envs"`
	App        string            `json:"app"`
	ConfigPath string            `json:"config_path"`
}

// Info writes the environment information to the controller's stdout as indented JSON.
// The caller resolves and passes configPath, so the controller reads only the
// environment variables it reports, via the injected getEnv.
// Token-bearing variables (GH_TOKEN, GITHUB_TOKEN, GHTKN_GITHUB_TOKEN) are
// redacted, and empty variables are omitted. appName, when non-empty, overrides
// GHTKN_APP.
func (c *Controller) Info(configPath, appName, version string) error {
	output := &Output{
		OS:         runtime.GOOS,
		Arch:       runtime.GOARCH,
		Version:    version,
		ConfigPath: configPath,
	}

	envs := map[string]string{}
	// Every GHTKN_* variable, taken from the single registry so this dump can never
	// drift from the set ghtkn actually reads. GHTKN_GITHUB_TOKEN is a credential, so
	// it is redacted.
	for _, name := range env.All {
		value := c.getEnv(name)
		if value == "" {
			continue
		}
		if name == env.GitHubToken {
			value = redacted
		}
		envs[name] = value
	}
	// Non-GHTKN variables ghtkn also honors: the XDG directories (shown) and the
	// ambient GitHub token variables (redacted).
	for _, name := range []string{"XDG_CACHE_HOME", "XDG_CONFIG_HOME", "XDG_RUNTIME_DIR"} {
		if value := c.getEnv(name); value != "" {
			envs[name] = value
		}
	}
	for _, name := range []string{"GH_TOKEN", "GITHUB_TOKEN"} {
		if c.getEnv(name) != "" {
			envs[name] = redacted
		}
	}
	output.Envs = envs

	output.App = c.getEnv(env.App)
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
