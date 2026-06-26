package repository

import (
	"context"

	"github.com/GenesisEducationKyiv/software-engineering-school-6-0-ReilEgor/services/subscription/internal/domain/model"
)

//go:generate mockery --name RepositoryUpdater --output ../../mocks --case underscore --outpkg mocks
type RepositoryUpdater interface {
	UpdateTag(ctx context.Context, fullName, tag string) error
}

//go:generate mockery --name RepositoryRepository --output ../../mocks --case underscore --outpkg mocks
type RepositoryRepository interface {
	GetOrCreate(ctx context.Context, fullName string) (*model.RepositoryRef, error)
}
