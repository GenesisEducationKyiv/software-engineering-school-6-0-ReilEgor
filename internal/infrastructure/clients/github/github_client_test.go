package github

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/ReilEgor/RepoNotifier/internal/domain/model"
	"github.com/ReilEgor/RepoNotifier/internal/domain/service"
	"github.com/ReilEgor/RepoNotifier/internal/mocks"
)

func TestGitHubClient_RepoExists(t *testing.T) {
	tests := []struct {
		name           string
		repoName       string
		mockCache      func(m *mocks.Cache)
		serverHandler  func(w http.ResponseWriter, r *http.Request)
		expectedResult bool
		expectedErr    error
	}{
		{
			name:     "success: exists (cache miss, api hit)",
			repoName: "golang/go",
			mockCache: func(m *mocks.Cache) {
				m.On("Get", mock.Anything, cacheKeyRepoExists+"golang/go").
					Return("", service.ErrCacheMiss).Once()
				m.On("Set", mock.Anything, cacheKeyRepoExists+"golang/go", "true", cacheTTL).
					Return(nil).Once()
			},
			serverHandler: func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, http.MethodHead, r.Method)
				w.WriteHeader(http.StatusOK)
			},
			expectedResult: true,
			expectedErr:    nil,
		},
		{
			name:     "success: found in cache (no api call)",
			repoName: "golang/go",
			mockCache: func(m *mocks.Cache) {
				m.On("Get", mock.Anything, cacheKeyRepoExists+"golang/go").
					Return("true", nil).Once()
			},
			serverHandler: func(w http.ResponseWriter, r *http.Request) {
				t.Fatal("API should not be called when data is in cache")
			},
			expectedResult: true,
			expectedErr:    nil,
		},
		{
			name:     "success: repo not found",
			repoName: "unknown/repo",
			mockCache: func(m *mocks.Cache) {
				m.On("Get", mock.Anything, cacheKeyRepoExists+"unknown/repo").
					Return("", service.ErrCacheMiss).Once()
				m.On("Set", mock.Anything, cacheKeyRepoExists+"unknown/repo", "false", cacheTTL).
					Return(nil).Once()
			},
			serverHandler: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusNotFound)
			},
			expectedResult: false,
			expectedErr:    nil,
		},
		{
			name:     "error: rate limit exceeded",
			repoName: "busy/repo",
			mockCache: func(m *mocks.Cache) {
				m.On("Get", mock.Anything, cacheKeyRepoExists+"busy/repo").
					Return("", service.ErrCacheMiss).Once()
			},
			serverHandler: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusForbidden)
			},
			expectedResult: false,
			expectedErr:    service.ErrRateLimitExceeded,
		},
		{
			name:     "success: api hit, cache set fails — result still returned",
			repoName: "golang/go",
			mockCache: func(m *mocks.Cache) {
				m.On("Get", mock.Anything, mock.Anything).Return("", service.ErrCacheMiss).Once()
				m.On("Set", mock.Anything, mock.Anything, mock.Anything, mock.Anything).
					Return(errors.New("redis unavailable")).Once()
			},
			serverHandler: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
			},
			expectedResult: true,
			expectedErr:    nil,
		},
		{
			name:     "error: invalid data in cache (fallback to api)",
			repoName: "golang/go",
			mockCache: func(m *mocks.Cache) {
				m.On("Get", mock.Anything, cacheKeyRepoExists+"golang/go").
					Return("some-garbage-data", nil).Once()
				m.On("Set", mock.Anything, cacheKeyRepoExists+"golang/go", "true", cacheTTL).
					Return(nil).Once()
			},
			serverHandler: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
			},
			expectedResult: true,
			expectedErr:    nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(tt.serverHandler))
			defer server.Close()

			cacheMock := mocks.NewCache(t)
			tt.mockCache(cacheMock)

			client := NewGitHubClient(cacheMock, "")
			client.apiBase = server.URL

			result, err := client.RepoExists(context.Background(), tt.repoName)

			if tt.expectedErr != nil {
				assert.Error(t, err)
				assert.ErrorIs(t, err, tt.expectedErr)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expectedResult, result)
			}

			cacheMock.AssertExpectations(t)
		})
	}
}

