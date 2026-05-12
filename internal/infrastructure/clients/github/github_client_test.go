package github

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/sony/gobreaker"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/GenesisEducationKyiv/software-engineering-school-6-0-ReilEgor/internal/config"
	"github.com/GenesisEducationKyiv/software-engineering-school-6-0-ReilEgor/internal/domain/model"
	"github.com/GenesisEducationKyiv/software-engineering-school-6-0-ReilEgor/internal/domain/service"
)

func newTestClient(t *testing.T, token string, server *httptest.Server) *GitHubClient {
	t.Helper()
	c := NewGitHubClient(config.GitHubTokenType(token))
	c.apiBase = server.URL
	return c
}

func tripBreaker(t *testing.T, client *GitHubClient) {
	t.Helper()
	for i := 0; i < cbFailureThreshold; i++ {
		_, err := client.RepoExists(context.Background(), "trip/repo")
		require.Error(t, err, "iteration %d must return an error to trip the breaker", i)
	}
}

func encodeJSON(w http.ResponseWriter, v any) {
	if err := json.NewEncoder(w).Encode(v); err != nil {
		panic(fmt.Sprintf("test: encode JSON: %v", err))
	}
}

func TestGitHubClient_RepoExists(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name            string
		token           string
		repoName        string
		serverHandler   func(w http.ResponseWriter, r *http.Request)
		useDeadCtx      bool
		wantResult      bool
		wantErr         bool
		wantErrIs       error
		wantErrContains string
	}{
		{
			name:     "success: repo exists (HTTP 200)",
			token:    "test-token",
			repoName: "golang/go",
			serverHandler: func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, http.MethodHead, r.Method)
				assert.Equal(t, "/repos/golang/go", r.URL.Path)
				assert.Equal(t, "application/vnd.github+json", r.Header.Get("Accept"))
				assert.Equal(t, githubAPIVersion, r.Header.Get("X-GitHub-Api-Version"))
				assert.Equal(t, userAgent, r.Header.Get("User-Agent"))
				assert.Equal(t, "Bearer test-token", r.Header.Get("Authorization"))
				w.WriteHeader(http.StatusOK)
			},
			wantResult: true,
		},
		{
			name:     "success: repo not found (HTTP 404)",
			token:    "",
			repoName: "unknown/repo",
			serverHandler: func(w http.ResponseWriter, r *http.Request) {
				assert.Empty(t, r.Header.Get("Authorization"))
				w.WriteHeader(http.StatusNotFound)
			},
			wantResult: false,
		},
		{
			name:     "error: HTTP 403 maps to ErrRateLimitExceeded",
			token:    "",
			repoName: "busy/repo",
			serverHandler: func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(http.StatusForbidden)
			},
			wantErr:   true,
			wantErrIs: service.ErrRateLimitExceeded,
		},
		{
			name:     "error: HTTP 500 maps to ErrUnexpectedStatus",
			token:    "",
			repoName: "server/error",
			serverHandler: func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(http.StatusInternalServerError)
			},
			wantErr:   true,
			wantErrIs: ErrUnexpectedStatus,
		},
		{
			name:     "error: HTTP 503 maps to ErrUnexpectedStatus",
			token:    "",
			repoName: "unavail/repo",
			serverHandler: func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(http.StatusServiceUnavailable)
			},
			wantErr:   true,
			wantErrIs: ErrUnexpectedStatus,
		},
		{
			name:     "error: unexpected status carries status text in message",
			token:    "",
			repoName: "teapot/repo",
			serverHandler: func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(http.StatusTeapot)
			},
			wantErr:         true,
			wantErrIs:       ErrUnexpectedStatus,
			wantErrContains: "418",
		},
		{
			name:       "error: cancelled context propagated as wrapped error",
			token:      "",
			repoName:   "any/repo",
			useDeadCtx: true,
			serverHandler: func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(http.StatusOK)
			},
			wantErr:         true,
			wantErrContains: "GitHubClient.RepoExists",
		},
		{
			name:     "boundary: repo name with special characters preserved in path",
			token:    "",
			repoName: "org/my-repo.v2",
			serverHandler: func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, "/repos/org/my-repo.v2", r.URL.Path)
				w.WriteHeader(http.StatusOK)
			},
			wantResult: true,
		},
		{
			name:     "boundary: empty repo name still issues a request",
			token:    "",
			repoName: "",
			serverHandler: func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, "/repos/", r.URL.Path)
				w.WriteHeader(http.StatusNotFound)
			},
			wantResult: false,
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			server := httptest.NewServer(http.HandlerFunc(tc.serverHandler))
			defer server.Close()

			client := newTestClient(t, tc.token, server)

			ctx := context.Background()
			if tc.useDeadCtx {
				var cancel context.CancelFunc
				ctx, cancel = context.WithCancel(ctx)
				cancel()
			}

			result, err := client.RepoExists(ctx, tc.repoName)

			if tc.wantErr {
				require.Error(t, err)
				assert.False(t, result, "result must be zero value on error")

				if tc.wantErrIs != nil {
					require.ErrorIs(t, err, tc.wantErrIs)
				}
				if tc.wantErrContains != "" {
					assert.Contains(t, err.Error(), tc.wantErrContains)
				}
			} else {
				require.NoError(t, err)
				assert.Equal(t, tc.wantResult, result)
			}
		})
	}
}

