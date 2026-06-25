package http

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	sharedModel "github.com/GenesisEducationKyiv/software-engineering-school-6-0-ReilEgor/shared/domain/model"
	nethttp "net/http"
)

type TagUpdatedPublisher struct {
	client  *nethttp.Client
	baseURL string
	apiKey  string
	logger  *slog.Logger
}

func NewTagUpdatedPublisher(client *nethttp.Client, baseURL, apiKey string) *TagUpdatedPublisher {
	return &TagUpdatedPublisher{
		client:  client,
		baseURL: baseURL,
		apiKey:  apiKey,
		logger:  slog.With(slog.String("component", "TagUpdatedHTTPPublisher")),
	}
}

func (p *TagUpdatedPublisher) Publish(ctx context.Context, event sharedModel.TagUpdatedEvent) error {
	body, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("marshal: %w", err)
	}

	req, err := nethttp.NewRequestWithContext(
		ctx,
		nethttp.MethodPost,
		p.baseURL+"/internal/repositories/tag",
		bytes.NewReader(body),
	)
	if err != nil {
		return fmt.Errorf("new request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-API-Key", p.apiKey)

	resp, err := p.client.Do(req)
	if err != nil {
		return fmt.Errorf("http update tag: %w", err)
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			p.logger.Warn("failed to close response body", slog.Any("error", err))
		}
	}()

	if resp.StatusCode != nethttp.StatusOK {
		return fmt.Errorf("http update tag: unexpected status %d", resp.StatusCode)
	}
	return nil
}
