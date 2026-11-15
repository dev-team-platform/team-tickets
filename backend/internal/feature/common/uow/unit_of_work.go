package uow

import (
	"context"

	userrepos "github.com/trannhanlv2004/team-tickets/internal/feature/user/repository"
)

type Repositories interface {
	User() userrepos.UserRepository
	// Context carries the transaction scope to nested UnitOfWork.Execute calls.
	Context() context.Context
}

type UnitOfWork interface {
	Execute(ctx context.Context, fn func(repos Repositories) error) error
	Shutdown(ctx context.Context) error
}
