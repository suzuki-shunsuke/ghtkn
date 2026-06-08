package info_test

import (
	"bytes"
	"encoding/json"
	"reflect"
	"runtime"
	"testing"

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
				Backend:    "keyring",
				App:        "",
				ConfigPath: configPath,
			},
		},
		{
			name: "GHTKN_BACKEND is reflected in Backend and Envs",
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
				Backend:    "text",
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
				Backend:    "keyring",
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
				Backend:    "keyring",
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
				Backend:    "keyring",
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
				Backend:    "agent",
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
				Backend:    "keyring",
				App:        "",
				ConfigPath: configPath,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			buf := &bytes.Buffer{}
			ctrl := info.New(buf, fakeEnv(tt.env))

			if err := ctrl.Info(configPath, tt.appName, tt.version); err != nil {
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
