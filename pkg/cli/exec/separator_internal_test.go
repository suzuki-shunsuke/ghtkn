package exec

import "testing"

func TestRequireSeparator(t *testing.T) { //nolint:funlen // A table of test cases.
	t.Parallel()
	tests := []struct {
		name    string
		rawArgs []string
		command []string
		wantErr bool
	}{
		{
			name:    "the separator is right in front of the command",
			rawArgs: []string{"ghtkn", "exec", "--", "gh", "pr", "view"},
			command: []string{"gh", "pr", "view"},
		},
		{
			name:    "flags come before the separator",
			rawArgs: []string{"ghtkn", "exec", "-e", "GH_TOKEN", "--", "gh"},
			command: []string{"gh"},
		},
		{
			name:    "the command has a separator of its own",
			rawArgs: []string{"ghtkn", "exec", "--", "git", "log", "--", "README.md"},
			command: []string{"git", "log", "--", "README.md"},
		},
		{
			name:    "no separator at all",
			rawArgs: []string{"ghtkn", "exec", "gh", "pr", "view"},
			command: []string{"gh", "pr", "view"},
			wantErr: true,
		},
		{
			name:    "no command",
			rawArgs: []string{"ghtkn", "exec"},
			command: nil,
			wantErr: true,
		},
		{
			name:    "no command after the separator",
			rawArgs: []string{"ghtkn", "exec", "--"},
			command: nil,
			wantErr: true,
		},
		{
			// The separator exists but the command starts before it, so ghtkn would
			// have parsed the command's flags.
			name:    "the separator is not where the command starts",
			rawArgs: []string{"ghtkn", "exec", "sh", "-c", "foo", "--", "bar"},
			command: []string{"sh", "-c", "foo", "--", "bar"},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			err := requireSeparator(tt.rawArgs, tt.command)
			if tt.wantErr {
				if err == nil {
					t.Fatal("an error must be returned")
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
		})
	}
}
