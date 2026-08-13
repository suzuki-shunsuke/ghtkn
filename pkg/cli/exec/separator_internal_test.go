package exec

import "testing"

func TestRequireSeparator(t *testing.T) { //nolint:funlen // A table of test cases.
	t.Parallel()
	tests := []struct {
		name string
		// commandLine is the 'ghtkn exec' command line the case stands for. It is not
		// passed to requireSeparator, which sees only what cobra made of it; it is here
		// so that the two fields below can be read as the parse of something.
		commandLine   string
		argsLenAtDash int
		command       []string
		wantErr       bool
	}{
		{
			name:          "the separator is right in front of the command",
			commandLine:   "ghtkn exec -- gh pr view",
			argsLenAtDash: 0,
			command:       []string{"gh", "pr", "view"},
		},
		{
			name:          "flags come before the separator",
			commandLine:   "ghtkn exec -e GH_TOKEN -- gh",
			argsLenAtDash: 0,
			command:       []string{"gh"},
		},
		{
			// Everything after the first '--' is the command, including a '--' of its
			// own, which pflag leaves in the arguments untouched.
			name:          "the command has a separator of its own",
			commandLine:   "ghtkn exec -- git log -- README.md",
			argsLenAtDash: 0,
			command:       []string{"git", "log", "--", "README.md"},
		},
		{
			name:          "no separator at all",
			commandLine:   "ghtkn exec gh pr view",
			argsLenAtDash: -1,
			command:       []string{"gh", "pr", "view"},
			wantErr:       true,
		},
		{
			name:          "no command",
			commandLine:   "ghtkn exec",
			argsLenAtDash: -1,
			command:       nil,
			wantErr:       true,
		},
		{
			name:          "no command after the separator",
			commandLine:   "ghtkn exec --",
			argsLenAtDash: 0,
			command:       nil,
			wantErr:       true,
		},
		{
			// The separator exists but the command starts before it, so ghtkn would
			// have parsed the command's flags. Here it did: the -c went to ghtkn's own
			// --config, which is why only 'sh' precedes the separator.
			name:          "the separator is not where the command starts",
			commandLine:   "ghtkn exec sh -c foo -- bar",
			argsLenAtDash: 1,
			command:       []string{"sh", "bar"},
			wantErr:       true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			err := requireSeparator(tt.argsLenAtDash, tt.command)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("an error must be returned: %s", tt.commandLine)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
		})
	}
}
