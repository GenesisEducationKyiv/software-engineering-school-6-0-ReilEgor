package postgres

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"github.com/jackc/pgx/v5"

	sharedPostgres "github.com/GenesisEducationKyiv/software-engineering-school-6-0-ReilEgor/shared/infrastructure/storage/postgres"

	subModel "github.com/GenesisEducationKyiv/software-engineering-school-6-0-ReilEgor/services/subscription/domain/model"
)

const (
	componentSubscriptionRepository = "SubscriptionRepository"
)

type SubscriptionRepository struct {
	db     sharedPostgres.PgxInterface
	logger *slog.Logger
}

func NewSubscriptionRepository(db sharedPostgres.PgxInterface) *SubscriptionRepository {
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

	res, err := sharedPostgres.Extract(ctx, r.db).Exec(ctx, deleteSubscriptionQuery, userID, repoName)
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

const deleteSubscriptionByIDQuery = `DELETE FROM subscriptions WHERE id = $1`

func (r *SubscriptionRepository) DeleteByID(ctx context.Context, subscriptionID int64) error {
	const op = "SubscriptionRepository.DeleteByID"

	res, err := sharedPostgres.Extract(ctx, r.db).Exec(ctx, deleteSubscriptionByIDQuery, subscriptionID)
	if err != nil {
		return fmt.Errorf("%s: exec: %w", op, err)
	}

	r.logger.DebugContext(ctx, "subscription deleted by id",
		slog.Int64("subscription_id", subscriptionID),
		slog.Int64("affected", res.RowsAffected()),
	)
	return nil
}

const getByTokenQuery = `
	SELECT s.id, s.user_id, u.email, s.repository_id, r.full_name, s.token, s.is_confirmed, s.created_at
	FROM subscriptions s
	JOIN repositories r ON s.repository_id = r.id
	JOIN users u ON s.user_id = u.id
	WHERE s.token = $1
`

func (r *SubscriptionRepository) GetByToken(ctx context.Context, token string) (*subModel.Subscription, error) {
	const op = "SubscriptionRepository.GetByToken"

	var sub subModel.Subscription
	err := r.db.QueryRow(ctx, getByTokenQuery, token).Scan(
		&sub.ID,
		&sub.UserID,
		&sub.Email,
		&sub.RepositoryID,
		&sub.RepositoryName,
		&sub.Token,
		&sub.Confirmed,
		&sub.CreatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("%s: %w", op, subModel.ErrInvalidToken)
		}
		return nil, fmt.Errorf("%s: query: %w", op, err)
	}

	return &sub, nil
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

func (r *SubscriptionRepository) GetByEmail(ctx context.Context, email string) ([]subModel.Subscription, error) {
	const op = "SubscriptionRepository.GetByEmail"

	rows, err := r.db.Query(ctx, listByEmailQuery, email)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}
	defer rows.Close()

	var subs []subModel.Subscription
	for rows.Next() {
		var s subModel.Subscription
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
		return []subModel.Subscription{}, nil
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

func (r *SubscriptionRepository) Save(ctx context.Context, sub *subModel.Subscription) error {
	const op = "SubscriptionRepository.Save"

	err := sharedPostgres.Extract(ctx, r.db).QueryRow(
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
