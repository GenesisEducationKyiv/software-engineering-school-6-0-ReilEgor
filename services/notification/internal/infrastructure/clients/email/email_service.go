package email

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/GenesisEducationKyiv/software-engineering-school-6-0-ReilEgor/shared/ctxlog"

	notifModel "github.com/GenesisEducationKyiv/software-engineering-school-6-0-ReilEgor/services/notification/internal/domain/model"
	"github.com/GenesisEducationKyiv/software-engineering-school-6-0-ReilEgor/services/notification/internal/domain/service"
)

const componentEmailService = "EmailService"

type EmailService struct {
	sender  service.EmailSender
	baseURL string
}

func NewEmailService(sender service.EmailSender, baseURL string) *EmailService {
	return &EmailService{
		sender:  sender,
		baseURL: baseURL,
	}
}

func (s *EmailService) log(ctx context.Context) *slog.Logger {
	return ctxlog.FromCtx(ctx).With(slog.String("component", componentEmailService))
}

func (s *EmailService) SendConfirmation(ctx context.Context, to, repoName, token string) error {
	const op = "EmailService.SendConfirmation"
	log := s.log(ctx).With(slog.String("op", op), slog.String("to", to), slog.String("repo", repoName))
	log.DebugContext(ctx, "called")

	msg := notifModel.EmailMessage{
		To:      to,
		Subject: fmt.Sprintf("Confirm subscription to %s", repoName),
		Body: fmt.Sprintf("Hello!\n\nTo confirm your subscription to %s, click here: %s/api/v1/confirm/%s",
			repoName, s.baseURL, token),
	}

	if err := s.sender.Send(ctx, msg); err != nil {
		log.ErrorContext(ctx, "failed to send confirmation email", slog.String("error", err.Error()))
		return fmt.Errorf("%s: %w", op, err)
	}

	log.DebugContext(ctx, "confirmation email sent")
	return nil
}

func (s *EmailService) SendNotification(ctx context.Context, to, repoName, tag, token string) error {
	const op = "EmailService.SendNotification"
	log := s.log(ctx).With(
		slog.String("op", op),
		slog.String("to", to),
		slog.String("repo", repoName),
		slog.String("tag", tag),
	)
	log.DebugContext(ctx, "called")

	msg := notifModel.EmailMessage{
		To:      to,
		Subject: fmt.Sprintf("New release: %s", repoName),
		Body: fmt.Sprintf(
			"Great news!\n\nA new version %s has been released for %s.\n\nUnsubscribe: %s/api/v1/unsubscribe/%s",
			tag,
			repoName,
			s.baseURL,
			token,
		),
	}
	if err := s.sender.Send(ctx, msg); err != nil {
		log.ErrorContext(ctx, "failed to send release notification email", slog.String("error", err.Error()))
		return fmt.Errorf("%s: %w", op, err)
	}

	log.DebugContext(ctx, "release notification email sent")
	return nil
}
