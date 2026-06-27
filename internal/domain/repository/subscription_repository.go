package repository

import (
	"context"

	"github.com/GenesisEducationKyiv/software-engineering-school-6-0-ReilEgor/internal/domain/model"
)

//go:generate mockery --name SubscriptionReader --output ../../mocks --case underscore --outpkg mocks
type SubscriptionReader interface {
	GetByToken(ctx context.Context, token string) (*model.Subscription, error)
	GetByRepoID(ctx context.Context, repoID int64) ([]model.Subscriber, error)
	GetByEmail(ctx context.Context, email string) ([]model.Subscription, error)
}

//go:generate mockery --name SubscriptionWriter --output ../../mocks --case underscore --outpkg mocks
type SubscriptionWriter interface {
	Save(ctx context.Context, sub *model.Subscription) error
	Delete(ctx context.Context, userID int64, repoName string) error
}

//go:generate mockery --name SubscriptionRepository --output ../../mocks --case underscore --outpkg mocks
type SubscriptionRepository interface {
	SubscriptionReader
	SubscriptionWriter
}
