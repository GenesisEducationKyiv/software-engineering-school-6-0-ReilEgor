package repository

import (
	"context"

	"github.com/GenesisEducationKyiv/software-engineering-school-6-0-ReilEgor/internal/domain/model"
)

//go:generate mockery --name SubscriptionRepository --output ../../mocks --case underscore --outpkg mocks
type SubscriptionRepository interface {
	Delete(ctx context.Context, userID int64, repo string) error

	CreatePending(ctx context.Context, userID, repoID int64, token string) (int64, error)
	Confirm(ctx context.Context, token string) error
	UnsubscribeByToken(ctx context.Context, token string) error
	GetSubscribersByRepoID(ctx context.Context, id int64) ([]model.Subscriber, error)
	GetByEmail(ctx context.Context, email string) ([]model.Subscription, error)
}
