package postgres

import (
	"context"
	"fmt"

	subModel "github.com/GenesisEducationKyiv/software-engineering-school-6-0-ReilEgor/internal/subscription/domain/model"
	sharedPostgres "github.com/GenesisEducationKyiv/software-engineering-school-6-0-ReilEgor/shared/storage/postgres"
)

type OutboxRepository struct {
	db sharedPostgres.PgxInterface
}

func NewOutboxRepository(db sharedPostgres.PgxInterface) *OutboxRepository {
	return &OutboxRepository{db: db}
}

const insertOutboxQuery = `INSERT INTO outbox_messages (queue, payload) VALUES ($1, $2)`

func (r *OutboxRepository) Insert(ctx context.Context, queue string, payload []byte) error {
	_, err := sharedPostgres.Extract(ctx, r.db).Exec(ctx, insertOutboxQuery, queue, payload)
	if err != nil {
		return fmt.Errorf("OutboxRepository.Insert: %w", err)
	}
	return nil
}

const fetchPendingQuery = `
	SELECT id, queue, payload
	FROM outbox_messages
	ORDER BY id ASC
	LIMIT $1
`

func (r *OutboxRepository) FetchPending(ctx context.Context, limit int) ([]subModel.OutboxMessage, error) {
	rows, err := r.db.Query(ctx, fetchPendingQuery, limit)
	if err != nil {
		return nil, fmt.Errorf("OutboxRepository.FetchPending: %w", err)
	}
	defer rows.Close()

	var msgs []subModel.OutboxMessage
	for rows.Next() {
		var msg subModel.OutboxMessage
		if err := rows.Scan(&msg.ID, &msg.Queue, &msg.Payload); err != nil {
			return nil, fmt.Errorf("OutboxRepository.FetchPending scan: %w", err)
		}
		msgs = append(msgs, msg)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("OutboxRepository.FetchPending rows: %w", err)
	}
	return msgs, nil
}

const deleteOutboxQuery = `DELETE FROM outbox_messages WHERE id = $1`

func (r *OutboxRepository) Delete(ctx context.Context, id int64) error {
	_, err := r.db.Exec(ctx, deleteOutboxQuery, id)
	if err != nil {
		return fmt.Errorf("OutboxRepository.Delete: %w", err)
	}
	return nil
}
