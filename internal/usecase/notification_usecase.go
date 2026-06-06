package usecase

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"golang.org/x/sync/errgroup"

	"github.com/GenesisEducationKyiv/software-engineering-school-6-0-ReilEgor/internal/shared/config"
	"github.com/GenesisEducationKyiv/software-engineering-school-6-0-ReilEgor/internal/shared/domain/model"
	repository2 "github.com/GenesisEducationKyiv/software-engineering-school-6-0-ReilEgor/internal/shared/domain/repository"
	"github.com/GenesisEducationKyiv/software-engineering-school-6-0-ReilEgor/internal/shared/domain/service"
	"github.com/GenesisEducationKyiv/software-engineering-school-6-0-ReilEgor/internal/shared/domain/usecase"
	"github.com/GenesisEducationKyiv/software-engineering-school-6-0-ReilEgor/internal/shared/metrics"
)

const (
	componentNotificationUseCase = "NotificationUseCase"

	errMsgFetchRelease   = "fetch latest release"
	errMsgGetSubscribers = "get subscribers"
)

type NotificationUseCase struct {
	logger       *slog.Logger
	subsRepo     repository2.SubscriptionRepository
	repoRepo     repository2.RepositoryRepository
	repoUC       usecase.RepositoryUseCase
	emailService service.EmailService
	workerCfg    config.WorkerConfig
}

func NewNotificationUseCase(
	sr repository2.SubscriptionRepository,
	rr repository2.RepositoryRepository,
	ru usecase.RepositoryUseCase,
	es service.EmailService,
	workerCfg config.WorkerConfig,
) *NotificationUseCase {
	return &NotificationUseCase{
		logger:       slog.With(slog.String("useCase", componentNotificationUseCase)),
		subsRepo:     sr,
		repoRepo:     rr,
		repoUC:       ru,
		emailService: es,
		workerCfg:    workerCfg,
	}
}

func (uc *NotificationUseCase) ProcessNotifications(ctx context.Context) (err error) {
	const op = "NotificationUseCase.ProcessNotifications"

	start := time.Now()
	defer func() {
		status := "success"
		if err != nil {
			status = "error"
		}
		metrics.NotificationsProcessedTotal.WithLabelValues(status).Inc()
		metrics.NotificationProcessingDurationSeconds.Observe(time.Since(start).Seconds())
	}()

	repos, err := uc.repoRepo.GetAll(ctx)
	if err != nil {
		return fmt.Errorf("%s: get repos: %w", op, err)
	}

	g, sendCtx := errgroup.WithContext(ctx)
	g.SetLimit(uc.workerCfg.MaxSendWorkers)

	for _, repo := range repos {
		updatedRepo, err := uc.repoUC.CheckForUpdates(ctx, repo)
		if err != nil {
			uc.logger.ErrorContext(ctx, errMsgFetchRelease,
				slog.String("repo", repo.FullName),
				slog.Any("error", err),
			)
			continue
		}

		if updatedRepo == nil {
			continue
		}

		subs, err := uc.subsRepo.GetByRepoID(ctx, updatedRepo.ID)
		if err != nil {
			uc.logger.ErrorContext(ctx, errMsgGetSubscribers,
				slog.String("repo", repo.FullName),
				slog.Any("error", err),
			)
			continue
		}

		for _, sub := range subs {
			g.Go(func() error {
				if err := uc.sendNotificationEmail(
					sendCtx,
					sub,
					updatedRepo.FullName,
					updatedRepo.LastSeenTag,
				); err != nil {
					uc.logger.WarnContext(sendCtx, "skipping failed notification",
						slog.String("email", sub.Email),
						slog.Any("error", err),
					)
				}
				return nil
			})
		}
	}

	if err := g.Wait(); err != nil {
		return fmt.Errorf("%s: group task: %w", op, err)
	}

	return nil
}

func (uc *NotificationUseCase) sendNotificationEmail(
	ctx context.Context,
	sub model.Subscriber,
	repoName, tag string,
) (err error) {
	const op = "NotificationUseCase.sendNotificationEmail"

	defer func() {
		status := "success"
		if err != nil {
			status = "error"
		}
		metrics.NotificationEmailsSentTotal.WithLabelValues(status).Inc()
	}()

	mailCtx, cancel := context.WithTimeout(ctx, uc.workerCfg.SendTimeout)
	defer cancel()

	if err := uc.emailService.SendNotification(mailCtx, sub.Email, repoName, tag, sub.Token); err != nil {
		uc.logger.ErrorContext(mailCtx, "failed to send email",
			slog.String("to", sub.Email),
			slog.Any("error", err),
		)
		return fmt.Errorf("%s: %w", op, err)
	}
	return nil
}
