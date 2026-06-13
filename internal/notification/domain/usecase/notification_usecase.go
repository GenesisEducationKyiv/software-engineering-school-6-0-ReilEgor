package usecase

import (
	"context"

	"github.com/GenesisEducationKyiv/software-engineering-school-6-0-ReilEgor/internal/shared/domain/model"
)

//go:generate mockery --name NotificationUseCase --output ../../mocks --case underscore --outpkg mocks
type NotificationUseCase interface {
	Send(ctx context.Context, cmd model.SendNotificationCommand) error
}
