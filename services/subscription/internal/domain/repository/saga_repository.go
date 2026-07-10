package repository

import (
	"context"

	sharedModel "github.com/GenesisEducationKyiv/software-engineering-school-6-0-ReilEgor/services/subscription/internal/domain/model"
)

//go:generate mockery --name SagaRepository --output ../../mocks --case underscore --outpkg mocks
type SagaRepository interface {
	Create(ctx context.Context, subscriptionID int64) (*sharedModel.SubscriptionSaga, error)
	GetByID(ctx context.Context, sagaID int64) (*sharedModel.SubscriptionSaga, error)
	UpdateStatus(ctx context.Context, sagaID int64, status sharedModel.SagaStatus) error
	UpdateStatusAndStep(
		ctx context.Context,
		sagaID int64,
		status sharedModel.SagaStatus,
		step sharedModel.SagaStep,
	) error
}
