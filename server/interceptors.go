package server

import (
	"context"

	"github.com/GoodCodingFriends/animekai/errors"
	grpc_zap "github.com/grpc-ecosystem/go-grpc-middleware/logging/zap"
	"github.com/grpc-ecosystem/go-grpc-middleware/logging/zap/ctxzap"
	grpc_recovery "github.com/grpc-ecosystem/go-grpc-middleware/recovery"
	"github.com/morikuni/failure"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func interceptors(logger *zap.Logger) []grpc.UnaryServerInterceptor {
	return []grpc.UnaryServerInterceptor{
		grpc_zap.UnaryServerInterceptor(logger),
		grpc_recovery.UnaryServerInterceptor(),
		convertErrorToCodeUnaryServerInterceptor,
	}
}

var failureCodeToGRPCCode = map[failure.Code]codes.Code{
	errors.Canceled:         codes.Canceled,
	errors.DeadlineExceeded: codes.DeadlineExceeded,
	errors.Internal:         codes.Internal,
}

func convertErrorToCodeUnaryServerInterceptor(
	ctx context.Context,
	req interface{},
	_ *grpc.UnaryServerInfo,
	handler grpc.UnaryHandler,
) (interface{}, error) {
	res, err := handler(ctx, req)
	if err == nil {
		return res, nil
	}

	fcode, _ := failure.CodeOf(err)
	code, ok := failureCodeToGRPCCode[fcode]
	if !ok {
		code = codes.Internal
	}

	ctxzap.Extract(ctx).Error(code.String(), zap.Error(err))

	return res, status.Error(code, code.String())
}
