package usecase

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"golang.org/x/sync/errgroup"

	"github.com/ReilEgor/RepoNotifier/internal/domain/model"
	"github.com/ReilEgor/RepoNotifier/internal/domain/repository"
	"github.com/ReilEgor/RepoNotifier/internal/domain/service"
)

const (
	componentSubscriptionUseCase = "SubscriptionUseCase"
)

const (
	errMsgGetUser        = "get user"
	errMsgDeleteSub      = "delete subscription"
	errMsgGetRepos       = "get repos"
	errMsgFetchRelease   = "fetch latest release"
	errMsgUpdateTag      = "update last seen tag"
	errMsgGetSubscribers = "get subscribers"
	// errMsgGetOrCreateUser = "get or create user".
	// errMsgGetOrCreateRepo = "get or create repo".
	// errMsgCreateSub       = "create subscription".
	// errMsgCheckRepoExists = "check repo exists".
)

const (
	maxSendWorkers = 10
)

type SubscriptionUseCase struct {
	logger      *slog.Logger
	subsRepo    repository.SubscriptionRepository
	userRepo    repository.UserRepository
	repoRepo    repository.RepositoryRepository
	emailSender service.EmailSender
	ghClient    service.GitHubClient
}

func NewSubscriptionUseCase(
	sr repository.SubscriptionRepository,
	gh service.GitHubClient,
	ur repository.UserRepository,
	rr repository.RepositoryRepository,
	es service.EmailSender,
) *SubscriptionUseCase {
	return &SubscriptionUseCase{
		logger:      slog.With(slog.String("useCase", componentSubscriptionUseCase)),
		subsRepo:    sr,
		ghClient:    gh,
		userRepo:    ur,
		repoRepo:    rr,
		emailSender: es,
	}
}

func (uc *SubscriptionUseCase) Subscribe(ctx context.Context, email, repoName string) error {
	const op = "SubscriptionUseCase.Subscribe"
	log := uc.logger.With(
		slog.String("op", op),
		slog.String("email", email),
		slog.String("repo", repoName),
	)

	exists, err := uc.ghClient.RepoExists(ctx, repoName)
	if err != nil {
		return fmt.Errorf("%s: check repo: %w", op, err)
	}
	if !exists {
		return service.ErrRepositoryNotFound
	}

	release, err := uc.ghClient.GetLatestRelease(ctx, repoName)
	if err != nil {
		return fmt.Errorf("%s: fetch release: %w", op, err)
	}

	repo, err := uc.repoRepo.GetOrCreate(ctx, repoName, release.TagName)
	if err != nil {
		return fmt.Errorf("%s: repo storage: %w", op, err)
	}

	user, err := uc.userRepo.GetOrCreate(ctx, email)
	if err != nil {
		return fmt.Errorf("%s: user storage: %w", op, err)
	}

	token := uuid.NewString()
	_, err = uc.subsRepo.CreatePending(ctx, user.ID, repo.ID, token)
	if err != nil {
		log.ErrorContext(ctx, "failed to create pending subscription", slog.String("error", err.Error()))
		return fmt.Errorf("%s: create pending: %w", op, err)
	}

	go func() {
		sendCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		if err := uc.emailSender.SendConfirmation(sendCtx, email, repoName, token); err != nil {
			log.Error("failed to send confirmation email", slog.String("error", err.Error()))
		}
	}()

	return nil
}

func (uc *SubscriptionUseCase) Unsubscribe(ctx context.Context, email, repoName string) error {
	const op = "SubscriptionUseCase.Unsubscribe"
	log := uc.logger.With(slog.String("op", op), slog.String("email", email), slog.String("repo", repoName))

	user, err := uc.userRepo.GetByEmail(ctx, email)
	if err != nil {
		if errors.Is(err, model.ErrUserNotFound) {
			log.DebugContext(ctx, "user not found, nothing to unsubscribe")
			return nil
		}
		log.ErrorContext(ctx, errMsgGetUser, slog.String("error", err.Error()))
		return fmt.Errorf("%s: %s: %w", op, errMsgGetUser, err)
	}

	if err = uc.subsRepo.Delete(ctx, user.ID, repoName); err != nil {
		log.ErrorContext(ctx, errMsgDeleteSub, slog.String("error", err.Error()))
		return fmt.Errorf("%s: %s: %w", op, errMsgDeleteSub, err)
	}

	log.InfoContext(ctx, "unsubscribed successfully")
	return nil
}

