package port

import (
	"context"

	sharedModel "github.com/GenesisEducationKyiv/software-engineering-school-6-0-ReilEgor/shared/domain/model"

	model2 "github.com/GenesisEducationKyiv/software-engineering-school-6-0-ReilEgor/services/tracking/domain/model"
)

//go:generate mockery --name RepositoryReader --output ../../mocks --case underscore --outpkg mocks
type RepositoryReader interface {
	GetAll(ctx context.Context) ([]model2.Repository, error)
}

//go:generate mockery --name SubscriberReader --output ../../mocks --case underscore --outpkg mocks
type SubscriberReader interface {
	GetByRepoID(ctx context.Context, repoID int64) ([]model2.Subscriber, error)
}

//go:generate mockery --name NotificationPublisher --output ../../mocks --case underscore --outpkg mocks
type NotificationPublisher interface {
	Publish(ctx context.Context, cmd sharedModel.SendNotificationCommand) error
}

//go:generate mockery --name TagUpdatedPublisher --output ../../mocks --case underscore --outpkg mocks
type TagUpdatedPublisher interface {
	Publish(ctx context.Context, event sharedModel.TagUpdatedEvent) error
}

//go:generate mockery --name TrackerSubscriptionWriter --output ../../mocks --case underscore --outpkg mocks
type TrackerSubscriptionWriter interface {
	Upsert(ctx context.Context, repoID int64, email, token string) error
}

//go:generate mockery --name TrackerSubscriptionDeleter --output ../../mocks --case underscore --outpkg mocks
type TrackerSubscriptionDeleter interface {
	DeleteByEmailAndRepo(ctx context.Context, email, repoName string) error
	DeleteByUserIDAndRepo(ctx context.Context, userID int64, repoName string) error
	HasSubscriptions(ctx context.Context, repoName string) (bool, error)
}
