package repository

import (
	"context"

	"github.com/ReilEgor/RepoNotifier/internal/domain/model"
)

//go:generate mockery --name UserRepository --output ../../mocks --case underscore --outpkg mocks
type UserRepository interface {
	GetByEmail(ctx context.Context, email string) (model.User, error)
	GetOrCreate(ctx context.Context, email string) (model.User, error)
}
