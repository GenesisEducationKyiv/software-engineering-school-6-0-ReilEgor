package github

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/GenesisEducationKyiv/software-engineering-school-6-0-ReilEgor/internal/domain/model"
	"github.com/GenesisEducationKyiv/software-engineering-school-6-0-ReilEgor/internal/domain/service"
	"github.com/GenesisEducationKyiv/software-engineering-school-6-0-ReilEgor/internal/mocks"
)

const (
	testRepo         = "owner/repo"
	repoExistsKey    = "repo_exists:" + testRepo
	latestReleaseKey = "release:" + testRepo
	cacheTTL         = time.Minute
)

var (
	ErrClient   = errors.New("github: rate limit exceeded")
	ErrCacheSet = errors.New("redis: write timeout")
)

func TestCachedGitHubClient_RepoExists(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name            string
		setupMock       func(mClient *mocks.GitHubClient, mCache *mocks.Cache)
		wantResult      bool
		wantErr         bool
		wantErrIs       error
		wantErrContains string
	}{
		{
			name: "cache hit: value 'true' returns true",
			setupMock: func(_ *mocks.GitHubClient, mCache *mocks.Cache) {
				mCache.On("Get", mock.Anything, repoExistsKey).
					Return([]byte("true"), nil).Once()
			},
			wantResult: true,
		},
		{
			name: "cache hit: value 'false' returns false",
			setupMock: func(_ *mocks.GitHubClient, mCache *mocks.Cache) {
				mCache.On("Get", mock.Anything, repoExistsKey).
					Return([]byte("false"), nil).Once()
			},
			wantResult: false,
		},
		{
			name: "cache hit: garbage bytes treated as false",
			setupMock: func(_ *mocks.GitHubClient, mCache *mocks.Cache) {
				mCache.On("Get", mock.Anything, repoExistsKey).
					Return([]byte("yes"), nil).Once()
			},
			wantResult: false,
		},
		{
			name: "cache miss: client returns true, stores 'true' in cache",
			setupMock: func(mClient *mocks.GitHubClient, mCache *mocks.Cache) {
				mCache.On("Get", mock.Anything, repoExistsKey).
					Return(nil, service.ErrCacheMiss).Once()
				mClient.On("RepoExists", mock.Anything, testRepo).
					Return(true, nil).Once()
				mCache.On("Set", mock.Anything, repoExistsKey, []byte("true"), cacheTTL).
					Return(nil).Once()
			},
			wantResult: true,
		},
		{
			name: "cache miss: client returns false, stores 'false' in cache",
			setupMock: func(mClient *mocks.GitHubClient, mCache *mocks.Cache) {
				mCache.On("Get", mock.Anything, repoExistsKey).
					Return(nil, service.ErrCacheMiss).Once()
				mClient.On("RepoExists", mock.Anything, testRepo).
					Return(false, nil).Once()
				mCache.On("Set", mock.Anything, repoExistsKey, []byte("false"), cacheTTL).
					Return(nil).Once()
			},
			wantResult: false,
		},
		{
			name: "cache generic error: falls back to client successfully",
			setupMock: func(mClient *mocks.GitHubClient, mCache *mocks.Cache) {
				mCache.On("Get", mock.Anything, repoExistsKey).
					Return(nil, errors.New("redis: connection refused")).Once()
				mClient.On("RepoExists", mock.Anything, testRepo).
					Return(true, nil).Once()
				mCache.On("Set", mock.Anything, repoExistsKey, []byte("true"), cacheTTL).
					Return(nil).Once()
			},
			wantResult: true,
		},
		{
			name: "client error: wrapped and propagated",
			setupMock: func(mClient *mocks.GitHubClient, mCache *mocks.Cache) {
				mCache.On("Get", mock.Anything, repoExistsKey).
					Return(nil, service.ErrCacheMiss).Once()
				mClient.On("RepoExists", mock.Anything, testRepo).
					Return(false, ErrClient).Once()
			},
			wantResult:      false,
			wantErr:         true,
			wantErrIs:       ErrClient,
			wantErrContains: "CachedGitHubClient.RepoExists: client:",
		},
		{
			name: "cache set error: wrapped and propagated after client success",
			setupMock: func(mClient *mocks.GitHubClient, mCache *mocks.Cache) {
				mCache.On("Get", mock.Anything, repoExistsKey).
					Return(nil, service.ErrCacheMiss).Once()
				mClient.On("RepoExists", mock.Anything, testRepo).
					Return(true, nil).Once()
				mCache.On("Set", mock.Anything, repoExistsKey, []byte("true"), cacheTTL).
					Return(ErrCacheSet).Once()
			},
			wantResult:      false,
			wantErr:         true,
			wantErrIs:       ErrCacheSet,
			wantErrContains: "CachedGitHubClient.RepoExists: cache set:",
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			mClient := mocks.NewGitHubClient(t)
			mCache := mocks.NewCache(t)
			tc.setupMock(mClient, mCache)

			c := NewCachedGitHubClient(mClient, mCache)
			result, err := c.RepoExists(context.Background(), testRepo)

			if tc.wantErr {
				require.Error(t, err)
				assert.False(t, result, "result must be zero value on error")

				if tc.wantErrIs != nil {
					require.ErrorIs(t, err, tc.wantErrIs,
						"errors.Is must reach root cause through wrapping")
				}
				if tc.wantErrContains != "" {
					assert.Contains(t, err.Error(), tc.wantErrContains,
						"error message must contain op+context prefix")
				}
			} else {
				require.NoError(t, err)
				assert.Equal(t, tc.wantResult, result)
			}

			mClient.AssertExpectations(t)
			mCache.AssertExpectations(t)
		})
	}
}

