package config_test

import (
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
