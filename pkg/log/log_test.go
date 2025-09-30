package log_test

import (
	"io"
	"log/slog"
	"testing"

	"github.com/suzuki-shunsuke/ghtkn/pkg/log"
)

func TestNew(t *testing.T) {
	t.Parallel()
	logger, _ := log.New(io.Discard, "v0.1.0")
	if logger == nil {
		t.Fatal("New() returned nil logger")
	}
}

func TestSetLevel(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name  string
		input string
	}{
		{
			name:  "parse debug level",
			input: "debug",
		},
		{
			name:  "unknown level",
			input: "unknown",
		},
		{
			name:  "empty string",
			input: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			log.SetLevel(slog.Default(), &slog.LevelVar{}, tt.input)
		})
	}
}
