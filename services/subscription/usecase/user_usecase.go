package usecase

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"strconv"
	"time"

	"github.com/google/uuid"

	model2 "github.com/GenesisEducationKyiv/software-engineering-school-6-0-ReilEgor/services/subscription/domain/model"
	"github.com/GenesisEducationKyiv/software-engineering-school-6-0-ReilEgor/services/subscription/domain/port"
	repository2 "github.com/GenesisEducationKyiv/software-engineering-school-6-0-ReilEgor/services/subscription/domain/repository"
	"github.com/GenesisEducationKyiv/software-engineering-school-6-0-ReilEgor/services/subscription/saga"
	sharedModel "github.com/GenesisEducationKyiv/software-engineering-school-6-0-ReilEgor/shared/domain/model"
	"github.com/GenesisEducationKyiv/software-engineering-school-6-0-ReilEgor/shared/infrastructure/broker/rabbitmq"
	"github.com/GenesisEducationKyiv/software-engineering-school-6-0-ReilEgor/shared/infrastructure/metrics"
)

const componentUserUseCase = "UserUseCase"

const (
	errMsgGetUser = "get user"
)

type UserUseCase struct {
	logger       *slog.Logger
	subsRepo     repository2.SubscriptionRepository
	userRepo     repository2.UserRepository
	repoUC       port.RepositoryUseCase
	orchestrator *saga.Orchestrator
	transactor   repository2.Transactor
	outBox       repository2.OutboxRepository
}

func NewUserUseCase(
	_ context.Context,
	sr repository2.SubscriptionRepository,
	ur repository2.UserRepository,
	ru port.RepositoryUseCase,
	orchestrator *saga.Orchestrator,
	transactor repository2.Transactor,
	outBox repository2.OutboxRepository,
) (*UserUseCase, func()) {
	uc := &UserUseCase{
		logger:       slog.With(slog.String("component", componentUserUseCase)),
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

	repoRef, err := uc.repoUC.GetOrCreate(ctx, repoName)
	if err != nil {
		return fmt.Errorf("%s: get or create repo: %w", op, err)
	}

	user, err := uc.userRepo.GetByEmail(ctx, email)
	if err != nil {
		if !errors.Is(err, model2.ErrUserNotFound) {
			return fmt.Errorf("%s: get user: %w", op, err)
		}

		user = model2.User{Email: email}
		if err = uc.userRepo.Create(ctx, &user); err != nil {
			return fmt.Errorf("%s: create user: %w", op, err)
		}
		log.InfoContext(ctx, "new user created", slog.String("id", strconv.FormatInt(user.ID, 10)))
	}

	token := uuid.NewString()
	sub := &model2.Subscription{
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
		log.ErrorContext(ctx, errMsgGetUser, slog.String("error", err.Error()))
		return fmt.Errorf("%s: %s: %w", op, errMsgGetUser, err)
	}

	if err = uc.transactor.WithinTransaction(ctx, func(txCtx context.Context) error {
		if txErr := uc.subsRepo.Delete(txCtx, user.ID, repoName); txErr != nil {
			log.ErrorContext(txCtx, "failed to delete subscription", slog.Any("error", txErr))
			return fmt.Errorf("delete pending: %w", txErr)
		}
		payload, marshalErr := json.Marshal(sharedModel.UnsubscriptionActivatedEvent{
			Email:    email,
			RepoName: repoName,
		})
		if marshalErr != nil {
			return fmt.Errorf("marshal confirmation command: %w", marshalErr)
		}

		return uc.outBox.Insert(txCtx, rabbitmq.QueueUnsubscriptionActivated, payload)
	}); err != nil {
		return fmt.Errorf("%s: %w", op, err)
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

	if err = uc.transactor.WithinTransaction(ctx, func(txCtx context.Context) error {
		if txErr := uc.subsRepo.Save(txCtx, sub); txErr != nil {
			return fmt.Errorf("save: %w", txErr)
		}

		payload, txErr := json.Marshal(sharedModel.SubscriptionActivatedEvent{
			FullName: sub.RepositoryName,
			Email:    sub.Email,
			Token:    sub.Token,
		})
		if txErr != nil {
			return fmt.Errorf("marshal activation event: %w", txErr)
		}

		return uc.outBox.Insert(txCtx, rabbitmq.QueueSubscriptionActivated, payload)
	}); err != nil {
		return fmt.Errorf("%s: %w", op, err)
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
			log.WarnContext(ctx, "invalid unsubscribe token")
		}
		return fmt.Errorf("%s: get by token: %w", op, err)
	}

	if err = uc.transactor.WithinTransaction(ctx, func(txCtx context.Context) error {
		if txErr := uc.subsRepo.Delete(txCtx, sub.UserID, sub.RepositoryName); txErr != nil {
			log.ErrorContext(txCtx, "failed to delete subscription", slog.Any("error", txErr))
			return fmt.Errorf("delete pending: %w", txErr)
		}
		payload, marshalErr := json.Marshal(sharedModel.UnsubscriptionActivatedEvent{
			Email:    sub.Email,
			RepoName: sub.RepositoryName,
		})
		if marshalErr != nil {
			return fmt.Errorf("marshal confirmation command: %w", marshalErr)
		}

		return uc.outBox.Insert(txCtx, rabbitmq.QueueUnsubscriptionActivated, payload)
	}); err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}

	log.InfoContext(ctx, "unsubscribed by token successfully")
	return nil
}
