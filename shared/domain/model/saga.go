package model

import "time"

type SagaStatus string

const (
	SagaStatusStarted     SagaStatus = "STARTED"
	SagaStatusCompleted   SagaStatus = "COMPLETED"
	SagaStatusCompensated SagaStatus = "COMPENSATED"
)

type SubscriptionSaga struct {
	ID             int64
	SubscriptionID int64
	Status         SagaStatus
	CreatedAt      time.Time
	UpdatedAt      time.Time
}
