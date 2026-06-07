package port

import (
	"context"

	"github.com/GenesisEducationKyiv/software-engineering-school-6-0-ReilEgor/internal/shared/domain/model"
)

//go:generate mockery --name RepositoryReader --output ../../../shared/mocks --case underscore --outpkg mocks
type RepositoryReader interface {
	GetAll(ctx context.Context) ([]model.Repository, error)
}

//go:generate mockery --name UpdateChecker --output ../../../shared/mocks --case underscore --outpkg mocks
type UpdateChecker interface {
	CheckForUpdates(ctx context.Context, repo model.Repository) (*model.Repository, error)
}

//go:generate mockery --name SubscriberReader --output ../../../shared/mocks --case underscore --outpkg mocks
type SubscriberReader interface {
	GetByRepoID(ctx context.Context, repoID int64) ([]model.Subscriber, error)
}
