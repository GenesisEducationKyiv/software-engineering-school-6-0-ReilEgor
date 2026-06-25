package usecase

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	sharedModel "github.com/GenesisEducationKyiv/software-engineering-school-6-0-ReilEgor/shared/domain/model"
	sharedRabbitmq "github.com/GenesisEducationKyiv/software-engineering-school-6-0-ReilEgor/shared/infrastructure/broker/rabbitmq"

	"github.com/GenesisEducationKyiv/software-engineering-school-6-0-ReilEgor/services/tracking/domain/model"
	"github.com/GenesisEducationKyiv/software-engineering-school-6-0-ReilEgor/services/tracking/domain/repository"
	domainUsecase "github.com/GenesisEducationKyiv/software-engineering-school-6-0-ReilEgor/services/tracking/domain/usecase"
)

//go:generate mockery --name RepositoryReader --output ../mocks --case underscore --outpkg mocks
type RepositoryReader interface {
	GetAll(ctx context.Context) ([]model.Repository, error)
}

//go:generate mockery --name SubscriberReader --output ../mocks --case underscore --outpkg mocks
type SubscriberReader interface {
	GetByRepoID(ctx context.Context, repoID int64) ([]model.Subscriber, error)
}

//go:generate mockery --name NotificationPublisher --output ../mocks --case underscore --outpkg mocks
type NotificationPublisher interface {
	Publish(ctx context.Context, cmd sharedModel.SendNotificationCommand) error
}

//go:generate mockery --name TagUpdatedPublisher --output ../mocks --case underscore --outpkg mocks
type TagUpdatedPublisher interface {
	Publish(ctx context.Context, event sharedModel.TagUpdatedEvent) error
}

type ReleaseProcessor struct {
	repoReader    RepositoryReader
	repoUC        domainUsecase.RepositoryUseCase
	subReader     SubscriberReader
	publisher     NotificationPublisher
	tagUpdatedPub TagUpdatedPublisher
	outboxRepo    repository.OutboxRepository
	transactor    repository.Transactor
	logger        *slog.Logger
}

func NewReleaseProcessor(
	repoReader RepositoryReader,
	repoUC domainUsecase.RepositoryUseCase,
	subReader SubscriberReader,
	publisher NotificationPublisher,
	tagUpdatedPub TagUpdatedPublisher,
	outboxRepo repository.OutboxRepository,
	transactor repository.Transactor,
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

func (rp *ReleaseProcessor) processRepo(ctx context.Context, repo model.Repository) error {
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

	if pubErr := rp.tagUpdatedPub.Publish(ctx, sharedModel.TagUpdatedEvent{
		FullName: updatedRepo.FullName,
		Tag:      updatedRepo.LastSeenTag,
	}); pubErr != nil {
		rp.logger.ErrorContext(ctx, "tag updated publish failed",
			slog.String("repo", updatedRepo.FullName),
			slog.Any("error", pubErr),
		)
	}

	return nil
}

func (rp *ReleaseProcessor) commitRelease(
	ctx context.Context,
	repo *model.Repository,
	subs []model.Subscriber,
) error {
	if err := rp.repoUC.UpdateRepo(ctx, repo); err != nil {
		return fmt.Errorf("update repo tag: %w", err)
	}

	for _, sub := range subs {
		if err := rp.insertNotificationOutbox(ctx, sub, repo); err != nil {
			return err
		}
	}
	return nil
}

func (rp *ReleaseProcessor) insertNotificationOutbox(
	ctx context.Context,
	sub model.Subscriber,
	repo *model.Repository,
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
