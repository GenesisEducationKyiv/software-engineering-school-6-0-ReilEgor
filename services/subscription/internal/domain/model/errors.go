package model

import "errors"

var (
	ErrRepositoryNotFound = errors.New("repository not found")
	ErrServiceUnavailable = errors.New("external service is temporarily unavailable")
)
