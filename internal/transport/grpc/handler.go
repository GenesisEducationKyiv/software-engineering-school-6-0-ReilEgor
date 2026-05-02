package grpc

import (
	"context"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/ReilEgor/RepoNotifier/internal/domain/usecase"
	pb "github.com/ReilEgor/RepoNotifier/internal/transport/grpc/proto/v1"
)

type SubscriptionHandler struct {
	pb.UnimplementedSubscriptionServiceServer
	usecase usecase.SubscriptionUseCase
}

func NewSubscriptionHandler(uc usecase.SubscriptionUseCase) *SubscriptionHandler {
	return &SubscriptionHandler{
		usecase: uc,
	}
}

func (h *SubscriptionHandler) Subscribe(ctx context.Context, req *pb.SubscribeRequest) (*pb.SubscribeResponse, error) {
	err := h.usecase.Subscribe(ctx, req.GetEmail(), req.GetRepository())
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to initiate subscription: %v", err)
	}

	return &pb.SubscribeResponse{
		Message: "Subscription initiated. Please check your email to confirm.",
		Success: true,
	}, nil
}

func (h *SubscriptionHandler) Unsubscribe(
	ctx context.Context,
	req *pb.UnsubscribeRequest,
) (*pb.UnsubscribeResponse, error) {
	token := req.GetToken()
	if token == "" {
		//nolint:wrapcheck // gRPC status errors shouldn't be wrapped
		return nil, status.Error(codes.InvalidArgument, "token is required")
	}

	err := h.usecase.UnsubscribeByToken(ctx, token)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to unsubscribe: %v", err)
	}

	return &pb.UnsubscribeResponse{
		Message: "Successfully unsubscribed",
		Success: true,
	}, nil
}

func (h *SubscriptionHandler) ListSubscriptions(
	ctx context.Context,
	req *pb.ListSubscriptionsRequest,
) (*pb.ListSubscriptionsResponse, error) {
	subs, err := h.usecase.ListByEmail(ctx, req.GetEmail())
	if err != nil {
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

	return &pb.ListSubscriptionsResponse{
		Subscriptions: pbSubs,
		Total:         int32(len(pbSubs)),
	}, nil
}
