package port

import (
	"context"

	model2 "github.com/GenesisEducationKyiv/software-engineering-school-6-0-ReilEgor/shared/domain/model"
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
	Publish(ctx context.Context, cmd model2.SendNotificationCommand) error
}
