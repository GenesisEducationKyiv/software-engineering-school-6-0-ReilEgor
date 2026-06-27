package usecase

import "context"

//go:generate mockery --name NotificationUseCase --output ../../mocks --case underscore --outpkg mocks
type NotificationUseCase interface {
	ProcessNotifications(ctx context.Context) error
}
