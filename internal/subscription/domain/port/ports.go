package port

import (
	"context"

	trackingModel "github.com/GenesisEducationKyiv/software-engineering-school-6-0-ReilEgor/internal/tracking/domain/model"
)

//go:generate mockery --name RepositoryUseCase --output ../../../shared/mocks --case underscore --outpkg mocks
type RepositoryUseCase interface {
	GetOrCreate(ctx context.Context, repoName string) (*trackingModel.Repository, error)
}

//go:generate mockery --name ConfirmationSender --output ../../../shared/mocks --case underscore --outpkg mocks
type ConfirmationSender interface {
	SendConfirmation(ctx context.Context, to, repoName, token string) error
}
