package config_test

import (
	"runtime"
	"testing"

	"github.com/suzuki-shunsuke/ghtkn/pkg/config"
)

func TestResolvePath_explicit(t *testing.T) {
	t.Parallel()
	// A non-empty path is returned unchanged without touching the environment.
	const path = "/explicit/ghtkn.yaml"
	got, err := config.ResolvePath(path)
	if err != nil {
		t.Fatalf("ResolvePath() returned an unexpected error: %v", err)
	}
	if got != path {
		t.Errorf("ResolvePath() = %q, want %q", got, path)
	}
}

func TestResolvePath_default(t *testing.T) {
	// GHTKN_CONFIG takes precedence in the SDK's default resolution, so setting
	// it makes the fallback deterministic across platforms.
	const path = "/default/ghtkn.yaml"
	t.Setenv("GHTKN_CONFIG", path)

	got, err := config.ResolvePath("")
	if err != nil {
		t.Fatalf("ResolvePath() returned an unexpected error: %v", err)
	}
	if got != path {
		t.Errorf("ResolvePath() = %q, want %q", got, path)
	}
}

func TestResolvePath_error(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("config path resolution differs on Windows")
	}
	// With GHTKN_CONFIG, XDG_CONFIG_HOME, and HOME all empty, the SDK can't
	// resolve a default path, so ResolvePath must return an error.
	t.Setenv("GHTKN_CONFIG", "")
	t.Setenv("XDG_CONFIG_HOME", "")
	t.Setenv("HOME", "")

	if _, err := config.ResolvePath(""); err == nil {
		t.Fatal("expected an error but got nil")
	}
}
