package grpc

import (
	"context"
	"log/slog"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"

	usecase2 "github.com/GenesisEducationKyiv/software-engineering-school-6-0-ReilEgor/services/subscription/domain/usecase"
	v2 "github.com/GenesisEducationKyiv/software-engineering-school-6-0-ReilEgor/shared/infrastructure/grpc/proto/v1"
)

type SubscriptionHandler struct {
	v2.UnimplementedSubscriptionServiceServer
	userUC usecase2.UserUseCase
	repoUC usecase2.RepositoryUseCase
	logger *slog.Logger
}

func NewSubscriptionHandler(userUC usecase2.UserUseCase, repoUC usecase2.RepositoryUseCase) *SubscriptionHandler {
	return &SubscriptionHandler{
		userUC: userUC,
		repoUC: repoUC,
		logger: slog.With(slog.String("component", "grpc_handler")),
	}
}

func (h *SubscriptionHandler) Subscribe(ctx context.Context, req *v2.SubscribeRequest) (*v2.SubscribeResponse, error) {
	log := h.logger.With(slog.String("handler", "Subscribe"))

	email, repo := req.GetEmail(), req.GetRepository()
	log.InfoContext(ctx, "subscribe request received",
		slog.String("email", email),
		slog.String("repo", repo),
	)

	err := h.userUC.Subscribe(ctx, email, repo)
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
	return &v2.SubscribeResponse{
		Message: "Subscription initiated. Please check your email to confirm.",
		Success: true,
	}, nil
}

func (h *SubscriptionHandler) Unsubscribe(
	ctx context.Context,
	req *v2.UnsubscribeRequest,
) (*v2.UnsubscribeResponse, error) {
	log := h.logger.With(slog.String("handler", "Unsubscribe"))

	token := req.GetToken()
	if token == "" {
		log.WarnContext(ctx, "unsubscribe request missing token")
		return nil, status.Error(codes.InvalidArgument, "token is required")
	}

	log.InfoContext(ctx, "unsubscribe request received")

	err := h.userUC.UnsubscribeByToken(ctx, token)
	if err != nil {
		log.ErrorContext(ctx, "failed to unsubscribe", slog.String("error", err.Error()))
		return nil, status.Errorf(codes.Internal, "failed to unsubscribe: %v", err)
	}

	log.InfoContext(ctx, "unsubscribed successfully")
	return &v2.UnsubscribeResponse{
		Message: "Successfully unsubscribed",
		Success: true,
	}, nil
}

func (h *SubscriptionHandler) ListSubscriptions(
	ctx context.Context,
	req *v2.ListSubscriptionsRequest,
) (*v2.ListSubscriptionsResponse, error) {
	log := h.logger.With(slog.String("handler", "ListSubscriptions"))

	email := req.GetEmail()
	log.InfoContext(ctx, "list subscriptions request received", slog.String("email", email))

	subs, err := h.userUC.ListByEmail(ctx, email)
	if err != nil {
		log.ErrorContext(ctx, "failed to list subscriptions",
			slog.String("email", email),
			slog.String("error", err.Error()),
		)
		return nil, status.Errorf(codes.Internal, "failed to list subscriptions: %v", err)
	}

	pbSubs := make([]*v2.Subscription, 0, len(subs))
	for _, s := range subs {
		pbSubs = append(pbSubs, &v2.Subscription{
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
	return &v2.ListSubscriptionsResponse{
		Subscriptions: pbSubs,
		Total:         int32(len(pbSubs)),
	}, nil
}

func (h *SubscriptionHandler) UpdateTag(ctx context.Context, req *v2.UpdateTagRequest) (*v2.UpdateTagResponse, error) {
	if err := h.repoUC.UpdateTag(ctx, req.GetFullName(), req.GetTag()); err != nil {
		h.logger.ErrorContext(ctx, "failed to update tag",
			slog.String("repo", req.GetFullName()),
			slog.Any("error", err),
		)
		return nil, status.Errorf(codes.Internal, "update tag: %v", err)
	}
	return &v2.UpdateTagResponse{}, nil
}
