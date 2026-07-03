package grpc

import (
	"context"
	"errors"
	"log/slog"
	"time"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/GenesisEducationKyiv/software-engineering-school-6-0-ReilEgor/shared/ctxlog"
	pb "github.com/GenesisEducationKyiv/software-engineering-school-6-0-ReilEgor/shared/infrastructure/grpc/proto/v1"

	"github.com/GenesisEducationKyiv/software-engineering-school-6-0-ReilEgor/services/tracking/internal/domain/model"
	trackingDomainUsecase "github.com/GenesisEducationKyiv/software-engineering-school-6-0-ReilEgor/services/tracking/internal/domain/usecase"
)

type TrackingHandler struct {
	pb.UnimplementedTrackingServiceServer
	repoUC trackingDomainUsecase.RepositoryUseCase
}

func NewTrackingHandler(repoUC trackingDomainUsecase.RepositoryUseCase) *TrackingHandler {
	return &TrackingHandler{repoUC: repoUC}
}

func (h *TrackingHandler) GetOrCreateRepository(
	ctx context.Context,
	req *pb.GetOrCreateRepositoryRequest,
) (_ *pb.GetOrCreateRepositoryResponse, retErr error) {
	start := time.Now()
	log := ctxlog.FromCtx(ctx).With(
		slog.String("handler", "GetOrCreateRepository"),
		slog.String("repo", req.GetFullName()),
	)
	log.DebugContext(ctx, "called")
	defer func() {
		log.InfoContext(
			ctx,
			"handler completed",
			slog.Duration("duration", time.Since(start)),
			slog.Bool("ok", retErr == nil),
		)
	}()

	repo, err := h.repoUC.GetOrCreate(ctx, req.GetFullName())
	if err != nil {
		if errors.Is(err, model.ErrRepositoryNotFound) {
			return nil, status.Errorf(codes.NotFound, "repository not found: %s", req.GetFullName())
		}
		if errors.Is(err, model.ErrGitHubUnavailable) || errors.Is(err, model.ErrRateLimitExceeded) {
			return nil, status.Errorf(codes.Unavailable, "external service unavailable: %v", err)
		}
		log.ErrorContext(ctx, "failed to get or create repository", slog.String("error", err.Error()))
		return nil, status.Errorf(codes.Internal, "get or create repository: %v", err)
	}

	log.InfoContext(ctx, "repository processed", slog.Int64("id", repo.ID))
	return &pb.GetOrCreateRepositoryResponse{
		Id:          repo.ID,
		FullName:    repo.FullName,
		LastSeenTag: repo.LastSeenTag,
	}, nil
}
