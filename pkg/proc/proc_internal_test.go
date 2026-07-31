package proc

import (
	"log/slog"
	"os"
	"syscall"
	"testing"
	"time"
)

// forwardTimeout bounds the wait for the forwarding goroutine so that a mistake in it
// fails the test instead of hanging it.
const forwardTimeout = 5 * time.Second

// fakeSignaler records what the forwarding goroutine does to the command, so that the
// goroutine can be tested without starting a process.
type fakeSignaler struct {
	calls chan string
}

func (f *fakeSignaler) Signal(sig os.Signal) error {
	f.calls <- "signal:" + sig.String()
	return nil
}

func (f *fakeSignaler) Kill() error {
	f.calls <- "kill"
	return nil
}

func TestForward(t *testing.T) { //nolint:funlen // A table of test cases.
	t.Parallel()
	tests := []struct {
		name      string
		signals   []os.Signal
		wantCalls []string
	}{
		{
			name:      "a signal is forwarded as it is",
			signals:   []os.Signal{syscall.SIGTERM},
			wantCalls: []string{"signal:" + syscall.SIGTERM.String()},
		},
		{
			// The command is ignoring the signal, so ghtkn stops waiting for it.
			name:      "the same signal twice kills the command",
			signals:   []os.Signal{syscall.SIGTERM, syscall.SIGTERM},
			wantCalls: []string{"signal:" + syscall.SIGTERM.String(), "kill"},
		},
		{
			name:    "different signals are both forwarded",
			signals: []os.Signal{syscall.SIGTERM, os.Interrupt},
			wantCalls: []string{
				"signal:" + syscall.SIGTERM.String(),
				"signal:" + os.Interrupt.String(),
			},
		},
		{
			// A second Ctrl-C must not kill an interactive command, which treats it as
			// "cancel the current line".
			name:    "SIGINT twice is forwarded twice",
			signals: []os.Signal{os.Interrupt, os.Interrupt},
			wantCalls: []string{
				"signal:" + os.Interrupt.String(),
				"signal:" + os.Interrupt.String(),
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			p := &fakeSignaler{calls: make(chan string, len(tt.wantCalls))}
			ch := make(chan os.Signal, len(tt.signals))
			done := make(chan struct{})
			finished := make(chan struct{})
			go func() {
				forward(slog.New(slog.DiscardHandler), p, ch, done)
				close(finished)
			}()

			for _, sig := range tt.signals {
				ch <- sig
			}
			for _, want := range tt.wantCalls {
				select {
				case got := <-p.calls:
					if got != want {
						t.Errorf("the command got %q, want %q", got, want)
					}
				case <-time.After(forwardTimeout):
					t.Fatalf("the command didn't get %q", want)
				}
			}

			// Closing done must stop the goroutine, or it would outlive the command.
			close(done)
			select {
			case <-finished:
			case <-time.After(forwardTimeout):
				t.Fatal("forwarding didn't stop")
			}
		})
	}
}
