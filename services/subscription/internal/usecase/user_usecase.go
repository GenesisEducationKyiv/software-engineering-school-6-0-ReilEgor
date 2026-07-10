package usecase

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"strconv"
	"time"

	"github.com/GenesisEducationKyiv/software-engineering-school-6-0-ReilEgor/shared/ctxlog"
	"github.com/GenesisEducationKyiv/software-engineering-school-6-0-ReilEgor/shared/infrastructure/metrics"
	"github.com/google/uuid"

	contracts "github.com/GenesisEducationKyiv/software-engineering-school-6-0-ReilEgor/shared/contracts"

	"github.com/GenesisEducationKyiv/software-engineering-school-6-0-ReilEgor/services/subscription/internal/domain/model"
	"github.com/GenesisEducationKyiv/software-engineering-school-6-0-ReilEgor/services/subscription/internal/domain/repository"
)

//go:generate mockery --name TrackingRepository --output ../mocks --case underscore --outpkg mocks
type TrackingRepository interface {
	GetOrCreate(ctx context.Context, repoName string) (*model.RepositoryRef, error)
}

//go:generate mockery --name SagaOrchestrator --output ../mocks --case underscore --outpkg mocks
type SagaOrchestrator interface {
	Start(ctx context.Context, subscriptionID int64, email, repoName, token string) error
}

const componentUserUseCase = "UserUseCase"

const (
	errMsgGetUser = "get user"
)

type UserUseCase struct {
	subsRepo     repository.SubscriptionRepository
	userRepo     repository.UserRepository
	repoUC       TrackingRepository
	orchestrator SagaOrchestrator
	transactor   repository.Transactor
	outBox       repository.OutboxRepository
}

func (uc *UserUseCase) log(ctx context.Context) *slog.Logger {
	return ctxlog.FromCtx(ctx).With(slog.String("component", componentUserUseCase))
}

func NewUserUseCase(
	_ context.Context,
	sr repository.SubscriptionRepository,
	ur repository.UserRepository,
	ru TrackingRepository,
	orchestrator SagaOrchestrator,
	transactor repository.Transactor,
	outBox repository.OutboxRepository,
) (*UserUseCase, func()) {
	uc := &UserUseCase{
		subsRepo:     sr,
		userRepo:     ur,
		repoUC:       ru,
		orchestrator: orchestrator,
		transactor:   transactor,
		outBox:       outBox,
	}
	return uc, func() {}
}

func (uc *UserUseCase) Subscribe(ctx context.Context, email, repoName string) (err error) {
	const op = "UserUseCase.Subscribe"
	start := time.Now()
	log := uc.log(ctx).With(
		slog.String("op", op),
		slog.String("email", email),
		slog.String("repo", repoName),
	)
	log.DebugContext(ctx, "called")

	defer func() {
		elapsed := time.Since(start)
		status := "success"
		if err != nil {
			status = "error"
		}
		metrics.UsecaseOperationsTotal.WithLabelValues(op, status).Inc()
		metrics.UsecaseOperationDurationSeconds.WithLabelValues(op).Observe(elapsed.Seconds())
		log.InfoContext(ctx, "done", slog.String("status", status), slog.Duration("duration", elapsed))
	}()

	repoRef, err := uc.repoUC.GetOrCreate(ctx, repoName)
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
		RepositoryID:   repoRef.ID,
		RepositoryName: repoRef.FullName,
		Token:          token,
		Confirmed:      false,
	}

	if err = uc.transactor.WithinTransaction(ctx, func(txCtx context.Context) error {
		if txErr := uc.subsRepo.Save(txCtx, sub); txErr != nil {
			log.ErrorContext(txCtx, "failed to save pending subscription", slog.Any("error", txErr))
			return fmt.Errorf("save pending: %w", txErr)
		}
		return uc.orchestrator.Start(txCtx, sub.ID, email, repoName, token)
	}); err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}

	return nil
}

func (uc *UserUseCase) Unsubscribe(ctx context.Context, email, repoName string) (err error) {
	const op = "UserUseCase.Unsubscribe"
	start := time.Now()
	log := uc.log(ctx).With(slog.String("op", op), slog.String("email", email), slog.String("repo", repoName))
	log.DebugContext(ctx, "called")

	defer func() {
		elapsed := time.Since(start)
		status := "success"
		if err != nil {
			status = "error"
		}
		metrics.UsecaseOperationsTotal.WithLabelValues(op, status).Inc()
		metrics.UsecaseOperationDurationSeconds.WithLabelValues(op).Observe(elapsed.Seconds())
		log.InfoContext(ctx, "done", slog.String("status", status), slog.Duration("duration", elapsed))
	}()

	user, err := uc.userRepo.GetByEmail(ctx, email)
	if err != nil {
		if errors.Is(err, model.ErrUserNotFound) {
			log.DebugContext(ctx, "user not found, nothing to unsubscribe")
			return nil
		}
		log.ErrorContext(ctx, errMsgGetUser, slog.String("error", err.Error()))
		return fmt.Errorf("%s: %s: %w", op, errMsgGetUser, err)
	}

	if err = uc.transactor.WithinTransaction(ctx, func(txCtx context.Context) error {
		if txErr := uc.subsRepo.Delete(txCtx, user.ID, repoName); txErr != nil {
			log.ErrorContext(txCtx, "failed to delete subscription", slog.Any("error", txErr))
			return fmt.Errorf("delete pending: %w", txErr)
		}
		payload, marshalErr := json.Marshal(contracts.UnsubscriptionActivatedEvent{
			Email:     email,
			RepoName:  repoName,
			RequestID: ctxlog.RequestID(ctx),
		})
		if marshalErr != nil {
			return fmt.Errorf("marshal confirmation command: %w", marshalErr)
		}

		return uc.outBox.Insert(txCtx, contracts.QueueUnsubscriptionActivated, payload)
	}); err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}
	log.InfoContext(ctx, "unsubscribed successfully")
	return nil
}

