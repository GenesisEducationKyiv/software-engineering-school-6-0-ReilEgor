package usecase

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/GenesisEducationKyiv/software-engineering-school-6-0-ReilEgor/internal/domain/model"
	"github.com/GenesisEducationKyiv/software-engineering-school-6-0-ReilEgor/internal/domain/repository"
	"github.com/GenesisEducationKyiv/software-engineering-school-6-0-ReilEgor/internal/domain/service"
)

const componentRepositoryUseCase = "RepositoryUseCase"

type RepositoryUseCase struct {
	logger   *slog.Logger
	repoRepo repository.RepositoryRepository
	ghClient service.GitHubClient
}

func NewRepositoryUseCase(
	repoRepo repository.RepositoryRepository,
	ghClient service.GitHubClient,
) *RepositoryUseCase {
	return &RepositoryUseCase{
		logger:   slog.With(slog.String("useCase", componentRepositoryUseCase)),
		repoRepo: repoRepo,
		ghClient: ghClient,
	}
}

var errMsgUpdateTag = errors.New("update last seen tag in database")

func (uc *RepositoryUseCase) GetOrCreate(ctx context.Context, repoName string) (*model.Repository, error) {
	const op = "RepositoryUseCase.GetOrCreate"
	log := uc.logger.With(
		slog.String("op", op),
		slog.String("repo", repoName),
	)

	log.DebugContext(ctx, "checking repository in local database")
	repo, err := uc.repoRepo.GetByName(ctx, repoName)
	if err == nil {
		log.DebugContext(ctx, "repository found in database")
		return repo, nil
	}

	if !errors.Is(err, model.ErrRepositoryNotFound) {
		log.ErrorContext(ctx, "failed to query repository from database", slog.Any("error", err))
		return nil, fmt.Errorf("%s: find in db: %w", op, err)
	}

	log.InfoContext(ctx, "repository not found in DB, checking external provider (GitHub)")

	exists, err := uc.ghClient.RepoExists(ctx, repoName)
	if err != nil {
		log.ErrorContext(ctx, "failed to check repository existence on GitHub", slog.Any("error", err))
		return nil, fmt.Errorf("%s: check repo: %w", op, err)
	}
	if !exists {
		log.WarnContext(ctx, "repository does not exist on GitHub")
		return nil, service.ErrRepositoryNotFound
	}

	release, err := uc.ghClient.GetLatestRelease(ctx, repoName)
	if err != nil {
		log.ErrorContext(ctx, "failed to fetch latest release from GitHub", slog.Any("error", err))
		return nil, fmt.Errorf("%s: fetch release: %w", op, err)
	}

	repo = &model.Repository{
		FullName:    repoName,
		LastSeenTag: release.TagName,
	}

	log.InfoContext(ctx, "creating new repository record", slog.String("tag", release.TagName))
	if err := uc.repoRepo.Create(ctx, repo); err != nil {
		log.ErrorContext(ctx, "failed to save new repository to database", slog.Any("error", err))
		return nil, fmt.Errorf("%s: create: %w", op, err)
	}

	log.InfoContext(ctx, "repository successfully registered in the system")
	return repo, nil
}

func (uc *RepositoryUseCase) CheckForUpdates(ctx context.Context, repo model.Repository) (*model.Repository, error) {
	const op = "RepositoryUseCase.CheckForUpdates"
	log := uc.logger.With(slog.String("repo", repo.FullName))

	repoCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	latestRelease, err := uc.ghClient.GetLatestRelease(repoCtx, repo.FullName)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	if latestRelease == nil || latestRelease.TagName == repo.LastSeenTag {
		return nil, nil
	}

	repo.LastSeenTag = latestRelease.TagName
	if err := uc.repoRepo.Update(ctx, &repo); err != nil {
		log.ErrorContext(ctx, errMsgUpdateTag.Error(), slog.String("error", err.Error()))
		return nil, fmt.Errorf("%s: update tag in db: %w", op, err)
	}

	log.InfoContext(ctx, "new release detected", slog.String("tag", repo.LastSeenTag))
	return &repo, nil
}
