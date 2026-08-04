package info_test

import (
	"bytes"
	"encoding/json"
	"reflect"
	"runtime"
	"strings"
	"testing"

	agentapi "github.com/suzuki-shunsuke/ghtkn-go-sdk/ghtkn/backend/agent"
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
		version string
		cfg     *config.Config
		want    *info.Output
	}{
		{
			name:    "minimal: no environment variables set",
			env:     map[string]string{},
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
			// GHTKN_APP selects a configured app rather than the default (first) one.
			name: "GHTKN_APP selects a configured app",
			env: map[string]string{
				"GHTKN_APP": "second",
			},
			version: "v1.0.0",
			cfg: &config.Config{
				Apps: []*config.App{
					{Name: "first", ClientID: "Iv1.first"},
					{Name: "second", ClientID: "Iv1.second"},
				},
			},
			want: &info.Output{
				OS:      runtime.GOOS,
				Arch:    runtime.GOARCH,
				Version: "v1.0.0",
				Envs: map[string]string{
					"GHTKN_APP": "second",
				},
				App:        "second",
				ConfigPath: configPath,
				// The config holds nothing but apps, which are omitted, so it
				// round-trips through JSON as an empty object.
				Config: &info.ConfigView{},
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
				// No GHTKN_APP, so the reported app is the default (first) app.
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

			if err := ctrl.Info(configPath, tt.version, tt.cfg, nil); err != nil {
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
// output's `agent` section, that the client's own protocol version accompanies it, and
// that a nil agent omits both. A zero protocol version must survive the encoding: 0 is a
// real version (a pre-versioning agent), not a missing value.
func TestController_Info_agent(t *testing.T) {
	t.Parallel()
	agent := &info.AgentStatus{
		Running:            true,
		Version:            "v0.3.1",
		ProtocolVersion:    new(1),
		MinProtocolVersion: new(0),
		Locked:             new(false),
		RefreshToken:       &info.AgentRefreshToken{Enabled: true, TTL: "3d"},
	}

	buf := &bytes.Buffer{}
	if err := info.New(buf, fakeEnv(nil)).Info("/x/ghtkn.yaml", "", nil, agent); err != nil {
		t.Fatal(err)
	}
	got := &info.Output{}
	if err := json.Unmarshal(buf.Bytes(), got); err != nil {
		t.Fatalf("output is not valid JSON: %v\n%s", err, buf)
	}
	if !reflect.DeepEqual(got.Agent, agent) {
		t.Errorf("agent section mismatch\n got: %+v\nwant: %+v", got.Agent, agent)
	}
	if !strings.Contains(buf.String(), `"min_protocol_version": 0`) {
		t.Errorf("a zero minimum protocol version must stay in the output, got:\n%s", buf)
	}
	if got.ProtocolVersion == nil || *got.ProtocolVersion != agentapi.ProtocolVersion {
		t.Errorf("protocol_version = %v, want %d", got.ProtocolVersion, agentapi.ProtocolVersion)
	}
}

// TestController_Info_noAgent verifies that a nil agent omits the `agent` section, and
// with it the client's own protocol version: it is meaningful only against a running
// agent.
func TestController_Info_noAgent(t *testing.T) {
	t.Parallel()
	buf := &bytes.Buffer{}
	if err := info.New(buf, fakeEnv(nil)).Info("/x/ghtkn.yaml", "", nil, nil); err != nil {
		t.Fatal(err)
	}
	got := &info.Output{}
	if err := json.Unmarshal(buf.Bytes(), got); err != nil {
		t.Fatal(err)
	}
	if got.Agent != nil {
		t.Errorf("a nil agent must be omitted, got %+v", got.Agent)
	}
	if got.ProtocolVersion != nil {
		t.Errorf("the protocol version must be omitted without an agent, got %d", *got.ProtocolVersion)
	}
}
