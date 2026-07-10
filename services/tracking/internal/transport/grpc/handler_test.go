package grpc

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	pb "github.com/GenesisEducationKyiv/software-engineering-school-6-0-ReilEgor/shared/infrastructure/grpc/proto/v1"

	"github.com/GenesisEducationKyiv/software-engineering-school-6-0-ReilEgor/services/tracking/internal/domain/model"
	"github.com/GenesisEducationKyiv/software-engineering-school-6-0-ReilEgor/services/tracking/internal/mocks"
)

func newTestTrackingHandler(repoUC *mocks.RepositoryUseCase) *TrackingHandler {
	return NewTrackingHandler(repoUC)
}

func TestTrackingHandler_GetOrCreateRepository(t *testing.T) {
	tests := []struct {
		name        string
		req         *pb.GetOrCreateRepositoryRequest
		setup       func(repoUC *mocks.RepositoryUseCase)
		wantCode    codes.Code
		checkResult func(t *testing.T, resp *pb.GetOrCreateRepositoryResponse)
	}{
		{
			name: "success - repository returned",
			req:  &pb.GetOrCreateRepositoryRequest{FullName: "golang/go"},
			setup: func(repoUC *mocks.RepositoryUseCase) {
				repoUC.On("GetOrCreate", mock.Anything, "golang/go").
					Return(&model.Repository{ID: 1, FullName: "golang/go", LastSeenTag: "v1.22.0"}, nil).Once()
			},
			wantCode: codes.OK,
			checkResult: func(t *testing.T, resp *pb.GetOrCreateRepositoryResponse) {
				require.Equal(t, int64(1), resp.GetId())
				require.Equal(t, "golang/go", resp.GetFullName())
				require.Equal(t, "v1.22.0", resp.GetLastSeenTag())
			},
		},
		{
			name: "error - repository not found",
			req:  &pb.GetOrCreateRepositoryRequest{FullName: "unknown/repo"},
			setup: func(repoUC *mocks.RepositoryUseCase) {
				repoUC.On("GetOrCreate", mock.Anything, "unknown/repo").
					Return((*model.Repository)(nil), model.ErrRepositoryNotFound).Once()
			},
			wantCode: codes.NotFound,
		},
		{
			name: "error - github unavailable",
			req:  &pb.GetOrCreateRepositoryRequest{FullName: "golang/go"},
			setup: func(repoUC *mocks.RepositoryUseCase) {
				repoUC.On("GetOrCreate", mock.Anything, "golang/go").
					Return((*model.Repository)(nil), model.ErrGitHubUnavailable).Once()
			},
			wantCode: codes.Unavailable,
		},
		{
			name: "error - rate limit exceeded",
			req:  &pb.GetOrCreateRepositoryRequest{FullName: "golang/go"},
			setup: func(repoUC *mocks.RepositoryUseCase) {
				repoUC.On("GetOrCreate", mock.Anything, "golang/go").
					Return((*model.Repository)(nil), model.ErrRateLimitExceeded).Once()
			},
			wantCode: codes.Unavailable,
		},
		{
			name: "error - unexpected failure maps to internal",
			req:  &pb.GetOrCreateRepositoryRequest{FullName: "golang/go"},
			setup: func(repoUC *mocks.RepositoryUseCase) {
				repoUC.On("GetOrCreate", mock.Anything, "golang/go").
					Return((*model.Repository)(nil), errors.New("db error")).Once()
			},
			wantCode: codes.Internal,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repoUC := mocks.NewRepositoryUseCase(t)
			tt.setup(repoUC)

			resp, err := newTestTrackingHandler(repoUC).GetOrCreateRepository(context.Background(), tt.req)

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
