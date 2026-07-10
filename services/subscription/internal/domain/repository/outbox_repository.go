package repository

import (
	"context"

	sharedModel "github.com/GenesisEducationKyiv/software-engineering-school-6-0-ReilEgor/shared/domain/model"
)

//go:generate mockery --name OutboxRepository --output ../../mocks --case underscore --outpkg mocks
type OutboxRepository interface {
	Insert(ctx context.Context, queue string, payload []byte) error
	FetchPending(ctx context.Context, limit int) ([]sharedModel.OutboxMessage, error)
	Delete(ctx context.Context, id int64) error
	Fail(ctx context.Context, id int64, maxAttempts int, lastErr string) error
}
