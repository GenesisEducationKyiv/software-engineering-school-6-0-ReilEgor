package postgres

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/GenesisEducationKyiv/software-engineering-school-6-0-ReilEgor/shared/ctxlog"

	sharedPostgres "github.com/GenesisEducationKyiv/software-engineering-school-6-0-ReilEgor/shared/infrastructure/storage/postgres"

	sharedModel "github.com/GenesisEducationKyiv/software-engineering-school-6-0-ReilEgor/services/subscription/internal/domain/model"
)

const componentSagaRepository = "SagaRepository"

type SagaRepository struct {
	db sharedPostgres.PgxInterface
}

func NewSagaRepository(db sharedPostgres.PgxInterface) *SagaRepository {
	return &SagaRepository{db: db}
}

func (sr *SagaRepository) log(ctx context.Context) *slog.Logger {
	return ctxlog.FromCtx(ctx).With(slog.String("component", componentSagaRepository))
}

const createSagaQuery = `
		INSERT INTO subscription_sagas (subscription_id, status, current_step)
		VALUES ($1, $2, $3)
		RETURNING id, subscription_id, status, current_step, created_at, updated_at
		`

func (sr *SagaRepository) Create(ctx context.Context, subscriptionID int64) (*sharedModel.SubscriptionSaga, error) {
	const op = "SagaRepository.Create"
	log := sr.log(ctx)
	log.DebugContext(ctx, "called", slog.String("op", op), slog.Int64("subscription_id", subscriptionID))

	saga := &sharedModel.SubscriptionSaga{}
	err := sharedPostgres.Extract(ctx, sr.db).
		QueryRow(ctx, createSagaQuery, subscriptionID, sharedModel.SagaStatusStarted, sharedModel.SagaStepSendConfirmation).
		Scan(&saga.ID, &saga.SubscriptionID, &saga.Status, &saga.CurrentStep, &saga.CreatedAt, &saga.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	log.DebugContext(
		ctx,
		"saga created",
		slog.Int64("saga_id", saga.ID),
		slog.Int64("subscription_id", subscriptionID),
		slog.String("step", string(saga.CurrentStep)),
	)
	return saga, nil
}

const getSagaByIDQuery = `
		SELECT id, subscription_id, status, current_step, created_at, updated_at
		FROM subscription_sagas
		WHERE id = $1
		`

func (sr *SagaRepository) GetByID(ctx context.Context, sagaID int64) (*sharedModel.SubscriptionSaga, error) {
	const op = "SagaRepository.GetByID"
	sr.log(ctx).DebugContext(ctx, "called", slog.String("op", op), slog.Int64("saga_id", sagaID))

	saga := &sharedModel.SubscriptionSaga{}
	err := sharedPostgres.Extract(ctx, sr.db).
		QueryRow(ctx, getSagaByIDQuery, sagaID).
		Scan(&saga.ID, &saga.SubscriptionID, &saga.Status, &saga.CurrentStep, &saga.CreatedAt, &saga.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}
	return saga, nil
}

const updateSagaStatusQuery = `
		UPDATE subscription_sagas
		SET status = $1, updated_at = NOW()
		WHERE id = $2
		`

func (sr *SagaRepository) UpdateStatus(ctx context.Context, sagaID int64, status sharedModel.SagaStatus) error {
	const op = "SagaRepository.UpdateStatus"
	log := sr.log(ctx)
	log.DebugContext(
		ctx,
		"called",
		slog.String("op", op),
		slog.Int64("saga_id", sagaID),
		slog.String("status", string(status)),
	)

	_, err := sharedPostgres.Extract(ctx, sr.db).Exec(ctx, updateSagaStatusQuery, status, sagaID)
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}

	log.DebugContext(
		ctx,
		"saga status updated",
		slog.Int64("saga_id", sagaID),
		slog.String("status", string(status)),
	)
	return nil
}

const updateSagaStatusAndStepQuery = `
		UPDATE subscription_sagas
		SET status = $1, current_step = $2, updated_at = NOW()
		WHERE id = $3
		`

func (sr *SagaRepository) UpdateStatusAndStep(
	ctx context.Context,
	sagaID int64,
	status sharedModel.SagaStatus,
	step sharedModel.SagaStep,
) error {
	const op = "SagaRepository.UpdateStatusAndStep"
	log := sr.log(ctx)
	log.DebugContext(ctx, "called",
		slog.String("op", op),
		slog.Int64("saga_id", sagaID),
		slog.String("status", string(status)),
		slog.String("step", string(step)),
	)

	_, err := sharedPostgres.Extract(ctx, sr.db).Exec(ctx, updateSagaStatusAndStepQuery, status, step, sagaID)
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}

	log.DebugContext(
		ctx,
		"saga status and step updated",
		slog.Int64("saga_id", sagaID),
		slog.String("status", string(status)),
		slog.String("step", string(step)),
	)
	return nil
}
