package github

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/sony/gobreaker"

	"github.com/GenesisEducationKyiv/software-engineering-school-6-0-ReilEgor/internal/config"
	"github.com/GenesisEducationKyiv/software-engineering-school-6-0-ReilEgor/internal/domain/model"
	"github.com/GenesisEducationKyiv/software-engineering-school-6-0-ReilEgor/internal/domain/service"
)

const (
	// HTTP client.
	httpClientTimeout = 10 * time.Second

	// Cache TTL.
	cacheTTL = 1 * time.Minute

	// Cache key prefixes.
	cacheKeyRepoExists    = "repo_exists:"
	cacheKeyLatestRelease = "release:"
	cacheValTrue          = "true"
	cacheValFalse         = "false"

	// Circuit breaker.
	cbName             = "GitHubAPI"
	cbMaxRequests      = 3
	cbInterval         = 5 * time.Second
	cbTimeout          = 30 * time.Second
	cbFailureThreshold = 3

	// GitHub API.
	githubAPIBase    = "https://api.github.com"
	githubAPIVersion = "2026-03-10"
	userAgent        = "RepoNotifier/1.0"

	// Component name.
	componentGithubClient = "GithubClient"
)

var ErrUnexpectedStatus = errors.New("unexpected github api status")

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
		return "", false, fmt.Errorf("get from cache (key: %s): %w", key, err)
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
		if v, valid := strToBool(cached); valid {
			return v, nil
		}
		log.WarnContext(ctx, "invalid cache value, falling back to api", slog.String("val", cached))
	}

	result, err := c.cb.Execute(func() (any, error) {
		return c.repoExistsRequest(ctx, log, fullName)
	})
	if err != nil {
		return false, c.handleCBError(ctx, log, op, err)
	}

	exists, ok := result.(bool)
	if !ok {
		return false, fmt.Errorf("%s: unexpected type: expected bool, got %T", op, result)
	}
	c.setCached(ctx, log, cacheKey, boolToStr(exists))
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
		if info := c.unmarshalCachedRelease(ctx, log, cacheKey, cached); info != nil {
			return info, nil
		}
	}

	result, err := c.cb.Execute(func() (any, error) {
		return c.latestReleaseRequest(ctx, log, fullName)
	})
	if err != nil {
		return nil, c.handleCBError(ctx, log, op, err)
	}

	info, ok := result.(*model.ReleaseInfo)

	if !ok {
		return nil, fmt.Errorf("%s: unexpected type: got %T, expected model.ReleaseInfo", op, result)
	}

	if err := c.marshalAndCache(ctx, log, cacheKey, info); err != nil {
		return nil, err
	}

	log.InfoContext(ctx, "latest release fetched",
		slog.String("tag", info.TagName),
		slog.Time("published_at", info.PublishedAt),
	)
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

func (c *GitHubClient) setCached(ctx context.Context, log *slog.Logger, key, val string) {
	if err := c.cache.Set(ctx, key, val, cacheTTL); err != nil {
		log.ErrorContext(ctx, "cache set failed",
			slog.String("key", key),
			slog.String("error", err.Error()),
		)
	}
}

func (c *GitHubClient) doRequest(ctx context.Context, method, url string) (*http.Response, error) {
	req, err := http.NewRequestWithContext(ctx, method, url, nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	c.setDefaultHeaders(req)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("do request: %w", err)
	}
	return resp, nil
}

func boolToStr(v bool) string {
	if v {
		return cacheValTrue
	}
	return cacheValFalse
}

func strToBool(s string) (bool, bool) {
	switch s {
	case cacheValTrue:
		return true, true
	case cacheValFalse:
		return false, true
	default:
		return false, false
	}
}

func (c *GitHubClient) repoExistsRequest(ctx context.Context, log *slog.Logger, fullName string) (bool, error) {
	url := fmt.Sprintf("%s/repos/%s", c.apiBase, fullName)

	resp, err := c.doRequest(ctx, http.MethodHead, url)
	if err != nil {
		log.ErrorContext(ctx, "request failed",
			slog.String("url", url),
			slog.String("error", err.Error()),
		)
		return false, err
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			log.ErrorContext(ctx, "failed to close response body", slog.Any("error", err))
		}
	}()

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
}

func (c *GitHubClient) unmarshalCachedRelease(
	ctx context.Context,
	log *slog.Logger,
	key, data string,
) *model.ReleaseInfo {
	var info model.ReleaseInfo
	if err := json.Unmarshal([]byte(data), &info); err != nil {
		log.WarnContext(ctx, "cache unmarshal failed, fetching from api",
			slog.String("key", key),
			slog.String("error", err.Error()),
		)
		return nil
	}
	return &info
}

func (c *GitHubClient) latestReleaseRequest(
	ctx context.Context,
	log *slog.Logger,
	fullName string,
) (*model.ReleaseInfo, error) {
	url := fmt.Sprintf("%s/repos/%s/releases/latest", c.apiBase, fullName)

	resp, err := c.doRequest(ctx, http.MethodGet, url)
	if err != nil {
		log.ErrorContext(ctx, "request failed",
			slog.String("url", url),
			slog.Any("error", err),
		)
		return nil, err
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			log.ErrorContext(ctx, "failed to close response body", slog.Any("error", err))
		}
	}()

	log.DebugContext(ctx, "response received",
		slog.String("url", url),
		slog.Int("status", resp.StatusCode),
	)

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
		log.ErrorContext(ctx, "response decode failed",
			slog.String("url", url),
			slog.Any("error", err),
		)
		return nil, fmt.Errorf("decode response: %w", err)
	}
	return &info, nil
}

func (c *GitHubClient) marshalAndCache(
	ctx context.Context,
	log *slog.Logger,
	key string,
	info *model.ReleaseInfo,
) error {
	jsonData, err := json.Marshal(info)
	if err != nil {
		log.ErrorContext(ctx, "marshal for cache failed", slog.Any("error", err))
		return fmt.Errorf("marshal release info: %w", err)
	}
	c.setCached(ctx, log, key, string(jsonData))
	return nil
}
