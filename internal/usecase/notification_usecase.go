package usecase

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"golang.org/x/sync/errgroup"

	"github.com/GenesisEducationKyiv/software-engineering-school-6-0-ReilEgor/internal/domain/model"
	"github.com/GenesisEducationKyiv/software-engineering-school-6-0-ReilEgor/internal/domain/repository"
	"github.com/GenesisEducationKyiv/software-engineering-school-6-0-ReilEgor/internal/domain/service"
	"github.com/GenesisEducationKyiv/software-engineering-school-6-0-ReilEgor/internal/domain/usecase"
)

const (
	componentNotificationUseCase = "NotificationUseCase"
	ctxTimeout                   = 5
)

const (
	errMsgGetRepos       = "get repos"
	errMsgFetchRelease   = "fetch latest release"
	errMsgGetSubscribers = "get subscribers"
)

const maxSendWorkers = 10

type NotificationUseCase struct {
	logger       *slog.Logger
	subsRepo     repository.SubscriptionRepository
	repoRepo     repository.RepositoryRepository
	repoUC       usecase.RepositoryUseCase
	emailService service.EmailService
}

func NewNotificationUseCase(
	sr repository.SubscriptionRepository,
	rr repository.RepositoryRepository,
	ru usecase.RepositoryUseCase,
	es service.EmailService,
) *NotificationUseCase {
	return &NotificationUseCase{
		logger:       slog.With(slog.String("useCase", componentNotificationUseCase)),
		subsRepo:     sr,
		repoRepo:     rr,
		repoUC:       ru,
		emailService: es,
	}
}

func (uc *NotificationUseCase) ProcessNotifications(ctx context.Context) error {
	repos, err := uc.repoRepo.GetAll(ctx)
	if err != nil {
		return fmt.Errorf("%s: %w", errMsgGetRepos, err)
	}

	g, sendCtx := errgroup.WithContext(ctx)
	g.SetLimit(maxSendWorkers)

	for _, repo := range repos {
		updatedRepo, err := uc.repoUC.CheckForUpdates(ctx, repo)
		if err != nil {
			uc.logger.ErrorContext(ctx, errMsgFetchRelease, "repo", repo.FullName, "err", err)
			continue
		}

		if updatedRepo == nil {
			continue
		}

		subs, err := uc.subsRepo.GetByRepoID(ctx, updatedRepo.ID)
		if err != nil {
			uc.logger.ErrorContext(ctx, errMsgGetSubscribers, "repo", updatedRepo.FullName, "err", err)
			continue
		}

		for _, sub := range subs {
			sub := sub
			g.Go(func() error {
				if err := uc.sendNotificationEmail(
					sendCtx,
					sub,
					updatedRepo.FullName,
					updatedRepo.LastSeenTag,
				); err != nil {
					uc.logger.WarnContext(sendCtx, "skipping failed notification", "email", sub.Email, "err", err)
				}
				return nil
			})
		}
	}

	if err := g.Wait(); err != nil {
		return fmt.Errorf("group task failed: %w", err)
	}

	return nil
}

func (uc *NotificationUseCase) sendNotificationEmail(
	ctx context.Context,
	sub model.Subscriber,
	repoName, tag string,
) error {
	mailCtx, cancel := context.WithTimeout(ctx, ctxTimeout*time.Second)
	defer cancel()

	if err := uc.emailService.SendNotification(mailCtx, sub.Email, repoName, tag, sub.Token); err != nil {
		uc.logger.ErrorContext(mailCtx, "failed to send email",
			slog.String("to", sub.Email),
			slog.Any("error", err),
		)
		return fmt.Errorf("send notification email: %w", err)
	}
	return nil
}
