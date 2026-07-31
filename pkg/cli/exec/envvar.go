package exec

import (
	"fmt"
	"strings"

	"github.com/suzuki-shunsuke/ghtkn/pkg/controller/exec"
)

// defaultEnvName is the environment variable the access token is set to when no -e is
// given.
const defaultEnvName = "GITHUB_TOKEN"

// parseEnvs converts the values of -e into the environment variables to set.
//
// Without -e, the access token of the app ghtkn selects automatically is set to
// GITHUB_TOKEN. Any -e replaces that default entirely, so GITHUB_TOKEN is not set
// implicitly once -e is given.
func parseEnvs(values []string) ([]*exec.EnvVar, error) {
	if len(values) == 0 {
		return []*exec.EnvVar{
			{
				Name: defaultEnvName,
			},
		}, nil
	}
	envVars := make([]*exec.EnvVar, 0, len(values))
	names := make(map[string]struct{}, len(values))
	for _, value := range values {
		envVar, err := parseEnv(value)
		if err != nil {
			return nil, err
		}
		if _, ok := names[envVar.Name]; ok {
			// Dropping one of them silently would leave it unclear which app the
			// environment variable gets its token from.
			return nil, fmt.Errorf("the environment variable is set by -e more than once: %s", envVar.Name)
		}
		names[envVar.Name] = struct{}{}
		envVars = append(envVars, envVar)
	}
	return envVars, nil
}

// parseEnv parses one -e value, which is an environment variable name optionally
// followed by ':' and the name of the app whose access token it receives.
func parseEnv(value string) (*exec.EnvVar, error) {
	// The name is cut at the first ':' so that an app name containing one still works.
	// Environment variable names don't contain ':' in practice.
	name, appName, hasApp := strings.Cut(value, ":")
	switch {
	case name == "":
		return nil, fmt.Errorf("-e requires an environment variable name, like -e GH_TOKEN or -e GH_TOKEN:my-app: %q", value)
	case strings.Contains(name, "="):
		// -e takes an app name, not a value, so accepting '=' would invite reading
		// 'GH_TOKEN=my-app' as "set GH_TOKEN to my-app".
		return nil, fmt.Errorf("-e takes <env name>[:<app name>], not <env name>=<value>. Separate the app name with ':', like -e GH_TOKEN:my-app: %q", value)
	case hasApp && appName == "":
		return nil, fmt.Errorf("the app name after ':' is empty. Remove the ':' to use the app ghtkn selects automatically: %q", value)
	}
	return &exec.EnvVar{
		Name:    name,
		AppName: appName,
	}, nil
}