func (uc *UserUseCase) ListByEmail(ctx context.Context, email string) (_ []model.Subscription, err error) {
	const op = "UserUseCase.ListByEmail"
	start := time.Now()
	log := uc.log(ctx).With(slog.String("op", op), slog.String("email", email))
	log.DebugContext(ctx, "called")

	defer func() {
		elapsed := time.Since(start)
		status := "success"
		if err != nil {
			status = "error"
		}
		metrics.UsecaseOperationsTotal.WithLabelValues(op, status).Inc()
		metrics.UsecaseOperationDurationSeconds.WithLabelValues(op).Observe(elapsed.Seconds())
		log.InfoContext(ctx, "done", slog.String("status", status), slog.Duration("duration", elapsed))
	}()

	subs, err := uc.subsRepo.GetByEmail(ctx, email)
	if err != nil {
		log.ErrorContext(ctx, "failed to list subscriptions", slog.String("error", err.Error()))
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	return subs, nil
}

func (uc *UserUseCase) Confirm(ctx context.Context, token string) (err error) {
	const op = "UserUseCase.Confirm"
	start := time.Now()
	log := uc.log(ctx).With(slog.String("op", op))
	log.DebugContext(ctx, "called")

	defer func() {
		elapsed := time.Since(start)
		status := "success"
		if err != nil {
			status = "error"
		}
		metrics.UsecaseOperationsTotal.WithLabelValues(op, status).Inc()
		metrics.UsecaseOperationDurationSeconds.WithLabelValues(op).Observe(elapsed.Seconds())
		log.InfoContext(ctx, "done", slog.String("status", status), slog.Duration("duration", elapsed))
	}()

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

	if err = uc.transactor.WithinTransaction(ctx, func(txCtx context.Context) error {
		if txErr := uc.subsRepo.Save(txCtx, sub); txErr != nil {
			return fmt.Errorf("save: %w", txErr)
		}

		payload, txErr := json.Marshal(contracts.SubscriptionActivatedEvent{
			FullName:  sub.RepositoryName,
			Email:     sub.Email,
			Token:     sub.Token,
			RequestID: ctxlog.RequestID(ctx),
		})
		if txErr != nil {
			return fmt.Errorf("marshal activation event: %w", txErr)
		}

		return uc.outBox.Insert(txCtx, contracts.QueueSubscriptionActivated, payload)
	}); err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}

	log.InfoContext(ctx, "subscription confirmed successfully")
	return nil
}

func (uc *UserUseCase) UnsubscribeByToken(ctx context.Context, token string) (err error) {
	const op = "UserUseCase.UnsubscribeByToken"
	start := time.Now()
	log := uc.log(ctx).With(slog.String("op", op))
	log.DebugContext(ctx, "called")

	defer func() {
		elapsed := time.Since(start)
		status := "success"
		if err != nil {
			status = "error"
		}
		metrics.UsecaseOperationsTotal.WithLabelValues(op, status).Inc()
		metrics.UsecaseOperationDurationSeconds.WithLabelValues(op).Observe(elapsed.Seconds())
		log.InfoContext(ctx, "done", slog.String("status", status), slog.Duration("duration", elapsed))
	}()

	if token == "" {
		return model.ErrInvalidToken
	}

	sub, err := uc.subsRepo.GetByToken(ctx, token)
	if err != nil {
		if errors.Is(err, model.ErrInvalidToken) {
			log.WarnContext(ctx, "invalid unsubscribe token")
		}
		return fmt.Errorf("%s: get by token: %w", op, err)
	}

	if err = uc.transactor.WithinTransaction(ctx, func(txCtx context.Context) error {
		if txErr := uc.subsRepo.Delete(txCtx, sub.UserID, sub.RepositoryName); txErr != nil {
			log.ErrorContext(txCtx, "failed to delete subscription", slog.Any("error", txErr))
			return fmt.Errorf("delete pending: %w", txErr)
		}
		payload, marshalErr := json.Marshal(contracts.UnsubscriptionActivatedEvent{
			Email:    sub.Email,
			RepoName: sub.RepositoryName,
		})
		if marshalErr != nil {
			return fmt.Errorf("marshal confirmation command: %w", marshalErr)
		}

		return uc.outBox.Insert(txCtx, contracts.QueueUnsubscriptionActivated, payload)
	}); err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}

	log.InfoContext(ctx, "unsubscribed by token successfully")
	return nil
}
