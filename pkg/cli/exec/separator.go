package exec

import "errors"

// usage is appended to the errors about how the command line is written, so that the
// user doesn't have to look the form up.
const usage = "ghtkn exec [<flags>] -- <command> [<args>...]"

// requireSeparator verifies that '--' separates the flags of 'ghtkn exec' from the
// command to run.
//
// urfave/cli consumes the '--' while parsing and appends everything after it to the
// positional arguments, so the raw command line is the only place it can be observed.
// Requiring it is what keeps the command's own flags from being parsed by ghtkn: in
// 'ghtkn exec gh pr view -h', the -h would be ghtkn's.
//
// The separator is identified by its position rather than by searching for '--', so
// that a '--' belonging to the command itself isn't mistaken for it. In
// 'ghtkn exec sh -c foo -- bar' there is a '--', but the command starts before it.
func requireSeparator(rawArgs, command []string) error {
	if len(command) == 0 {
		return errors.New("a command to run is required: " + usage)
	}
	i := len(rawArgs) - len(command) - 1
	if i < 0 || rawArgs[i] != "--" {
		return errors.New("'--' is required in front of the command so that ghtkn doesn't parse the command's own flags: " + usage)
	}
	return nil
}
