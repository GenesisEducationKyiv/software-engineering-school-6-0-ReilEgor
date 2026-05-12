package usecase

import (
	"context"

	"github.com/GenesisEducationKyiv/software-engineering-school-6-0-ReilEgor/internal/domain/model"
)

//go:generate mockery --name UserUseCase --output ../../mocks --case underscore --outpkg mocks
type UserUseCase interface {
	Subscribe(ctx context.Context, email, repoName string) error
	Unsubscribe(ctx context.Context, email, repoName string) error
	UnsubscribeByToken(ctx context.Context, token string) error
	Confirm(ctx context.Context, token string) error
	ListByEmail(ctx context.Context, email string) ([]model.Subscription, error)
}
