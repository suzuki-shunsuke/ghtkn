package exec

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/suzuki-shunsuke/ghtkn/pkg/controller/exec"
)

func TestParseEnvs(t *testing.T) { //nolint:funlen // A table of test cases.
	t.Parallel()
	tests := []struct {
		name    string
		values  []string
		want    []*exec.EnvVar
		wantErr bool
	}{
		{
			name:   "no -e sets GITHUB_TOKEN with the app ghtkn selects",
			values: nil,
			want:   []*exec.EnvVar{{Name: "GITHUB_TOKEN"}},
		},
		{
			name:   "an -e without an app name replaces the default",
			values: []string{"GH_TOKEN"},
			want:   []*exec.EnvVar{{Name: "GH_TOKEN"}},
		},
		{
			name:   "an app name",
			values: []string{"GH_TOKEN:suzuki-shunsuke/write"},
			want:   []*exec.EnvVar{{Name: "GH_TOKEN", AppName: "suzuki-shunsuke/write"}},
		},
		{
			name:   "several environment variables keep their order",
			values: []string{"A:app-a", "B", "C:app-c"},
			want: []*exec.EnvVar{
				{Name: "A", AppName: "app-a"},
				{Name: "B"},
				{Name: "C", AppName: "app-c"},
			},
		},
		{
			// The name is cut at the first colon, so an app name containing one works.
			name:   "an app name containing a colon",
			values: []string{"A:foo:bar"},
			want:   []*exec.EnvVar{{Name: "A", AppName: "foo:bar"}},
		},
		{
			name:    "an empty environment variable name",
			values:  []string{":app"},
			wantErr: true,
		},
		{
			name:    "an empty app name",
			values:  []string{"A:"},
			wantErr: true,
		},
		{
			name:    "'=' instead of ':'",
			values:  []string{"GH_TOKEN=my-app"},
			wantErr: true,
		},
		{
			name:    "the same environment variable twice",
			values:  []string{"GH_TOKEN:a", "GH_TOKEN:b"},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			envVars, err := parseEnvs(tt.values)
			if tt.wantErr {
				if err == nil {
					t.Fatal("an error must be returned")
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if diff := cmp.Diff(tt.want, envVars); diff != "" {
				t.Errorf("env vars are unexpected (-want +got):\n%s", diff)
			}
		})
	}
}
