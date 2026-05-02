package model

import (
	"errors"
	"time"
)

var ErrInvalidToken = errors.New("invalid or expired token")

type Subscription struct {
	ID             int64
	UserID         int64
	LastSeenTag    string
	CreatedAt      time.Time
	RepositoryID   int64
	RepositoryName string
	Confirmed      bool `json:"confirmed"`
}
type Subscriber struct {
	Email string
	Token string
}
