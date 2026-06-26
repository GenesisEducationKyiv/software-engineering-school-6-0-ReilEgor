package usecase

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	contracts "github.com/GenesisEducationKyiv/software-engineering-school-6-0-ReilEgor/shared/contracts"

	model2 "github.com/GenesisEducationKyiv/software-engineering-school-6-0-ReilEgor/services/tracking/internal/domain/model"
	repository2 "github.com/GenesisEducationKyiv/software-engineering-school-6-0-ReilEgor/services/tracking/internal/domain/repository"
	domainUsecase "github.com/GenesisEducationKyiv/software-engineering-school-6-0-ReilEgor/services/tracking/internal/domain/usecase"
)

//go:generate mockery --name RepositoryReader --output ../mocks --case underscore --outpkg mocks
type RepositoryReader interface {
	GetAll(ctx context.Context) ([]model2.Repository, error)
}

//go:generate mockery --name SubscriberReader --output ../mocks --case underscore --outpkg mocks
type SubscriberReader interface {
	GetByRepoID(ctx context.Context, repoID int64) ([]model2.Subscriber, error)
}

//go:generate mockery --name NotificationPublisher --output ../mocks --case underscore --outpkg mocks
type NotificationPublisher interface {
	Publish(ctx context.Context, cmd contracts.SendNotificationCommand) error
}

//go:generate mockery --name TagUpdatedPublisher --output ../mocks --case underscore --outpkg mocks
type TagUpdatedPublisher interface {
	Publish(ctx context.Context, event contracts.TagUpdatedEvent) error
}

type ReleaseProcessor struct {
	repoReader    RepositoryReader
	repoUC        domainUsecase.RepositoryUseCase
	subReader     SubscriberReader
	publisher     NotificationPublisher
	tagUpdatedPub TagUpdatedPublisher
	outboxRepo    repository2.OutboxRepository
	transactor    repository2.Transactor
	logger        *slog.Logger
}

func NewReleaseProcessor(
	repoReader RepositoryReader,
	repoUC domainUsecase.RepositoryUseCase,
	subReader SubscriberReader,
	publisher NotificationPublisher,
	tagUpdatedPub TagUpdatedPublisher,
	outboxRepo repository2.OutboxRepository,
	transactor repository2.Transactor,
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

func (rp *ReleaseProcessor) processRepo(ctx context.Context, repo model2.Repository) error {
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

	if pubErr := rp.tagUpdatedPub.Publish(ctx, contracts.TagUpdatedEvent{
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
	repo *model2.Repository,
	subs []model2.Subscriber,
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
	sub model2.Subscriber,
	repo *model2.Repository,
) error {
	payload, err := json.Marshal(contracts.SendNotificationCommand{
		Email:    sub.Email,
		RepoName: repo.FullName,
		Tag:      repo.LastSeenTag,
		Token:    sub.Token,
	})
	if err != nil {
		return fmt.Errorf("marshal notification: %w", err)
	}
	if err = rp.outboxRepo.Insert(ctx, contracts.QueueNotifications, payload); err != nil {
		return fmt.Errorf("insert notification outbox: %w", err)
	}
	return nil
}
