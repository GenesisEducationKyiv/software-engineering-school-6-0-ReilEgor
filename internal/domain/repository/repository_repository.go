package repository

import (
	"context"

	"github.com/GenesisEducationKyiv/software-engineering-school-6-0-ReilEgor/internal/domain/model"
)

//go:generate mockery --name RepositoryRepository --output ../../mocks --case underscore --outpkg mocks
type RepositoryRepository interface {
	GetAll(ctx context.Context) ([]model.Repository, error)
	UpdateLastSeenTag(ctx context.Context, name, tag string) error
	GetOrCreate(ctx context.Context, name, tagName string) (*model.Repository, error)
}
