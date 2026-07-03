package usecase

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	"github.com/GenesisEducationKyiv/software-engineering-school-6-0-ReilEgor/shared/ctxlog"

	contracts "github.com/GenesisEducationKyiv/software-engineering-school-6-0-ReilEgor/shared/contracts"

	"github.com/GenesisEducationKyiv/software-engineering-school-6-0-ReilEgor/services/tracking/internal/domain/model"
	"github.com/GenesisEducationKyiv/software-engineering-school-6-0-ReilEgor/services/tracking/internal/domain/repository"
	domainUsecase "github.com/GenesisEducationKyiv/software-engineering-school-6-0-ReilEgor/services/tracking/internal/domain/usecase"
)

const componentReleaseProcessor = "ReleaseProcessor"

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
	outboxRepo    repository.OutboxRepository
	transactor    repository.Transactor
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
	}
}

func (rp *ReleaseProcessor) log(ctx context.Context) *slog.Logger {
	return ctxlog.FromCtx(ctx).With(slog.String("component", componentReleaseProcessor))
}

func (rp *ReleaseProcessor) ProcessReleases(ctx context.Context) error {
	const op = "ReleaseProcessor.ProcessReleases"
	log := rp.log(ctx).With(slog.String("op", op))
	log.DebugContext(ctx, "called")

	repos, err := rp.repoReader.GetAll(ctx)
	if err != nil {
		log.ErrorContext(ctx, "failed to get all repos", slog.String("error", err.Error()))
		return fmt.Errorf("%s: get all repos: %w", op, err)
	}

	for _, repo := range repos {
		if err := rp.processRepo(ctx, repo); err != nil {
			log.ErrorContext(ctx, "process release failed",
				slog.String("repo", repo.FullName),
				slog.String("error", err.Error()),
			)
		}
	}

	log.InfoContext(ctx, "done", slog.Int("repos_checked", len(repos)))
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

	if pubErr := rp.tagUpdatedPub.Publish(ctx, contracts.TagUpdatedEvent{
		FullName: updatedRepo.FullName,
		Tag:      updatedRepo.LastSeenTag,
	}); pubErr != nil {
		rp.log(ctx).ErrorContext(ctx, "tag updated publish failed",
			slog.String("repo", updatedRepo.FullName),
			slog.String("error", pubErr.Error()),
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
