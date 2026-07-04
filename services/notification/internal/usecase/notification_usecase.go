package usecase

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"github.com/GenesisEducationKyiv/software-engineering-school-6-0-ReilEgor/shared/ctxlog"

	contracts "github.com/GenesisEducationKyiv/software-engineering-school-6-0-ReilEgor/shared/contracts"

	"github.com/GenesisEducationKyiv/software-engineering-school-6-0-ReilEgor/services/notification/internal/domain/service"
	domainUsecase "github.com/GenesisEducationKyiv/software-engineering-school-6-0-ReilEgor/services/notification/internal/domain/usecase"
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

func (uc *NotificationUseCase) SendConfirmation(ctx context.Context, cmd contracts.SendConfirmationCommand) error {
	const op = "NotificationUseCase.SendConfirmation"
	log := uc.log(ctx).With(slog.String("op", op), slog.String("email", cmd.Email), slog.String("repo", cmd.RepoName))
	log.DebugContext(ctx, "called")

	if err := uc.emailService.SendConfirmation(ctx, cmd.Email, cmd.RepoName, cmd.Token); err != nil {
		if errors.Is(err, service.ErrSMTPUnavailable) {
			log.ErrorContext(ctx, "email service unavailable", slog.String("error", err.Error()))
			return fmt.Errorf("%s: %w", op, domainUsecase.ErrEmailUnavailable)
		}
		log.ErrorContext(ctx, "failed to send confirmation", slog.String("error", err.Error()))
		return fmt.Errorf("%s: %w", op, err)
	}

	log.InfoContext(ctx, "confirmation sent")
	return nil
}
