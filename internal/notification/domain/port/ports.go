package port

import (
	"context"

	"github.com/GenesisEducationKyiv/software-engineering-school-6-0-ReilEgor/shared/domain/model"
)

//go:generate mockery --name MessagePublisher --output ../../../shared/mocks --case underscore --outpkg mocks
type MessagePublisher interface {
	Publish(ctx context.Context, cmd model.SendNotificationCommand) error
}

type MessageConsumer interface {
	Start(ctx context.Context) error
}

//go:generate mockery --name SagaResultPublisher --output ../../mocks --case underscore --outpkg mocks
type SagaResultPublisher interface {
	Publish(ctx context.Context, event model.ConfirmationResultEvent) error
}
