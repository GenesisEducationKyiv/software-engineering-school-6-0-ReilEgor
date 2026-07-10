package usecase

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/GenesisEducationKyiv/software-engineering-school-6-0-ReilEgor/shared/ctxlog"

	contracts "github.com/GenesisEducationKyiv/software-engineering-school-6-0-ReilEgor/shared/contracts"

	"github.com/GenesisEducationKyiv/software-engineering-school-6-0-ReilEgor/services/notification/internal/domain/service"
)

const componentNotificationUseCase = "NotificationUseCase"

type NotificationUseCase struct {
	emailService service.EmailService
}

func NewNotificationUseCase(emailService service.EmailService) *NotificationUseCase {
	return &NotificationUseCase{emailService: emailService}
}

func (uc *NotificationUseCase) log(ctx context.Context) *slog.Logger {
	return ctxlog.FromCtx(ctx).With(slog.String("component", componentNotificationUseCase))
}

func (uc *NotificationUseCase) Send(ctx context.Context, cmd contracts.SendNotificationCommand) error {
	const op = "NotificationUseCase.Send"
	log := uc.log(ctx).With(slog.String("op", op), slog.String("email", cmd.Email), slog.String("repo", cmd.RepoName))
	log.DebugContext(ctx, "called")

	if err := uc.emailService.SendNotification(ctx, cmd.Email, cmd.RepoName, cmd.Tag, cmd.Token); err != nil {
		log.ErrorContext(ctx, "failed to send notification", slog.String("error", err.Error()))
		return fmt.Errorf("%s: %w", op, err)
	}

	log.InfoContext(ctx, "notification sent")
	return nil
}
