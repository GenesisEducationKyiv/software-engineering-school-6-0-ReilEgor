package usecase

import (
	"context"

	"github.com/GenesisEducationKyiv/software-engineering-school-6-0-ReilEgor/internal/domain/model"
)

//go:generate mockery --name UserUseCase --output ../../mocks --case underscore --outpkg mocks
type UserUseCase interface {
	GetByEmail(ctx context.Context, email string) (model.User, error)
	Create(ctx context.Context, email string) (int, error)
}
