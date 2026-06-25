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

	"github.com/GenesisEducationKyiv/software-engineering-school-6-0-ReilEgor/services/notification/domain/service"
	"github.com/GenesisEducationKyiv/software-engineering-school-6-0-ReilEgor/services/notification/mocks"
	"github.com/GenesisEducationKyiv/software-engineering-school-6-0-ReilEgor/shared/domain/model"
)

func newTestConfirmationConsumer(
	t *testing.T,
) (*ConfirmationConsumer, *mocks.EmailService, *mocks.SagaResultPublisher) {
	t.Helper()
	emailSvc := mocks.NewEmailService(t)
	sagaPub := mocks.NewSagaResultPublisher(t)
	return NewConfirmationConsumer(nil, emailSvc, time.Second, sagaPub), emailSvc, sagaPub
}

func TestConfirmationConsumer_handle(t *testing.T) {
	cmd := model.SendConfirmationCommand{
		Email:    "alice@example.com",
		RepoName: "golang/go",
		Token:    "tok-b",
	}
	validBody, err := json.Marshal(cmd)
	require.NoError(t, err)

	failureEvent := mock.MatchedBy(func(e model.ConfirmationResultEvent) bool {
		return e.Success == false
	})
	successEvent := mock.MatchedBy(func(e model.ConfirmationResultEvent) bool {
		return e.Success == true
	})

	tests := []struct {
		name       string
		body       []byte
		setupEmail func(svc *mocks.EmailService)
		setupSaga  func(pub *mocks.SagaResultPublisher)
		setupAck   func(ack *mockAck)
	}{
		{
			name: "success — acks delivery",
			body: validBody,
			setupEmail: func(svc *mocks.EmailService) {
				svc.On("SendConfirmation", mock.Anything, cmd.Email, cmd.RepoName, cmd.Token).Return(nil).Once()
			},
			setupSaga: func(pub *mocks.SagaResultPublisher) {
				pub.On("Publish", mock.Anything, successEvent).Return(nil).Once()
			},
			setupAck: func(ack *mockAck) {
				ack.On("Ack", uint64(0), false).Return(nil).Once()
			},
		},
		{
			name: "transient SMTP error — nacks with requeue, saga not notified",
			body: validBody,
			setupEmail: func(svc *mocks.EmailService) {
				svc.On("SendConfirmation", mock.Anything, cmd.Email, cmd.RepoName, cmd.Token).
					Return(service.ErrSMTPUnavailable).Once()
			},
			setupSaga: func(_ *mocks.SagaResultPublisher) {},
			setupAck: func(ack *mockAck) {
				ack.On("Nack", uint64(0), false, true).Return(nil).Once()
			},
		},
		{
			name: "permanent SMTP error — publishes saga failure, nacks without requeue",
			body: validBody,
			setupEmail: func(svc *mocks.EmailService) {
				svc.On("SendConfirmation", mock.Anything, cmd.Email, cmd.RepoName, cmd.Token).
					Return(service.ErrAuthFailed).Once()
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
			setupEmail: func(svc *mocks.EmailService) {
				svc.On("SendConfirmation", mock.Anything, cmd.Email, cmd.RepoName, cmd.Token).Return(nil).Once()
			},
			setupSaga: func(pub *mocks.SagaResultPublisher) {
				pub.On("Publish", mock.Anything, successEvent).Return(errors.New("broker unavailable")).Once()
			},
			setupAck: func(ack *mockAck) {
				ack.On("Nack", uint64(0), false, true).Return(nil).Once()
			},
		},
		{
			name:       "invalid JSON — nacks without requeue",
			body:       []byte("not-json"),
			setupEmail: func(_ *mocks.EmailService) {},
			setupSaga:  func(_ *mocks.SagaResultPublisher) {},
			setupAck: func(ack *mockAck) {
				ack.On("Nack", uint64(0), false, false).Return(nil).Once()
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c, emailSvc, sagaPub := newTestConfirmationConsumer(t)
			ack := newMockAck(t)

			tt.setupEmail(emailSvc)
			tt.setupSaga(sagaPub)
			tt.setupAck(ack)

			d := amqp.Delivery{Acknowledger: ack, Body: tt.body}
			c.handle(context.Background(), d)
		})
	}
}
