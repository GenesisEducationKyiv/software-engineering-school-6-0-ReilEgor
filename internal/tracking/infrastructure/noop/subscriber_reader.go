package noop

import (
	"context"

	"github.com/GenesisEducationKyiv/software-engineering-school-6-0-ReilEgor/internal/tracking/domain/model"
)

type SubscriberReader struct{}

func NewSubscriberReader() *SubscriberReader {
	return &SubscriberReader{}
}

func (r *SubscriberReader) GetByRepoID(_ context.Context, _ int64) ([]model.Subscriber, error) {
	return []model.Subscriber{}, nil
}
