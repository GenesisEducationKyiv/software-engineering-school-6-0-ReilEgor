package usecase

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strconv"
	"time"

	"github.com/google/uuid"

	"github.com/GenesisEducationKyiv/software-engineering-school-6-0-ReilEgor/internal/subscription/domain/model"
	"github.com/GenesisEducationKyiv/software-engineering-school-6-0-ReilEgor/internal/subscription/domain/port"
	"github.com/GenesisEducationKyiv/software-engineering-school-6-0-ReilEgor/internal/subscription/domain/repository"
	"github.com/GenesisEducationKyiv/software-engineering-school-6-0-ReilEgor/shared/metrics"
)

const componentUserUseCase = "UserUseCase"

const (
	errMsgGetUser   = "get user"
	errMsgDeleteSub = "delete subscription"
)

type UserUseCase struct {
	logger        *slog.Logger
	subsRepo      repository.SubscriptionRepository
	userRepo      repository.UserRepository
	repoUC        port.RepositoryUseCase
	confirmSender port.ConfirmationSender
	sagaRepo      repository.SagaRepository
	transactor    repository.Transactor
}

func NewUserUseCase(
	_ context.Context,
	sr repository.SubscriptionRepository,
	ur repository.UserRepository,
	ru port.RepositoryUseCase,
	cs port.ConfirmationSender,
	sagaRepo repository.SagaRepository,
	transactor repository.Transactor,
) (*UserUseCase, func()) {
	uc := &UserUseCase{
		logger:        slog.With(slog.String("component", componentUserUseCase)),
		subsRepo:      sr,
		userRepo:      ur,
		repoUC:        ru,
		confirmSender: cs,
		sagaRepo:      sagaRepo,
		transactor:    transactor,
	}
	return uc, func() {}
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
		if !errors.Is(err, model.ErrUserNotFound) {
			return fmt.Errorf("%s: get user: %w", op, err)
		}

		user = model.User{Email: email}
		if err = uc.userRepo.Create(ctx, &user); err != nil {
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

	var sagaID int64
	if err = uc.transactor.WithinTransaction(ctx, func(txCtx context.Context) error {
		if txErr := uc.subsRepo.Save(txCtx, sub); txErr != nil {
			log.ErrorContext(txCtx, "failed to save pending subscription", slog.Any("error", txErr))
			return fmt.Errorf("save pending: %w", txErr)
		}
		saga, txErr := uc.sagaRepo.Create(txCtx, sub.ID)
		if txErr != nil {
			return fmt.Errorf("create saga: %w", txErr)
		}
		sagaID = saga.ID
		return nil
	}); err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}

	if err := uc.confirmSender.SendConfirmation(ctx, email, repoName, token, sagaID, sub.ID); err != nil {
		log.ErrorContext(ctx, "failed to send confirmation", slog.Any("error", err))
		return fmt.Errorf("%s: send confirmation: %w", op, err)
	}

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

func (uc *UserUseCase) ListByEmail(ctx context.Context, email string) (_ []model.Subscription, err error) {
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
