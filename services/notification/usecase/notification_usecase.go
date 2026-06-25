package usecase

import (
	"context"
	"fmt"

	"github.com/GenesisEducationKyiv/software-engineering-school-6-0-ReilEgor/services/notification/domain/service"
	"github.com/GenesisEducationKyiv/software-engineering-school-6-0-ReilEgor/shared/domain/model"
)

type NotificationUseCase struct {
	emailService service.EmailService
}

func NewNotificationUseCase(emailService service.EmailService) *NotificationUseCase {
	return &NotificationUseCase{emailService: emailService}
}

func (uc *NotificationUseCase) Send(ctx context.Context, cmd model.SendNotificationCommand) error {
	if err := uc.emailService.SendNotification(ctx, cmd.Email, cmd.RepoName, cmd.Tag, cmd.Token); err != nil {
		return fmt.Errorf("notification: send email to %s: %w", cmd.Email, err)
	}
	return nil
}
