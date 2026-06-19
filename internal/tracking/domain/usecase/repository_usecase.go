package usecase

import (
	"context"

	"github.com/GenesisEducationKyiv/software-engineering-school-6-0-ReilEgor/internal/tracking/domain/model"
)

//go:generate mockery --name RepositoryUseCase --output ../../mocks --case underscore --outpkg mocks
type RepositoryUseCase interface {
	GetOrCreate(ctx context.Context, repoName string) (*model.Repository, error)
	CheckForUpdates(ctx context.Context, repo model.Repository) (*model.Repository, error)
	Delete(ctx context.Context, repoName string) error
}

type ReleaseProcessorUseCase interface {
	ProcessReleases(ctx context.Context) error
}
