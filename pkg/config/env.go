package config

// Env represents environment variables used by ghtkn.
// It contains configuration paths and app selection settings.
type Env struct {
	XDGConfigHome string
	App           string
}

// NewEnv creates a new Env struct by reading environment variables.
func NewEnv(getEnv func(string) string) *Env {
	return &Env{
		XDGConfigHome: getEnv("XDG_CONFIG_HOME"),
		App:           getEnv("GHTKN_APP"),
	}
}
