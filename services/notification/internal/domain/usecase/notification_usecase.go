package usecase

import (
	"context"
	"errors"

	contracts "github.com/GenesisEducationKyiv/software-engineering-school-6-0-ReilEgor/shared/contracts"
)

// ErrEmailUnavailable signals a transient email-delivery failure that the caller should retry.
var ErrEmailUnavailable = errors.New("email service temporarily unavailable")

//go:generate mockery --name NotificationUseCase --output ../../mocks --case underscore --outpkg mocks
type NotificationUseCase interface {
	Send(ctx context.Context, cmd contracts.SendNotificationCommand) error
	SendConfirmation(ctx context.Context, cmd contracts.SendConfirmationCommand) error
}
