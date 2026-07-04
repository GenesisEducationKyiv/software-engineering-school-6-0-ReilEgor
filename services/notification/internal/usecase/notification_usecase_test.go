package usecase

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"

	contracts "github.com/GenesisEducationKyiv/software-engineering-school-6-0-ReilEgor/shared/contracts"

	"github.com/GenesisEducationKyiv/software-engineering-school-6-0-ReilEgor/services/notification/internal/domain/service"
	domainUsecase "github.com/GenesisEducationKyiv/software-engineering-school-6-0-ReilEgor/services/notification/internal/domain/usecase"
	"github.com/GenesisEducationKyiv/software-engineering-school-6-0-ReilEgor/services/notification/internal/mocks"
)

func newNotificationUseCase(t *testing.T) (*NotificationUseCase, *mocks.EmailService) {
	t.Helper()
	emailService := mocks.NewEmailService(t)
	return NewNotificationUseCase(emailService), emailService
}

func TestNotificationUseCase_Send(t *testing.T) {
	cmd := contracts.SendNotificationCommand{
		Email:    "alice@example.com",
		RepoName: "golang/go",
		Tag:      "v1.22.0",
		Token:    "tok-a",
	}

	tests := []struct {
		name      string
		mockErr   error
		expectErr bool
	}{
		{
			name:      "success - email sent",
			mockErr:   nil,
			expectErr: false,
		},
		{
			name:      "error - smtp fails, error wrapped and returned",
			mockErr:   errors.New("smtp error"),
			expectErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			notificationUseCase, emailService := newNotificationUseCase(t)
			emailService.On("SendNotification", context.Background(), cmd.Email, cmd.RepoName, cmd.Tag, cmd.Token).
				Return(tt.mockErr).Once()

			err := notificationUseCase.Send(context.Background(), cmd)

			if tt.expectErr {
				require.Error(t, err)
				require.ErrorIs(t, err, tt.mockErr)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestNotificationUseCase_SendConfirmation(t *testing.T) {
	cmd := contracts.SendConfirmationCommand{
		Email:    "alice@example.com",
		RepoName: "golang/go",
		Token:    "tok-b",
	}

	tests := []struct {
		name      string
		mockErr   error
		expectErr error
	}{
		{
			name:      "success - confirmation sent",
			mockErr:   nil,
			expectErr: nil,
		},
		{
			name:      "smtp unavailable - translated to ErrEmailUnavailable for transport to retry",
			mockErr:   service.ErrSMTPUnavailable,
			expectErr: domainUsecase.ErrEmailUnavailable,
		},
		{
			name:      "other email error - returned as-is",
			mockErr:   service.ErrAuthFailed,
			expectErr: service.ErrAuthFailed,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			notificationUseCase, emailService := newNotificationUseCase(t)
			emailService.On("SendConfirmation", context.Background(), cmd.Email, cmd.RepoName, cmd.Token).
				Return(tt.mockErr).Once()

			err := notificationUseCase.SendConfirmation(context.Background(), cmd)

			if tt.expectErr != nil {
				require.Error(t, err)
				require.ErrorIs(t, err, tt.expectErr)
			} else {
				require.NoError(t, err)
			}
		})
	}
}