func TestGitHubClient_GetLatestRelease(t *testing.T) {
	tests := []struct {
		name           string
		repoName       string
		mockCache      func(m *mocks.Cache)
		serverHandler  func(w http.ResponseWriter, r *http.Request)
		expectedResult *model.ReleaseInfo
		expectedErr    error
	}{
		{
			name:     "success: fetched from api and cached",
			repoName: "golang/go",
			mockCache: func(m *mocks.Cache) {
				cacheKey := cacheKeyLatestRelease + "golang/go"
				m.On("Get", mock.Anything, cacheKey).Return("", service.ErrCacheMiss).Once()
				m.On("Set", mock.Anything, cacheKey, mock.Anything, cacheTTL).Return(nil).Once()
			},
			serverHandler: func(w http.ResponseWriter, r *http.Request) {
				resp := model.ReleaseInfo{
					TagName:     "v1.22.0",
					PublishedAt: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
				}
				w.WriteHeader(http.StatusOK)
				if err := json.NewEncoder(w).Encode(resp); err != nil {
					t.Errorf("failed to encode response: %v", err)
				}
			},
			expectedResult: &model.ReleaseInfo{
				TagName:     "v1.22.0",
				PublishedAt: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
			},
		},
		{
			name:     "success: found in cache",
			repoName: "golang/go",
			mockCache: func(m *mocks.Cache) {
				cacheKey := cacheKeyLatestRelease + "golang/go"
				cachedData := `{"tag_name":"v1.21.0","published_at":"2023-01-01T00:00:00Z"}`
				m.On("Get", mock.Anything, cacheKey).Return(cachedData, nil).Once()
			},
			serverHandler: func(w http.ResponseWriter, r *http.Request) {
				t.Fatal("API should not be called")
			},
			expectedResult: &model.ReleaseInfo{
				TagName:     "v1.21.0",
				PublishedAt: time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC),
			},
		},
		{
			name:     "error: release not found (404)",
			repoName: "no-releases/repo",
			mockCache: func(m *mocks.Cache) {
				m.On("Get", mock.Anything, cacheKeyLatestRelease+"no-releases/repo").
					Return("", service.ErrCacheMiss).Once()
			},
			serverHandler: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusNotFound)
			},
			expectedErr: service.ErrReleaseNotFound,
		},
		{
			name:     "error: unexpected status with message",
			repoName: "error/repo",
			mockCache: func(m *mocks.Cache) {
				m.On("Get", mock.Anything, cacheKeyLatestRelease+"error/repo").
					Return("", service.ErrCacheMiss).Once()
			},
			serverHandler: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusInternalServerError)
				_, _ = w.Write([]byte(`{"message": "something went wrong on github"}`))
			},
			expectedErr: ErrUnexpectedStatus,
		},
		{
			name:     "error: invalid json in cache (should fallback to api)",
			repoName: "golang/go",
			mockCache: func(m *mocks.Cache) {
				cacheKey := cacheKeyLatestRelease + "golang/go"
				m.On("Get", mock.Anything, cacheKey).Return("{invalid-json}", nil).Once()
				m.On("Set", mock.Anything, cacheKey, mock.Anything, cacheTTL).Return(nil).Once()
			},
			serverHandler: func(w http.ResponseWriter, r *http.Request) {
				resp := model.ReleaseInfo{TagName: "v1.22.0"}
				w.WriteHeader(http.StatusOK)
				_ = json.NewEncoder(w).Encode(resp)
			},
			expectedResult: &model.ReleaseInfo{TagName: "v1.22.0"},
			expectedErr:    nil,
		},
		{
			name:     "success: api hit, cache set fails — release still returned",
			repoName: "golang/go",
			mockCache: func(m *mocks.Cache) {
				cacheKey := cacheKeyLatestRelease + "golang/go"
				m.On("Get", mock.Anything, cacheKey).Return("", service.ErrCacheMiss).Once()
				m.On("Set", mock.Anything, cacheKey, mock.Anything, cacheTTL).
					Return(errors.New("redis connection reset")).Once()
			},
			serverHandler: func(w http.ResponseWriter, r *http.Request) {
				resp := model.ReleaseInfo{
					TagName:     "v1.22.0",
					PublishedAt: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
				}
				w.WriteHeader(http.StatusOK)

				_ = json.NewEncoder(w).Encode(resp)
			},
			expectedResult: &model.ReleaseInfo{
				TagName:     "v1.22.0",
				PublishedAt: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
			},
			expectedErr: nil,
		},
		{
			name:     "error: rate limit exceeded (403)",
			repoName: "golang/go",
			mockCache: func(m *mocks.Cache) {
				m.On("Get", mock.Anything, cacheKeyLatestRelease+"golang/go").
					Return("", service.ErrCacheMiss).Once()
			},
			serverHandler: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusForbidden)
			},
			expectedErr: service.ErrRateLimitExceeded,
		},
		{
			name:     "error: unexpected status without body (502)",
			repoName: "golang/go",
			mockCache: func(m *mocks.Cache) {
				m.On("Get", mock.Anything, cacheKeyLatestRelease+"golang/go").
					Return("", service.ErrCacheMiss).Once()
			},
			serverHandler: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusBadGateway)
			},
			expectedErr: ErrUnexpectedStatus,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(tt.serverHandler))
			defer server.Close()

			cacheMock := mocks.NewCache(t)
			tt.mockCache(cacheMock)

			client := NewGitHubClient(cacheMock, "")
			client.apiBase = server.URL

			result, err := client.GetLatestRelease(context.Background(), tt.repoName)

			if tt.expectedErr != nil {
				assert.Error(t, err)
				assert.ErrorIs(t, err, tt.expectedErr)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expectedResult.TagName, result.TagName)
				assert.True(t, tt.expectedResult.PublishedAt.Equal(result.PublishedAt))
			}
			cacheMock.AssertExpectations(t)
		})
	}
}

func TestGitHubClient_CircuitBreaker_Recovery(t *testing.T) {
	var shouldFail atomic.Bool
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if shouldFail.Load() {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	cacheMock := mocks.NewCache(t)
	cacheMock.On("Get", mock.Anything, mock.Anything).Return("", service.ErrCacheMiss)
	cacheMock.On("Set", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil).Maybe()

	client := NewGitHubClient(cacheMock, "")
	client.apiBase = server.URL

	ctx := context.Background()
	repo := "recovery/repo"

	shouldFail.Store(true)
	for i := 0; i < cbFailureThreshold; i++ {
		_, _ = client.RepoExists(ctx, repo)
	}

	_, err := client.RepoExists(ctx, repo)
	require.ErrorIs(t, err, service.ErrGitHubUnavailable)

	fmt.Println("Waiting for Circuit Breaker timeout...")
	time.Sleep(cbTimeout + 100*time.Millisecond)

	shouldFail.Store(false)

	exists, err := client.RepoExists(ctx, repo)

	assert.NoError(t, err, "CB should allow request in Half-Open state")
	assert.True(t, exists)

	_, err = client.RepoExists(ctx, repo)
	assert.NoError(t, err, "CB should be CLOSED now")
}
