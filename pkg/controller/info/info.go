package info

import (
	"encoding/json"
	"fmt"
	"runtime"

	"github.com/suzuki-shunsuke/ghtkn-go-sdk/ghtkn/config"
	"github.com/suzuki-shunsuke/ghtkn-go-sdk/ghtkn/env"
)

// redacted replaces the value of credential-bearing environment variables in the output.
const redacted = "<REDACTED>"

// ConfigView is the effective configuration (config file plus environment overrides)
// as shown in the info output. It embeds config.Config but hides the apps list, which
// is noise for a settings dump, without forcing omitempty on the SDK's Config.Apps
// field. The shallow Apps field shadows the embedded config.Config.Apps and, left nil
// with omitempty, drops apps from the JSON. (json:"-" would not work here: it removes
// this field from the JSON entirely, leaving the embedded apps to be marshaled.)
type ConfigView struct {
	*config.Config
	Apps []*config.App `json:"apps,omitempty"`
}

// Output is the troubleshooting information printed by the info command as JSON.
type Output struct {
	OS         string            `json:"os"`
	Arch       string            `json:"arch"`
	Version    string            `json:"version"`
	Envs       map[string]string `json:"envs"`
	App        string            `json:"app"`
	ConfigPath string            `json:"config_path"`
	Config     *ConfigView       `json:"config,omitempty"`
}

// Info writes the environment information to the controller's stdout as indented JSON.
// The caller resolves and passes configPath, so the controller reads only the
// environment variables it reports, via the injected getEnv.
// Token-bearing variables (GH_TOKEN, GITHUB_TOKEN, GHTKN_GITHUB_TOKEN) are
// redacted, and empty variables are omitted. appName, when non-empty, overrides
// GHTKN_APP; when neither is set, the reported app falls back to the default app (the
// first configured app), matching what ghtkn would actually use.
func (c *Controller) Info(configPath, appName, version string, cfg *config.Config) error {
	output := &Output{
		OS:         runtime.GOOS,
		Arch:       runtime.GOARCH,
		Version:    version,
		ConfigPath: configPath,
	}
	// The caller resolves the effective config via ghtkn.LoadConfig (file plus env) and
	// passes it in, so the controller stays free of config-file and environment lookups.
	if cfg != nil {
		output.Config = &ConfigView{Config: cfg}
	}

	output.Envs = c.collectEnvs()

	// Resolve the app the same way token retrieval does, via the SDK's ResolveApp,
	// instead of reimplementing the default-app rule here. key is the requested app
	// (argument overrides GHTKN_APP); when it resolves to a configured app the resolved
	// name is reported (an empty key selects the default app), otherwise the requested
	// key is reported as is so an unknown app is still visible.
	key := c.getEnv(env.App)
	if appName != "" {
		key = appName
	}
	output.App = key
	if app := config.ResolveApp(cfg, key, ""); app != nil {
		output.App = app.Name
	}

	return c.output(output)
}

// collectEnvs reads the environment variables the info command reports. Every variable
// ghtkn reads comes from the single registry (env.All)—both the GHTKN_* variables and the
// OS/XDG path variables—so this dump can never drift from that set; GHTKN_GITHUB_TOKEN is
// a credential, so it is redacted. HOME is omitted (rarely useful and embeds the
// username). It additionally reports the ambient GH_TOKEN and GITHUB_TOKEN (redacted):
// ghtkn does not read them, but they are relevant to troubleshooting. Empty variables
// are omitted.
func (c *Controller) collectEnvs() map[string]string {
	envs := map[string]string{}
	for _, name := range env.All {
		// HOME is omitted: it is rarely useful for troubleshooting and typically embeds
		// the username, which makes the output awkward to share.
		if name == env.Home {
			continue
		}
		value := c.getEnv(name)
		if value == "" {
			continue
		}
		if name == env.GitHubToken {
			value = redacted
		}
		envs[name] = value
	}
	for _, name := range []string{"GH_TOKEN", "GITHUB_TOKEN"} {
		if c.getEnv(name) != "" {
			envs[name] = redacted
		}
	}
	return envs
}

func (c *Controller) output(output *Output) error {
	encoder := json.NewEncoder(c.stdout)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(output); err != nil {
		return fmt.Errorf("encode info as JSON: %w", err)
	}
	return nil
}
