package exec

import "errors"

// usage is appended to the errors about how the command line is written, so that the
// user doesn't have to look the form up.
const usage = "ghtkn exec [<flags>] -- <command> [<args>...]"

// requireSeparator verifies that '--' separates the flags of 'ghtkn exec' from the
// command to run.
//
// Requiring it is what keeps the command's own flags from being parsed by ghtkn: in
// 'ghtkn exec gh pr view -h', the -h would be ghtkn's.
//
// argsLenAtDash is cobra's ArgsLenAtDash: the number of positional arguments that
// came before the '--', or -1 when there is none. Only 0 is the form we want, which
// is also what distinguishes the separator from a '--' belonging to the command
// itself: in 'ghtkn exec sh -c foo -- bar' there is a '--', but the command starts
// before it, so the count is not 0.
func requireSeparator(argsLenAtDash int, command []string) error {
	if len(command) == 0 {
		return errors.New("a command to run is required: " + usage)
	}
	if argsLenAtDash != 0 {
		return errors.New("'--' is required in front of the command so that ghtkn doesn't parse the command's own flags: " + usage)
	}
	return nil
}