func TestCachedGitHubClient_GetLatestRelease(t *testing.T) {
	t.Parallel()

	baseRelease := &model.ReleaseInfo{TagName: "v1.2.3"}
	baseReleaseJSON, err := json.Marshal(baseRelease)
	require.NoError(t, err, "test fixture: marshalling baseRelease must not fail")

	tests := []struct {
		name            string
		setupMock       func(mClient *mocks.GitHubClient, mCache *mocks.Cache)
		wantTag         string
		wantErr         bool
		wantErrIs       error
		wantErrContains string
	}{
		{
			name: "cache hit: valid JSON returns release without client call",
			setupMock: func(_ *mocks.GitHubClient, mCache *mocks.Cache) {
				mCache.On("Get", mock.Anything, latestReleaseKey).
					Return(baseReleaseJSON, nil).Once()
			},
			wantTag: "v1.2.3",
		},
		{
			name: "cache hit: corrupted JSON falls back to client",
			setupMock: func(mClient *mocks.GitHubClient, mCache *mocks.Cache) {
				mCache.On("Get", mock.Anything, latestReleaseKey).
					Return([]byte("{invalid-json}"), nil).Once()
				mClient.On("GetLatestRelease", mock.Anything, testRepo).
					Return(baseRelease, nil).Once()
				mCache.On("Set", mock.Anything, latestReleaseKey,
					mock.MatchedBy(func(b []byte) bool {
						var r model.ReleaseInfo
						return json.Unmarshal(b, &r) == nil && r.TagName == baseRelease.TagName
					}), cacheTTL).
					Return(nil).Once()
			},
			wantTag: "v1.2.3",
		},
		{
			name: "cache hit: empty JSON object returns zero-value release",
			setupMock: func(_ *mocks.GitHubClient, mCache *mocks.Cache) {
				mCache.On("Get", mock.Anything, latestReleaseKey).
					Return([]byte("{}"), nil).Once()
			},
			wantTag: "",
		},
		{
			name: "cache miss: client called and result stored in cache",
			setupMock: func(mClient *mocks.GitHubClient, mCache *mocks.Cache) {
				mCache.On("Get", mock.Anything, latestReleaseKey).
					Return(nil, service.ErrCacheMiss).Once()
				mClient.On("GetLatestRelease", mock.Anything, testRepo).
					Return(baseRelease, nil).Once()
				mCache.On("Set", mock.Anything, latestReleaseKey,
					mock.MatchedBy(func(b []byte) bool {
						var r model.ReleaseInfo
						return json.Unmarshal(b, &r) == nil && r.TagName == baseRelease.TagName
					}), cacheTTL).
					Return(nil).Once()
			},
			wantTag: "v1.2.3",
		},
		{
			name: "cache generic error: falls back to client successfully",
			setupMock: func(mClient *mocks.GitHubClient, mCache *mocks.Cache) {
				mCache.On("Get", mock.Anything, latestReleaseKey).
					Return(nil, errors.New("redis: i/o timeout")).Once()
				mClient.On("GetLatestRelease", mock.Anything, testRepo).
					Return(baseRelease, nil).Once()
				mCache.On("Set", mock.Anything, latestReleaseKey,
					mock.Anything, cacheTTL).
					Return(nil).Once()
			},
			wantTag: "v1.2.3",
		},
		{
			name: "client error: wrapped and propagated",
			setupMock: func(mClient *mocks.GitHubClient, mCache *mocks.Cache) {
				mCache.On("Get", mock.Anything, latestReleaseKey).
					Return(nil, service.ErrCacheMiss).Once()
				mClient.On("GetLatestRelease", mock.Anything, testRepo).
					Return(nil, ErrClient).Once()
			},
			wantErr:         true,
			wantErrIs:       ErrClient,
			wantErrContains: "CachedGitHubClient.GetLatestRelease: client:",
		},
		{
			name: "cache set error: wrapped and propagated after client success",
			setupMock: func(mClient *mocks.GitHubClient, mCache *mocks.Cache) {
				mCache.On("Get", mock.Anything, latestReleaseKey).
					Return(nil, service.ErrCacheMiss).Once()
				mClient.On("GetLatestRelease", mock.Anything, testRepo).
					Return(baseRelease, nil).Once()
				mCache.On("Set", mock.Anything, latestReleaseKey,
					mock.Anything, cacheTTL).
					Return(ErrCacheSet).Once()
			},
			wantErr:         true,
			wantErrIs:       ErrCacheSet,
			wantErrContains: "CachedGitHubClient.GetLatestRelease: cache set:",
		},
		{
			name: "cache set error after corrupted-cache fallback: wrapped and propagated",
			setupMock: func(mClient *mocks.GitHubClient, mCache *mocks.Cache) {
				mCache.On("Get", mock.Anything, latestReleaseKey).
					Return([]byte("!!!"), nil).Once()
				mClient.On("GetLatestRelease", mock.Anything, testRepo).
					Return(baseRelease, nil).Once()
				mCache.On("Set", mock.Anything, latestReleaseKey,
					mock.Anything, cacheTTL).
					Return(ErrCacheSet).Once()
			},
			wantErr:         true,
			wantErrIs:       ErrCacheSet,
			wantErrContains: "CachedGitHubClient.GetLatestRelease: cache set:",
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			mClient := mocks.NewGitHubClient(t)
			mCache := mocks.NewCache(t)
			tc.setupMock(mClient, mCache)

			c := NewCachedGitHubClient(mClient, mCache)
			res, err := c.GetLatestRelease(context.Background(), testRepo)

			if tc.wantErr {
				require.Error(t, err)
				assert.Nil(t, res, "result must be nil on error")

				if tc.wantErrIs != nil {
					require.ErrorIs(t, err, tc.wantErrIs,
						"errors.Is must reach root cause through wrapping")
				}
				if tc.wantErrContains != "" {
					assert.Contains(t, err.Error(), tc.wantErrContains,
						"error message must contain op+context prefix")
				}
			} else {
				require.NoError(t, err)
				require.NotNil(t, res, "result must not be nil on success")
				assert.Equal(t, tc.wantTag, res.TagName)
			}

			mClient.AssertExpectations(t)
			mCache.AssertExpectations(t)
		})
	}
}
