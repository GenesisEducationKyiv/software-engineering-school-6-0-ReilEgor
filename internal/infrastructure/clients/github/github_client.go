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

type GitHubClient struct {
	httpClient *http.Client
	logger     *slog.Logger
	cb         *gobreaker.CircuitBreaker
	apiBase    string
	token      config.GitHubTokenType
}

func NewGitHubClient(token config.GitHubTokenType) *GitHubClient {
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
		logger:     slog.With(slog.String("component", componentGithubClient)),
		cb:         gobreaker.NewCircuitBreaker(settings),
		apiBase:    githubAPIBase,
		token:      token,
	}
}

func (c *GitHubClient) RepoExists(ctx context.Context, fullName string) (bool, error) {
	const op = "GitHubClient.RepoExists"

	result, err := c.cb.Execute(func() (any, error) {
		return c.repoExistsRequest(ctx, fullName)
	})
	if err != nil {
		return false, c.handleCBError(ctx, op, err)
	}

	exists, ok := result.(bool)
	if !ok {
		return false, fmt.Errorf("%s: unexpected result type: %T", op, result)
	}

	return exists, nil
}

func (c *GitHubClient) GetLatestRelease(ctx context.Context, fullName string) (*model.ReleaseInfo, error) {
	const op = "GitHubClient.GetLatestRelease"

	result, err := c.cb.Execute(func() (any, error) {
		return c.latestReleaseRequest(ctx, fullName)
	})
	if err != nil {
		return nil, c.handleCBError(ctx, op, err)
	}

	info, ok := result.(*model.ReleaseInfo)
	if !ok {
		return nil, fmt.Errorf("%s: unexpected result type: %T", op, result)
	}

	return info, nil
}

func (c *GitHubClient) handleCBError(ctx context.Context, op string, err error) error {
	if errors.Is(err, gobreaker.ErrOpenState) || errors.Is(err, gobreaker.ErrTooManyRequests) {
		c.logger.WarnContext(ctx, "circuit breaker is open", slog.String("error", err.Error()))
		return service.ErrGitHubUnavailable
	}
	return fmt.Errorf("%s: %w", op, err)
}

func (c *GitHubClient) repoExistsRequest(ctx context.Context, fullName string) (bool, error) {
	url := fmt.Sprintf("%s/repos/%s", c.apiBase, fullName)
	resp, err := c.doRequest(ctx, http.MethodHead, url)
	if err != nil {
		c.logger.ErrorContext(ctx, "github request failed",
			slog.String("url", url),
			slog.String("error", err.Error()),
		)
		return false, err
	}
	defer func() {
		if closeErr := resp.Body.Close(); closeErr != nil {
			c.logger.WarnContext(ctx, "failed to close response body", slog.String("error", closeErr.Error()))
		}
	}()

	c.logger.DebugContext(ctx, "github response received",
		slog.Int("status", resp.StatusCode),
		slog.String("method", http.MethodHead),
		slog.String("url", url),
	)

	switch resp.StatusCode {
	case http.StatusOK:
		return true, nil
	case http.StatusNotFound:
		return false, nil
	case http.StatusForbidden:
		c.logger.WarnContext(ctx, "github rate limit exceeded", slog.String("repo", fullName))
		return false, service.ErrRateLimitExceeded
	default:
		return false, fmt.Errorf("%w: %s", ErrUnexpectedStatus, resp.Status)
	}
}

func (c *GitHubClient) latestReleaseRequest(
	ctx context.Context,
	fullName string,
) (*model.ReleaseInfo, error) {
	url := fmt.Sprintf("%s/repos/%s/releases/latest", c.apiBase, fullName)
	resp, err := c.doRequest(ctx, http.MethodGet, url)
	if err != nil {
		return nil, err
	}
	defer func() {
		if closeErr := resp.Body.Close(); closeErr != nil {
			c.logger.WarnContext(ctx, "failed to close response body", slog.String("error", closeErr.Error()))
		}
	}()

	if resp.StatusCode != http.StatusOK {
		if resp.StatusCode == http.StatusNotFound {
			return nil, service.ErrReleaseNotFound
		}
		return nil, fmt.Errorf("%w: %s", ErrUnexpectedStatus, resp.Status)
	}

	var info model.ReleaseInfo
	if err := json.NewDecoder(resp.Body).Decode(&info); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}
	return &info, nil
}

func (c *GitHubClient) doRequest(ctx context.Context, method, url string) (*http.Response, error) {
	req, err := http.NewRequestWithContext(ctx, method, url, nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("X-GitHub-Api-Version", githubAPIVersion)
	req.Header.Set("User-Agent", userAgent)
	if c.token != "" {
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", c.token))
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("do request: %w", err)
	}

	return resp, nil
}
