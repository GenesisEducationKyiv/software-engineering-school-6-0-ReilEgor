package usecase

import (
	"context"
	"fmt"

	contracts "github.com/GenesisEducationKyiv/software-engineering-school-6-0-ReilEgor/shared/contracts"

	"github.com/GenesisEducationKyiv/software-engineering-school-6-0-ReilEgor/services/notification/internal/domain/service"
)

type NotificationUseCase struct {
	emailService service.EmailService
}

func NewNotificationUseCase(emailService service.EmailService) *NotificationUseCase {
	return &NotificationUseCase{emailService: emailService}
}

func (uc *NotificationUseCase) Send(ctx context.Context, cmd contracts.SendNotificationCommand) error {
	if err := uc.emailService.SendNotification(ctx, cmd.Email, cmd.RepoName, cmd.Tag, cmd.Token); err != nil {
		return fmt.Errorf("NotificationUseCase.Send: %w", err)
	}
	return nil
}
