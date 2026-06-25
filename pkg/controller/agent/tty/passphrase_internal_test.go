package tty

import (
	"errors"
	"testing"
)

func TestPromptPassphrase(t *testing.T) {
	t.Parallel()

	t.Run("existing prompts once", func(t *testing.T) {
		t.Parallel()
		calls := 0
		read := func(string) ([]byte, error) {
			calls++
			return []byte("pass"), nil
		}
		got, err := PromptPassphrase(read, true)
		if err != nil {
			t.Fatal(err)
		}
		if calls != 1 {
			t.Fatalf("read called %d times, want 1", calls)
		}
		if string(got) != "pass" {
			t.Fatalf("passphrase = %q, want %q", got, "pass")
		}
	})

	t.Run("first run confirms", func(t *testing.T) {
		t.Parallel()
		calls := 0
		read := func(string) ([]byte, error) {
			calls++
			return []byte("pass"), nil
		}
		if _, err := PromptPassphrase(read, false); err != nil {
			t.Fatal(err)
		}
		if calls != 2 {
			t.Fatalf("read called %d times, want 2", calls)
		}
	})

	t.Run("first run mismatch", func(t *testing.T) {
		t.Parallel()
		seq := [][]byte{[]byte("a"), []byte("b")}
		i := 0
		read := func(string) ([]byte, error) {
			v := seq[i]
			i++
			return v, nil
		}
		if _, err := PromptPassphrase(read, false); !errors.Is(err, ErrPassphraseMismatch) {
			t.Fatalf("err = %v, want ErrPassphraseMismatch", err)
		}
	})
}
