package agent

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

// configDir resolves the base config directory. On Windows it is %AppData%;
// otherwise it honors XDG_CONFIG_HOME and falls back to $HOME/.config.
func configDir(getEnv func(string) string, goos string) (string, error) {
	if goos == goosWindows {
		if d := getEnv("AppData"); d != "" {
			return d, nil
		}
		return "", errors.New("AppData is required to use the agent backend on Windows")
	}
	if d := getEnv("XDG_CONFIG_HOME"); d != "" {
		return d, nil
	}
	if home := getEnv("HOME"); home != "" {
		return filepath.Join(home, ".config"), nil
	}
	return "", errors.New("XDG_CONFIG_HOME or HOME is required to use the agent backend")
}

// tokenDir resolves the directory that stores encrypted token files:
// ${cache dir}/ghtkn/agent.
func tokenDir(getEnv func(string) string, goos string) (string, error) {
	dir, err := cacheDir(getEnv, goos)
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "ghtkn", "agent"), nil
}

// keyPath resolves the path of the wrapped data key file:
// ${config dir}/ghtkn/key.
func keyPath(getEnv func(string) string, goos string) (string, error) {
	dir, err := configDir(getEnv, goos)
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "ghtkn", "key"), nil
}
