package redis

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/go-redis/redismock/v9"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/GenesisEducationKyiv/software-engineering-school-6-0-ReilEgor/internal/domain/service"
)

func TestCache_Get(t *testing.T) {
	tests := []struct {
		name        string
		key         string
		mockSetup   func(m redismock.ClientMock)
		expectedVal string
		expectedErr error
	}{
		{
			name: "success: value found",
			key:  "test-key",
			mockSetup: func(m redismock.ClientMock) {
				m.ExpectGet("test-key").SetVal("test-value")
			},
			expectedVal: "test-value",
			expectedErr: nil,
		},
		{
			name: "error: cache miss",
			key:  "missing-key",
			mockSetup: func(m redismock.ClientMock) {
				m.ExpectGet("missing-key").RedisNil()
			},
			expectedVal: "",
			expectedErr: service.ErrCacheMiss,
		},
		{
			name: "error: redis internal error",
			key:  "error-key",
			mockSetup: func(m redismock.ClientMock) {
				m.ExpectGet("error-key").SetErr(errors.New("connection refused"))
			},
			expectedVal: "",
			expectedErr: redis.ErrClosed,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db, mock := redismock.NewClientMock()
			tt.mockSetup(mock)

			cache := NewCache(db)
			val, err := cache.Get(context.Background(), tt.key)

			if tt.expectedErr != nil {
				require.Error(t, err)
				if errors.Is(tt.expectedErr, service.ErrCacheMiss) {
					assert.ErrorIs(t, err, service.ErrCacheMiss)
				}
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.expectedVal, val)
			}
			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestCache_Set(t *testing.T) {
	const (
		key = "test-key"
		val = "test-value"
		ttl = time.Minute
	)

	tests := []struct {
		name      string
		mockSetup func(m redismock.ClientMock)
		wantErr   bool
	}{
		{
			name: "success",
			mockSetup: func(m redismock.ClientMock) {
				m.ExpectSet(key, val, ttl).SetVal("OK")
			},
			wantErr: false,
		},
		{
			name: "error: set failed",
			mockSetup: func(m redismock.ClientMock) {
				m.ExpectSet(key, val, ttl).SetErr(errors.New("redis out of memory"))
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db, mock := redismock.NewClientMock()
			tt.mockSetup(mock)

			cache := NewCache(db)
			err := cache.Set(context.Background(), key, val, ttl)

			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}
