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
	componentRepositoryRepository = "RepositoryRepository"
)

type RepositoryRepository struct {
	db     PgxInterface
	logger *slog.Logger
}

func NewRepositoryRepository(db PgxInterface) *RepositoryRepository {
	return &RepositoryRepository{
		db:     db,
		logger: slog.With(slog.String("component", componentRepositoryRepository)),
	}
}

const getActiveRepositoriesQuery = `
	SELECT r.id, r.full_name, r.last_seen_tag, r.updated_at 
	FROM repositories r
	WHERE EXISTS (
		SELECT 1 FROM subscriptions s WHERE s.repository_id = r.id
	)
`

func (r *RepositoryRepository) GetAll(ctx context.Context) ([]model.Repository, error) {
	const op = "RepositoryRepository.GetAll"
	log := r.logger.With(slog.String("op", op))

	rows, err := r.db.Query(ctx, getActiveRepositoriesQuery)
	if err != nil {
		log.ErrorContext(ctx, "query failed", slog.String("error", err.Error()))
		return nil, fmt.Errorf("%s: query: %w", op, err)
	}
	defer rows.Close()

	repos := make([]model.Repository, 0)
	for rows.Next() {
		var repo model.Repository
		if err := rows.Scan(&repo.ID, &repo.FullName, &repo.LastSeenTag, &repo.UpdatedAt); err != nil {
			log.ErrorContext(ctx, "scan failed", slog.String("error", err.Error()))
			return nil, fmt.Errorf("%s: scan: %w", op, err)
		}
		repos = append(repos, repo)
	}

	if err := rows.Err(); err != nil {
		log.ErrorContext(ctx, "rows iteration failed", slog.String("error", err.Error()))
		return nil, fmt.Errorf("%s: rows: %w", op, err)
	}

	log.DebugContext(ctx, "repositories fetched", slog.Int("count", len(repos)))
	return repos, nil
}

const getRepositoryByNameQuery = `
	SELECT id, full_name, last_seen_tag, updated_at 
	FROM repositories 
	WHERE full_name = $1
`

func (r *RepositoryRepository) GetByName(ctx context.Context, name string) (*model.Repository, error) {
	const op = "RepositoryRepository.GetByName"
	log := r.logger.With(slog.String("op", op), slog.String("name", name))

	var repo model.Repository
	err := r.db.QueryRow(ctx, getRepositoryByNameQuery, name).Scan(
		&repo.ID,
		&repo.FullName,
		&repo.LastSeenTag,
		&repo.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf(
				"%s: %w",
				op,
				model.ErrRepositoryNotFound,
			)
		}
		log.ErrorContext(ctx, "query row failed", slog.String("error", err.Error()))
		return nil, fmt.Errorf("%s: query row: %w", op, err)
	}

	return &repo, nil
}

const createRepositoryQuery = `
	INSERT INTO repositories (full_name, last_seen_tag)
	VALUES ($1, $2)
	RETURNING id, updated_at
`

func (r *RepositoryRepository) Create(ctx context.Context, repo *model.Repository) error {
	const op = "RepositoryRepository.Create"
	log := r.logger.With(slog.String("op", op), slog.String("name", repo.FullName))

	err := r.db.QueryRow(ctx, createRepositoryQuery, repo.FullName, repo.LastSeenTag).Scan(
		&repo.ID,
		&repo.UpdatedAt,
	)
	if err != nil {
		log.ErrorContext(ctx, "insert failed", slog.String("error", err.Error()))
		return fmt.Errorf("%s: insert: %w", op, err)
	}

	log.DebugContext(ctx, "repository created", slog.Int64("id", repo.ID))
	return nil
}

const updateRepositoryQuery = `
	UPDATE repositories 
	SET last_seen_tag = $1, updated_at = CURRENT_TIMESTAMP 
	WHERE id = $2
`

func (r *RepositoryRepository) Update(ctx context.Context, repo *model.Repository) error {
	const op = "RepositoryRepository.Update"
	log := r.logger.With(slog.String("op", op), slog.Int64("id", repo.ID))

	commandTag, err := r.db.Exec(ctx, updateRepositoryQuery, repo.LastSeenTag, repo.ID)
	if err != nil {
		log.ErrorContext(ctx, "exec failed", slog.String("error", err.Error()))
		return fmt.Errorf("%s: exec: %w", op, err)
	}

	if commandTag.RowsAffected() == 0 {
		return fmt.Errorf("%s: %w", op, model.ErrRepositoryNotFound)
	}

	log.DebugContext(ctx, "repository updated")
	return nil
}
