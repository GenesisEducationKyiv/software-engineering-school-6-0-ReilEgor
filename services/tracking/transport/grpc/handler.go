package grpc

import (
	"context"
	"errors"
	"log/slog"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	trackingModel "github.com/GenesisEducationKyiv/software-engineering-school-6-0-ReilEgor/services/tracking/domain/model"
	trackingDomainUsecase "github.com/GenesisEducationKyiv/software-engineering-school-6-0-ReilEgor/services/tracking/domain/usecase"
	pb "github.com/GenesisEducationKyiv/software-engineering-school-6-0-ReilEgor/shared/infrastructure/grpc/proto/v1"
)

type TrackingHandler struct {
	pb.UnimplementedTrackingServiceServer
	repoUC trackingDomainUsecase.RepositoryUseCase
	logger *slog.Logger
}

func NewTrackingHandler(repoUC trackingDomainUsecase.RepositoryUseCase) *TrackingHandler {
	return &TrackingHandler{
		repoUC: repoUC,
		logger: slog.With(slog.String("component", "tracking_grpc_handler")),
	}
}

func (h *TrackingHandler) GetOrCreateRepository(
	ctx context.Context,
	req *pb.GetOrCreateRepositoryRequest,
) (*pb.GetOrCreateRepositoryResponse, error) {
	log := h.logger.With(
		slog.String("handler", "GetOrCreateRepository"),
		slog.String("repo", req.GetFullName()),
	)
	log.InfoContext(ctx, "request received")

	repo, err := h.repoUC.GetOrCreate(ctx, req.GetFullName())
	if err != nil {
		if errors.Is(err, trackingModel.ErrRepositoryNotFound) {
			return nil, status.Errorf(codes.NotFound, "repository not found: %s", req.GetFullName())
		}
		if errors.Is(err, trackingModel.ErrGitHubUnavailable) || errors.Is(err, trackingModel.ErrRateLimitExceeded) {
			return nil, status.Errorf(codes.Unavailable, "external service unavailable: %v", err)
		}
		log.ErrorContext(ctx, "failed to get or create repository", slog.Any("error", err))
		return nil, status.Errorf(codes.Internal, "get or create repository: %v", err)
	}

	log.InfoContext(ctx, "repository processed", slog.Int64("id", repo.ID))
	return &pb.GetOrCreateRepositoryResponse{
		Id:       repo.ID,
		FullName: repo.FullName,
	}, nil
}
