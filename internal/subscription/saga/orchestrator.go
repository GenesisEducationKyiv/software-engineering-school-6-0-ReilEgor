package saga

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	"github.com/GenesisEducationKyiv/software-engineering-school-6-0-ReilEgor/internal/subscription/domain/repository"
	sharedRabbit "github.com/GenesisEducationKyiv/software-engineering-school-6-0-ReilEgor/shared/broker/rabbitmq"
	sharedModel "github.com/GenesisEducationKyiv/software-engineering-school-6-0-ReilEgor/shared/domain/model"
)

type Orchestrator struct {
	sagaRepo   repository.SagaRepository
	subsRepo   repository.SubscriptionWriter
	outboxRepo repository.OutboxRepository
	transactor repository.Transactor
	logger     *slog.Logger
}

func NewOrchestrator(
	sagaRepo repository.SagaRepository,
	subsRepo repository.SubscriptionWriter,
	outboxRepo repository.OutboxRepository,
	transactor repository.Transactor,
) *Orchestrator {
	return &Orchestrator{
		sagaRepo:   sagaRepo,
		subsRepo:   subsRepo,
		outboxRepo: outboxRepo,
		transactor: transactor,
		logger:     slog.With(slog.String("component", "SagaOrchestrator")),
	}
}

func (o *Orchestrator) Start(
	txCtx context.Context,
	subscriptionID int64,
	email, repoName, token string,
) error {
	saga, err := o.sagaRepo.Create(txCtx, subscriptionID)
	if err != nil {
		return fmt.Errorf("saga orchestrator: create: %w", err)
	}

	payload, err := json.Marshal(sharedModel.SendConfirmationCommand{
		SagaID:         saga.ID,
		SubscriptionID: subscriptionID,
		Email:          email,
		RepoName:       repoName,
		Token:          token,
	})
	if err != nil {
		return fmt.Errorf("saga orchestrator: marshal confirmation command: %w", err)
	}

	if err := o.outboxRepo.Insert(txCtx, sharedRabbit.QueueConfirmations, payload); err != nil {
		return fmt.Errorf("saga orchestrator: insert outbox: %w", err)
	}

	o.logger.DebugContext(txCtx, "saga started",
		slog.Int64("saga_id", saga.ID),
		slog.Int64("subscription_id", subscriptionID),
	)
	return nil
}

func (o *Orchestrator) HandleConfirmationReply(
	ctx context.Context,
	reply sharedModel.ConfirmationResultEvent,
) error {
	if reply.Success {
		return o.stepActivateSubscription(ctx, reply)
	}
	return o.compensate(ctx, reply)
}

func (o *Orchestrator) stepActivateSubscription(
	ctx context.Context,
	reply sharedModel.ConfirmationResultEvent,
) error {
	current, err := o.sagaRepo.GetByID(ctx, reply.SagaID)
	if err != nil {
		return fmt.Errorf("saga orchestrator: get saga: %w", err)
	}
	if current.Status == sharedModel.SagaStatusCompleted {
		o.logger.InfoContext(ctx, "saga already completed, skipping duplicate",
			slog.Int64("saga_id", reply.SagaID),
		)
		return nil
	}

	err = o.transactor.WithinTransaction(ctx, func(txCtx context.Context) error {
		if txErr := o.sagaRepo.UpdateStatusAndStep(
			txCtx,
			reply.SagaID,
			sharedModel.SagaStatusCompleted,
			sharedModel.SagaStepActivateSubscription,
		); txErr != nil {
			return fmt.Errorf("update saga: %w", txErr)
		}

		payload, txErr := json.Marshal(sharedModel.SubscriptionActivatedEvent{
			FullName: reply.RepoName,
			Email:    reply.Email,
			Token:    reply.Token,
		})
		if txErr != nil {
			return fmt.Errorf("marshal activation command: %w", txErr)
		}

		return o.outboxRepo.Insert(txCtx, sharedRabbit.QueueSubscriptionActivated, payload)
	})
	if err != nil {
		return fmt.Errorf("saga orchestrator: step activate: %w", err)
	}

	o.logger.InfoContext(ctx, "saga completed — activation command enqueued",
		slog.Int64("saga_id", reply.SagaID),
		slog.Int64("subscription_id", reply.SubscriptionID),
	)
	return nil
}

func (o *Orchestrator) compensate(
	ctx context.Context,
	reply sharedModel.ConfirmationResultEvent,
) error {
	o.logger.WarnContext(ctx, "saga: confirmation failed, compensating",
		slog.Int64("saga_id", reply.SagaID),
		slog.Int64("subscription_id", reply.SubscriptionID),
	)
	if err := o.transactor.WithinTransaction(ctx, func(txCtx context.Context) error {
		if err := o.subsRepo.DeleteByID(txCtx, reply.SubscriptionID); err != nil {
			return fmt.Errorf("saga orchestrator: compensate delete subscription: %w", err)
		}
		if err := o.sagaRepo.UpdateStatus(txCtx, reply.SagaID, sharedModel.SagaStatusCompensated); err != nil {
			return fmt.Errorf("saga orchestrator: compensate update saga status: %w", err)
		}
		return nil
	}); err != nil {
		return fmt.Errorf("saga orchestrator: compensate: %w", err)
	}

	o.logger.InfoContext(ctx, "saga compensated",
		slog.Int64("saga_id", reply.SagaID),
	)
	return nil
}
