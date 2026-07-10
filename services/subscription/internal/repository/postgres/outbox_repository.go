package postgres

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/GenesisEducationKyiv/software-engineering-school-6-0-ReilEgor/shared/ctxlog"

	sharedModel "github.com/GenesisEducationKyiv/software-engineering-school-6-0-ReilEgor/shared/domain/model"
	sharedPostgres "github.com/GenesisEducationKyiv/software-engineering-school-6-0-ReilEgor/shared/infrastructure/storage/postgres"
)

const componentOutboxRepository = "OutboxRepository"

type OutboxRepository struct {
	db sharedPostgres.PgxInterface
}

func NewOutboxRepository(db sharedPostgres.PgxInterface) *OutboxRepository {
	return &OutboxRepository{db: db}
}

func (r *OutboxRepository) log(ctx context.Context) *slog.Logger {
	return ctxlog.FromCtx(ctx).With(slog.String("component", componentOutboxRepository))
}

const insertOutboxQuery = `INSERT INTO outbox_messages (queue, payload) VALUES ($1, $2)`

func (r *OutboxRepository) Insert(ctx context.Context, queue string, payload []byte) error {
	const op = "OutboxRepository.Insert"
	r.log(ctx).DebugContext(ctx, "called", slog.String("op", op), slog.String("queue", queue))

	_, err := sharedPostgres.Extract(ctx, r.db).Exec(ctx, insertOutboxQuery, queue, payload)
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}
	return nil
}

const fetchPendingQuery = `
	SELECT id, queue, payload
	FROM outbox_messages
	WHERE status = 'PENDING'
	ORDER BY id ASC
	LIMIT $1
	FOR UPDATE SKIP LOCKED
`

func (r *OutboxRepository) FetchPending(ctx context.Context, limit int) ([]sharedModel.OutboxMessage, error) {
	const op = "OutboxRepository.FetchPending"
	r.log(ctx).DebugContext(ctx, "called", slog.String("op", op), slog.Int("limit", limit))

	rows, err := sharedPostgres.Extract(ctx, r.db).Query(ctx, fetchPendingQuery, limit)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}
	defer rows.Close()

	var msgs []sharedModel.OutboxMessage
	for rows.Next() {
		var msg sharedModel.OutboxMessage
		if err := rows.Scan(&msg.ID, &msg.Queue, &msg.Payload); err != nil {
			return nil, fmt.Errorf("%s: scan: %w", op, err)
		}
		msgs = append(msgs, msg)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("%s: rows: %w", op, err)
	}
	return msgs, nil
}

const deleteOutboxQuery = `DELETE FROM outbox_messages WHERE id = $1`

func (r *OutboxRepository) Delete(ctx context.Context, id int64) error {
	const op = "OutboxRepository.Delete"
	r.log(ctx).DebugContext(ctx, "called", slog.String("op", op), slog.Int64("id", id))

	_, err := sharedPostgres.Extract(ctx, r.db).Exec(ctx, deleteOutboxQuery, id)
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}
	return nil
}

const failOutboxQuery = `
	UPDATE outbox_messages
	SET attempts   = attempts + 1,
	    last_error = $3,
	    status     = CASE WHEN attempts + 1 >= $2 THEN 'FAILED' ELSE 'PENDING' END
	WHERE id = $1
`

func (r *OutboxRepository) Fail(ctx context.Context, id int64, maxAttempts int, lastErr string) error {
	const op = "OutboxRepository.Fail"
	r.log(ctx).DebugContext(ctx, "called", slog.String("op", op), slog.Int64("id", id))

	_, err := sharedPostgres.Extract(ctx, r.db).Exec(ctx, failOutboxQuery, id, maxAttempts, lastErr)
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}
	return nil
}
