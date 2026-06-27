package usecase

import (
	"context"

	contracts "github.com/GenesisEducationKyiv/software-engineering-school-6-0-ReilEgor/shared/contracts"
)

//go:generate mockery --name NotificationUseCase --output ../../mocks --case underscore --outpkg mocks
type NotificationUseCase interface {
	Send(ctx context.Context, cmd contracts.SendNotificationCommand) error
}
