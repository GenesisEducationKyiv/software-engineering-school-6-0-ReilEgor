package github

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/GenesisEducationKyiv/software-engineering-school-6-0-ReilEgor/internal/domain/model"
	"github.com/GenesisEducationKyiv/software-engineering-school-6-0-ReilEgor/internal/domain/service"
)

type CachedGitHubClient struct {
	client service.GitHubClient
	cache  service.Cache
	logger *slog.Logger
}

func NewCachedGitHubClient(client service.GitHubClient, cache service.Cache) *CachedGitHubClient {
	return &CachedGitHubClient{
		client: client,
		cache:  cache,
		logger: slog.With(slog.String("component", "CachedGitHubClient")),
	}
}

func (c *CachedGitHubClient) RepoExists(ctx context.Context, fullName string) (bool, error) {
	const op = "CachedGitHubClient.RepoExists"
	key := "repo_exists:" + fullName

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

	if data, err := c.cache.Get(ctx, key); err == nil {
		var info model.ReleaseInfo
		if err := json.Unmarshal(data, &info); err == nil {
			return &info, nil
		}
	}

	info, err := c.client.GetLatestRelease(ctx, fullName)
	if err != nil {
		return nil, fmt.Errorf("%s: client: %w", op, err)
	}

	if jsonData, err := json.Marshal(info); err == nil {
		err = c.cache.Set(ctx, key, jsonData, time.Minute)
		if err != nil {
			return nil, fmt.Errorf("%s: cache set: %w", op, err)
		}
	}

	return info, nil
}
