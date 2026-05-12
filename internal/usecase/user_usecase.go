package usecase

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strconv"
	"time"

	"github.com/google/uuid"

	"github.com/GenesisEducationKyiv/software-engineering-school-6-0-ReilEgor/internal/domain/model"
	"github.com/GenesisEducationKyiv/software-engineering-school-6-0-ReilEgor/internal/domain/repository"
	"github.com/GenesisEducationKyiv/software-engineering-school-6-0-ReilEgor/internal/domain/service"
	"github.com/GenesisEducationKyiv/software-engineering-school-6-0-ReilEgor/internal/domain/usecase"
)

const componentUserUseCase = "UserUseCase"

const (
	ErrMsgGetUser   = "get user"
	ErrMsgDeleteSub = "delete subscription"
)

type UserUseCase struct {
	logger       *slog.Logger
	subsRepo     repository.SubscriptionRepository
	userRepo     repository.UserRepository
	repoUC       usecase.RepositoryUseCase
	emailService service.EmailService
}

func NewUserUseCase(
	sr repository.SubscriptionRepository,
	ur repository.UserRepository,
	ru usecase.RepositoryUseCase,
	es service.EmailService,
) *UserUseCase {
	return &UserUseCase{
		logger:       slog.With(slog.String("useCase", componentUserUseCase)),
		subsRepo:     sr,
		userRepo:     ur,
		repoUC:       ru,
		emailService: es,
	}
}

func (uc *UserUseCase) Subscribe(ctx context.Context, email, repoName string) error {
	const op = "UserUseCase.Subscribe"
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

func (uc *UserUseCase) Unsubscribe(ctx context.Context, email, repoName string) error {
	const op = "UserUseCase.Unsubscribe"
	log := uc.logger.With(slog.String("op", op), slog.String("email", email), slog.String("repo", repoName))

	user, err := uc.userRepo.GetByEmail(ctx, email)
	if err != nil {
		if errors.Is(err, model.ErrUserNotFound) {
			log.DebugContext(ctx, "user not found, nothing to unsubscribe")
			return nil
		}
		log.ErrorContext(ctx, ErrMsgGetUser, slog.String("error", err.Error()))
		return fmt.Errorf("%s: %s: %w", op, ErrMsgGetUser, err)
	}

	if err = uc.subsRepo.Delete(ctx, user.ID, repoName); err != nil {
		log.ErrorContext(ctx, ErrMsgDeleteSub, slog.String("error", err.Error()))
		return fmt.Errorf("%s: %s: %w", op, ErrMsgDeleteSub, err)
	}

	log.InfoContext(ctx, "unsubscribed successfully")
	return nil
}

func (uc *UserUseCase) ListByEmail(ctx context.Context, email string) ([]model.Subscription, error) {
	const op = "UserUseCase.ListByEmail"

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

func (uc *UserUseCase) Confirm(ctx context.Context, token string) error {
	const op = "UserUseCase.Confirm"
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

func (uc *UserUseCase) UnsubscribeByToken(ctx context.Context, token string) error {
	const op = "UserUseCase.UnsubscribeByToken"
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

func (uc *UserUseCase) sendConfirmationEmail(email, repo, token string) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := uc.emailService.SendConfirmation(ctx, email, repo, token); err != nil {
		uc.logger.Error("failed to send confirmation email",
			slog.String("to", email),
			slog.Any("error", err),
		)
	}
}
