package info_test

import (
	"bytes"
	"encoding/json"
	"reflect"
	"runtime"
	"testing"

	"github.com/suzuki-shunsuke/ghtkn-go-sdk/ghtkn/config"
	"github.com/suzuki-shunsuke/ghtkn/pkg/controller/info"
)

func fakeEnv(m map[string]string) func(string) string {
	return func(k string) string {
		return m[k]
	}
}

func TestController_Info(t *testing.T) { //nolint:funlen
	t.Parallel()
	const configPath = "/test/ghtkn/ghtkn.yaml"

	tests := []struct {
		name    string
		env     map[string]string
		appName string
		version string
		cfg     *config.Config
		want    *info.Output
	}{
		{
			name:    "minimal: no environment variables set",
			env:     map[string]string{},
			appName: "",
			version: "v1.0.0",
			want: &info.Output{
				OS:         runtime.GOOS,
				Arch:       runtime.GOARCH,
				Version:    "v1.0.0",
				Envs:       map[string]string{},
				App:        "",
				ConfigPath: configPath,
			},
		},
		{
			name: "GHTKN_BACKEND is reflected in Envs",
			env: map[string]string{
				"GHTKN_BACKEND": "text",
			},
			version: "v1.2.3",
			want: &info.Output{
				OS:      runtime.GOOS,
				Arch:    runtime.GOARCH,
				Version: "v1.2.3",
				Envs: map[string]string{
					"GHTKN_BACKEND": "text",
				},
				App:        "",
				ConfigPath: configPath,
			},
		},
		{
			name: "App is taken from GHTKN_APP",
			env: map[string]string{
				"GHTKN_APP": "env-app",
			},
			version: "v1.0.0",
			want: &info.Output{
				OS:      runtime.GOOS,
				Arch:    runtime.GOARCH,
				Version: "v1.0.0",
				Envs: map[string]string{
					"GHTKN_APP": "env-app",
				},
				App:        "env-app",
				ConfigPath: configPath,
			},
		},
		{
			name: "appName argument overrides GHTKN_APP",
			env: map[string]string{
				"GHTKN_APP": "env-app",
			},
			appName: "arg-app",
			version: "v1.0.0",
			want: &info.Output{
				OS:      runtime.GOOS,
				Arch:    runtime.GOARCH,
				Version: "v1.0.0",
				Envs: map[string]string{
					"GHTKN_APP": "env-app",
				},
				App:        "arg-app",
				ConfigPath: configPath,
			},
		},
		{
			name: "token environment variables are redacted",
			env: map[string]string{
				"GH_TOKEN":           "secret-1",
				"GITHUB_TOKEN":       "secret-2",
				"GHTKN_GITHUB_TOKEN": "secret-3",
			},
			version: "v1.0.0",
			want: &info.Output{
				OS:      runtime.GOOS,
				Arch:    runtime.GOARCH,
				Version: "v1.0.0",
				Envs: map[string]string{
					"GH_TOKEN":           "<REDACTED>",
					"GITHUB_TOKEN":       "<REDACTED>",
					"GHTKN_GITHUB_TOKEN": "<REDACTED>",
				},
				App:        "",
				ConfigPath: configPath,
			},
		},
		{
			name: "empty environment variables are omitted",
			env: map[string]string{
				"GHTKN_CONFIG":  "",
				"GHTKN_APP":     "",
				"GH_TOKEN":      "",
				"GHTKN_BACKEND": "agent",
			},
			version: "v1.0.0",
			want: &info.Output{
				OS:      runtime.GOOS,
				Arch:    runtime.GOARCH,
				Version: "v1.0.0",
				Envs: map[string]string{
					"GHTKN_BACKEND": "agent",
				},
				App:        "",
				ConfigPath: configPath,
			},
		},
		{
			name: "multiple known environment variables are collected",
			env: map[string]string{
				"GHTKN_LOG_LEVEL":        "debug",
				"GHTKN_MIN_EXPIRATION":   "10m",
				"GHTKN_AGENT_SOCKET":     "/tmp/agent.sock",
				"XDG_CACHE_HOME":         "/home/user/.cache",
				"UNRELATED_ENV_VARIABLE": "ignored",
			},
			version: "v1.0.0",
			want: &info.Output{
				OS:      runtime.GOOS,
				Arch:    runtime.GOARCH,
				Version: "v1.0.0",
				Envs: map[string]string{
					"GHTKN_LOG_LEVEL":      "debug",
					"GHTKN_MIN_EXPIRATION": "10m",
					"GHTKN_AGENT_SOCKET":   "/tmp/agent.sock",
					"XDG_CACHE_HOME":       "/home/user/.cache",
				},
				App:        "",
				ConfigPath: configPath,
			},
		},
		{
			name: "HOME is omitted but other path vars are reported",
			env: map[string]string{
				"HOME":            "/home/alice",
				"XDG_CONFIG_HOME": "/home/alice/.config",
			},
			version: "v1.0.0",
			want: &info.Output{
				OS:      runtime.GOOS,
				Arch:    runtime.GOARCH,
				Version: "v1.0.0",
				Envs: map[string]string{
					// HOME is intentionally absent; XDG_CONFIG_HOME is reported.
					"XDG_CONFIG_HOME": "/home/alice/.config",
				},
				App:        "",
				ConfigPath: configPath,
			},
		},
		{
			name:    "effective config is reported with the apps list omitted",
			env:     map[string]string{},
			version: "v1.0.0",
			cfg: &config.Config{
				Apps:          []*config.App{{Name: "example", ClientID: "Iv1.example"}},
				Backend:       &config.Backend{Type: "agent"},
				MinExpiration: "1h",
			},
			want: &info.Output{
				OS:      runtime.GOOS,
				Arch:    runtime.GOARCH,
				Version: "v1.0.0",
				Envs:    map[string]string{},
				// No GHTKN_APP/argument, so the reported app is the default (first) app.
				App:        "example",
				ConfigPath: configPath,
				// apps must not appear: the round-trip through JSON drops it.
				Config: &info.ConfigView{Config: &config.Config{
					Backend:       &config.Backend{Type: "agent"},
					MinExpiration: "1h",
				}},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			buf := &bytes.Buffer{}
			ctrl := info.New(buf, fakeEnv(tt.env))

			if err := ctrl.Info(configPath, tt.appName, tt.version, tt.cfg, nil); err != nil {
				t.Fatalf("Info() returned an unexpected error: %v", err)
			}

			got := &info.Output{}
			if err := json.Unmarshal(buf.Bytes(), got); err != nil {
				t.Fatalf("the output is not valid JSON: %v\noutput: %s", err, buf.String())
			}

			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Info() output mismatch\n got: %+v\nwant: %+v", got, tt.want)
			}
		})
	}
}

