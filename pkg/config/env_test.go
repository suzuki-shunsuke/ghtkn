package config_test

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/suzuki-shunsuke/ghtkn/pkg/config"
)

func TestNewEnv(t *testing.T) { //nolint:funlen
	t.Parallel()
	tests := []struct {
		name   string
		envMap map[string]string
		want   *config.Env
	}{
		{
			name: "all environment variables set",
			envMap: map[string]string{
				"XDG_CONFIG_HOME": "/home/user/.config",
				"GHTKN_APP":       "my-app",
			},
			want: &config.Env{
				XDGConfigHome: "/home/user/.config",
				App:           "my-app",
			},
		},
		{
			name: "XDG_CONFIG_HOME only",
			envMap: map[string]string{
				"XDG_CONFIG_HOME": "/custom/config",
			},
			want: &config.Env{
				XDGConfigHome: "/custom/config",
				App:           "",
			},
		},
		{
			name: "GHTKN_APP only",
			envMap: map[string]string{
				"GHTKN_APP": "test-app",
			},
			want: &config.Env{
				XDGConfigHome: "",
				App:           "test-app",
			},
		},
		{
			name:   "no environment variables set",
			envMap: map[string]string{},
			want: &config.Env{
				XDGConfigHome: "",
				App:           "",
			},
		},
		{
			name: "with other environment variables",
			envMap: map[string]string{
				"XDG_CONFIG_HOME": "/home/user/.config",
				"GHTKN_APP":       "app1",
				"PATH":            "/usr/bin:/bin",
				"HOME":            "/home/user",
			},
			want: &config.Env{
				XDGConfigHome: "/home/user/.config",
				App:           "app1",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			getEnv := func(key string) string {
				return tt.envMap[key]
			}
			got := config.NewEnv(getEnv)
			if diff := cmp.Diff(tt.want, got); diff != "" {
				t.Errorf("NewEnv() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}
