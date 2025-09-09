package log_test

import (
	"errors"
	"log/slog"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/suzuki-shunsuke/ghtkn/pkg/log"
)

func TestNew(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		version string
		level   slog.Level
	}{
		{
			name:    "debug level logger",
			version: "1.0.0",
			level:   slog.LevelDebug,
		},
		{
			name:    "info level logger",
			version: "2.0.0",
			level:   slog.LevelInfo,
		},
		{
			name:    "warn level logger",
			version: "3.0.0",
			level:   slog.LevelWarn,
		},
		{
			name:    "error level logger",
			version: "4.0.0",
			level:   slog.LevelError,
		},
		{
			name:    "empty version",
			version: "",
			level:   slog.LevelInfo,
		},
		{
			name:    "version with special characters",
			version: "v1.2.3-beta+build123",
			level:   slog.LevelDebug,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			logger := log.New(tt.version, tt.level)
			if logger == nil {
				t.Fatal("New() returned nil logger")
			}
			// We can't easily test the internal structure of slog.Logger,
			// but we can verify it doesn't panic and returns a valid logger
		})
	}
}

func TestParseLevel(t *testing.T) { //nolint:funlen
	t.Parallel()
	tests := []struct {
		name    string
		input   string
		want    slog.Level
		wantErr error
	}{
		{
			name:    "parse debug level",
			input:   "debug",
			want:    slog.LevelDebug,
			wantErr: nil,
		},
		{
			name:    "parse info level",
			input:   "info",
			want:    slog.LevelInfo,
			wantErr: nil,
		},
		{
			name:    "parse warn level",
			input:   "warn",
			want:    slog.LevelWarn,
			wantErr: nil,
		},
		{
			name:    "parse error level",
			input:   "error",
			want:    slog.LevelError,
			wantErr: nil,
		},
		{
			name:    "unknown level",
			input:   "unknown",
			want:    0,
			wantErr: log.ErrUnknownLogLevel,
		},
		{
			name:    "empty string",
			input:   "",
			want:    0,
			wantErr: log.ErrUnknownLogLevel,
		},
		{
			name:    "uppercase level",
			input:   "DEBUG",
			want:    0,
			wantErr: log.ErrUnknownLogLevel,
		},
		{
			name:    "mixed case level",
			input:   "Info",
			want:    0,
			wantErr: log.ErrUnknownLogLevel,
		},
		{
			name:    "level with spaces",
			input:   " info ",
			want:    0,
			wantErr: log.ErrUnknownLogLevel,
		},
		{
			name:    "numeric level",
			input:   "0",
			want:    0,
			wantErr: log.ErrUnknownLogLevel,
		},
		{
			name:    "verbose level not supported",
			input:   "verbose",
			want:    0,
			wantErr: log.ErrUnknownLogLevel,
		},
		{
			name:    "trace level not supported",
			input:   "trace",
			want:    0,
			wantErr: log.ErrUnknownLogLevel,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, err := log.ParseLevel(tt.input)

			if tt.wantErr != nil {
				if err == nil {
					t.Fatalf("ParseLevel() error = nil, wantErr %v", tt.wantErr)
				}
				if !errors.Is(err, tt.wantErr) {
					t.Errorf("ParseLevel() error = %v, wantErr %v", err, tt.wantErr)
				}
				return
			}

			if err != nil {
				t.Fatalf("ParseLevel() unexpected error = %v", err)
			}

			if diff := cmp.Diff(tt.want, got); diff != "" {
				t.Errorf("ParseLevel() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestErrUnknownLogLevel(t *testing.T) {
	t.Parallel()
	// Test that the error has the expected message
	if got := log.ErrUnknownLogLevel.Error(); got != "unknown log level" {
		t.Errorf("ErrUnknownLogLevel.Error() = %q, want %q", got, "unknown log level")
	}
}
