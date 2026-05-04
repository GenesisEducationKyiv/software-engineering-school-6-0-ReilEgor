package postgres

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/GenesisEducationKyiv/software-engineering-school-6-0-ReilEgor/internal/domain/model"
)

const (
	componentSubscriptionRepository = "SubscriptionRepository"
)

type SubscriptionRepository struct {
	db     PgxInterface
	logger *slog.Logger
}

func NewSubscriptionRepository(db PgxInterface) *SubscriptionRepository {
	return &SubscriptionRepository{
		db:     db,
		logger: slog.With(slog.String("component", componentSubscriptionRepository)),
	}
}

const deleteSubscriptionQuery = `
	DELETE FROM subscriptions 
	WHERE user_id = $1 AND repository_id = (SELECT id FROM repositories WHERE full_name = $2)
`

func (r *SubscriptionRepository) Delete(ctx context.Context, userID int64, repo string) error {
	const op = "SubscriptionRepository.Delete"
	log := r.logger.With(slog.String("op", op))

	res, err := r.db.Exec(ctx, deleteSubscriptionQuery, userID, repo)
	if err != nil {
		log.ErrorContext(ctx, "failed to delete subscription",
			slog.Int64("user_id", userID),
			slog.String("repo", repo),
			slog.String("error", err.Error()),
		)
		return fmt.Errorf("%s: exec: %w", op, err)
	}

	log.DebugContext(ctx, "subscription deleted",
		slog.Int64("user_id", userID),
		slog.String("repo", repo),
		slog.Int64("affected", res.RowsAffected()),
	)
	return nil
}

const confirmSubscriptionQuery = `
    UPDATE subscriptions 
    SET is_confirmed = TRUE 
    WHERE token = $1 
    RETURNING id
`

func (r *SubscriptionRepository) Confirm(ctx context.Context, token string) error {
	const op = "SubscriptionRepository.Confirm"

	result, err := r.db.Exec(ctx, confirmSubscriptionQuery, token)
	if err != nil {
		return fmt.Errorf("%s: exec: %w", op, err)
	}

	if result.RowsAffected() == 0 {
		return fmt.Errorf("%s: %w", op, model.ErrInvalidToken)
	}

	return nil
}

const unsubscribeByTokenQuery = `
    DELETE FROM subscriptions 
    WHERE token = $1 
    RETURNING id
`

func (r *SubscriptionRepository) UnsubscribeByToken(ctx context.Context, token string) error {
	const op = "SubscriptionRepository.UnsubscribeByToken"

	result, err := r.db.Exec(ctx, unsubscribeByTokenQuery, token)
	if err != nil {
		return fmt.Errorf("%s: exec: %w", op, err)
	}

	if result.RowsAffected() == 0 {
		return fmt.Errorf("%s: %w", op, model.ErrInvalidToken)
	}

	return nil
}

const getSubscribersQuery = `
    SELECT u.email, s.token 
    FROM subscriptions s
    JOIN users u ON s.user_id = u.id
    WHERE s.repository_id = $1 AND s.is_confirmed = TRUE
`

func (r *SubscriptionRepository) GetSubscribersByRepoID(ctx context.Context, id int64) ([]model.Subscriber, error) {
	const op = "SubscriptionRepository.GetSubscribersByRepoID"

	rows, err := r.db.Query(ctx, getSubscribersQuery, id)
	if err != nil {
		return nil, fmt.Errorf("%s: query: %w", op, err)
	}
	defer rows.Close()

	var subscribers []model.Subscriber
	for rows.Next() {
		var sub model.Subscriber
		if err := rows.Scan(&sub.Email, &sub.Token); err != nil {
			return nil, fmt.Errorf("%s: scan: %w", op, err)
		}
		subscribers = append(subscribers, sub)
	}

	return subscribers, nil
}

const listByEmailQuery = `
    SELECT 
        s.id, 
        r.id as repository_id,
        r.full_name, 
        s.is_confirmed, 
        r.last_seen_tag, 
        s.created_at
    FROM subscriptions s
    JOIN users u ON s.user_id = u.id
    JOIN repositories r ON s.repository_id = r.id
    WHERE u.email = $1
    ORDER BY s.created_at DESC
`

func (r *SubscriptionRepository) GetByEmail(ctx context.Context, email string) ([]model.Subscription, error) {
	const op = "SubscriptionRepository.GetByEmail"

	rows, err := r.db.Query(ctx, listByEmailQuery, email)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}
	defer rows.Close()

	var subs []model.Subscription
	for rows.Next() {
		var s model.Subscription
		err = rows.Scan(
			&s.ID,
			&s.RepositoryID,
			&s.RepositoryName,
			&s.Confirmed,
			&s.LastSeenTag,
			&s.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("%s: scan: %w", op, err)
		}
		subs = append(subs, s)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("%s: rows err: %w", op, err)
	}

	if subs == nil {
		return []model.Subscription{}, nil
	}

	return subs, nil
}
