package usecase

import "context"

//go:generate mockery --name RepositoryUseCase --output ../../mocks --case underscore --outpkg mocks
type RepositoryUseCase interface {
	UpdateTag(ctx context.Context, fullName, tag string) error
}
