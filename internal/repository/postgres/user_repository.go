package postgres

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"github.com/ReilEgor/RepoNotifier/internal/domain/model"
	"github.com/jackc/pgx/v5"
)

const (
	componentUserRepository = "UserRepository"
)

type UserRepository struct {
	db     PgxInterface
	logger *slog.Logger
}

func NewUserRepository(db PgxInterface) *UserRepository {
	return &UserRepository{
		db:     db,
		logger: slog.With(slog.String("component", componentUserRepository)),
	}
}

const getByEmailUserRepositoryQuery = `SELECT id, email, created_at FROM users WHERE email = $1`

func (r *UserRepository) GetByEmail(ctx context.Context, email string) (model.User, error) {
	const op = "UserRepository.GetByEmail"
	log := r.logger.With(slog.String("op", op))

	var user model.User
	err := r.db.QueryRow(ctx, getByEmailUserRepositoryQuery, email).Scan(&user.ID, &user.Email, &user.CreatedAt)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			log.DebugContext(ctx, "user not found",
				slog.String("email", email),
			)
			return model.User{}, fmt.Errorf("%s: %w", op, model.ErrUserNotFound)
		}
		log.ErrorContext(ctx, "query failed",
			slog.String("email", email),
			slog.String("error", err.Error()),
		)
		return model.User{}, fmt.Errorf("%s: query row: %w", op, err)
	}

	return user, nil
}

const getOrCreateUserRepositoryQuery = `
	INSERT INTO users (email)
	VALUES ($1)
	ON CONFLICT (email) DO UPDATE SET email = EXCLUDED.email
	RETURNING id, email, created_at
`

func (r *UserRepository) GetOrCreate(ctx context.Context, email string) (model.User, error) {
	const op = "UserRepository.GetOrCreate"
	log := r.logger.With(slog.String("op", op))

	var user model.User
	if err := r.db.QueryRow(ctx, getOrCreateUserRepositoryQuery, email).Scan(&user.ID, &user.Email, &user.CreatedAt); err != nil {
		log.ErrorContext(ctx, "query failed",
			slog.String("email", email),
			slog.String("error", err.Error()),
		)
		return model.User{}, fmt.Errorf("%s: query row: %w", op, err)
	}

	log.DebugContext(ctx, "user get or created",
		slog.String("email", email),
		slog.Int64("id", user.ID),
	)
	return user, nil
}
