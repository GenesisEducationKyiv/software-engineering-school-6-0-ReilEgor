package port

import (
	"context"

	"github.com/GenesisEducationKyiv/software-engineering-school-6-0-ReilEgor/internal/tracking/domain/model"
	sharedModel "github.com/GenesisEducationKyiv/software-engineering-school-6-0-ReilEgor/shared/domain/model"
)

//go:generate mockery --name RepositoryReader --output ../../mocks --case underscore --outpkg mocks
type RepositoryReader interface {
	GetAll(ctx context.Context) ([]model.Repository, error)
}

//go:generate mockery --name SubscriberReader --output ../../mocks --case underscore --outpkg mocks
type SubscriberReader interface {
	GetByRepoID(ctx context.Context, repoID int64) ([]model.Subscriber, error)
}

//go:generate mockery --name NotificationPublisher --output ../../mocks --case underscore --outpkg mocks
type NotificationPublisher interface {
	Publish(ctx context.Context, cmd sharedModel.SendNotificationCommand) error
}
