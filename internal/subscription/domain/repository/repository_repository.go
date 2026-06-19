package repository

import "context"

//go:generate mockery --name RepositoryUpdater --output ../../mocks --case underscore --outpkg mocks
type RepositoryUpdater interface {
	UpdateTag(ctx context.Context, fullName, tag string) error
}
