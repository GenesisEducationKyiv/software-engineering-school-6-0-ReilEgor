package repository

import (
	"context"

	"github.com/GenesisEducationKyiv/software-engineering-school-6-0-ReilEgor/internal/domain/model"
)

//go:generate mockery --name RepositoryReader --output ../../mocks --case underscore --outpkg mocks
type RepositoryReader interface {
	GetAll(ctx context.Context) ([]model.Repository, error)
	GetByName(ctx context.Context, name string) (*model.Repository, error)
}

//go:generate mockery --name RepositoryWriter --output ../../mocks --case underscore --outpkg mocks
type RepositoryWriter interface {
	Create(ctx context.Context, repo *model.Repository) error
	Update(ctx context.Context, repo *model.Repository) error
}

//go:generate mockery --name RepositoryRepository --output ../../mocks --case underscore --outpkg mocks
type RepositoryRepository interface {
	RepositoryWriter
	RepositoryReader
}
