package grpc

import (
	"context"
	"log/slog"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/GenesisEducationKyiv/software-engineering-school-6-0-ReilEgor/internal/domain/usecase"
	pb "github.com/GenesisEducationKyiv/software-engineering-school-6-0-ReilEgor/internal/transport/grpc/proto/v1"
)

type SubscriptionHandler struct {
	pb.UnimplementedSubscriptionServiceServer
	usecase usecase.UserUseCase
	logger  *slog.Logger
}

func NewSubscriptionHandler(uc usecase.UserUseCase) *SubscriptionHandler {
	return &SubscriptionHandler{
		usecase: uc,
		logger:  slog.With(slog.String("component", "grpc_handler")),
	}
}

func (h *SubscriptionHandler) Subscribe(ctx context.Context, req *pb.SubscribeRequest) (*pb.SubscribeResponse, error) {
	log := h.logger.With(slog.String("handler", "Subscribe"))

	email, repo := req.GetEmail(), req.GetRepository()
	log.InfoContext(ctx, "subscribe request received",
		slog.String("email", email),
		slog.String("repo", repo),
	)

	err := h.usecase.Subscribe(ctx, email, repo)
	if err != nil {
		log.ErrorContext(ctx, "failed to initiate subscription",
			slog.String("email", email),
			slog.String("repo", repo),
			slog.String("error", err.Error()),
		)
		return nil, status.Errorf(codes.Internal, "failed to initiate subscription: %v", err)
	}

	log.InfoContext(ctx, "subscription initiated",
		slog.String("email", email),
		slog.String("repo", repo),
	)
	return &pb.SubscribeResponse{
		Message: "Subscription initiated. Please check your email to confirm.",
		Success: true,
	}, nil
}

func (h *SubscriptionHandler) Unsubscribe(
	ctx context.Context,
	req *pb.UnsubscribeRequest,
) (*pb.UnsubscribeResponse, error) {
	log := h.logger.With(slog.String("handler", "Unsubscribe"))

	token := req.GetToken()
	if token == "" {
		log.WarnContext(ctx, "unsubscribe request missing token")
		return nil, status.Error(codes.InvalidArgument, "token is required")
	}

	log.InfoContext(ctx, "unsubscribe request received")

	err := h.usecase.UnsubscribeByToken(ctx, token)
	if err != nil {
		log.ErrorContext(ctx, "failed to unsubscribe", slog.String("error", err.Error()))
		return nil, status.Errorf(codes.Internal, "failed to unsubscribe: %v", err)
	}

	log.InfoContext(ctx, "unsubscribed successfully")
	return &pb.UnsubscribeResponse{
		Message: "Successfully unsubscribed",
		Success: true,
	}, nil
}

func (h *SubscriptionHandler) ListSubscriptions(
	ctx context.Context,
	req *pb.ListSubscriptionsRequest,
) (*pb.ListSubscriptionsResponse, error) {
	log := h.logger.With(slog.String("handler", "ListSubscriptions"))

	email := req.GetEmail()
	log.InfoContext(ctx, "list subscriptions request received", slog.String("email", email))

	subs, err := h.usecase.ListByEmail(ctx, email)
	if err != nil {
		log.ErrorContext(ctx, "failed to list subscriptions",
			slog.String("email", email),
			slog.String("error", err.Error()),
		)
		return nil, status.Errorf(codes.Internal, "failed to list subscriptions: %v", err)
	}

	pbSubs := make([]*pb.Subscription, 0, len(subs))
	for _, s := range subs {
		pbSubs = append(pbSubs, &pb.Subscription{
			Id:          s.ID,
			Repo:        s.RepositoryName,
			Confirmed:   s.Confirmed,
			LastSeenTag: s.LastSeenTag,
			CreatedAt:   timestamppb.New(s.CreatedAt),
		})
	}

	log.InfoContext(ctx, "list subscriptions completed",
		slog.String("email", email),
		slog.Int("count", len(pbSubs)),
	)
	return &pb.ListSubscriptionsResponse{
		Subscriptions: pbSubs,
		Total:         int32(len(pbSubs)),
	}, nil
}
