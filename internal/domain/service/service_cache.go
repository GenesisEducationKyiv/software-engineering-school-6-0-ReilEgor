package service

import (
	"context"
	"errors"
	"time"
)

var ErrCacheMiss = errors.New("key not found in cache")

//go:generate mockery --name Cache --output ../../mocks --case underscore --outpkg mocks
type Cache interface {
	Get(ctx context.Context, key string) (string, error)
	Set(ctx context.Context, key, value string, ttl time.Duration) error
}
