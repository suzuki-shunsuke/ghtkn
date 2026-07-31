package exec

import "strings"

// buildEnv returns ghtkn's environment with the access tokens in vars added to it.
//
// A variable of the same name inherited from ghtkn's environment is replaced rather
// than duplicated, so the command sees one entry per name. A variable whose token
// couldn't be acquired is not in vars, so the value inherited from ghtkn's
// environment, if any, is passed through untouched; that is what -continue-on-error
// means for it.
//
// Environment variable names are case insensitive on Windows, which this doesn't
// handle: os/exec folds the case there itself and keeps the last entry of a name, and
// the entries built here are appended last, so they win. Folding the case here as
// well would only diverge from the platform.
func buildEnv(environ []string, vars []*envVar) []string {
	replaced := make(map[string]struct{}, len(vars))
	for _, v := range vars {
		replaced[v.name] = struct{}{}
	}
	envs := make([]string, 0, len(environ)+len(vars))
	for _, env := range environ {
		name, _, _ := strings.Cut(env, "=")
		if _, ok := replaced[name]; ok {
			continue
		}
		envs = append(envs, env)
	}
	for _, v := range vars {
		envs = append(envs, v.name+"="+v.value)
	}
	return envs
}
