package usecase

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strconv"
	"time"

	"github.com/google/uuid"
	"golang.org/x/sync/errgroup"

	"github.com/GenesisEducationKyiv/software-engineering-school-6-0-ReilEgor/internal/domain/model"
	"github.com/GenesisEducationKyiv/software-engineering-school-6-0-ReilEgor/internal/domain/repository"
	"github.com/GenesisEducationKyiv/software-engineering-school-6-0-ReilEgor/internal/domain/service"
	"github.com/GenesisEducationKyiv/software-engineering-school-6-0-ReilEgor/internal/domain/usecase"
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
)

const (
	maxSendWorkers = 10
)

type SubscriptionUseCase struct {
	logger       *slog.Logger
	subsRepo     repository.SubscriptionRepository
	repoUC       usecase.RepositoryUseCase
	userRepo     repository.UserRepository
	emailService service.EmailService
	ghClient     service.GitHubClient
	repoRepo     repository.RepositoryRepository
}

func NewSubscriptionUseCase(
	sr repository.SubscriptionRepository,
	gh service.GitHubClient,
	ru usecase.RepositoryUseCase,
	ur repository.UserRepository,
	es service.EmailService,
	rr repository.RepositoryRepository,
) *SubscriptionUseCase {
	return &SubscriptionUseCase{
		logger:       slog.With(slog.String("useCase", componentSubscriptionUseCase)),
		subsRepo:     sr,
		ghClient:     gh,
		repoUC:       ru,
		emailService: es,
		userRepo:     ur,
		repoRepo:     rr,
	}
}

func (uc *SubscriptionUseCase) Subscribe(ctx context.Context, email, repoName string) error {
	const op = "SubscriptionUseCase.Subscribe"
	log := uc.logger.With(
		slog.String("op", op),
		slog.String("email", email),
		slog.String("repo", repoName),
	)

	repo, err := uc.repoUC.GetOrCreate(ctx, repoName)
	if err != nil {
		return fmt.Errorf("%s: get or create repo: %w", op, err)
	}

	user, err := uc.userRepo.GetByEmail(ctx, email)
	if err != nil {
		if !errors.Is(err, model.ErrUserNotFound) {
			return fmt.Errorf("%s: get user: %w", op, err)
		}

		user = model.User{Email: email}
		if err := uc.userRepo.Create(ctx, &user); err != nil {
			return fmt.Errorf("%s: create user: %w", op, err)
		}
		log.InfoContext(ctx, "new user created", slog.String("id", strconv.FormatInt(user.ID, 10)))
	}

	token := uuid.NewString()
	sub := &model.Subscription{
		UserID:         user.ID,
		RepositoryID:   repo.ID,
		RepositoryName: repo.FullName,
		Token:          token,
		Confirmed:      false,
	}

	if err := uc.subsRepo.Save(ctx, sub); err != nil {
		log.ErrorContext(ctx, "failed to save pending subscription", slog.Any("error", err))
		return fmt.Errorf("%s: save pending: %w", op, err)
	}

	go uc.sendConfirmationEmail(email, repoName, token)

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
				return uc.sendNotificationEmail(sendCtx, sub, updatedRepo.FullName, updatedRepo.LastSeenTag)
			})
		}
	}

	if err := g.Wait(); err != nil {
		return fmt.Errorf("group task failed: %w", err)
	}

	return nil
}

func (uc *SubscriptionUseCase) Confirm(ctx context.Context, token string) error {
	const op = "SubscriptionUseCase.Confirm"
	log := uc.logger.With(slog.String("op", op))

	if token == "" {
		return model.ErrInvalidToken
	}

	sub, err := uc.subsRepo.GetByToken(ctx, token)
	if err != nil {
		if errors.Is(err, model.ErrInvalidToken) {
			log.WarnContext(ctx, "attempt to confirm with invalid token")
		}
		return fmt.Errorf("%s: %w", op, err)
	}

	sub.Confirmed = true

	if err := uc.subsRepo.Save(ctx, sub); err != nil {
		return fmt.Errorf("%s: save: %w", op, err)
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

	sub, err := uc.subsRepo.GetByToken(ctx, token)
	if err != nil {
		if errors.Is(err, model.ErrInvalidToken) {
			log.WarnContext(ctx, "invalid unsubscribe token", slog.String("token", token))
		}
		return fmt.Errorf("%s: get by token: %w", op, err)
	}

	if err := uc.subsRepo.Delete(ctx, sub.UserID, sub.RepositoryName); err != nil {
		return fmt.Errorf("%s: delete: %w", op, err)
	}

	log.InfoContext(ctx, "unsubscribed by token successfully")
	return nil
}

func (uc *SubscriptionUseCase) sendConfirmationEmail(email, repo, token string) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := uc.emailService.SendConfirmation(ctx, email, repo, token); err != nil {
		uc.logger.Error("failed to send confirmation email",
			slog.String("to", email),
			slog.Any("error", err),
		)
	}
}

func (uc *SubscriptionUseCase) sendNotificationEmail(
	ctx context.Context,
	sub model.Subscriber,
	repoName, tag string,
) error {
	mailCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	if err := uc.emailService.SendNotification(mailCtx, sub.Email, repoName, tag, sub.Token); err != nil {
		uc.logger.ErrorContext(mailCtx, "failed to send email",
			slog.String("to", sub.Email),
			slog.Any("error", err),
		)
	}
	return nil
}
