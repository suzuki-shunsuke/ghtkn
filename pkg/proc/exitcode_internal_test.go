package proc

import (
	"errors"
	"fmt"
	"io/fs"
	"os/exec"
	"testing"
)

func TestStartExitCode(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		err  error
		want int
	}{
		{
			name: "the command isn't in PATH",
			err:  &exec.Error{Name: "gh", Err: exec.ErrNotFound},
			want: exitCodeNotFound,
		},
		{
			name: "the command doesn't exist",
			err:  fmt.Errorf("fork/exec ./gh: %w", fs.ErrNotExist),
			want: exitCodeNotFound,
		},
		{
			// A file found in PATH but not executable is reported as a permission
			// error, so it must not be taken for a missing command.
			name: "the command isn't executable",
			err:  &exec.Error{Name: "gh", Err: fs.ErrPermission},
			want: exitCodeNotExecutable,
		},
		{
			name: "some other failure",
			err:  errors.New("too many open files"),
			want: exitCodeUnclassified,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if code := startExitCode(tt.err); code != tt.want {
				t.Errorf("the exit code is %d, want %d", code, tt.want)
			}
		})
	}
}