func TestGitHubClient_GetLatestRelease(t *testing.T) {
	t.Parallel()

	baseRelease := model.ReleaseInfo{
		TagName:     "v1.2.3",
		PublishedAt: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
	}

	tests := []struct {
		name string

		token    string
		repoName string

		serverHandler func(w http.ResponseWriter, r *http.Request)

		useDeadCtx bool

		wantTag         string
		wantPublishedAt time.Time
		wantErr         bool
		wantErrIs       error
		wantErrContains string
	}{
		{
			name:     "success: full release payload decoded correctly",
			token:    "tok",
			repoName: "owner/repo",
			serverHandler: func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, http.MethodGet, r.Method)
				assert.Equal(t, "/repos/owner/repo/releases/latest", r.URL.Path)
				assert.Equal(t, "Bearer tok", r.Header.Get("Authorization"))
				w.WriteHeader(http.StatusOK)
				encodeJSON(w, baseRelease)
			},
			wantTag:         "v1.2.3",
			wantPublishedAt: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
		},
		{
			name:     "success: no token means no Authorization header",
			token:    "",
			repoName: "owner/repo",
			serverHandler: func(w http.ResponseWriter, r *http.Request) {
				assert.Empty(t, r.Header.Get("Authorization"))
				w.WriteHeader(http.StatusOK)
				encodeJSON(w, baseRelease)
			},
			wantTag: "v1.2.3",
		},
		{
			name:     "error: HTTP 404 maps to ErrReleaseNotFound",
			repoName: "owner/repo",
			serverHandler: func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(http.StatusNotFound)
			},
			wantErr:   true,
			wantErrIs: service.ErrReleaseNotFound,
		},
		{
			name:     "error: HTTP 500 maps to ErrUnexpectedStatus",
			repoName: "owner/repo",
			serverHandler: func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(http.StatusInternalServerError)
			},
			wantErr:   true,
			wantErrIs: ErrUnexpectedStatus,
		},
		{
			name:     "error: HTTP 503 carries status text in message",
			repoName: "owner/repo",
			serverHandler: func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(http.StatusServiceUnavailable)
			},
			wantErr:         true,
			wantErrIs:       ErrUnexpectedStatus,
			wantErrContains: "503",
		},
		{
			name:     "error: malformed JSON body returns decode error",
			repoName: "owner/repo",
			serverHandler: func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(http.StatusOK)
				_, err := w.Write([]byte("{invalid json"))
				if err != nil {
					t.Fatal(err)
				}
			},
			wantErr:         true,
			wantErrContains: "decode response",
		},
		{
			name:     "error: empty body returns decode error",
			repoName: "owner/repo",
			serverHandler: func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(http.StatusOK)
			},
			wantErr:         true,
			wantErrContains: "decode response",
		},
		{
			name:       "error: cancelled context propagated as wrapped error",
			repoName:   "owner/repo",
			useDeadCtx: true,
			serverHandler: func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(http.StatusOK)
			},
			wantErr:         true,
			wantErrContains: "GitHubClient.GetLatestRelease",
		},
		{
			name:     "boundary: repo name embedded verbatim in releases URL",
			repoName: "org/my-app.v3",
			serverHandler: func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, "/repos/org/my-app.v3/releases/latest", r.URL.Path)
				w.WriteHeader(http.StatusOK)
				encodeJSON(w, baseRelease)
			},
			wantTag: "v1.2.3",
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			server := httptest.NewServer(http.HandlerFunc(tc.serverHandler))
			defer server.Close()

			client := newTestClient(t, tc.token, server)

			ctx := context.Background()
			if tc.useDeadCtx {
				var cancel context.CancelFunc
				ctx, cancel = context.WithCancel(ctx)
				cancel()
			}

			res, err := client.GetLatestRelease(ctx, tc.repoName)

			if tc.wantErr {
				require.Error(t, err)
				assert.Nil(t, res, "result must be nil on error")

				if tc.wantErrIs != nil {
					require.ErrorIs(t, err, tc.wantErrIs)
				}
				if tc.wantErrContains != "" {
					assert.Contains(t, err.Error(), tc.wantErrContains)
				}
			} else {
				require.NoError(t, err)
				require.NotNil(t, res, "result must not be nil on success")
				assert.Equal(t, tc.wantTag, res.TagName)
				if !tc.wantPublishedAt.IsZero() {
					assert.Equal(t, tc.wantPublishedAt, res.PublishedAt)
				}
			}
		})
	}
}

