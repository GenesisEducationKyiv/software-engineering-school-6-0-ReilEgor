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
