package saga

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/GenesisEducationKyiv/software-engineering-school-6-0-ReilEgor/shared/ctxlog"

	contracts "github.com/GenesisEducationKyiv/software-engineering-school-6-0-ReilEgor/shared/contracts"

	sharedModel "github.com/GenesisEducationKyiv/software-engineering-school-6-0-ReilEgor/services/subscription/internal/domain/model"
	"github.com/GenesisEducationKyiv/software-engineering-school-6-0-ReilEgor/services/subscription/internal/domain/repository"
)

type Orchestrator struct {
	sagaRepo   repository.SagaRepository
	subsRepo   repository.SubscriptionWriter
	outboxRepo repository.OutboxRepository
	transactor repository.Transactor
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
	}
}

func (o *Orchestrator) log(ctx context.Context) *slog.Logger {
	return ctxlog.FromCtx(ctx).With(slog.String("component", "SagaOrchestrator"))
}

func (o *Orchestrator) Start(
	txCtx context.Context,
	subscriptionID int64,
	email, repoName, token string,
) error {
	const op = "Orchestrator.Start"
	start := time.Now()
	o.log(txCtx).DebugContext(txCtx, "called",
		slog.String("op", op),
		slog.Int64("subscription_id", subscriptionID),
	)

	saga, err := o.sagaRepo.Create(txCtx, subscriptionID)
	if err != nil {
		return fmt.Errorf("saga orchestrator: create: %w", err)
	}

	payload, err := json.Marshal(contracts.SendConfirmationCommand{
		SagaID:         saga.ID,
		SubscriptionID: subscriptionID,
		Email:          email,
		RepoName:       repoName,
		Token:          token,
		RequestID:      ctxlog.RequestID(txCtx),
	})
	if err != nil {
		return fmt.Errorf("saga orchestrator: marshal confirmation command: %w", err)
	}

	if err := o.outboxRepo.Insert(txCtx, contracts.QueueConfirmations, payload); err != nil {
		return fmt.Errorf("saga orchestrator: insert outbox: %w", err)
	}

	o.log(txCtx).DebugContext(txCtx, "saga started",
		slog.String("op", op),
		slog.Int64("saga_id", saga.ID),
		slog.Int64("subscription_id", subscriptionID),
		slog.Duration("duration", time.Since(start)),
	)
	return nil
}

func (o *Orchestrator) HandleConfirmationReply(
	ctx context.Context,
	reply contracts.ConfirmationResultEvent,
) error {
	const op = "Orchestrator.HandleConfirmationReply"
	o.log(ctx).DebugContext(ctx, "called",
		slog.String("op", op),
		slog.Int64("saga_id", reply.SagaID),
		slog.Bool("success", reply.Success),
	)

	if reply.Success {
		return o.stepActivateSubscription(ctx, reply)
	}
	return o.compensate(ctx, reply)
}

func (o *Orchestrator) stepActivateSubscription(
	ctx context.Context,
	reply contracts.ConfirmationResultEvent,
) error {
	const op = "Orchestrator.stepActivateSubscription"
	start := time.Now()
	log := o.log(ctx).With(slog.String("op", op))
	log.DebugContext(ctx, "called", slog.Int64("saga_id", reply.SagaID))

	current, err := o.sagaRepo.GetByID(ctx, reply.SagaID)
	if err != nil {
		return fmt.Errorf("saga orchestrator: get saga: %w", err)
	}
	if current.Status == sharedModel.SagaStatusCompleted {
		log.InfoContext(ctx, "saga already completed, skipping duplicate",
			slog.Int64("saga_id", reply.SagaID),
			slog.Duration("duration", time.Since(start)),
		)
		return nil
	}

	if err = o.sagaRepo.UpdateStatusAndStep(
		ctx,
		reply.SagaID,
		sharedModel.SagaStatusCompleted,
		sharedModel.SagaStepActivateSubscription,
	); err != nil {
		return fmt.Errorf("saga orchestrator: step activate: update saga: %w", err)
	}

	log.InfoContext(ctx, "saga completed — awaiting user confirmation",
		slog.Int64("saga_id", reply.SagaID),
		slog.Int64("subscription_id", reply.SubscriptionID),
		slog.Duration("duration", time.Since(start)),
	)
	return nil
}

func (o *Orchestrator) compensate(
	ctx context.Context,
	reply contracts.ConfirmationResultEvent,
) error {
	const op = "Orchestrator.compensate"
	start := time.Now()
	log := o.log(ctx).With(slog.String("op", op))
	log.DebugContext(ctx, "called", slog.Int64("saga_id", reply.SagaID))

	log.WarnContext(ctx, "saga: confirmation failed, compensating",
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

	log.InfoContext(ctx, "saga compensated",
		slog.Int64("saga_id", reply.SagaID),
		slog.Duration("duration", time.Since(start)),
	)
	return nil
}
