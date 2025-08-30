package config

type Env struct {
	XDGConfigHome string
	App           string
}

func NewEnv(getEnv func(string) string) *Env {
	return &Env{
		XDGConfigHome: getEnv("XDG_CONFIG_HOME"),
		App:           getEnv("GHTKN_APP"),
	}
}
