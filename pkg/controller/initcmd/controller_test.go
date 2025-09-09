package initcmd_test

import (
	"testing"

	"github.com/spf13/afero"
	"github.com/suzuki-shunsuke/ghtkn/pkg/controller/initcmd"
)

func TestNew(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		fs   afero.Fs
	}{
		{
			name: "create controller with memory filesystem",
			fs:   afero.NewMemMapFs(),
		},
		{
			name: "create controller with nil env",
			fs:   afero.NewMemMapFs(),
		},
		{
			name: "create controller with empty env",
			fs:   afero.NewMemMapFs(),
		},
		{
			name: "create controller with os filesystem",
			fs:   afero.NewOsFs(),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if ctrl := initcmd.New(tt.fs); ctrl == nil {
				t.Fatal("New() returned nil controller")
			}
		})
	}
}
