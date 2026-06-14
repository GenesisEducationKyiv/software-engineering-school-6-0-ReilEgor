package usecase

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/GenesisEducationKyiv/software-engineering-school-6-0-ReilEgor/internal/tracking/domain/port"
	domainUsecase "github.com/GenesisEducationKyiv/software-engineering-school-6-0-ReilEgor/internal/tracking/domain/usecase"
	"github.com/GenesisEducationKyiv/software-engineering-school-6-0-ReilEgor/shared/domain/model"
)

type ReleaseProcessor struct {
	repoReader port.RepositoryReader
	repoUC     domainUsecase.RepositoryUseCase
	subReader  port.SubscriberReader
	publisher  port.NotificationPublisher
	logger     *slog.Logger
}

func NewReleaseProcessor(
	repoReader port.RepositoryReader,
	repoUC domainUsecase.RepositoryUseCase,
	subReader port.SubscriberReader,
	publisher port.NotificationPublisher,
) *ReleaseProcessor {
	return &ReleaseProcessor{
		repoReader: repoReader,
		repoUC:     repoUC,
		subReader:  subReader,
		publisher:  publisher,
		logger:     slog.With(slog.String("component", "ReleaseProcessor")),
	}
}

func (rp *ReleaseProcessor) ProcessReleases(ctx context.Context) error {
	repos, err := rp.repoReader.GetAll(ctx)
	if err != nil {
		return fmt.Errorf("ReleaseProcessor.ProcessReleases: get all repos: %w", err)
	}

	for _, repo := range repos {
		updatedRepo, err := rp.repoUC.CheckForUpdates(ctx, repo)
		if err != nil {
			rp.logger.ErrorContext(ctx, "check for updates failed",
				slog.String("repo", repo.FullName),
				slog.Any("error", err),
			)
			continue
		}

		if updatedRepo == nil {
			continue
		}

		subs, err := rp.subReader.GetByRepoID(ctx, updatedRepo.ID)
		if err != nil {
			rp.logger.ErrorContext(ctx, "get subscribers failed",
				slog.String("repo", repo.FullName),
				slog.Any("error", err),
			)
			continue
		}

		for _, sub := range subs {
			cmd := model.SendNotificationCommand{
				Email:    sub.Email,
				RepoName: updatedRepo.FullName,
				Tag:      updatedRepo.LastSeenTag,
				Token:    sub.Token,
			}
			if err := rp.publisher.Publish(ctx, cmd); err != nil {
				rp.logger.WarnContext(ctx, "publish notification failed",
					slog.String("email", sub.Email),
					slog.Any("error", err),
				)
			}
		}
	}

	return nil
}
