package rabbitmq

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	contracts "github.com/GenesisEducationKyiv/software-engineering-school-6-0-ReilEgor/shared/contracts"
	amqp "github.com/rabbitmq/amqp091-go"

	"github.com/GenesisEducationKyiv/software-engineering-school-6-0-ReilEgor/services/notification/internal/domain/usecase"
	"github.com/GenesisEducationKyiv/software-engineering-school-6-0-ReilEgor/services/notification/internal/mocks"
)

func newTestConfirmationConsumer(
	t *testing.T,
) (*ConfirmationConsumer, *mocks.NotificationUseCase, *mocks.SagaResultPublisher) {
	t.Helper()
	notificationUC := mocks.NewNotificationUseCase(t)
	sagaPub := mocks.NewSagaResultPublisher(t)
	return NewConfirmationConsumer(nil, notificationUC, time.Second, sagaPub), notificationUC, sagaPub
}

func TestConfirmationConsumer_handle(t *testing.T) {
	cmd := contracts.SendConfirmationCommand{
		Email:    "alice@example.com",
		RepoName: "golang/go",
		Token:    "tok-b",
	}
	validBody, err := json.Marshal(cmd)
	require.NoError(t, err)

	failureEvent := mock.MatchedBy(func(e contracts.ConfirmationResultEvent) bool {
		return e.Success == false
	})
	successEvent := mock.MatchedBy(func(e contracts.ConfirmationResultEvent) bool {
		return e.Success == true
	})

	tests := []struct {
		name         string
		body         []byte
		setupUseCase func(uc *mocks.NotificationUseCase)
		setupSaga    func(pub *mocks.SagaResultPublisher)
		setupAck     func(ack *mockAck)
	}{
		{
			name: "success — acks delivery",
			body: validBody,
			setupUseCase: func(uc *mocks.NotificationUseCase) {
				uc.On("SendConfirmation", mock.Anything, cmd).Return(nil).Once()
			},
			setupSaga: func(pub *mocks.SagaResultPublisher) {
				pub.On("Publish", mock.Anything, successEvent).Return(nil).Once()
			},
			setupAck: func(ack *mockAck) {
				ack.On("Ack", uint64(0), false).Return(nil).Once()
			},
		},
		{
			name: "transient email error — nacks with requeue, saga not notified",
			body: validBody,
			setupUseCase: func(uc *mocks.NotificationUseCase) {
				uc.On("SendConfirmation", mock.Anything, cmd).
					Return(usecase.ErrEmailUnavailable).Once()
			},
			setupSaga: func(_ *mocks.SagaResultPublisher) {},
			setupAck: func(ack *mockAck) {
				ack.On("Nack", uint64(0), false, true).Return(nil).Once()
			},
		},
		{
			name: "permanent email error — publishes saga failure, nacks without requeue",
			body: validBody,
			setupUseCase: func(uc *mocks.NotificationUseCase) {
				uc.On("SendConfirmation", mock.Anything, cmd).
					Return(errors.New("permanent failure")).Once()
			},
			setupSaga: func(pub *mocks.SagaResultPublisher) {
				pub.On("Publish", mock.Anything, failureEvent).Return(nil).Once()
			},
			setupAck: func(ack *mockAck) {
				ack.On("Nack", uint64(0), false, false).Return(nil).Once()
			},
		},
		{
			name: "saga result publish fails after email success — nacks with requeue",
			body: validBody,
			setupUseCase: func(uc *mocks.NotificationUseCase) {
				uc.On("SendConfirmation", mock.Anything, cmd).Return(nil).Once()
			},
			setupSaga: func(pub *mocks.SagaResultPublisher) {
				pub.On("Publish", mock.Anything, successEvent).Return(errors.New("broker unavailable")).Once()
			},
			setupAck: func(ack *mockAck) {
				ack.On("Nack", uint64(0), false, true).Return(nil).Once()
			},
		},
		{
			name:         "invalid JSON — nacks without requeue",
			body:         []byte("not-json"),
			setupUseCase: func(_ *mocks.NotificationUseCase) {},
			setupSaga:    func(_ *mocks.SagaResultPublisher) {},
			setupAck: func(ack *mockAck) {
				ack.On("Nack", uint64(0), false, false).Return(nil).Once()
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c, notificationUC, sagaPub := newTestConfirmationConsumer(t)
			ack := newMockAck(t)

			tt.setupUseCase(notificationUC)
			tt.setupSaga(sagaPub)
			tt.setupAck(ack)

			d := amqp.Delivery{Acknowledger: ack, Body: tt.body}
			c.handle(context.Background(), d)
		})
	}
}
