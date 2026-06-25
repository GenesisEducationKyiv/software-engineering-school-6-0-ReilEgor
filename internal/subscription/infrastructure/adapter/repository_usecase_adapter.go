package adapter

import (
	"context"
	"errors"
	"fmt"

	subModel "github.com/GenesisEducationKyiv/software-engineering-school-6-0-ReilEgor/internal/subscription/domain/model"
	trackingModel "github.com/GenesisEducationKyiv/software-engineering-school-6-0-ReilEgor/internal/tracking/domain/model"
	trackingUsecase "github.com/GenesisEducationKyiv/software-engineering-school-6-0-ReilEgor/internal/tracking/usecase"
)

type RepositoryUseCaseAdapter struct {
	tracking *trackingUsecase.RepositoryUseCase
}

func NewRepositoryUseCaseAdapter(uc *trackingUsecase.RepositoryUseCase) *RepositoryUseCaseAdapter {
	return &RepositoryUseCaseAdapter{tracking: uc}
}

func (a *RepositoryUseCaseAdapter) GetOrCreate(ctx context.Context, repoName string) (*subModel.RepositoryRef, error) {
	repo, err := a.tracking.GetOrCreate(ctx, repoName)
	if err != nil {
		return nil, translateError(err)
	}
	return &subModel.RepositoryRef{
		ID:       repo.ID,
		FullName: repo.FullName,
	}, nil
}

func translateError(err error) error {
	switch {
	case errors.Is(err, trackingModel.ErrRepositoryNotFound):
		return fmt.Errorf("%w", subModel.ErrRepositoryNotFound)
	case errors.Is(err, trackingModel.ErrGitHubUnavailable):
		return fmt.Errorf("%w", subModel.ErrServiceUnavailable)
	case errors.Is(err, trackingModel.ErrRateLimitExceeded):
		return fmt.Errorf("%w", subModel.ErrServiceUnavailable)
	default:
		return err
	}
}
