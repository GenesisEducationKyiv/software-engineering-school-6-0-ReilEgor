package model

import "time"

type Repository struct {
	ID          int64
	FullName    string
	LastSeenTag string
	UpdatedAt   time.Time
}
