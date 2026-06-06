package usecase

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strconv"
	"sync"
	"time"

	"github.com/google/uuid"

	"github.com/GenesisEducationKyiv/software-engineering-school-6-0-ReilEgor/internal/notification/domain/service"
	model2 "github.com/GenesisEducationKyiv/software-engineering-school-6-0-ReilEgor/internal/shared/domain/model"
	"github.com/GenesisEducationKyiv/software-engineering-school-6-0-ReilEgor/internal/shared/metrics"
	"github.com/GenesisEducationKyiv/software-engineering-school-6-0-ReilEgor/internal/subscription/domain/repository"
	"github.com/GenesisEducationKyiv/software-engineering-school-6-0-ReilEgor/internal/tracking/domain/usecase"
)

const (
	componentUserUseCase            = "UserUseCase"
	sendConfirmationEmailctxTimeout = 10
)

const (
	ErrMsgGetUser   = "get user"
	ErrMsgDeleteSub = "delete subscription"
)

type UserUseCase struct {
	appCtx       context.Context
	logger       *slog.Logger
	subsRepo     repository.SubscriptionRepository
	userRepo     repository.UserRepository
	repoUC       usecase.RepositoryUseCase
	emailService service.EmailService
	wg           sync.WaitGroup
}

func NewUserUseCase(
	appCtx context.Context,
	sr repository.SubscriptionRepository,
	ur repository.UserRepository,
	ru usecase.RepositoryUseCase,
	es service.EmailService,
) (*UserUseCase, func()) {
	uc := &UserUseCase{
		logger:       slog.With(slog.String("useCase", componentUserUseCase)),
		subsRepo:     sr,
		userRepo:     ur,
		repoUC:       ru,
		emailService: es,
		appCtx:       appCtx,
	}
	return uc, func() { uc.wg.Wait() }
}

func (uc *UserUseCase) Subscribe(ctx context.Context, email, repoName string) (err error) {
	const op = "UserUseCase.Subscribe"
	start := time.Now()

	defer func() {
		status := "success"
		if err != nil {
			status = "error"
		}
		metrics.UsecaseOperationsTotal.WithLabelValues(op, status).Inc()
		metrics.UsecaseOperationDurationSeconds.WithLabelValues(op).Observe(time.Since(start).Seconds())
	}()

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
		if !errors.Is(err, model2.ErrUserNotFound) {
			return fmt.Errorf("%s: get user: %w", op, err)
		}

		user = model2.User{Email: email}
		if err := uc.userRepo.Create(ctx, &user); err != nil {
			return fmt.Errorf("%s: create user: %w", op, err)
		}
		log.InfoContext(ctx, "new user created", slog.String("id", strconv.FormatInt(user.ID, 10)))
	}

	token := uuid.NewString()
	sub := &model2.Subscription{
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

	uc.wg.Add(1)
	go func() {
		defer uc.wg.Done()
		uc.sendConfirmationEmail(email, repoName, token)
	}()

	return nil
}

func (uc *UserUseCase) Unsubscribe(ctx context.Context, email, repoName string) (err error) {
	const op = "UserUseCase.Unsubscribe"

	start := time.Now()

	defer func() {
		status := "success"
		if err != nil {
			status = "error"
		}
		metrics.UsecaseOperationsTotal.WithLabelValues(op, status).Inc()
		metrics.UsecaseOperationDurationSeconds.WithLabelValues(op).Observe(time.Since(start).Seconds())
	}()

	log := uc.logger.With(slog.String("op", op), slog.String("email", email), slog.String("repo", repoName))

	user, err := uc.userRepo.GetByEmail(ctx, email)
	if err != nil {
		if errors.Is(err, model2.ErrUserNotFound) {
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

func (uc *UserUseCase) ListByEmail(ctx context.Context, email string) (_ []model2.Subscription, err error) {
	const op = "UserUseCase.ListByEmail"

	start := time.Now()

	defer func() {
		status := "success"
		if err != nil {
			status = "error"
		}
		metrics.UsecaseOperationsTotal.WithLabelValues(op, status).Inc()
		metrics.UsecaseOperationDurationSeconds.WithLabelValues(op).Observe(time.Since(start).Seconds())
	}()

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

func (uc *UserUseCase) Confirm(ctx context.Context, token string) (err error) {
	const op = "UserUseCase.Confirm"

	start := time.Now()

	defer func() {
		status := "success"
		if err != nil {
			status = "error"
		}
		metrics.UsecaseOperationsTotal.WithLabelValues(op, status).Inc()
		metrics.UsecaseOperationDurationSeconds.WithLabelValues(op).Observe(time.Since(start).Seconds())
	}()

	log := uc.logger.With(slog.String("op", op))

	if token == "" {
		return model2.ErrInvalidToken
	}

	sub, err := uc.subsRepo.GetByToken(ctx, token)
	if err != nil {
		if errors.Is(err, model2.ErrInvalidToken) {
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

func (uc *UserUseCase) UnsubscribeByToken(ctx context.Context, token string) (err error) {
	const op = "UserUseCase.UnsubscribeByToken"

	start := time.Now()

	defer func() {
		status := "success"
		if err != nil {
			status = "error"
		}
		metrics.UsecaseOperationsTotal.WithLabelValues(op, status).Inc()
		metrics.UsecaseOperationDurationSeconds.WithLabelValues(op).Observe(time.Since(start).Seconds())
	}()

	log := uc.logger.With(slog.String("op", op))

	if token == "" {
		return model2.ErrInvalidToken
	}

	sub, err := uc.subsRepo.GetByToken(ctx, token)
	if err != nil {
		if errors.Is(err, model2.ErrInvalidToken) {
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
	ctx, cancel := context.WithTimeout(uc.appCtx, sendConfirmationEmailctxTimeout*time.Second)
	defer cancel()

	err := uc.emailService.SendConfirmation(ctx, email, repo, token)
	status := "success"
	if err != nil {
		status = "error"
		uc.logger.Error("failed to send confirmation email",
			slog.String("to", email),
			slog.Any("error", err),
		)
	}
	metrics.ConfirmationEmailsSentTotal.WithLabelValues(status).Inc()
}
