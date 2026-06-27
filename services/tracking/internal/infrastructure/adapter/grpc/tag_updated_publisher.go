package grpc

import (
	"context"
	"fmt"

	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"

	contracts "github.com/GenesisEducationKyiv/software-engineering-school-6-0-ReilEgor/shared/contracts"
	v2 "github.com/GenesisEducationKyiv/software-engineering-school-6-0-ReilEgor/shared/infrastructure/grpc/proto/v1"
)

type TagUpdatedPublisher struct {
	client v2.SubscriptionServiceClient
	apiKey string
}

func NewTagUpdatedPublisher(conn *grpc.ClientConn, apiKey string) *TagUpdatedPublisher {
	return &TagUpdatedPublisher{
		client: v2.NewSubscriptionServiceClient(conn),
		apiKey: apiKey,
	}
}

func (p *TagUpdatedPublisher) Publish(ctx context.Context, event contracts.TagUpdatedEvent) error {
	ctx = metadata.AppendToOutgoingContext(ctx, "x-api-key", p.apiKey)
	_, err := p.client.UpdateTag(ctx, &v2.UpdateTagRequest{
		FullName: event.FullName,
		Tag:      event.Tag,
	})
	if err != nil {
		return fmt.Errorf("grpc update tag: %w", err)
	}
	return nil
}
