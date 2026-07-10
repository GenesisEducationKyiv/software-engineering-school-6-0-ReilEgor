package http

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"

	contracts "github.com/GenesisEducationKyiv/software-engineering-school-6-0-ReilEgor/shared/contracts"
)

type TagUpdatedPublisher struct {
	client  *http.Client
	baseURL string
	apiKey  string
	logger  *slog.Logger
}

func NewTagUpdatedPublisher(client *http.Client, baseURL, apiKey string) *TagUpdatedPublisher {
	return &TagUpdatedPublisher{
		client:  client,
		baseURL: baseURL,
		apiKey:  apiKey,
		logger:  slog.With(slog.String("component", "TagUpdatedHTTPPublisher")),
	}
}

func (p *TagUpdatedPublisher) Publish(ctx context.Context, event contracts.TagUpdatedEvent) error {
	body, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("marshal: %w", err)
	}

	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
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

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("http update tag: unexpected status %d", resp.StatusCode)
	}
	return nil
}
