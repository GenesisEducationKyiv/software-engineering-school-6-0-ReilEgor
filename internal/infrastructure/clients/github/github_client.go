package github

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/ReilEgor/RepoNotifier/internal/config"
	"github.com/ReilEgor/RepoNotifier/internal/domain/model"
	"github.com/ReilEgor/RepoNotifier/internal/domain/service"
	"github.com/sony/gobreaker"
)

const (
	// HTTP client
	httpClientTimeout = 10 * time.Second

	// Cache TTL
	cacheTTL = 1 * time.Minute

	// Cache key prefixes
	cacheKeyRepoExists    = "repo_exists:"
	cacheKeyLatestRelease = "release:"
	cacheValTrue          = "true"
	cacheValFalse         = "false"

	// Circuit breaker
	cbName             = "GitHubAPI"
	cbMaxRequests      = 3
	cbInterval         = 5 * time.Second
	cbTimeout          = 30 * time.Second
	cbFailureThreshold = 3

	// GitHub API
	githubAPIBase    = "https://api.github.com"
	githubAPIVersion = "2026-03-10"
	userAgent        = "RepoNotifier/1.0"

	// Component name
	componentGithubClient = "GithubClient"
)

var (
	ErrUnexpectedStatus = errors.New("unexpected github api status")
)

func (c *GitHubClient) getCached(ctx context.Context, log *slog.Logger, key string) (string, bool, error) {
	val, err := c.cache.Get(ctx, key)
	if err != nil {
		if errors.Is(err, service.ErrCacheMiss) {
			log.DebugContext(ctx, "cache miss", slog.String("key", key))
			return "", false, nil
		}
		log.WarnContext(ctx, "cache get failed",
			slog.String("key", key),
			slog.String("error", err.Error()),
		)
		return "", false, err
	}
	if val == "" {
		log.DebugContext(ctx, "cache miss", slog.String("key", key))
		return "", false, nil
	}
	log.DebugContext(ctx, "cache hit", slog.String("key", key))
	return val, true, nil
}

func (c *GitHubClient) handleCBError(ctx context.Context, log *slog.Logger, op string, err error) error {
	if errors.Is(err, gobreaker.ErrOpenState) || errors.Is(err, gobreaker.ErrTooManyRequests) {
		log.WarnContext(ctx, "circuit breaker is open", slog.String("error", err.Error()))
		return service.ErrGitHubUnavailable
	}
	log.ErrorContext(ctx, "github request failed", slog.String("error", err.Error()))
	return fmt.Errorf("%s: %w", op, err)
}

type GitHubClient struct {
	httpClient *http.Client
	cache      service.Cache
	logger     *slog.Logger
	cb         *gobreaker.CircuitBreaker
	apiBase    string
	token      config.GitHubTokenType
}

func NewGitHubClient(cache service.Cache, token config.GitHubTokenType) *GitHubClient {
	settings := gobreaker.Settings{
		Name:        cbName,
		MaxRequests: cbMaxRequests,
		Interval:    cbInterval,
		Timeout:     cbTimeout,
		ReadyToTrip: func(counts gobreaker.Counts) bool {
			return counts.ConsecutiveFailures >= cbFailureThreshold
		},
		OnStateChange: func(name string, from, to gobreaker.State) {
			slog.Warn("circuit breaker state changed",
				slog.String("component", componentGithubClient),
				slog.String("breaker", name),
				slog.String("from", from.String()),
				slog.String("to", to.String()),
			)
		},
	}
	return &GitHubClient{
		httpClient: &http.Client{Timeout: httpClientTimeout},
		cache:      cache,
		logger:     slog.With(slog.String("component", componentGithubClient)),
		cb:         gobreaker.NewCircuitBreaker(settings),
		apiBase:    githubAPIBase,
		token:      token,
	}
}

