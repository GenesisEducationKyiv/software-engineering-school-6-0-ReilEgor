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

const createUserQuery = `
	INSERT INTO users (email)
	VALUES ($1)
	RETURNING id, created_at
`

func (r *UserRepository) Create(ctx context.Context, user *model.User) error {
	const op = "UserRepository.Create"
	log := r.logger.With(slog.String("op", op), slog.String("email", user.Email))

	err := r.db.QueryRow(ctx, createUserQuery, user.Email).Scan(&user.ID, &user.CreatedAt)
	if err != nil {
		log.ErrorContext(ctx, "insert failed",
			slog.String("email", user.Email),
			slog.String("error", err.Error()),
		)
		return fmt.Errorf("%s: insert: %w", op, err)
	}

	log.DebugContext(ctx, "user created",
		slog.String("email", user.Email),
		slog.Int64("id", user.ID),
	)
	return nil
}
