package adapter

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/GenesisEducationKyiv/software-engineering-school-6-0-ReilEgor/shared/config"
	"github.com/GenesisEducationKyiv/software-engineering-school-6-0-ReilEgor/shared/ctxlog"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"

	pb "github.com/GenesisEducationKyiv/software-engineering-school-6-0-ReilEgor/shared/infrastructure/grpc/proto/v1"

	"github.com/GenesisEducationKyiv/software-engineering-school-6-0-ReilEgor/services/subscription/internal/domain/model"
	"github.com/GenesisEducationKyiv/software-engineering-school-6-0-ReilEgor/services/subscription/internal/domain/repository"
)

const componentRepositoryUseCaseAdapter = "RepositoryUseCaseAdapter"

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

func (a *RepositoryUseCaseAdapter) GetOrCreate(ctx context.Context, repoName string) (*model.RepositoryRef, error) {
	const op = "RepositoryUseCaseAdapter.GetOrCreate"
	ctxlog.FromCtx(ctx).With(slog.String("component", componentRepositoryUseCaseAdapter)).
		DebugContext(ctx, "called", slog.String("op", op), slog.String("repo", repoName))

	ctx = metadata.AppendToOutgoingContext(ctx, "x-api-key", a.apiKey)
	if reqID := ctxlog.RequestID(ctx); reqID != "" {
		ctx = metadata.AppendToOutgoingContext(ctx, "x-request-id", reqID)
	}
	resp, err := a.client.GetOrCreateRepository(ctx, &pb.GetOrCreateRepositoryRequest{FullName: repoName})
	if err != nil {
		return nil, translateGRPCError(err)
	}

	ref, err := a.repoStore.GetOrCreate(ctx, repoName, resp.GetLastSeenTag())
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
		return fmt.Errorf("%w", model.ErrRepositoryNotFound)
	case codes.Unavailable, codes.ResourceExhausted:
		return fmt.Errorf("%w", model.ErrServiceUnavailable)
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
