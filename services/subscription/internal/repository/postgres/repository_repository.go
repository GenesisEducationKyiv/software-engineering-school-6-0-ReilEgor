package postgres

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/GenesisEducationKyiv/software-engineering-school-6-0-ReilEgor/shared/ctxlog"

	sharedPostgres "github.com/GenesisEducationKyiv/software-engineering-school-6-0-ReilEgor/shared/infrastructure/storage/postgres"

	subModel "github.com/GenesisEducationKyiv/software-engineering-school-6-0-ReilEgor/services/subscription/internal/domain/model"
)

const componentRepositoryRepository = "SubRepositoryRepository"

type RepositoryRepository struct {
	db sharedPostgres.PgxInterface
}

func NewRepositoryRepository(db sharedPostgres.PgxInterface) *RepositoryRepository {
	return &RepositoryRepository{db: db}
}

func (r *RepositoryRepository) log(ctx context.Context) *slog.Logger {
	return ctxlog.FromCtx(ctx).With(slog.String("component", componentRepositoryRepository))
}

const updateTagQuery = `
	UPDATE repositories SET last_seen_tag = $1 WHERE full_name = $2
`

const getOrCreateRepositoryQuery = `
	INSERT INTO repositories (full_name, last_seen_tag) VALUES ($1, NULLIF($2, ''))
	ON CONFLICT (full_name) DO UPDATE SET full_name = EXCLUDED.full_name
	RETURNING id, full_name
`

func (r *RepositoryRepository) GetOrCreate(
	ctx context.Context,
	fullName, lastSeenTag string,
) (*subModel.RepositoryRef, error) {
	const op = "SubRepositoryRepository.GetOrCreate"
	log := r.log(ctx)
	log.DebugContext(ctx, "called", slog.String("op", op), slog.String("repo", fullName))

	var ref subModel.RepositoryRef
	err := r.db.QueryRow(ctx, getOrCreateRepositoryQuery, fullName, lastSeenTag).Scan(&ref.ID, &ref.FullName)
	if err != nil {
		log.ErrorContext(ctx, "get or create repository failed",
			slog.String("op", op),
			slog.String("repo", fullName),
			slog.Any("error", err),
		)
		return nil, fmt.Errorf("%s: %w", op, err)
	}
	return &ref, nil
}

func (r *RepositoryRepository) UpdateTag(ctx context.Context, fullName, tag string) error {
	const op = "SubRepositoryRepository.UpdateTag"
	log := r.log(ctx)
	log.DebugContext(ctx, "called", slog.String("op", op), slog.String("repo", fullName), slog.String("tag", tag))

	_, err := r.db.Exec(ctx, updateTagQuery, tag, fullName)
	if err != nil {
		log.ErrorContext(ctx, "update tag failed",
			slog.String("op", op),
			slog.String("repo", fullName),
			slog.Any("error", err),
		)
		return fmt.Errorf("%s: exec: %w", op, err)
	}

	log.DebugContext(ctx, "tag updated in subscription DB",
		slog.String("repo", fullName),
		slog.String("tag", tag),
	)
	return nil
}
