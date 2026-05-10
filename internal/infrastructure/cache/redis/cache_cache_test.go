package redis

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/go-redis/redismock/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/GenesisEducationKyiv/software-engineering-school-6-0-ReilEgor/internal/domain/service"
)

func cancelledCtx() context.Context {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	return ctx
}

func TestCache_Get(t *testing.T) {
	t.Parallel()

	someRedisErr := errors.New("connection refused")

	tests := []struct {
		name string
		ctx  context.Context
		key  string

		mockSetup func(m redismock.ClientMock)

		wantVal []byte

		wantErr     error
		wantErrIs   error
		wantErrWrap string
	}{
		{
			name: "success: ascii value returned verbatim",
			ctx:  context.Background(),
			key:  "rate:USD",
			mockSetup: func(m redismock.ClientMock) {
				m.ExpectGet("rate:USD").SetVal("42.10")
			},
			wantVal: []byte("42.10"),
		},
		{
			name: "success: binary / JSON value returned intact",
			ctx:  context.Background(),
			key:  "obj:1",
			mockSetup: func(m redismock.ClientMock) {
				m.ExpectGet("obj:1").SetVal(`{"price":1.23}`)
			},
			wantVal: []byte(`{"price":1.23}`),
		},
		{
			name: "success: empty-string value is not treated as a miss",
			ctx:  context.Background(),
			key:  "empty-val",
			mockSetup: func(m redismock.ClientMock) {
				m.ExpectGet("empty-val").SetVal("")
			},
			wantVal: nil,
		},
		{
			name: "miss: redis.Nil mapped to service.ErrCacheMiss",
			ctx:  context.Background(),
			key:  "missing-key",
			mockSetup: func(m redismock.ClientMock) {
				m.ExpectGet("missing-key").RedisNil()
			},
			wantErr:   service.ErrCacheMiss,
			wantErrIs: service.ErrCacheMiss,
		},
		{
			name: "error: redis internal error is wrapped and propagated",
			ctx:  context.Background(),
			key:  "error-key",
			mockSetup: func(m redismock.ClientMock) {
				m.ExpectGet("error-key").SetErr(someRedisErr)
			},
			wantErr:     someRedisErr,
			wantErrWrap: someRedisErr.Error(),
		},
		{
			name: "error: wrapped error still satisfies errors.Is on original",
			ctx:  context.Background(),
			key:  "sentinel-key",
			mockSetup: func(m redismock.ClientMock) {
				m.ExpectGet("sentinel-key").SetErr(someRedisErr)
			},
			wantErr:   someRedisErr,
			wantErrIs: someRedisErr,
		},
		{
			name: "boundary: empty key — redis nil maps to miss",
			ctx:  context.Background(),
			key:  "",
			mockSetup: func(m redismock.ClientMock) {
				m.ExpectGet("").RedisNil()
			},
			wantErr:   service.ErrCacheMiss,
			wantErrIs: service.ErrCacheMiss,
		},
		{
			name: "boundary: very long key — success",
			ctx:  context.Background(),
			key:  fmt.Sprintf("%0512d", 0),
			mockSetup: func(m redismock.ClientMock) {
				longKey := fmt.Sprintf("%0512d", 0)
				m.ExpectGet(longKey).SetVal("v")
			},
			wantVal: []byte("v"),
		},
		{
			name: "error: cancelled context propagated",
			ctx:  cancelledCtx(),
			key:  "ctx-key",
			mockSetup: func(m redismock.ClientMock) {
				m.ExpectGet("ctx-key").SetErr(context.Canceled)
			},
			wantErr:     context.Canceled,
			wantErrWrap: context.Canceled.Error(),
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			db, mock := redismock.NewClientMock()
			tc.mockSetup(mock)

			cache := NewCache(db)
			got, err := cache.Get(tc.ctx, tc.key)

			if tc.wantErr != nil {
				require.Error(t, err)

				if tc.wantErrIs != nil {
					require.ErrorIs(t, err, tc.wantErrIs,
						"expected errors.Is(err, %v) to hold", tc.wantErrIs)
				}
				if tc.wantErrWrap != "" {
					assert.Contains(t, err.Error(), tc.wantErrWrap,
						"error message should contain the original cause")
				}

				assert.Nil(t, got, "no value should be returned alongside an error")
			} else {
				require.NoError(t, err)
				assert.Equal(t, tc.wantVal, got,
					"returned bytes must match what Redis stored")
			}
			require.NoError(t, mock.ExpectationsWereMet(),
				"not all Redis mock expectations were satisfied")
		})
	}
}

