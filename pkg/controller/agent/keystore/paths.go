package keystore

import (
	"errors"
	"path/filepath"
)

// goosWindows is the runtime.GOOS value for Windows.
const goosWindows = "windows"

// cacheDir resolves the base cache directory. On Windows it is %LocalAppData%\cache;
// otherwise it honors XDG_CACHE_HOME and falls back to $HOME/.cache. This mirrors how
// the ghtkn SDK's text backend resolves its storage directory.
func cacheDir(getEnv func(string) string, goos string) (string, error) {
	if goos == goosWindows {
		if d := getEnv("LocalAppData"); d != "" {
			return filepath.Join(d, "cache"), nil
		}
		return "", errors.New("LocalAppData is required to use the agent backend on Windows")
	}
	if d := getEnv("XDG_CACHE_HOME"); d != "" {
		return d, nil
	}
	if home := getEnv("HOME"); home != "" {
		return filepath.Join(home, ".cache"), nil
	}
	return "", errors.New("XDG_CACHE_HOME or HOME is required to use the agent backend")
}

// dataDir resolves the base data directory. On Windows it is %LocalAppData%;
// otherwise it honors XDG_DATA_HOME and falls back to $HOME/.local/share.
// The key file is persistent data (losing it makes the encrypted tokens
// unrecoverable), not user-editable config, so it lives here rather than under the
// config dir.
func dataDir(getEnv func(string) string, goos string) (string, error) {
	if goos == goosWindows {
		if d := getEnv("LocalAppData"); d != "" {
			return d, nil
		}
		return "", errors.New("LocalAppData is required to use the agent backend on Windows")
	}
	if d := getEnv("XDG_DATA_HOME"); d != "" {
		return d, nil
	}
	if home := getEnv("HOME"); home != "" {
		return filepath.Join(home, ".local", "share"), nil
	}
	return "", errors.New("XDG_DATA_HOME or HOME is required to use the agent backend")
}

// TokenDir resolves the directory that stores encrypted token files.
// GHTKN_AGENT_TOKEN_DIR takes precedence; otherwise it is ${cache dir}/ghtkn/agent.
func TokenDir(getEnv func(string) string, goos string) (string, error) {
	if dir := getEnv("GHTKN_AGENT_TOKEN_DIR"); dir != "" {
		return dir, nil
	}
	dir, err := cacheDir(getEnv, goos)
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "ghtkn", "agent"), nil
}

// KeyPath resolves the path of the wrapped data key file.
// GHTKN_AGENT_KEY takes precedence; otherwise it is ${data dir}/ghtkn/key.
func KeyPath(getEnv func(string) string, goos string) (string, error) {
	if path := getEnv("GHTKN_AGENT_KEY"); path != "" {
		return path, nil
	}
	dir, err := dataDir(getEnv, goos)
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "ghtkn", "key"), nil
}
