package get_test

import (
	"testing"

	"github.com/suzuki-shunsuke/ghtkn/pkg/config"
	"github.com/suzuki-shunsuke/ghtkn/pkg/controller/get"
)

type mockConfigReader struct {
	cfg *config.Config
	err error
}

func (m *mockConfigReader) Read(cfg *config.Config, _ string) error {
	if m.err != nil {
		return m.err
	}
	if m.cfg != nil {
		*cfg = *m.cfg
	}
	return nil
}

func TestNew(t *testing.T) {
	t.Parallel()

	input := &get.Input{}
	controller := get.New(input)
	if controller == nil {
		t.Error("New() returned nil")
	}
}

func TestNewInput(t *testing.T) {
	t.Parallel()

	input := get.NewInput("/path/to/config")
	if input == nil {
		t.Error("NewInput() returned nil")
		return
	}

	if input.ConfigFilePath != "/path/to/config" {
		t.Errorf("NewInput().ConfigFilePath = %v, want /path/to/config", input.ConfigFilePath)
	}

	if input.FS == nil {
		t.Error("NewInput().FS is nil")
	}

	if input.ConfigReader == nil {
		t.Error("NewInput().ConfigReader is nil")
	}

	if input.Env == nil {
		t.Error("NewInput().Env is nil")
	}

	if input.Stdout == nil {
		t.Error("NewInput().Stdout is nil")
	}
}

func TestInput_IsJSON(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		outputFormat string
		want         bool
	}{
		{
			name:         "json format",
			outputFormat: "json",
			want:         true,
		},
		{
			name:         "empty format",
			outputFormat: "",
			want:         false,
		},
		{
			name:         "other format",
			outputFormat: "yaml",
			want:         false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			input := &get.Input{
				OutputFormat: tt.outputFormat,
			}

			got := input.IsJSON()
			if got != tt.want {
				t.Errorf("IsJSON() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestInput_Validate(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		outputFormat string
		wantErr      bool
	}{
		{
			name:         "valid json format",
			outputFormat: "json",
			wantErr:      false,
		},
		{
			name:         "valid empty format",
			outputFormat: "",
			wantErr:      false,
		},
		{
			name:         "invalid format",
			outputFormat: "yaml",
			wantErr:      true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			input := &get.Input{
				OutputFormat: tt.outputFormat,
			}

			err := input.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