func (uc *SubscriptionUseCase) ListByEmail(ctx context.Context, email string) ([]model.Subscription, error) {
	const op = "SubscriptionUseCase.ListByEmail"

	subs, err := uc.subsRepo.GetByEmail(ctx, email)
	if err != nil {
		uc.logger.ErrorContext(ctx, "failed to list subscriptions",
			slog.String("op", op),
			slog.String("email", email),
			slog.String("error", err.Error()),
		)
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	return subs, nil
}

func (uc *SubscriptionUseCase) ProcessNotifications(ctx context.Context) error {
	const op = "SubscriptionUseCase.ProcessNotifications"
	log := uc.logger.With(slog.String("op", op))

	repos, err := uc.repoRepo.GetAll(ctx)
	if err != nil {
		return fmt.Errorf("%s: %s: %w", op, errMsgGetRepos, err)
	}

	g, sendCtx := errgroup.WithContext(context.Background())
	g.SetLimit(maxSendWorkers)

	for _, repo := range repos {
		log = log.With(slog.String("repo", repo.FullName))

		repoCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
		latestRelease, err := uc.ghClient.GetLatestRelease(repoCtx, repo.FullName)
		cancel()

		if err != nil {
			log.ErrorContext(ctx, errMsgFetchRelease, slog.String("error", err.Error()))
			continue
		}

		if latestRelease == nil {
			log.WarnContext(ctx, "latest release is nil")
			continue
		}

		if latestRelease.TagName == repo.LastSeenTag {
			continue
		}

		if err = uc.repoRepo.UpdateLastSeenTag(ctx, repo.FullName, latestRelease.TagName); err != nil {
			log.ErrorContext(ctx, errMsgUpdateTag, slog.String("error", err.Error()))
			continue
		}

		subs, err := uc.subsRepo.GetSubscribersByRepoID(ctx, repo.ID)
		if err != nil {
			log.ErrorContext(ctx, errMsgGetSubscribers, slog.String("error", err.Error()))
			continue
		}

		for _, sub := range subs {
			g.Go(func() error {
				mailCtx, mailCancel := context.WithTimeout(sendCtx, 5*time.Second)
				defer mailCancel()

				if err := uc.emailSender.SendNotification(
					mailCtx,
					sub.Email,
					repo.FullName,
					latestRelease.TagName,
					sub.Token,
				); err != nil {
					log.ErrorContext(mailCtx, "failed to send email",
						slog.String("to", sub.Email),
						slog.String("error", err.Error()),
					)
				}
				return nil
			})
		}
	}
	if err := g.Wait(); err != nil {
		return fmt.Errorf("%s: wait group: %w", op, err)
	}
	return nil
}

func (uc *SubscriptionUseCase) Confirm(ctx context.Context, token string) error {
	const op = "SubscriptionUseCase.Confirm"
	log := uc.logger.With(slog.String("op", op))

	if token == "" {
		return model.ErrInvalidToken
	}

	err := uc.subsRepo.Confirm(ctx, token)
	if err != nil {
		if errors.Is(err, model.ErrInvalidToken) {
			log.WarnContext(ctx, "attempt to confirm with invalid token")
			return fmt.Errorf("%s: %w", op, err)
		}
		return fmt.Errorf("%s: %w", op, err)
	}

	log.InfoContext(ctx, "subscription confirmed successfully")
	return nil
}

func (uc *SubscriptionUseCase) UnsubscribeByToken(ctx context.Context, token string) error {
	const op = "SubscriptionUseCase.UnsubscribeByToken"
	log := uc.logger.With(slog.String("op", op))

	if token == "" {
		return model.ErrInvalidToken
	}

	if err := uc.subsRepo.UnsubscribeByToken(ctx, token); err != nil {
		if errors.Is(err, model.ErrInvalidToken) {
			log.WarnContext(ctx, "invalid unsubscribe token", slog.String("token", token))
			return fmt.Errorf("%s: %w", op, err)
		}
		return fmt.Errorf("%s: %w", op, err)
	}

	log.InfoContext(ctx, "unsubscribed by token successfully")
	return nil
}
