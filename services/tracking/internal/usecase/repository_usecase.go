package usecase

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/GenesisEducationKyiv/software-engineering-school-6-0-ReilEgor/shared/ctxlog"
	"github.com/GenesisEducationKyiv/software-engineering-school-6-0-ReilEgor/shared/infrastructure/metrics"

	"github.com/GenesisEducationKyiv/software-engineering-school-6-0-ReilEgor/services/tracking/internal/domain/model"
	"github.com/GenesisEducationKyiv/software-engineering-school-6-0-ReilEgor/services/tracking/internal/domain/repository"
	"github.com/GenesisEducationKyiv/software-engineering-school-6-0-ReilEgor/services/tracking/internal/domain/service"
)

const (
	componentRepositoryUseCase = "RepositoryUseCase"
	checkForUpdatesCtxTimeout  = 10
)

type RepositoryUseCase struct {
	repoRepo repository.RepositoryRepository
	ghClient service.GitHubClient
}

func NewRepositoryUseCase(
	repoRepo repository.RepositoryRepository,
	ghClient service.GitHubClient,
) *RepositoryUseCase {
	return &RepositoryUseCase{
		repoRepo: repoRepo,
		ghClient: ghClient,
	}
}

func (uc *RepositoryUseCase) log(ctx context.Context) *slog.Logger {
	return ctxlog.FromCtx(ctx).With(slog.String("component", componentRepositoryUseCase))
}

func (uc *RepositoryUseCase) GetOrCreate(ctx context.Context, repoName string) (_ *model.Repository, err error) {
	const op = "RepositoryUseCase.GetOrCreate"
	start := time.Now()
	log := uc.log(ctx).With(slog.String("op", op), slog.String("repo", repoName))
	log.DebugContext(ctx, "called")

	defer func() {
		elapsed := time.Since(start)
		status := "success"
		if err != nil {
			status = "error"
		}
		metrics.UsecaseOperationsTotal.WithLabelValues(op, status).Inc()
		metrics.UsecaseOperationDurationSeconds.WithLabelValues(op).Observe(elapsed.Seconds())
		log.InfoContext(ctx, "done", slog.String("status", status), slog.Duration("duration", elapsed))
	}()

	log.DebugContext(ctx, "checking repository in local database")
	repo, err := uc.repoRepo.GetByName(ctx, repoName)
	if err == nil {
		log.DebugContext(ctx, "repository found in database")
		return repo, nil
	}

	if !errors.Is(err, model.ErrRepositoryNotFound) {
		log.ErrorContext(ctx, "failed to query repository from database", slog.String("error", err.Error()))
		return nil, fmt.Errorf("%s: find in db: %w", op, err)
	}

	log.InfoContext(ctx, "repository not found in DB, checking external provider (GitHub)")

	exists, err := uc.ghClient.RepoExists(ctx, repoName)
	if err != nil {
		log.ErrorContext(ctx, "failed to check repository existence on GitHub", slog.String("error", err.Error()))
		return nil, fmt.Errorf("%s: check repo: %w", op, err)
	}
	if !exists {
		log.WarnContext(ctx, "repository does not exist on GitHub")
		return nil, model.ErrRepositoryNotFound
	}

	release, err := uc.ghClient.GetLatestRelease(ctx, repoName)
	if err != nil {
		log.ErrorContext(ctx, "failed to fetch latest release from GitHub", slog.String("error", err.Error()))
		return nil, fmt.Errorf("%s: fetch release: %w", op, err)
	}

	repo = &model.Repository{
		FullName:    repoName,
		LastSeenTag: release.TagName,
	}

	log.InfoContext(ctx, "creating new repository record", slog.String("tag", release.TagName))
	if err := uc.repoRepo.Create(ctx, repo); err != nil {
		log.ErrorContext(ctx, "failed to save new repository to database", slog.String("error", err.Error()))
		return nil, fmt.Errorf("%s: create: %w", op, err)
	}

	log.InfoContext(ctx, "repository successfully registered in the system")
	return repo, nil
}

func (uc *RepositoryUseCase) CheckForUpdates(
	ctx context.Context,
	repo model.Repository,
) (_ *model.Repository, err error) {
	const op = "RepositoryUseCase.CheckForUpdates"
	start := time.Now()
	log := uc.log(ctx).With(slog.String("op", op), slog.String("repo", repo.FullName))
	log.DebugContext(ctx, "called")

	defer func() {
		elapsed := time.Since(start)
		status := "success"
		if err != nil {
			status = "error"
		}
		metrics.UsecaseOperationsTotal.WithLabelValues(op, status).Inc()
		metrics.UsecaseOperationDurationSeconds.WithLabelValues(op).Observe(elapsed.Seconds())
		log.InfoContext(ctx, "done", slog.String("status", status), slog.Duration("duration", elapsed))
	}()

	repoCtx, cancel := context.WithTimeout(ctx, checkForUpdatesCtxTimeout*time.Second)
	defer cancel()

	latestRelease, err := uc.ghClient.GetLatestRelease(repoCtx, repo.FullName)
	if err != nil {
		log.ErrorContext(ctx, "failed to fetch latest release from GitHub", slog.String("error", err.Error()))
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	if latestRelease == nil || latestRelease.TagName == repo.LastSeenTag {
		log.DebugContext(ctx, "no new release found")
		return nil, nil
	}

	repo.LastSeenTag = latestRelease.TagName
	log.InfoContext(ctx, "new release detected", slog.String("tag", repo.LastSeenTag))
	return &repo, nil
}

func (uc *RepositoryUseCase) UpdateRepo(ctx context.Context, repo *model.Repository) error {
	const op = "RepositoryUseCase.UpdateRepo"
	log := uc.log(ctx).With(slog.String("op", op), slog.String("repo", repo.FullName))
	log.DebugContext(ctx, "called")

	if err := uc.repoRepo.Update(ctx, repo); err != nil {
		log.ErrorContext(ctx, "failed to update repository", slog.String("error", err.Error()))
		return fmt.Errorf("%s: %w", op, err)
	}

	log.InfoContext(ctx, "repository updated")
	return nil
}

func (uc *RepositoryUseCase) Delete(ctx context.Context, repoName string) error {
	const op = "RepositoryUseCase.Delete"
	log := uc.log(ctx).With(slog.String("op", op), slog.String("repo", repoName))
	log.DebugContext(ctx, "called")

	if err := uc.repoRepo.Delete(ctx, repoName); err != nil {
		log.ErrorContext(ctx, "failed to delete repository", slog.String("error", err.Error()))
		return fmt.Errorf("%s: %w", op, err)
	}

	log.InfoContext(ctx, "repository deleted")
	return nil
}
