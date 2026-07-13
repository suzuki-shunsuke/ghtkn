package keyfile

import (
	"errors"
	"path/filepath"

	"github.com/suzuki-shunsuke/ghtkn-go-sdk/ghtkn/env"
)

// goosWindows is the runtime.GOOS value for Windows.
const goosWindows = "windows"

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

// KeyPath resolves the path of the wrapped data key file.
// GHTKN_AGENT_KEY takes precedence; otherwise it is ${data dir}/ghtkn/key.
func KeyPath(getEnv func(string) string, goos string) (string, error) {
	if path := getEnv(env.AgentKey); path != "" {
		return path, nil
	}
	dir, err := dataDir(getEnv, goos)
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "ghtkn", "key"), nil
}