func TestCache_Set(t *testing.T) {
	t.Parallel()

	const regularKey = "rate:USD"

	regularVal := []byte("42.10")
	regularTTL := time.Minute
	redisErr := errors.New("redis out of memory")

	tests := []struct {
		name string
		ctx  context.Context
		key  string
		val  []byte
		ttl  time.Duration

		mockSetup func(m redismock.ClientMock)

		wantErr     bool
		wantErrIs   error
		wantErrWrap string
	}{
		{
			name: "success: typical key/value/ttl",
			ctx:  context.Background(),
			key:  regularKey,
			val:  regularVal,
			ttl:  regularTTL,
			mockSetup: func(m redismock.ClientMock) {
				m.ExpectSet(regularKey, regularVal, regularTTL).SetVal("OK")
			},
		},
		{
			name: "success: zero TTL (no expiry)",
			ctx:  context.Background(),
			key:  "persistent-key",
			val:  []byte("v"),
			ttl:  0,
			mockSetup: func(m redismock.ClientMock) {
				m.ExpectSet("persistent-key", []byte("v"), 0).SetVal("OK")
			},
		},
		{
			name: "success: nil value slice",
			ctx:  context.Background(),
			key:  "nil-val",
			val:  nil,
			ttl:  regularTTL,
			mockSetup: func(m redismock.ClientMock) {
				m.ExpectSet("nil-val", []byte(nil), regularTTL).SetVal("OK")
			},
		},
		{
			name: "success: empty value slice",
			ctx:  context.Background(),
			key:  "empty-val",
			val:  []byte{},
			ttl:  regularTTL,
			mockSetup: func(m redismock.ClientMock) {
				m.ExpectSet("empty-val", []byte{}, regularTTL).SetVal("OK")
			},
		},
		{
			name: "error: redis failure is wrapped with op prefix",
			ctx:  context.Background(),
			key:  regularKey,
			val:  regularVal,
			ttl:  regularTTL,
			mockSetup: func(m redismock.ClientMock) {
				m.ExpectSet(regularKey, regularVal, regularTTL).SetErr(redisErr)
			},
			wantErr:     true,
			wantErrIs:   redisErr,
			wantErrWrap: "Cache.Set",
		},
		{
			name: "error: original cause message present in wrapped error",
			ctx:  context.Background(),
			key:  regularKey,
			val:  regularVal,
			ttl:  regularTTL,
			mockSetup: func(m redismock.ClientMock) {
				m.ExpectSet(regularKey, regularVal, regularTTL).SetErr(redisErr)
			},
			wantErr:     true,
			wantErrWrap: redisErr.Error(),
		},
		{
			name: "boundary: empty key — success",
			ctx:  context.Background(),
			key:  "",
			val:  regularVal,
			ttl:  regularTTL,
			mockSetup: func(m redismock.ClientMock) {
				m.ExpectSet("", regularVal, regularTTL).SetVal("OK")
			},
		},
		{
			name: "boundary: very large TTL",
			ctx:  context.Background(),
			key:  regularKey,
			val:  regularVal,
			ttl:  365 * 24 * time.Hour,
			mockSetup: func(m redismock.ClientMock) {
				m.ExpectSet(regularKey, regularVal, 365*24*time.Hour).SetVal("OK")
			},
		},
		{
			name: "error: cancelled context propagated",
			ctx:  cancelledCtx(),
			key:  regularKey,
			val:  regularVal,
			ttl:  regularTTL,
			mockSetup: func(m redismock.ClientMock) {
				m.ExpectSet(regularKey, regularVal, regularTTL).SetErr(context.Canceled)
			},
			wantErr:     true,
			wantErrIs:   context.Canceled,
			wantErrWrap: "Cache.Set",
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			db, mock := redismock.NewClientMock()
			tc.mockSetup(mock)

			cache := NewCache(db)
			err := cache.Set(tc.ctx, tc.key, tc.val, tc.ttl)

			if tc.wantErr {
				require.Error(t, err)

				if tc.wantErrIs != nil {
					require.ErrorIs(t, err, tc.wantErrIs,
						"errors.Is must reach the root cause through wrapping")
				}
				if tc.wantErrWrap != "" {
					assert.Contains(t, err.Error(), tc.wantErrWrap,
						"wrapped error message must contain expected substring")
				}
			} else {
				require.NoError(t, err)
			}

			require.NoError(t, mock.ExpectationsWereMet(),
				"not all Redis mock expectations were satisfied")
		})
	}
}
