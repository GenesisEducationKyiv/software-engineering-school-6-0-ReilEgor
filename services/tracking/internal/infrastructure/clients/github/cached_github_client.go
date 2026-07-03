package github

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/GenesisEducationKyiv/software-engineering-school-6-0-ReilEgor/shared/ctxlog"
	sharedcache "github.com/GenesisEducationKyiv/software-engineering-school-6-0-ReilEgor/shared/domain/cache"

	"github.com/GenesisEducationKyiv/software-engineering-school-6-0-ReilEgor/services/tracking/internal/domain/model"
	"github.com/GenesisEducationKyiv/software-engineering-school-6-0-ReilEgor/services/tracking/internal/domain/service"
)

const componentCachedGithubClient = "CachedGitHubClient"

type CachedGitHubClient struct {
	client service.GitHubClient
	cache  sharedcache.Cache
}

func NewCachedGitHubClient(client service.GitHubClient, cache sharedcache.Cache) *CachedGitHubClient {
	return &CachedGitHubClient{
		client: client,
		cache:  cache,
	}
}

func (c *CachedGitHubClient) log(ctx context.Context) *slog.Logger {
	return ctxlog.FromCtx(ctx).With(slog.String("component", componentCachedGithubClient))
}

func (c *CachedGitHubClient) RepoExists(ctx context.Context, fullName string) (bool, error) {
	const op = "CachedGitHubClient.RepoExists"
	key := "repo_exists:" + fullName
	c.log(ctx).DebugContext(ctx, "called", slog.String("op", op), slog.String("repo", fullName))

	if data, err := c.cache.Get(ctx, key); err == nil {
		return string(data) == "true", nil
	}

	exists, err := c.client.RepoExists(ctx, fullName)
	if err != nil {
		return false, fmt.Errorf("%s: client: %w", op, err)
	}

	val := []byte("false")
	if exists {
		val = []byte("true")
	}
	err = c.cache.Set(ctx, key, val, time.Minute)
	if err != nil {
		return false, fmt.Errorf("%s: cache set: %w", op, err)
	}
	return exists, nil
}

func (c *CachedGitHubClient) GetLatestRelease(ctx context.Context, fullName string) (*model.ReleaseInfo, error) {
	const op = "CachedGitHubClient.GetLatestRelease"
	key := "release:" + fullName
	log := c.log(ctx).With(slog.String("op", op), slog.String("repo", fullName))
	log.DebugContext(ctx, "called")

	data, err := c.cache.Get(ctx, key)
	if err == nil {
		var info model.ReleaseInfo
		if err = json.Unmarshal(data, &info); err == nil {
			return &info, nil
		}

		log.WarnContext(ctx, "cache unmarshal failed, falling back to API",
			slog.String("key", key),
			slog.String("error", err.Error()),
		)
	}

	info, err := c.client.GetLatestRelease(ctx, fullName)
	if err != nil {
		return nil, fmt.Errorf("%s: client: %w", op, err)
	}

	if jsonData, err := json.Marshal(info); err == nil {
		err = c.cache.Set(ctx, key, jsonData, 5*time.Minute)
		if err != nil {
			return nil, fmt.Errorf("%s: cache set: %w", op, err)
		}
	}

	return info, nil
}
