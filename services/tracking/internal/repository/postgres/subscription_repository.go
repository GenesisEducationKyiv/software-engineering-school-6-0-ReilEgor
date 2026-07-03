package postgres

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/GenesisEducationKyiv/software-engineering-school-6-0-ReilEgor/shared/ctxlog"

	sharedPostgres "github.com/GenesisEducationKyiv/software-engineering-school-6-0-ReilEgor/shared/infrastructure/storage/postgres"

	"github.com/GenesisEducationKyiv/software-engineering-school-6-0-ReilEgor/services/tracking/internal/domain/model"
)

const componentTrackerSubscriptionRepository = "TrackerSubscriptionRepository"

type SubscriptionRepository struct {
	db sharedPostgres.PgxInterface
}

func NewTrackerSubscriptionRepository(db sharedPostgres.PgxInterface) *SubscriptionRepository {
	return &SubscriptionRepository{db: db}
}

func (r *SubscriptionRepository) log(ctx context.Context) *slog.Logger {
	return ctxlog.FromCtx(ctx).With(slog.String("component", componentTrackerSubscriptionRepository))
}

const getSubscribersByRepoIDQuery = `
	SELECT email, token FROM subscriptions WHERE repository_id = $1
`

func (r *SubscriptionRepository) GetByRepoID(ctx context.Context, repoID int64) ([]model.Subscriber, error) {
	const op = "TrackerSubscriptionRepository.GetByRepoID"
	log := r.log(ctx).With(slog.String("op", op), slog.Int64("repo_id", repoID))
	log.DebugContext(ctx, "called")

	rows, err := r.db.Query(ctx, getSubscribersByRepoIDQuery, repoID)
	if err != nil {
		log.ErrorContext(ctx, "query failed", slog.String("error", err.Error()))
		return nil, fmt.Errorf("%s: query: %w", op, err)
	}
	defer rows.Close()

	var subscribers []model.Subscriber
	for rows.Next() {
		var s model.Subscriber
		if err := rows.Scan(&s.Email, &s.Token); err != nil {
			log.ErrorContext(ctx, "scan failed", slog.String("error", err.Error()))
			return nil, fmt.Errorf("%s: scan: %w", op, err)
		}
		subscribers = append(subscribers, s)
	}

	if err := rows.Err(); err != nil {
		log.ErrorContext(ctx, "rows iteration failed", slog.String("error", err.Error()))
		return nil, fmt.Errorf("%s: rows: %w", op, err)
	}

	log.DebugContext(ctx, "subscribers fetched", slog.Int("count", len(subscribers)))
	return subscribers, nil
}

const deleteSubscriptionByEmailAndRepoQuery = `
	DELETE FROM subscriptions
	WHERE email = $1 AND repository_id = (SELECT id FROM repositories WHERE full_name = $2)
`

func (r *SubscriptionRepository) DeleteByEmailAndRepo(ctx context.Context, email, repoName string) error {
	const op = "TrackerSubscriptionRepository.DeleteByEmailAndRepo"
	log := r.log(ctx).With(slog.String("op", op), slog.String("email", email), slog.String("repo", repoName))
	log.DebugContext(ctx, "called")

	_, err := r.db.Exec(ctx, deleteSubscriptionByEmailAndRepoQuery, email, repoName)
	if err != nil {
		log.ErrorContext(ctx, "delete subscription failed", slog.String("error", err.Error()))
		return fmt.Errorf("%s: exec: %w", op, err)
	}

	log.DebugContext(ctx, "subscription deleted")
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
	log := r.log(ctx).With(slog.String("op", op), slog.String("repo", repoName))
	log.DebugContext(ctx, "called")

	var exists bool
	err := r.db.QueryRow(ctx, hasSubscriptionsQuery, repoName).Scan(&exists)
	if err != nil {
		log.ErrorContext(ctx, "query failed", slog.String("error", err.Error()))
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
	log := r.log(ctx).With(slog.String("op", op), slog.Int64("repo_id", repoID), slog.String("email", email))
	log.DebugContext(ctx, "called")

	_, err := r.db.Exec(ctx, upsertSubscriptionQuery, repoID, email, token)
	if err != nil {
		log.ErrorContext(ctx, "upsert subscription failed", slog.String("error", err.Error()))
		return fmt.Errorf("%s: exec: %w", op, err)
	}

	log.DebugContext(ctx, "subscription upserted in tracker DB")
	return nil
}

const deleteSubscriptionByUserIDAndRepo = `
	DELETE FROM subscriptions
	WHERE user_id = $1 AND repository_id = (SELECT id FROM repositories WHERE full_name = $2)
`

func (r *SubscriptionRepository) DeleteByUserIDAndRepo(ctx context.Context, userID int64, repoName string) error {
	const op = "TrackerSubscriptionRepository.DeleteByUserIDAndRepo"
	log := r.log(ctx).With(slog.String("op", op), slog.Int64("user_id", userID), slog.String("repo", repoName))
	log.DebugContext(ctx, "called")

	_, err := r.db.Exec(ctx, deleteSubscriptionByUserIDAndRepo, userID, repoName)
	if err != nil {
		log.ErrorContext(ctx, "delete subscription failed", slog.String("error", err.Error()))
		return fmt.Errorf("%s: exec: %w", op, err)
	}

	log.DebugContext(ctx, "subscription deleted")
	return nil
}
