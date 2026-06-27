package model

import "time"

type SagaStatus string

const (
	SagaStatusStarted     SagaStatus = "STARTED"
	SagaStatusCompleted   SagaStatus = "COMPLETED"
	SagaStatusCompensated SagaStatus = "COMPENSATED"
)

type SagaStep string

const (
	SagaStepSendConfirmation     SagaStep = "SEND_CONFIRMATION"
	SagaStepActivateSubscription SagaStep = "ACTIVATE_SUBSCRIPTION"
)

type SubscriptionSaga struct {
	ID             int64
	SubscriptionID int64
	Status         SagaStatus
	CurrentStep    SagaStep
	CreatedAt      time.Time
	UpdatedAt      time.Time
}
