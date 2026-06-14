package usecase

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/GenesisEducationKyiv/software-engineering-school-6-0-ReilEgor/shared/domain/model"
	"github.com/GenesisEducationKyiv/software-engineering-school-6-0-ReilEgor/shared/mocks"
)

func newNotificationUseCase(t *testing.T) (*NotificationUseCase, *mocks.EmailService) {
	t.Helper()
	emailService := mocks.NewEmailService(t)
	return NewNotificationUseCase(emailService), emailService
}

func TestNotificationUseCase_Send(t *testing.T) {
	cmd := model.SendNotificationCommand{
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
