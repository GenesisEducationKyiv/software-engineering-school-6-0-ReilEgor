package grpc

import (
	"context"
	"errors"
	"log/slog"
	"time"

	"github.com/GenesisEducationKyiv/software-engineering-school-6-0-ReilEgor/shared/ctxlog"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"

	v1 "github.com/GenesisEducationKyiv/software-engineering-school-6-0-ReilEgor/shared/infrastructure/grpc/proto/v1"

	"github.com/GenesisEducationKyiv/software-engineering-school-6-0-ReilEgor/services/subscription/internal/domain/model"
	"github.com/GenesisEducationKyiv/software-engineering-school-6-0-ReilEgor/services/subscription/internal/domain/usecase"
)

type SubscriptionHandler struct {
	v1.UnimplementedSubscriptionServiceServer
	userUC usecase.UserUseCase
	repoUC usecase.RepositoryUseCase
}

func NewSubscriptionHandler(userUC usecase.UserUseCase, repoUC usecase.RepositoryUseCase) *SubscriptionHandler {
	return &SubscriptionHandler{
		userUC: userUC,
		repoUC: repoUC,
	}
}

func (h *SubscriptionHandler) Subscribe(
	ctx context.Context,
	req *v1.SubscribeRequest,
) (_ *v1.SubscribeResponse, retErr error) {
	start := time.Now()
	log := ctxlog.FromCtx(ctx).With(slog.String("handler", "Subscribe"))
	log.DebugContext(ctx, "called")
	defer func() {
		log.InfoContext(
			ctx,
			"handler completed",
			slog.Duration("duration", time.Since(start)),
			slog.Bool("ok", retErr == nil),
		)
	}()

	email, repo := req.GetEmail(), req.GetRepository()
	log.InfoContext(ctx, "subscribe request received",
		slog.String("email", email),
		slog.String("repo", repo),
	)

	if err := h.userUC.Subscribe(ctx, email, repo); err != nil {
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
	return &v1.SubscribeResponse{
		Message: "Subscription initiated. Please check your email to confirm.",
		Success: true,
	}, nil
}

func (h *SubscriptionHandler) Unsubscribe(
	ctx context.Context,
	req *v1.UnsubscribeRequest,
) (_ *v1.UnsubscribeResponse, retErr error) {
	start := time.Now()
	log := ctxlog.FromCtx(ctx).With(slog.String("handler", "Unsubscribe"))
	log.DebugContext(ctx, "called")
	defer func() {
		log.InfoContext(
			ctx,
			"handler completed",
			slog.Duration("duration", time.Since(start)),
			slog.Bool("ok", retErr == nil),
		)
	}()

	token := req.GetToken()
	if token == "" {
		log.WarnContext(ctx, "unsubscribe request missing token")
		return nil, status.Error(codes.InvalidArgument, "token is required")
	}

	log.InfoContext(ctx, "unsubscribe request received")

	if err := h.userUC.UnsubscribeByToken(ctx, token); err != nil {
		log.ErrorContext(ctx, "failed to unsubscribe", slog.String("error", err.Error()))
		return nil, status.Errorf(codes.Internal, "failed to unsubscribe: %v", err)
	}

	log.InfoContext(ctx, "unsubscribed successfully")
	return &v1.UnsubscribeResponse{
		Message: "Successfully unsubscribed",
		Success: true,
	}, nil
}

func (h *SubscriptionHandler) ListSubscriptions(
	ctx context.Context,
	req *v1.ListSubscriptionsRequest,
) (_ *v1.ListSubscriptionsResponse, retErr error) {
	start := time.Now()
	log := ctxlog.FromCtx(ctx).With(slog.String("handler", "ListSubscriptions"))
	log.DebugContext(ctx, "called")
	defer func() {
		log.InfoContext(
			ctx,
			"handler completed",
			slog.Duration("duration", time.Since(start)),
			slog.Bool("ok", retErr == nil),
		)
	}()

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

	pbSubs := make([]*v1.Subscription, 0, len(subs))
	for _, s := range subs {
		var lastSeenTag string
		if s.LastSeenTag != nil {
			lastSeenTag = *s.LastSeenTag
		}
		pbSubs = append(pbSubs, &v1.Subscription{
			Id:          s.ID,
			Repo:        s.RepositoryName,
			Confirmed:   s.Confirmed,
			LastSeenTag: lastSeenTag,
			CreatedAt:   timestamppb.New(s.CreatedAt),
		})
	}

	log.InfoContext(ctx, "list subscriptions completed",
		slog.String("email", email),
		slog.Int("count", len(pbSubs)),
	)
	return &v1.ListSubscriptionsResponse{
		Subscriptions: pbSubs,
		Total:         int32(len(pbSubs)),
	}, nil
}

func (h *SubscriptionHandler) UpdateTag(
	ctx context.Context,
	req *v1.UpdateTagRequest,
) (_ *v1.UpdateTagResponse, retErr error) {
	start := time.Now()
	log := ctxlog.FromCtx(ctx).With(slog.String("handler", "UpdateTag"))
	log.DebugContext(ctx, "called",
		slog.String("repo", req.GetFullName()),
		slog.String("tag", req.GetTag()),
	)
	defer func() {
		log.InfoContext(
			ctx,
			"handler completed",
			slog.Duration("duration", time.Since(start)),
			slog.Bool("ok", retErr == nil),
		)
	}()

	if req.GetFullName() == "" {
		return nil, status.Error(codes.InvalidArgument, "full_name is required")
	}
	if req.GetTag() == "" {
		return nil, status.Error(codes.InvalidArgument, "tag is required")
	}

	if err := h.repoUC.UpdateTag(ctx, req.GetFullName(), req.GetTag()); err != nil {
		if errors.Is(err, model.ErrRepositoryNotFound) {
			log.WarnContext(ctx, "repository not found",
				slog.String("repo", req.GetFullName()),
			)
			return nil, status.Error(codes.NotFound, "repository not found")
		}
		log.ErrorContext(ctx, "failed to update tag",
			slog.String("repo", req.GetFullName()),
			slog.Any("error", err),
		)
		return nil, status.Errorf(codes.Internal, "update tag: %v", err)
	}
	return &v1.UpdateTagResponse{}, nil
}