// TestController_Info_agent verifies that a passed-in agent status is rendered in the
// output's `agent` section, and that a nil agent omits the section.
func TestController_Info_agent(t *testing.T) {
	t.Parallel()
	agent := &info.AgentStatus{
		Running:      true,
		Locked:       new(false),
		RefreshToken: &info.AgentRefreshToken{Enabled: true, TTL: "3d"},
	}

	buf := &bytes.Buffer{}
	if err := info.New(buf, fakeEnv(nil)).Info("/x/ghtkn.yaml", "", "", nil, agent); err != nil {
		t.Fatal(err)
	}
	got := &info.Output{}
	if err := json.Unmarshal(buf.Bytes(), got); err != nil {
		t.Fatalf("output is not valid JSON: %v\n%s", err, buf)
	}
	if !reflect.DeepEqual(got.Agent, agent) {
		t.Errorf("agent section mismatch\n got: %+v\nwant: %+v", got.Agent, agent)
	}

	// A nil agent omits the section entirely.
	buf.Reset()
	if err := info.New(buf, fakeEnv(nil)).Info("/x/ghtkn.yaml", "", "", nil, nil); err != nil {
		t.Fatal(err)
	}
	got2 := &info.Output{}
	if err := json.Unmarshal(buf.Bytes(), got2); err != nil {
		t.Fatal(err)
	}
	if got2.Agent != nil {
		t.Errorf("a nil agent must be omitted, got %+v", got2.Agent)
	}
}
