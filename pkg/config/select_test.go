package config_test

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/suzuki-shunsuke/ghtkn/pkg/config"
)

func TestConfig_SelectApp(t *testing.T) { //nolint:funlen
	t.Parallel()
	tests := []struct {
		name string
		cfg  *config.Config
		key  string
		want *config.App
	}{
		{
			name: "nil config returns nil",
			cfg:  nil,
			key:  "test",
			want: nil,
		},
		{
			name: "empty apps returns nil",
			cfg: &config.Config{
				Apps: []*config.App{},
			},
			key:  "test",
			want: nil,
		},
		{
			name: "select by key match",
			cfg: &config.Config{
				Apps: []*config.App{
					{
						Name:     "app1",
						ClientID: "client1",
					},
					{
						Name:     "app2",
						ClientID: "client2",
						Default:  true,
					},
					{
						Name:     "app3",
						ClientID: "client3",
					},
				},
			},
			key: "app3",
			want: &config.App{
				Name:     "app3",
				ClientID: "client3",
			},
		},
		{
			name: "select default when no key match",
			cfg: &config.Config{
				Apps: []*config.App{
					{
						Name:     "app1",
						ClientID: "client1",
					},
					{
						Name:     "app2",
						ClientID: "client2",
						Default:  true,
					},
					{
						Name:     "app3",
						ClientID: "client3",
					},
				},
			},
			key: "nonexistent",
			want: &config.App{
				Name:     "app2",
				ClientID: "client2",
				Default:  true,
			},
		},
		{
			name: "select default when empty key",
			cfg: &config.Config{
				Apps: []*config.App{
					{
						Name:     "app1",
						ClientID: "client1",
					},
					{
						Name:     "app2",
						ClientID: "client2",
						Default:  true,
					},
				},
			},
			key: "",
			want: &config.App{
				Name:     "app2",
				ClientID: "client2",
				Default:  true,
			},
		},
		{
			name: "select first when no default and no key match",
			cfg: &config.Config{
				Apps: []*config.App{
					{
						Name:     "app1",
						ClientID: "client1",
					},
					{
						Name:     "app2",
						ClientID: "client2",
					},
					{
						Name:     "app3",
						ClientID: "client3",
					},
				},
			},
			key: "nonexistent",
			want: &config.App{
				Name:     "app1",
				ClientID: "client1",
			},
		},
		{
			name: "select first when no default and empty key",
			cfg: &config.Config{
				Apps: []*config.App{
					{
						Name:     "app1",
						ClientID: "client1",
					},
					{
						Name:     "app2",
						ClientID: "client2",
					},
				},
			},
			key: "",
			want: &config.App{
				Name:     "app1",
				ClientID: "client1",
			},
		},
		{
			name: "single app without default",
			cfg: &config.Config{
				Apps: []*config.App{
					{
						Name:     "only-app",
						ClientID: "client123",
					},
				},
			},
			key: "",
			want: &config.App{
				Name:     "only-app",
				ClientID: "client123",
			},
		},
		{
			name: "single app with default",
			cfg: &config.Config{
				Apps: []*config.App{
					{
						Name:     "only-app",
						ClientID: "client123",
						Default:  true,
					},
				},
			},
			key: "",
			want: &config.App{
				Name:     "only-app",
				ClientID: "client123",
				Default:  true,
			},
		},
		{
			name: "multiple defaults - select first default",
			cfg: &config.Config{
				Apps: []*config.App{
					{
						Name:     "app1",
						ClientID: "client1",
					},
					{
						Name:     "app2",
						ClientID: "client2",
						Default:  true,
					},
					{
						Name:     "app3",
						ClientID: "client3",
						Default:  true,
					},
				},
			},
			key: "",
			want: &config.App{
				Name:     "app2",
				ClientID: "client2",
				Default:  true,
			},
		},
		{
			name: "key takes precedence over default",
			cfg: &config.Config{
				Apps: []*config.App{
					{
						Name:     "app1",
						ClientID: "client1",
					},
					{
						Name:     "app2",
						ClientID: "client2",
						Default:  true,
					},
				},
			},
			key: "app1",
			want: &config.App{
				Name:     "app1",
				ClientID: "client1",
			},
		},
		{
			name: "case sensitive key matching",
			cfg: &config.Config{
				Apps: []*config.App{
					{
						Name:     "MyApp",
						ClientID: "client1",
					},
					{
						Name:     "myapp",
						ClientID: "client2",
					},
				},
			},
			key: "myapp",
			want: &config.App{
				Name:     "myapp",
				ClientID: "client2",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := tt.cfg.SelectApp(tt.key)
			if diff := cmp.Diff(tt.want, got); diff != "" {
				t.Errorf("SelectApp() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}
