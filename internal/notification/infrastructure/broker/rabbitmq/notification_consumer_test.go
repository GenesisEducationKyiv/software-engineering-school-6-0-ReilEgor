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

func newTestConsumer(t *testing.T) (*NotificationConsumer, *notificationmocks.NotificationUseCase) {
	t.Helper()
	uc := notificationmocks.NewNotificationUseCase(t)
	return NewNotificationConsumer(nil, uc, time.Second), uc
}

func TestConsumer_handle(t *testing.T) {
	cmd := model.SendNotificationCommand{
		Email:    "alice@example.com",
		RepoName: "golang/go",
		Tag:      "v1.22.0",
		Token:    "tok-a",
	}
	validBody, err := json.Marshal(cmd)
	require.NoError(t, err)

	tests := []struct {
		name     string
		body     []byte
		setupUC  func(uc *notificationmocks.NotificationUseCase)
		setupAck func(ack *mockAck)
	}{
		{
			name: "success — acks delivery",
			body: validBody,
			setupUC: func(uc *notificationmocks.NotificationUseCase) {
				uc.On("Send", mock.Anything, cmd).Return(nil).Once()
			},
			setupAck: func(ack *mockAck) {
				ack.On("Ack", uint64(0), false).Return(nil).Once()
			},
		},
		{
			name: "use case error — nacks with requeue",
			body: validBody,
			setupUC: func(uc *notificationmocks.NotificationUseCase) {
				uc.On("Send", mock.Anything, cmd).Return(errors.New("smtp error")).Once()
			},
			setupAck: func(ack *mockAck) {
				ack.On("Nack", uint64(0), false, true).Return(nil).Once()
			},
		},
		{
			name:    "invalid JSON — nacks without requeue",
			body:    []byte("not-json"),
			setupUC: func(_ *notificationmocks.NotificationUseCase) {},
			setupAck: func(ack *mockAck) {
				ack.On("Nack", uint64(0), false, false).Return(nil).Once()
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c, uc := newTestConsumer(t)
			ack := newMockAck(t)

			tt.setupUC(uc)
			tt.setupAck(ack)

			d := amqp.Delivery{Acknowledger: ack, Body: tt.body}
			c.handle(context.Background(), d)
		})
	}
}
