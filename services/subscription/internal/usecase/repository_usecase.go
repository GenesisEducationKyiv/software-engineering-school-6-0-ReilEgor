package usecase

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/GenesisEducationKyiv/software-engineering-school-6-0-ReilEgor/shared/ctxlog"

	"github.com/GenesisEducationKyiv/software-engineering-school-6-0-ReilEgor/services/subscription/internal/domain/repository"
)

type RepositoryUseCase struct {
	repoUpdater repository.RepositoryUpdater
}

func NewRepositoryUseCase(repoUpdater repository.RepositoryUpdater) *RepositoryUseCase {
	return &RepositoryUseCase{repoUpdater: repoUpdater}
}

func (uc *RepositoryUseCase) UpdateTag(ctx context.Context, fullName, tag string) error {
	const op = "RepositoryUseCase.UpdateTag"
	log := ctxlog.FromCtx(ctx).With(slog.String("op", op), slog.String("repo", fullName), slog.String("tag", tag))
	log.DebugContext(ctx, "called")

	if err := uc.repoUpdater.UpdateTag(ctx, fullName, tag); err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}
	return nil
}
