package exec

import (
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestBuildEnv(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		environ []string
		vars    []*envVar
		want    []string
	}{
		{
			name:    "the token is appended",
			environ: []string{"PATH=/bin", "HOME=/home/foo"},
			vars:    []*envVar{{name: "GITHUB_TOKEN", value: "token-1"}},
			want:    []string{"PATH=/bin", "HOME=/home/foo", "GITHUB_TOKEN=token-1"},
		},
		{
			name:    "an inherited variable of the same name is replaced",
			environ: []string{"PATH=/bin", "GITHUB_TOKEN=old", "HOME=/home/foo"},
			vars:    []*envVar{{name: "GITHUB_TOKEN", value: "token-1"}},
			want:    []string{"PATH=/bin", "HOME=/home/foo", "GITHUB_TOKEN=token-1"},
		},
		{
			name:    "the order of the tokens follows the order of the variables",
			environ: []string{"PATH=/bin"},
			vars: []*envVar{
				{name: "A", value: "token-a"},
				{name: "B", value: "token-b"},
			},
			want: []string{"PATH=/bin", "A=token-a", "B=token-b"},
		},
		{
			name:    "the environment is passed through when there is no token",
			environ: []string{"PATH=/bin", "GITHUB_TOKEN=inherited"},
			vars:    nil,
			want:    []string{"PATH=/bin", "GITHUB_TOKEN=inherited"},
		},
		{
			// os.Environ can hold an entry without '=' on some platforms, which is
			// neither a name to replace nor a reason to drop it.
			name:    "an entry without a value",
			environ: []string{"PATH=/bin", "broken"},
			vars:    []*envVar{{name: "GITHUB_TOKEN", value: "token-1"}},
			want:    []string{"PATH=/bin", "broken", "GITHUB_TOKEN=token-1"},
		},
		{
			name:    "a token whose value contains '='",
			environ: []string{"PATH=/bin"},
			vars:    []*envVar{{name: "GITHUB_TOKEN", value: "a=b"}},
			want:    []string{"PATH=/bin", "GITHUB_TOKEN=a=b"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if diff := cmp.Diff(tt.want, buildEnv(tt.environ, tt.vars)); diff != "" {
				t.Errorf("the environment is unexpected (-want +got):\n%s", diff)
			}
		})
	}
}
