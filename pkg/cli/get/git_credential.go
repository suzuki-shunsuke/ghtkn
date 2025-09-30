package get

import (
	"bufio"
	"context"
	"strings"
)

type scanResult struct {
	Protocol string
	Host     string
	Username string
	Path     string
	Password string
	Owner    string
	Err      error
}

func (r *runner) readStdinForGitCredentialHelper(ctx context.Context) (*scanResult, error) { //nolint:cyclop
	inputCh := make(chan *scanResult, 1)

	go func() {
		scanner := bufio.NewScanner(r.stdin)
		result := &scanResult{}
		for scanner.Scan() {
			line := scanner.Text()
			if line == "" {
				break // empty line means the end of input
			}
			key, value, ok := strings.Cut(line, "=")
			if !ok {
				continue // ignore invalid stdin
			}
			if key != "password" {
				r.logger.Debug("read a parameter from stdin for Git Credential Helper", key, value)
			}
			switch key {
			case "protocol":
				result.Protocol = value
			case "host":
				result.Host = value
			case "username":
				result.Username = value
			case "path":
				// path is used to switch GitHub Apps by repository
				// But path may not be passed.
				// To guarantee the path is passed, you can configure Git like below:
				//
				//   $ git config credential.useHttpPath true
				a, _, ok := strings.Cut(value, "/")
				if !ok {
					r.logger.Warn("the path from stdin for Git Credential Helper is unexpected", "path", value)
					continue
				}
				result.Path = value
				result.Owner = a
			case "password":
				result.Password = value
			default:
				continue // ignore unknown keys
			}
		}
		result.Err = scanner.Err()
		inputCh <- result
		close(inputCh)
	}()

	select {
	case <-ctx.Done():
		return nil, ctx.Err() //nolint:wrapcheck
	case result := <-inputCh:
		return result, result.Err
	}
}
