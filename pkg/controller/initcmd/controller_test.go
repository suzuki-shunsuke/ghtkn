package initcmd_test

import (
	"testing"

	"github.com/spf13/afero"
	"github.com/suzuki-shunsuke/ghtkn/pkg/config"
	"github.com/suzuki-shunsuke/ghtkn/pkg/controller/initcmd"
)

func TestNew(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		fs   afero.Fs
		env  *config.Env
	}{
		{
			name: "create controller with memory filesystem",
			fs:   afero.NewMemMapFs(),
			env: &config.Env{
				XDGConfigHome: "/home/user/.config",
				App:           "test-app",
			},
		},
		{
			name: "create controller with nil env",
			fs:   afero.NewMemMapFs(),
			env:  nil,
		},
		{
			name: "create controller with empty env",
			fs:   afero.NewMemMapFs(),
			env:  &config.Env{},
		},
		{
			name: "create controller with os filesystem",
			fs:   afero.NewOsFs(),
			env: &config.Env{
				XDGConfigHome: "/tmp/.config",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if ctrl := initcmd.New(tt.fs, tt.env); ctrl == nil {
				t.Fatal("New() returned nil controller")
			}
		})
	}
}
