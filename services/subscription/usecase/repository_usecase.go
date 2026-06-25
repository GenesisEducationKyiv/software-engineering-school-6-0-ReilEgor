package usecase

import (
	"context"
	"fmt"

	"github.com/GenesisEducationKyiv/software-engineering-school-6-0-ReilEgor/services/subscription/domain/repository"
)

type RepositoryUseCase struct {
	repoUpdater repository.RepositoryUpdater
}

func NewRepositoryUseCase(repoUpdater repository.RepositoryUpdater) *RepositoryUseCase {
	return &RepositoryUseCase{repoUpdater: repoUpdater}
}

func (uc *RepositoryUseCase) UpdateTag(ctx context.Context, fullName, tag string) error {
	if err := uc.repoUpdater.UpdateTag(ctx, fullName, tag); err != nil {
		return fmt.Errorf("RepositoryUseCase.UpdateTag: %w", err)
	}
	return nil
}
