package server

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/GoodCodingFriends/animekai/api"
	"github.com/golang/protobuf/proto" //nolint:staticcheck
	"github.com/grpc-ecosystem/go-grpc-middleware/logging/zap/ctxzap"
	"github.com/rs/cors"
	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// New returns a handler for statistics server.
func New(logger *zap.Logger, statisticsService api.StatisticsServer, enableCORS bool) http.Handler {
	ints := interceptors(logger)

	srv := newStatisticsServer(statisticsService)
	mux := http.NewServeMux()
	mux.Handle(endpoint(srv.GetDashboardWithName(appendGRPCStatusToHeader, ints...)))
	mux.Handle(endpoint(srv.ListWorksWithName(appendGRPCStatusToHeader, ints...)))

	if enableCORS {
		logger.Info("enable CORS")
		return cors.Default().Handler(mux)
	}

	return mux
}

func newStatisticsServer(srv api.StatisticsServer) *api.StatisticsHTTPConverter {
	return api.NewStatisticsHTTPConverter(srv)
}

func endpoint(service, method string, handlerFunc http.HandlerFunc) (string, http.HandlerFunc) {
	return fmt.Sprintf("/%s/%s", strings.ToLower(service), strings.ToLower(method)), handlerFunc
}

func appendGRPCStatusToHeader(
	ctx context.Context,
	w http.ResponseWriter,
	_ *http.Request,
	_, _ proto.Message,
	err error,
) {
	if err == nil {
		// When err is nil, body is already written.
		return
	}

	code := status.Code(err)
	if code == codes.Unknown {
		code = codes.Internal
		ctxzap.Extract(ctx).Warn("unknown gRPC code returned", zap.Error(err))
	}
	w.Header().Set("grpc-status", strconv.Itoa(int(code)))
}
