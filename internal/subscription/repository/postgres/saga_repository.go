package postgres

import (
	"context"
	"fmt"
	"log/slog"

	sharedModel "github.com/GenesisEducationKyiv/software-engineering-school-6-0-ReilEgor/shared/domain/model"
	sharedPostgres "github.com/GenesisEducationKyiv/software-engineering-school-6-0-ReilEgor/shared/storage/postgres"
)

type SagaRepository struct {
	db     sharedPostgres.PgxInterface
	logger *slog.Logger
}

func NewSagaRepository(db sharedPostgres.PgxInterface) *SagaRepository {
	return &SagaRepository{
		db:     db,
		logger: slog.With(slog.String("component", "SagaRepository")),
	}
}

const createSagaQuery = `
		INSERT INTO subscription_sagas (subscription_id, status)
		VALUES ($1, $2)
		RETURNING id, subscription_id, status, created_at, updated_at
		`

func (sr *SagaRepository) Create(ctx context.Context, subscriptionID int64) (*sharedModel.SubscriptionSaga, error) {
	const op = "SagaRepository.Create"

	saga := &sharedModel.SubscriptionSaga{}
	err := sr.db.QueryRow(ctx, createSagaQuery, subscriptionID, sharedModel.SagaStatusStarted).
		Scan(&saga.ID, &saga.SubscriptionID, &saga.Status, &saga.CreatedAt, &saga.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	sr.logger.DebugContext(
		ctx,
		"saga created",
		slog.Int64("saga_id", saga.ID),
		slog.Int64("subscription_id", subscriptionID),
	)
	return saga, nil
}

const updateSagaStatusQuery = `
		UPDATE subscription_sagas
		SET status = $1, updated_at = NOW()
		WHERE id = $2
		`

func (sr *SagaRepository) UpdateStatus(ctx context.Context, sagaID int64, status sharedModel.SagaStatus) error {
	const op = "SagaRepository.UpdateStatus"

	_, err := sr.db.Exec(ctx, updateSagaStatusQuery, status, sagaID)
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}

	sr.logger.DebugContext(
		ctx,
		"saga status updated",
		slog.Int64("saga_id", sagaID),
		slog.String("status", string(status)),
	)
	return nil
}
