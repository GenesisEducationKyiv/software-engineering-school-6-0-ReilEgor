package usecase

import (
	"context"

	"github.com/GenesisEducationKyiv/software-engineering-school-6-0-ReilEgor/internal/domain/model"
)

var (
// ErrRepoNotFound      = errors.New("repository not found on GitHub").
// ErrAlreadySubscribed = errors.New("user already subscribed to this repository").
// ErrInvalidFormat     = errors.New("invalid repository format").
)

//go:generate mockery --name SubscriptionUseCase --output ../../mocks --case underscore --outpkg mocks
type SubscriptionUseCase interface {
	Subscribe(ctx context.Context, email, repoName string) error
	ProcessNotifications(ctx context.Context) error
	ListByEmail(ctx context.Context, email string) ([]model.Subscription, error)

	Confirm(ctx context.Context, token string) error
	UnsubscribeByToken(ctx context.Context, token string) error
}
