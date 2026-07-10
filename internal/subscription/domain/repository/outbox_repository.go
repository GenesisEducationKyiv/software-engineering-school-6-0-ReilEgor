package repository

import (
	"context"

	"github.com/GenesisEducationKyiv/software-engineering-school-6-0-ReilEgor/internal/subscription/domain/model"
)

//go:generate mockery --name OutboxRepository --output ../../mocks --case underscore --outpkg mocks
type OutboxRepository interface {
	Insert(ctx context.Context, queue string, payload []byte) error
	FetchPending(ctx context.Context, limit int) ([]model.OutboxMessage, error)
	Delete(ctx context.Context, id int64) error
}