func TestGitHubClient_CircuitBreaker(t *testing.T) {
	t.Parallel()
	t.Run("opens after consecutive failures and blocks RepoExists", func(t *testing.T) {
		t.Parallel()

		var callCount atomic.Int32
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			callCount.Add(1)
			w.WriteHeader(http.StatusInternalServerError)
		}))
		defer server.Close()

		client := newTestClient(t, "", server)
		tripBreaker(t, client)

		countBeforeOpenCall := callCount.Load()

		_, err := client.RepoExists(context.Background(), "test/repo")

		require.ErrorIs(t, err, service.ErrGitHubUnavailable,
			"open breaker must return ErrGitHubUnavailable")
		assert.Equal(t, countBeforeOpenCall, callCount.Load(),
			"server must not receive any request when breaker is open")
	})

	t.Run("open breaker blocks GetLatestRelease", func(t *testing.T) {
		t.Parallel()

		var callCount atomic.Int32
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			callCount.Add(1)
			w.WriteHeader(http.StatusInternalServerError)
		}))
		defer server.Close()

		client := newTestClient(t, "", server)
		tripBreaker(t, client)

		countBefore := callCount.Load()

		_, err := client.GetLatestRelease(context.Background(), "owner/repo")

		require.ErrorIs(t, err, service.ErrGitHubUnavailable,
			"open breaker must return ErrGitHubUnavailable for GetLatestRelease too")
		assert.Equal(t, countBefore, callCount.Load(),
			"server must not receive any request when breaker is open")
	})

	t.Run("failures below threshold do not open breaker", func(t *testing.T) {
		t.Parallel()

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
		}))
		defer server.Close()

		client := newTestClient(t, "", server)

		for i := 0; i < cbFailureThreshold-1; i++ {
			_, err := client.RepoExists(context.Background(), "test/repo")
			require.Error(t, err)
			assert.NotErrorIs(t, err, service.ErrGitHubUnavailable,
				"iteration %d: breaker must still be closed", i)
		}
	})

	t.Run("breaker transitions to half-open after timeout and allows one probe", func(t *testing.T) {
		t.Parallel()

		var callCount atomic.Int32
		failMode := atomic.Bool{}
		failMode.Store(true)

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			callCount.Add(1)
			if failMode.Load() {
				w.WriteHeader(http.StatusInternalServerError)
			} else {
				w.WriteHeader(http.StatusOK)
			}
		}))
		defer server.Close()

		client := newTestClient(t, "", server)
		client.cb = newTestCircuitBreaker(50 * time.Millisecond)

		for i := 0; i < cbFailureThreshold; i++ {
			_, err := client.RepoExists(context.Background(), "trip/repo")
			require.Error(t, err, "iteration %d: trip request must fail to increment failure counter", i)
			require.NotErrorIs(t, err, service.ErrGitHubUnavailable,
				"iteration %d: breaker must not be open yet during trip phase", i)
		}

		_, err := client.RepoExists(context.Background(), "trip/repo")
		require.ErrorIs(t, err, service.ErrGitHubUnavailable, "breaker must be open")

		time.Sleep(100 * time.Millisecond)

		failMode.Store(false)
		countBefore := callCount.Load()

		_, err = client.RepoExists(context.Background(), "trip/repo")
		require.NoError(t, err, "probe request in half-open state must succeed")
		assert.Equal(t, countBefore+1, callCount.Load(),
			"exactly one probe must reach the server during half-open")
	})
}

func newTestCircuitBreaker(timeout time.Duration) *gobreaker.CircuitBreaker {
	settings := gobreaker.Settings{
		Name:        cbName,
		MaxRequests: cbMaxRequests,
		Interval:    cbInterval,
		Timeout:     timeout,
		ReadyToTrip: func(counts gobreaker.Counts) bool {
			return counts.ConsecutiveFailures >= cbFailureThreshold
		},
	}
	return gobreaker.NewCircuitBreaker(settings)
}
