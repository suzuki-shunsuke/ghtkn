package tokenstore

import (
	"errors"
	"path/filepath"

	"github.com/suzuki-shunsuke/ghtkn-go-sdk/ghtkn/env"
)

// goosWindows is the runtime.GOOS value for Windows.
const goosWindows = "windows"

// cacheDir resolves the base cache directory. On Windows it is %LocalAppData%\cache;
// otherwise it honors XDG_CACHE_HOME and falls back to $HOME/.cache. This mirrors how
// the ghtkn SDK's text backend resolves its storage directory.
func cacheDir(getEnv func(string) string, goos string) (string, error) {
	if goos == goosWindows {
		if d := getEnv(env.LocalAppData); d != "" {
			return filepath.Join(d, "cache"), nil
		}
		return "", errors.New("LocalAppData is required to use the agent backend on Windows")
	}
	if d := getEnv(env.XDGCacheHome); d != "" {
		return d, nil
	}
	if home := getEnv(env.Home); home != "" {
		return filepath.Join(home, ".cache"), nil
	}
	return "", errors.New("XDG_CACHE_HOME or HOME is required to use the agent backend")
}

// TokenDir resolves the directory that stores encrypted token files.
// GHTKN_AGENT_TOKEN_DIR takes precedence; otherwise it is ${cache dir}/ghtkn/agent.
func TokenDir(getEnv func(string) string, goos string) (string, error) {
	if dir := getEnv(env.AgentTokenDir); dir != "" {
		return dir, nil
	}
	dir, err := cacheDir(getEnv, goos)
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "ghtkn", "agent"), nil
}
