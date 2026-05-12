package service

import (
	"context"
	"errors"
	"time"
)

var ErrCacheMiss = errors.New("key not found in cache")

//go:generate mockery --name Cache --output ../../mocks --case underscore --outpkg mocks
type Cache interface {
	Get(ctx context.Context, key string) ([]byte, error)
	Set(ctx context.Context, key string, value []byte, ttl time.Duration) error
}
