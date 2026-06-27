package repository

import (
	"context"

	"github.com/GenesisEducationKyiv/software-engineering-school-6-0-ReilEgor/internal/domain/model"
)

//go:generate mockery --name UserReader --output ../../mocks --case underscore --outpkg mocks
type UserReader interface {
	GetByEmail(ctx context.Context, email string) (model.User, error)
}

//go:generate mockery --name UserWriter --output ../../mocks --case underscore --outpkg mocks
type UserWriter interface {
	Create(ctx context.Context, user *model.User) error
}

//go:generate mockery --name UserRepository --output ../../mocks --case underscore --outpkg mocks
type UserRepository interface {
	UserReader
	UserWriter
}
