package usecase

import (
	"context"

	"github.com/GenesisEducationKyiv/software-engineering-school-6-0-ReilEgor/internal/domain/model"
)

var (
// ErrInvalidEmail      = errors.New("invalid email format").
// ErrUserNotFound      = errors.New("user not found").
// ErrUserAlreadyExists = errors.New("user with this email already exists").
// ErrInternalServer    = errors.New("internal server error").
)

//go:generate mockery --name UserUseCase --output ../../mocks --case underscore --outpkg mocks
type UserUseCase interface {
	GetByEmail(ctx context.Context, email string) (model.User, error)
	Create(ctx context.Context, email string) (int, error)
}
