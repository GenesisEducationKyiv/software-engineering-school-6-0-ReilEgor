package model

import (
	"errors"
	"time"
)

var (
	ErrUserNotFound = errors.New("user not found")
)

type User struct {
	ID        int64
	Email     string
	CreatedAt time.Time
}
