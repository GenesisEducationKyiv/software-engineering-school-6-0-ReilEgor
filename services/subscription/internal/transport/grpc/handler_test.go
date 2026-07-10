package grpc

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	v1 "github.com/GenesisEducationKyiv/software-engineering-school-6-0-ReilEgor/shared/infrastructure/grpc/proto/v1"

	"github.com/GenesisEducationKyiv/software-engineering-school-6-0-ReilEgor/services/subscription/internal/domain/model"
	"github.com/GenesisEducationKyiv/software-engineering-school-6-0-ReilEgor/services/subscription/internal/mocks"
)

func strPtr(s string) *string { return &s }

type handlerMockFields struct {
	userUC *mocks.UserUseCase
	repoUC *mocks.RepositoryUseCase
}

func newHandlerMockFields(t *testing.T) handlerMockFields {
	t.Helper()
	return handlerMockFields{
		userUC: mocks.NewUserUseCase(t),
		repoUC: mocks.NewRepositoryUseCase(t),
	}
}

func newTestHandler(f handlerMockFields) *SubscriptionHandler {
	return NewSubscriptionHandler(f.userUC, f.repoUC)
}

func TestSubscriptionHandler_Subscribe(t *testing.T) {
	tests := []struct {
		name     string
		req      *v1.SubscribeRequest
		setup    func(f handlerMockFields)
		wantCode codes.Code
	}{
		{
			name: "success - subscription initiated",
			req:  &v1.SubscribeRequest{Email: "user@example.com", Repository: "golang/go"},
			setup: func(f handlerMockFields) {
				f.userUC.On("Subscribe", mock.Anything, "user@example.com", "golang/go").
					Return(nil).Once()
			},
			wantCode: codes.OK,
		},
		{
			name: "error - usecase fails",
			req:  &v1.SubscribeRequest{Email: "user@example.com", Repository: "golang/go"},
			setup: func(f handlerMockFields) {
				f.userUC.On("Subscribe", mock.Anything, "user@example.com", "golang/go").
					Return(errors.New("db error")).Once()
			},
			wantCode: codes.Internal,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := newHandlerMockFields(t)
			tt.setup(f)

			resp, err := newTestHandler(f).Subscribe(context.Background(), tt.req)

			if tt.wantCode == codes.OK {
				require.NoError(t, err)
				require.NotNil(t, resp)
				require.True(t, resp.GetSuccess())
			} else {
				require.Error(t, err)
				require.Equal(t, tt.wantCode, status.Code(err))
				require.Nil(t, resp)
			}
		})
	}
}

func TestSubscriptionHandler_Unsubscribe(t *testing.T) {
	tests := []struct {
		name     string
		req      *v1.UnsubscribeRequest
		setup    func(f handlerMockFields)
		wantCode codes.Code
	}{
		{
			name:     "error - missing token",
			req:      &v1.UnsubscribeRequest{Token: ""},
			setup:    func(_ handlerMockFields) {},
			wantCode: codes.InvalidArgument,
		},
		{
			name: "success - unsubscribed",
			req:  &v1.UnsubscribeRequest{Token: "valid-token"},
			setup: func(f handlerMockFields) {
				f.userUC.On("UnsubscribeByToken", mock.Anything, "valid-token").
					Return(nil).Once()
			},
			wantCode: codes.OK,
		},
		{
			name: "error - usecase fails",
			req:  &v1.UnsubscribeRequest{Token: "valid-token"},
			setup: func(f handlerMockFields) {
				f.userUC.On("UnsubscribeByToken", mock.Anything, "valid-token").
					Return(errors.New("db error")).Once()
			},
			wantCode: codes.Internal,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := newHandlerMockFields(t)
			tt.setup(f)

			resp, err := newTestHandler(f).Unsubscribe(context.Background(), tt.req)

			if tt.wantCode == codes.OK {
				require.NoError(t, err)
				require.NotNil(t, resp)
				require.True(t, resp.GetSuccess())
			} else {
				require.Error(t, err)
				require.Equal(t, tt.wantCode, status.Code(err))
				require.Nil(t, resp)
			}
		})
	}
}

