package postgres

import (
	"context"
	"fmt"
	"log/slog"

	sharedPostgres "github.com/GenesisEducationKyiv/software-engineering-school-6-0-ReilEgor/shared/infrastructure/storage/postgres"

	"github.com/GenesisEducationKyiv/software-engineering-school-6-0-ReilEgor/services/tracking/internal/domain/model"
)

type SubscriptionRepository struct {
	db     sharedPostgres.PgxInterface
	logger *slog.Logger
}

func NewTrackerSubscriptionRepository(db sharedPostgres.PgxInterface) *SubscriptionRepository {
	return &SubscriptionRepository{
		db:     db,
		logger: slog.With(slog.String("component", "TrackerSubscriptionRepository")),
	}
}

const getSubscribersByRepoIDQuery = `
	SELECT email, token FROM subscriptions WHERE repository_id = $1
`

func (r *SubscriptionRepository) GetByRepoID(ctx context.Context, repoID int64) ([]model.Subscriber, error) {
	const op = "TrackerSubscriptionRepository.GetByRepoID"

	rows, err := r.db.Query(ctx, getSubscribersByRepoIDQuery, repoID)
	if err != nil {
		return nil, fmt.Errorf("%s: query: %w", op, err)
	}
	defer rows.Close()

	var subscribers []model.Subscriber
	for rows.Next() {
		var s model.Subscriber
		if err := rows.Scan(&s.Email, &s.Token); err != nil {
			return nil, fmt.Errorf("%s: scan: %w", op, err)
		}
		subscribers = append(subscribers, s)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("%s: rows: %w", op, err)
	}

	return subscribers, nil
}

const deleteSubscriptionByEmailAndRepoQuery = `
	DELETE FROM subscriptions
	WHERE email = $1 AND repository_id = (SELECT id FROM repositories WHERE full_name = $2)
`

func (r *SubscriptionRepository) DeleteByEmailAndRepo(ctx context.Context, email, repoName string) error {
	const op = "TrackerSubscriptionRepository.DeleteByEmailAndRepo"

	_, err := r.db.Exec(ctx, deleteSubscriptionByEmailAndRepoQuery, email, repoName)
	if err != nil {
		r.logger.ErrorContext(ctx, "delete subscription failed",
			slog.String("op", op),
			slog.String("email", email),
			slog.String("repo", repoName),
			slog.Any("error", err),
		)
		return fmt.Errorf("%s: exec: %w", op, err)
	}
	return nil
}

const hasSubscriptionsQuery = `
	SELECT EXISTS (
		SELECT 1 FROM subscriptions
		WHERE repository_id = (SELECT id FROM repositories WHERE full_name = $1)
	)
`

func (r *SubscriptionRepository) HasSubscriptions(ctx context.Context, repoName string) (bool, error) {
	const op = "TrackerSubscriptionRepository.HasSubscriptions"

	var exists bool
	err := r.db.QueryRow(ctx, hasSubscriptionsQuery, repoName).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("%s: query: %w", op, err)
	}
	return exists, nil
}

const upsertSubscriptionQuery = `
	INSERT INTO subscriptions (repository_id, email, token)
	VALUES ($1, $2, $3)
	ON CONFLICT (repository_id, email) DO UPDATE SET token = EXCLUDED.token
`

func (r *SubscriptionRepository) Upsert(ctx context.Context, repoID int64, email, token string) error {
	const op = "TrackerSubscriptionRepository.Upsert"

	_, err := r.db.Exec(ctx, upsertSubscriptionQuery, repoID, email, token)
	if err != nil {
		r.logger.ErrorContext(ctx, "upsert subscription failed",
			slog.String("op", op),
			slog.Int64("repo_id", repoID),
			slog.String("email", email),
			slog.Any("error", err),
		)
		return fmt.Errorf("%s: exec: %w", op, err)
	}

	r.logger.DebugContext(ctx, "subscription upserted in tracker DB",
		slog.Int64("repo_id", repoID),
		slog.String("email", email),
	)
	return nil
}

const deleteSubscriptionByUserIDAndRepo = `
	DELETE FROM subscriptions
	WHERE user_id = $1 AND repository_id = (SELECT id FROM repositories WHERE full_name = $2)
`

func (r *SubscriptionRepository) DeleteByUserIDAndRepo(ctx context.Context, userID int64, repoName string) error {
	const op = "TrackerSubscriptionRepository.DeleteByUserIDAndRepo"

	_, err := r.db.Exec(ctx, deleteSubscriptionByUserIDAndRepo, userID, repoName)
	if err != nil {
		r.logger.ErrorContext(ctx, "delete subscription failed",
			slog.String("op", op),
			slog.Int64("userID", userID),
			slog.String("repo", repoName),
			slog.Any("error", err),
		)
		return fmt.Errorf("%s: exec: %w", op, err)
	}
	return nil
}
