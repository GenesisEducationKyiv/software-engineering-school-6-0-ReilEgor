package postgres

import (
	"context"
	"fmt"
	"log/slog"

	sharedPostgres "github.com/GenesisEducationKyiv/software-engineering-school-6-0-ReilEgor/shared/storage/postgres"
)

type RepositoryRepository struct {
	db     sharedPostgres.PgxInterface
	logger *slog.Logger
}

func NewRepositoryRepository(db sharedPostgres.PgxInterface) *RepositoryRepository {
	return &RepositoryRepository{
		db:     db,
		logger: slog.With(slog.String("component", "SubRepositoryRepository")),
	}
}

const updateTagQuery = `
	UPDATE repositories SET last_seen_tag = $1 WHERE full_name = $2
`

func (r *RepositoryRepository) UpdateTag(ctx context.Context, fullName, tag string) error {
	const op = "SubRepositoryRepository.UpdateTag"

	_, err := r.db.Exec(ctx, updateTagQuery, tag, fullName)
	if err != nil {
		r.logger.ErrorContext(ctx, "update tag failed",
			slog.String("op", op),
			slog.String("repo", fullName),
			slog.Any("error", err),
		)
		return fmt.Errorf("%s: exec: %w", op, err)
	}

	r.logger.DebugContext(ctx, "tag updated in subscription DB",
		slog.String("repo", fullName),
		slog.String("tag", tag),
	)
	return nil
}
