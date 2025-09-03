package github

import (
	"context"
)

type Mock struct {
	user *User
	err  error
}

func (m *Mock) Get(_ context.Context) (*User, error) {
	return m.user, m.err
}

func NewMock(user *User, err error) func(ctx context.Context, _ string) *Mock {
	return func(_ context.Context, _ string) *Mock {
		return &Mock{
			user: user,
			err:  err,
		}
	}
}
