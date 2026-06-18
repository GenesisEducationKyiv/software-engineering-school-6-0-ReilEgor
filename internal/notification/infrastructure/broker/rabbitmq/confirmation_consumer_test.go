package rabbitmq

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	amqp "github.com/rabbitmq/amqp091-go"

	notificationmocks "github.com/GenesisEducationKyiv/software-engineering-school-6-0-ReilEgor/internal/notification/mocks"
	"github.com/GenesisEducationKyiv/software-engineering-school-6-0-ReilEgor/shared/domain/model"
)

func newTestConfirmationConsumer(t *testing.T) (*ConfirmationConsumer, *notificationmocks.EmailService) {
	t.Helper()
	emailSvc := notificationmocks.NewEmailService(t)
	return NewConfirmationConsumer(nil, emailSvc, time.Second), emailSvc
}

func TestConfirmationConsumer_handle(t *testing.T) {
	cmd := model.SendConfirmationCommand{
		Email:    "alice@example.com",
		RepoName: "golang/go",
		Token:    "tok-b",
	}
	validBody, err := json.Marshal(cmd)
	require.NoError(t, err)

	tests := []struct {
		name       string
		body       []byte
		setupEmail func(svc *notificationmocks.EmailService)
		setupAck   func(ack *mockAck)
	}{
		{
			name: "success — acks delivery",
			body: validBody,
			setupEmail: func(svc *notificationmocks.EmailService) {
				svc.On("SendConfirmation", mock.Anything, cmd.Email, cmd.RepoName, cmd.Token).Return(nil).Once()
			},
			setupAck: func(ack *mockAck) {
				ack.On("Ack", uint64(0), false).Return(nil).Once()
			},
		},
		{
			name: "email service error — nacks with requeue",
			body: validBody,
			setupEmail: func(svc *notificationmocks.EmailService) {
				svc.On("SendConfirmation", mock.Anything, cmd.Email, cmd.RepoName, cmd.Token).
					Return(errors.New("smtp error")).
					Once()
			},
			setupAck: func(ack *mockAck) {
				ack.On("Nack", uint64(0), false, true).Return(nil).Once()
			},
		},
		{
			name:       "invalid JSON — nacks without requeue",
			body:       []byte("not-json"),
			setupEmail: func(_ *notificationmocks.EmailService) {},
			setupAck: func(ack *mockAck) {
				ack.On("Nack", uint64(0), false, false).Return(nil).Once()
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c, emailSvc := newTestConfirmationConsumer(t)
			ack := newMockAck(t)

			tt.setupEmail(emailSvc)
			tt.setupAck(ack)

			d := amqp.Delivery{Acknowledger: ack, Body: tt.body}
			c.handle(context.Background(), d)
		})
	}
}
