package grpc

import (
	"context"
	"fmt"

	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"

	sharedModel "github.com/GenesisEducationKyiv/software-engineering-school-6-0-ReilEgor/shared/domain/model"
	v1 "github.com/GenesisEducationKyiv/software-engineering-school-6-0-ReilEgor/shared/grpc/proto/v1"
)

type TagUpdatedPublisher struct {
	client v1.SubscriptionServiceClient
	apiKey string
}

func NewTagUpdatedPublisher(conn *grpc.ClientConn, apiKey string) *TagUpdatedPublisher {
	return &TagUpdatedPublisher{
		client: v1.NewSubscriptionServiceClient(conn),
		apiKey: apiKey,
	}
}

func (p *TagUpdatedPublisher) Publish(ctx context.Context, event sharedModel.TagUpdatedEvent) error {
	ctx = metadata.AppendToOutgoingContext(ctx, "x-api-key", p.apiKey)
	_, err := p.client.UpdateTag(ctx, &v1.UpdateTagRequest{
		FullName: event.FullName,
		Tag:      event.Tag,
	})
	if err != nil {
		return fmt.Errorf("grpc update tag: %w", err)
	}
	return nil
}
