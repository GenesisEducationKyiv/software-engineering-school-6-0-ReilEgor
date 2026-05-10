package postgres

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"github.com/jackc/pgx/v5"

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

func (r *SubscriptionRepository) Delete(ctx context.Context, userID int64, repoName string) error {
	const op = "SubscriptionRepository.Delete"
	log := r.logger.With(slog.String("op", op))

	res, err := r.db.Exec(ctx, deleteSubscriptionQuery, userID, repoName)
	if err != nil {
		log.ErrorContext(ctx, "failed to delete subscription",
			slog.Int64("user_id", userID),
			slog.String("repo", repoName),
			slog.String("error", err.Error()),
		)
		return fmt.Errorf("%s: exec: %w", op, err)
	}

	log.DebugContext(ctx, "subscription deleted",
		slog.Int64("user_id", userID),
		slog.String("repo", repoName),
		slog.Int64("affected", res.RowsAffected()),
	)
	return nil
}

const getByTokenQuery = `
	SELECT s.id, s.user_id, s.repository_id, r.full_name, s.token, s.is_confirmed, s.created_at
	FROM subscriptions s
	JOIN repositories r ON s.repository_id = r.id
	WHERE s.token = $1
`

func (r *SubscriptionRepository) GetByToken(ctx context.Context, token string) (*model.Subscription, error) {
	const op = "SubscriptionRepository.GetByToken"

	var sub model.Subscription
	err := r.db.QueryRow(ctx, getByTokenQuery, token).Scan(
		&sub.ID,
		&sub.UserID,
		&sub.RepositoryID,
		&sub.RepositoryName,
		&sub.Token,
		&sub.Confirmed,
		&sub.CreatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("%s: %w", op, model.ErrInvalidToken)
		}
		return nil, fmt.Errorf("%s: query: %w", op, err)
	}

	return &sub, nil
}

const getSubscribersQuery = `
	SELECT u.email, s.token 
	FROM subscriptions s
	JOIN users u ON s.user_id = u.id
	WHERE s.repository_id = $1 AND s.is_confirmed = TRUE
`

func (r *SubscriptionRepository) GetByRepoID(ctx context.Context, repoID int64) ([]model.Subscriber, error) {
	const op = "SubscriptionRepository.GetByRepoID"

	rows, err := r.db.Query(ctx, getSubscribersQuery, repoID)
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
		s.token,
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
			&s.Token,
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

const saveSubscriptionQuery = `
	INSERT INTO subscriptions (user_id, repository_id, token, is_confirmed)
	VALUES ($1, $2, $3, $4)
	ON CONFLICT (user_id, repository_id) 
	DO UPDATE SET token = EXCLUDED.token, is_confirmed = EXCLUDED.is_confirmed
	RETURNING id
`

func (r *SubscriptionRepository) Save(ctx context.Context, sub *model.Subscription) error {
	const op = "SubscriptionRepository.Save"

	err := r.db.QueryRow(
		ctx,
		saveSubscriptionQuery,
		sub.UserID,
		sub.RepositoryID,
		sub.Token,
		sub.Confirmed,
	).Scan(&sub.ID)
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}

	return nil
}
