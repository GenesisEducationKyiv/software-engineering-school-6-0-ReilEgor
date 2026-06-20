package usecase

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	trackingModel "github.com/GenesisEducationKyiv/software-engineering-school-6-0-ReilEgor/internal/tracking/domain/model"
	"github.com/GenesisEducationKyiv/software-engineering-school-6-0-ReilEgor/internal/tracking/domain/port"
	trackingRepo "github.com/GenesisEducationKyiv/software-engineering-school-6-0-ReilEgor/internal/tracking/domain/repository"
	domainUsecase "github.com/GenesisEducationKyiv/software-engineering-school-6-0-ReilEgor/internal/tracking/domain/usecase"
	sharedRabbitmq "github.com/GenesisEducationKyiv/software-engineering-school-6-0-ReilEgor/shared/broker/rabbitmq"
	sharedModel "github.com/GenesisEducationKyiv/software-engineering-school-6-0-ReilEgor/shared/domain/model"
)

type ReleaseProcessor struct {
	repoReader    port.RepositoryReader
	repoUC        domainUsecase.RepositoryUseCase
	subReader     port.SubscriberReader
	publisher     port.NotificationPublisher
	tagUpdatedPub port.TagUpdatedPublisher
	outboxRepo    trackingRepo.OutboxRepository
	transactor    trackingRepo.Transactor
	logger        *slog.Logger
}

func NewReleaseProcessor(
	repoReader port.RepositoryReader,
	repoUC domainUsecase.RepositoryUseCase,
	subReader port.SubscriberReader,
	publisher port.NotificationPublisher,
	tagUpdatedPub port.TagUpdatedPublisher,
	outboxRepo trackingRepo.OutboxRepository,
	transactor trackingRepo.Transactor,
) *ReleaseProcessor {
	return &ReleaseProcessor{
		repoReader:    repoReader,
		repoUC:        repoUC,
		subReader:     subReader,
		publisher:     publisher,
		tagUpdatedPub: tagUpdatedPub,
		outboxRepo:    outboxRepo,
		transactor:    transactor,
		logger:        slog.With(slog.String("component", "ReleaseProcessor")),
	}
}

func (rp *ReleaseProcessor) ProcessReleases(ctx context.Context) error {
	repos, err := rp.repoReader.GetAll(ctx)
	if err != nil {
		return fmt.Errorf("ReleaseProcessor.ProcessReleases: get all repos: %w", err)
	}

	for _, repo := range repos {
		if err := rp.processRepo(ctx, repo); err != nil {
			rp.logger.ErrorContext(ctx, "process release failed",
				slog.String("repo", repo.FullName),
				slog.Any("error", err),
			)
		}
	}
	return nil
}

func (rp *ReleaseProcessor) processRepo(ctx context.Context, repo trackingModel.Repository) error {
	updatedRepo, err := rp.repoUC.CheckForUpdates(ctx, repo)
	if err != nil {
		return fmt.Errorf("check for updates: %w", err)
	}
	if updatedRepo == nil {
		return nil
	}

	subs, err := rp.subReader.GetByRepoID(ctx, updatedRepo.ID)
	if err != nil {
		return fmt.Errorf("get subscribers: %w", err)
	}

	if err = rp.transactor.WithinTransaction(ctx, func(txCtx context.Context) error {
		return rp.commitRelease(txCtx, updatedRepo, subs)
	}); err != nil {
		return fmt.Errorf("commit release: %w", err)
	}
	return nil
}

func (rp *ReleaseProcessor) commitRelease(
	ctx context.Context,
	repo *trackingModel.Repository,
	subs []trackingModel.Subscriber,
) error {
	if err := rp.repoUC.UpdateRepo(ctx, repo); err != nil {
		return fmt.Errorf("update repo tag: %w", err)
	}

	payload, err := json.Marshal(sharedModel.TagUpdatedEvent{
		FullName: repo.FullName,
		Tag:      repo.LastSeenTag,
	})
	if err != nil {
		return fmt.Errorf("marshal tag updated: %w", err)
	}
	if err = rp.outboxRepo.Insert(ctx, sharedRabbitmq.QueueTagUpdated, payload); err != nil {
		return fmt.Errorf("insert tag updated outbox: %w", err)
	}

	for _, sub := range subs {
		if err = rp.insertNotificationOutbox(ctx, sub, repo); err != nil {
			return err
		}
	}
	return nil
}

func (rp *ReleaseProcessor) insertNotificationOutbox(
	ctx context.Context,
	sub trackingModel.Subscriber,
	repo *trackingModel.Repository,
) error {
	payload, err := json.Marshal(sharedModel.SendNotificationCommand{
		Email:    sub.Email,
		RepoName: repo.FullName,
		Tag:      repo.LastSeenTag,
		Token:    sub.Token,
	})
	if err != nil {
		return fmt.Errorf("marshal notification: %w", err)
	}
	if err = rp.outboxRepo.Insert(ctx, sharedRabbitmq.QueueNotifications, payload); err != nil {
		return fmt.Errorf("insert notification outbox: %w", err)
	}
	return nil
}
