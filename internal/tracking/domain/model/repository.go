package model

import (
	"errors"
	"time"
)

var ErrRepositoryNotFound = errors.New("repository not found")

type Repository struct {
	ID          int64
	FullName    string
	LastSeenTag string
	UpdatedAt   time.Time
}
