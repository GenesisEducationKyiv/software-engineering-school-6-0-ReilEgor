package adapter

import (
	"context"
	"fmt"

	"github.com/GenesisEducationKyiv/software-engineering-school-6-0-ReilEgor/shared/config"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"

	pb "github.com/GenesisEducationKyiv/software-engineering-school-6-0-ReilEgor/shared/infrastructure/grpc/proto/v1"

	subModel "github.com/GenesisEducationKyiv/software-engineering-school-6-0-ReilEgor/services/subscription/domain/model"
	"github.com/GenesisEducationKyiv/software-engineering-school-6-0-ReilEgor/services/subscription/domain/repository"
)

type RepositoryUseCaseAdapter struct {
	client    pb.TrackingServiceClient
	repoStore repository.RepositoryRepository
	apiKey    string
}

func NewRepositoryUseCaseAdapter(
	client pb.TrackingServiceClient,
	repoStore repository.RepositoryRepository,
	cfg config.TrackingClientConfig,
) *RepositoryUseCaseAdapter {
	return &RepositoryUseCaseAdapter{
		client:    client,
		repoStore: repoStore,
		apiKey:    cfg.APIKey,
	}
}

func (a *RepositoryUseCaseAdapter) GetOrCreate(ctx context.Context, repoName string) (*subModel.RepositoryRef, error) {
	ctx = metadata.AppendToOutgoingContext(ctx, "x-api-key", a.apiKey)
	_, err := a.client.GetOrCreateRepository(ctx, &pb.GetOrCreateRepositoryRequest{FullName: repoName})
	if err != nil {
		return nil, translateGRPCError(err)
	}

	ref, err := a.repoStore.GetOrCreate(ctx, repoName)
	if err != nil {
		return nil, fmt.Errorf("upsert local repo: %w", err)
	}
	return ref, nil
}

func translateGRPCError(err error) error {
	s, ok := status.FromError(err)
	if !ok {
		return err
	}
	switch s.Code() {
	case codes.NotFound:
		return fmt.Errorf("%w", subModel.ErrRepositoryNotFound)
	case codes.Unavailable, codes.ResourceExhausted:
		return fmt.Errorf("%w", subModel.ErrServiceUnavailable)
	case codes.OK,
		codes.Canceled,
		codes.Unknown,
		codes.InvalidArgument,
		codes.DeadlineExceeded,
		codes.AlreadyExists,
		codes.PermissionDenied,
		codes.FailedPrecondition,
		codes.Aborted,
		codes.OutOfRange,
		codes.Unimplemented,
		codes.Internal,
		codes.DataLoss,
		codes.Unauthenticated:
		return err
	}
	return err
}