func (c *GitHubClient) RepoExists(ctx context.Context, fullName string) (bool, error) {
	const op = "GitHubClient.RepoExists"
	log := c.logger.With(slog.String("op", op), slog.String("repo", fullName))
	cacheKey := cacheKeyRepoExists + fullName

	if cached, ok, err := c.getCached(ctx, log, cacheKey); err != nil {
		return false, fmt.Errorf("%s: cache get: %w", op, err)
	} else if ok {
		if cached == cacheValTrue {
			return true, nil
		}
		if cached == cacheValFalse {
			return false, nil
		}
		log.WarnContext(ctx, "invalid cache value, falling back to api", slog.String("val", cached))
	}

	result, err := c.cb.Execute(func() (interface{}, error) {
		url := fmt.Sprintf("%s/repos/%s", c.apiBase, fullName)
		req, err := http.NewRequestWithContext(ctx, http.MethodHead, url, nil)
		if err != nil {
			return false, fmt.Errorf("create request: %w", err)
		}
		c.setDefaultHeaders(req)

		resp, err := c.httpClient.Do(req)
		if err != nil {
			log.ErrorContext(ctx, "request failed",
				slog.String("url", url),
				slog.String("error", err.Error()),
			)
			return false, fmt.Errorf("do request: %w", err)
		}
		defer resp.Body.Close()

		log.DebugContext(ctx, "response received",
			slog.String("url", url),
			slog.Int("status", resp.StatusCode),
		)

		switch resp.StatusCode {
		case http.StatusOK:
			return true, nil
		case http.StatusNotFound:
			return false, nil
		case http.StatusForbidden:
			return false, service.ErrRateLimitExceeded
		default:
			return false, fmt.Errorf("%w: %s", ErrUnexpectedStatus, resp.Status)
		}
	})
	if err != nil {
		return false, c.handleCBError(ctx, log, op, err)
	}

	exists := result.(bool)
	val := "false"
	if exists {
		val = "true"
	}
	if err = c.cache.Set(ctx, cacheKey, val, cacheTTL); err != nil {
		log.ErrorContext(ctx, "cache set failed",
			slog.String("key", cacheKey),
			slog.String("error", err.Error()),
		)
		//return false, fmt.Errorf("%s: cache set: %w", op, err)
	}
	log.InfoContext(ctx, "repo existence checked", slog.Bool("exists", exists))
	return exists, nil
}

func (c *GitHubClient) GetLatestRelease(ctx context.Context, fullName string) (*model.ReleaseInfo, error) {
	const op = "GitHubClient.GetLatestRelease"
	log := c.logger.With(slog.String("op", op), slog.String("repo", fullName))

	cacheKey := cacheKeyLatestRelease + fullName
	if cached, ok, err := c.getCached(ctx, log, cacheKey); err != nil {
		return nil, fmt.Errorf("%s: cache get: %w", op, err)
	} else if ok {
		var info model.ReleaseInfo
		if err := json.Unmarshal([]byte(cached), &info); err != nil {
			log.WarnContext(ctx, "cache unmarshal failed, fetching from api",
				slog.String("key", cacheKey),
				slog.String("error", err.Error()),
			)
		} else {
			return &info, nil
		}
	}

	url := fmt.Sprintf("%s/repos/%s/releases/latest", c.apiBase, fullName)

	result, err := c.cb.Execute(func() (interface{}, error) {
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
		if err != nil {
			return nil, fmt.Errorf("create request: %w", err)
		}
		c.setDefaultHeaders(req)
		log.DebugContext(ctx, "sending request", slog.String("method", http.MethodGet), slog.String("url", url))
		resp, err := c.httpClient.Do(req)
		if err != nil {
			log.ErrorContext(ctx, "request failed", slog.String("url", url), slog.Any("error", err))
			return nil, err
		}
		defer resp.Body.Close()
		log.DebugContext(ctx, "response received", slog.String("url", url), slog.Int("status", resp.StatusCode))
		switch resp.StatusCode {
		case http.StatusNotFound:
			return nil, service.ErrReleaseNotFound
		case http.StatusForbidden:
			return nil, service.ErrRateLimitExceeded
		case http.StatusOK:
		default:
			var apiErr struct {
				Message string `json:"message"`
			}
			if err := json.NewDecoder(resp.Body).Decode(&apiErr); err == nil && apiErr.Message != "" {
				return nil, fmt.Errorf("%w (%d): %s", ErrUnexpectedStatus, resp.StatusCode, apiErr.Message)
			}
			return nil, fmt.Errorf("%w: %s", ErrUnexpectedStatus, resp.Status)
		}
		var info model.ReleaseInfo
		if err := json.NewDecoder(resp.Body).Decode(&info); err != nil {
			log.ErrorContext(ctx, "response decode failed", slog.String("url", url), slog.Any("error", err))
			return nil, fmt.Errorf("decode response: %w", err)
		}
		return &info, nil
	})
	if err != nil {
		return nil, c.handleCBError(ctx, log, op, err)
	}

	info := result.(*model.ReleaseInfo)

	jsonData, err := json.Marshal(info)
	if err != nil {
		log.ErrorContext(ctx, "marshal for cache failed", slog.Any("error", err))
		return nil, fmt.Errorf("marshal release info: %w", err)
	}
	if err = c.cache.Set(ctx, cacheKey, string(jsonData), cacheTTL); err != nil {
		log.ErrorContext(ctx, "cache set failed", slog.String("key", cacheKey), slog.Any("error", err))
	}

	log.InfoContext(ctx, "latest release fetched", slog.String("tag", info.TagName), slog.Time("published_at", info.PublishedAt))
	return info, nil
}

func (c *GitHubClient) setDefaultHeaders(req *http.Request) {
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("X-GitHub-Api-Version", githubAPIVersion)
	req.Header.Set("User-Agent", userAgent)

	if c.token != "" {
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", c.token))
	}
}