func TestSubscriptionHandler_ListSubscriptions(t *testing.T) {
	createdAt := time.Date(2026, 1, 15, 12, 0, 0, 0, time.UTC)

	tests := []struct {
		name        string
		req         *v1.ListSubscriptionsRequest
		setup       func(f handlerMockFields)
		wantCode    codes.Code
		checkResult func(t *testing.T, resp *v1.ListSubscriptionsResponse)
	}{
		{
			name: "success - maps subscriptions to proto",
			req:  &v1.ListSubscriptionsRequest{Email: "user@example.com"},
			setup: func(f handlerMockFields) {
				f.userUC.On("ListByEmail", mock.Anything, "user@example.com").
					Return([]model.Subscription{
						{
							ID:             1,
							RepositoryName: "golang/go",
							Confirmed:      true,
							LastSeenTag:    strPtr("v1.2.0"),
							CreatedAt:      createdAt,
						},
						{
							ID:             2,
							RepositoryName: "torvalds/linux",
							Confirmed:      false,
							LastSeenTag:    nil,
							CreatedAt:      createdAt,
						},
					}, nil).Once()
			},
			wantCode: codes.OK,
			checkResult: func(t *testing.T, resp *v1.ListSubscriptionsResponse) {
				require.Equal(t, int32(2), resp.GetTotal())
				require.Len(t, resp.GetSubscriptions(), 2)

				first := resp.GetSubscriptions()[0]
				require.Equal(t, int64(1), first.GetId())
				require.Equal(t, "golang/go", first.GetRepo())
				require.True(t, first.GetConfirmed())
				require.Equal(t, "v1.2.0", first.GetLastSeenTag())
				require.True(t, createdAt.Equal(first.GetCreatedAt().AsTime()))

				second := resp.GetSubscriptions()[1]
				require.Empty(t, second.GetLastSeenTag())
				require.False(t, second.GetConfirmed())
			},
		},
		{
			name: "success - empty list",
			req:  &v1.ListSubscriptionsRequest{Email: "empty@example.com"},
			setup: func(f handlerMockFields) {
				f.userUC.On("ListByEmail", mock.Anything, "empty@example.com").
					Return([]model.Subscription{}, nil).Once()
			},
			wantCode: codes.OK,
			checkResult: func(t *testing.T, resp *v1.ListSubscriptionsResponse) {
				require.Equal(t, int32(0), resp.GetTotal())
				require.Empty(t, resp.GetSubscriptions())
			},
		},
		{
			name: "error - usecase fails",
			req:  &v1.ListSubscriptionsRequest{Email: "user@example.com"},
			setup: func(f handlerMockFields) {
				f.userUC.On("ListByEmail", mock.Anything, "user@example.com").
					Return(nil, errors.New("db error")).Once()
			},
			wantCode: codes.Internal,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := newHandlerMockFields(t)
			tt.setup(f)

			resp, err := newTestHandler(f).ListSubscriptions(context.Background(), tt.req)

			if tt.wantCode == codes.OK {
				require.NoError(t, err)
				require.NotNil(t, resp)
				tt.checkResult(t, resp)
			} else {
				require.Error(t, err)
				require.Equal(t, tt.wantCode, status.Code(err))
				require.Nil(t, resp)
			}
		})
	}
}

func TestSubscriptionHandler_UpdateTag(t *testing.T) {
	tests := []struct {
		name     string
		req      *v1.UpdateTagRequest
		setup    func(f handlerMockFields)
		wantCode codes.Code
	}{
		{
			name:     "error - missing full_name",
			req:      &v1.UpdateTagRequest{FullName: "", Tag: "v1.0.0"},
			setup:    func(_ handlerMockFields) {},
			wantCode: codes.InvalidArgument,
		},
		{
			name:     "error - missing tag",
			req:      &v1.UpdateTagRequest{FullName: "golang/go", Tag: ""},
			setup:    func(_ handlerMockFields) {},
			wantCode: codes.InvalidArgument,
		},
		{
			name: "error - repository not found",
			req:  &v1.UpdateTagRequest{FullName: "golang/go", Tag: "v1.0.0"},
			setup: func(f handlerMockFields) {
				f.repoUC.On("UpdateTag", mock.Anything, "golang/go", "v1.0.0").
					Return(model.ErrRepositoryNotFound).Once()
			},
			wantCode: codes.NotFound,
		},
		{
			name: "error - usecase fails",
			req:  &v1.UpdateTagRequest{FullName: "golang/go", Tag: "v1.0.0"},
			setup: func(f handlerMockFields) {
				f.repoUC.On("UpdateTag", mock.Anything, "golang/go", "v1.0.0").
					Return(errors.New("db error")).Once()
			},
			wantCode: codes.Internal,
		},
		{
			name: "success - tag updated",
			req:  &v1.UpdateTagRequest{FullName: "golang/go", Tag: "v1.0.0"},
			setup: func(f handlerMockFields) {
				f.repoUC.On("UpdateTag", mock.Anything, "golang/go", "v1.0.0").
					Return(nil).Once()
			},
			wantCode: codes.OK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := newHandlerMockFields(t)
			tt.setup(f)

			resp, err := newTestHandler(f).UpdateTag(context.Background(), tt.req)

			if tt.wantCode == codes.OK {
				require.NoError(t, err)
				require.NotNil(t, resp)
			} else {
				require.Error(t, err)
				require.Equal(t, tt.wantCode, status.Code(err))
				require.Nil(t, resp)
			}
		})
	}
}
