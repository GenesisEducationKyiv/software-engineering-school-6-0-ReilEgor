package port

import (
	"context"

	"github.com/GenesisEducationKyiv/software-engineering-school-6-0-ReilEgor/internal/shared/domain/model"
)

type RepositoryReader interface {
	GetAll(ctx context.Context) ([]model.Repository, error)
}

type UpdateChecker interface {
	CheckForUpdates(ctx context.Context, repo model.Repository) (*model.Repository, error)
}

type SubscriberReader interface {
	GetByRepoID(ctx context.Context, repoID int64) ([]model.Subscriber, error)
}
